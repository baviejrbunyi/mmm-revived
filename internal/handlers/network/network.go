package network

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/database"
	"mmm-revived/internal/handlers/friend"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
	"mmm-revived/internal/wire"
)

func network[T any](s service, decode func(protocol.Call) (T, error), handle func(p *server.Player, request T, now int64) error) rpc.Endpoint[*server.Player] {
	return rpc.Bind(decode, func(p *server.Player, request T) error {
		now := time.Now().Unix()
		if s.Store.Rules == nil {
			return fmt.Errorf("%w: network rules unavailable", account.ErrProgression)
		}
		p.Account.PrepareNetwork(s.Store.Rules.Network, now)
		return handle(p, request, now)
	})
}

func sendWorld[T any](p *server.Player, method string, value T) error {
	data, err := json.Marshal(struct {
		Code int `json:"error_code"`
		Data T   `json:"data"`
	}{0, value})
	if err != nil {
		return err
	}
	return p.Send(method, string(data))
}

func (s service) claimSupport(p *server.Player) error {
	var amount int64
	err := s.Store.Update(p.Account, func(tx *sql.Tx) error {
		var err error
		amount, err = s.Store.ClaimSupport(p.Account, tx)
		return err
	})
	if err != nil {
		return err
	}
	if err = s.Sync(p, "itemComp"); err != nil {
		return err
	}
	if err = friend.Sync(s.Service, p.Account.UID); err != nil {
		return err
	}
	return p.Send("notifyGetSupportHeroReward", 1, amount)
}

func (s service) claimNetwork(p *server.Player, claim func() (map[model.ItemID]int64, error)) error {
	before := *p.Account
	var quantities map[model.ItemID]int64
	err := s.Store.Update(p.Account, func(*sql.Tx) error {
		var err error
		quantities, err = claim()
		return err
	})
	if err != nil {
		return err
	}
	if err = s.Sync(p, "itemComp", "heroComp", "taskComp"); err != nil {
		return err
	}
	if err = s.syncNetwork(p); err != nil {
		return err
	}
	return p.ShowRewards(s.ItemRewards(before, p.Account, quantities))
}

func (s service) personalRanks(p *server.Player, q RankQuery, guild bool, now int64) error {
	entries, err := s.Store.PersonalRanks(p.Account.UID, model.RankMode(q.Mode), q.Rank, guild)
	if err != nil {
		return err
	}
	encoded := make(map[model.UserID]string, len(entries))
	for uid, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		encoded[uid] = string(data)
	}
	scope := 1
	if guild {
		scope = 2
	}
	err = s.Store.Update(p.Account, func(*sql.Tx) error {
		if p.Account.Network.RankCooldowns == nil {
			p.Account.Network.RankCooldowns = map[string]int64{}
		}
		p.Account.Network.RankCooldowns[strconv.Itoa(scope)+"-"+q.Mode+":"+strconv.Itoa(q.Rank)] = now
		return nil
	})
	if err != nil {
		return err
	}
	if err = p.Update([]string{"historyComp", "rankCdRecord"}, p.Account.Network.RankCooldowns); err != nil {
		return err
	}
	return p.Send("onGetPersonalRank", scope, q.Mode, q.Rank, encoded)
}

func (s service) syncNetwork(p *server.Player) error {
	profile := p.Account.NetworkSnapshot(time.Now().Unix())
	if s.Store.Rules != nil {
		rule := s.Store.Rules.Network.Activities[p.Account.Network.Activity]
		profile.Environments = slices.Clone(rule.Environments)
	}
	ctx := context.Background()
	var err error
	profile.Battles, err = s.Store.Database.Queries.ListAsyncRecords(ctx, database.ListAsyncRecordsParams{UID: p.Account.UID, Friend: 0})
	if err != nil {
		return err
	}
	profile.FriendGames, err = s.Store.Database.Queries.ListAsyncRecords(ctx, database.ListAsyncRecordsParams{UID: p.Account.UID, Friend: 1})
	if err != nil {
		return err
	}
	return p.Update([]string{"pvpComp"}, profile)
}

func (s service) networkProfiles(ids []model.UserID) (map[model.UserID]account.PublicProfile, error) {
	profiles, err := s.Store.NetworkProfiles(ids)
	if err != nil {
		return nil, err
	}
	for _, uid := range ids {
		if live := s.Online(uid); live != nil {
			entry := profiles[uid]
			entry.Account = *live.Account
			profiles[uid] = entry
		}
	}
	return profiles, nil
}

func supportHeroData(a *account.Account, id model.HeroID) (map[string]any, error) {
	hero := a.Heroes[id]
	fields, err := wire.StructFields(hero)
	if err != nil {
		return nil, err
	}
	artifacts := map[model.InstanceID]model.Artifact{}
	highlights := map[model.InstanceID]model.Highlight{}
	for _, instance := range hero.Artifacts {
		if value, ok := a.Artifacts[instance]; ok {
			artifacts[instance] = value
		}
	}
	for _, instance := range hero.Highlights {
		if value, ok := a.Highlights[instance]; ok {
			highlights[instance] = value
		}
	}
	fields["artifacts"], fields["highlights"], fields["bondAttr"] = artifacts, highlights, model.Empty{}
	return fields, nil
}

func SupportHeroPayload(a *account.Account, id model.HeroID) (string, error) {
	fields, err := supportHeroData(a, id)
	if err != nil {
		return "", err
	}
	return server.EncodePayload(fields)
}

func (s service) recommendSupport(p *server.Player, now int64) error {
	ctx := context.Background()
	ids, err := s.Store.Database.Queries.ListSupportCandidates(ctx, p.Account.UID)
	if err != nil {
		return err
	}
	profiles, err := s.networkProfiles(ids)
	if err != nil {
		return err
	}
	result := map[string]string{}
	for _, uid := range ids {
		profile := profiles[uid].Account
		for _, id := range profile.Profile.SupportHeroes {
			hero, owned := profile.Heroes[id]
			if !owned || hero.Level > p.Account.Level+10 {
				continue
			}
			encoded, err := SupportHeroPayload(&profile, id)
			if err != nil {
				return err
			}
			result[strconv.FormatUint(uint64(uid), 10)+":"+strconv.Itoa(int(id))] = encoded
		}
	}
	if err = s.Store.Database.Queries.SetSupportRefresh(ctx, database.SetSupportRefreshParams{UID: p.Account.UID, Refreshed: now}); err != nil {
		return err
	}
	if err = friend.Sync(s.Service, p.Account.UID); err != nil {
		return err
	}
	return p.Send("notifyRecommendSupportHeroes", result)
}

func (s service) chooseSupport(p *server.Player, r Support, guildWar bool) error {
	if p.State != server.Deployment {
		return fmt.Errorf("%w: support cannot change during combat", account.ErrProgression)
	}
	if r.Owner == 0 {
		if err := s.Store.Update(p.Account, func(*sql.Tx) error {
			p.Account.Network.Support.Owner, p.Account.Network.Support.Hero = 0, 0
			return nil
		}); err != nil {
			return err
		}
		return server.Acknowledge(p)
	}
	if r.Owner == p.Account.UID {
		return fmt.Errorf("%w: cannot borrow own support hero", account.ErrProgression)
	}
	profiles, err := s.networkProfiles([]model.UserID{r.Owner})
	if err != nil {
		return err
	}
	profile, exists := profiles[r.Owner]
	hero, owned := profile.Account.Heroes[r.Hero]
	if !exists || !owned || !slices.Contains(profile.Account.Profile.SupportHeroes, r.Hero) || hero.Level > p.Account.Level+10 || guildWar && hero.Level-p.Account.Level >= 10 {
		return fmt.Errorf("%w: support hero unavailable", account.ErrProgression)
	}
	if !guildWar {
		if err = s.Store.SupportAvailable(p.Account, r.Owner, time.Now().Unix()); err != nil {
			return err
		}
	}
	if guildWar {
		if p.Account.Network.Pending == nil || p.Account.Network.Pending.Guild == 0 {
			return fmt.Errorf("%w: guild support requires a guild war deployment", account.ErrProgression)
		}
		guild, err := s.Store.Social.GuildSnapshot(p.Account.UID)
		if err != nil {
			return err
		}
		if guild.ID == 0 || guild.ID != profile.Guild {
			return fmt.Errorf("%w: guild support requires same guild", account.ErrProgression)
		}
	}
	if err = s.Store.Update(p.Account, func(*sql.Tx) error {
		p.Account.Network.Support.Owner, p.Account.Network.Support.Hero, p.Account.Network.Support.Guild = r.Owner, r.Hero, guildWar
		return nil
	}); err != nil {
		return err
	}
	data, err := supportHeroData(&profile.Account, r.Hero)
	if err != nil {
		return err
	}
	return p.Send("assistantActorChange", r.Owner, r.Hero, data)
}

func (s service) asyncFriends(p *server.Player, target model.UserID) error {
	state, err := s.Store.Social.Snapshot(p.Account.UID)
	if err != nil {
		return err
	}
	ids := make([]model.UserID, 0, len(state.Friends))
	if target != 0 {
		if _, ok := state.Friends[target]; !ok {
			return fmt.Errorf("%w: challenge target is not a friend", account.ErrProgression)
		}
		ids = append(ids, target)
	} else {
		ids = slices.AppendSeq(ids, maps.Keys(state.Friends))
	}
	profiles, err := s.networkProfiles(ids)
	if err != nil {
		return err
	}
	targets := map[model.UserID]model.AsyncTarget{}
	for uid, entry := range profiles {
		if len(entry.Account.Network.Defense) == 0 {
			continue
		}
		defense, err := server.EncodePayload(entry.Account.NetworkDefense(s.Store.Rules))
		if err != nil {
			return err
		}
		targets[uid] = model.AsyncTarget{UID: int64(uid), Score: entry.Account.Network.Score, Level: entry.Account.Level, Defense: defense, Shown: defense}
	}
	if target != 0 {
		if _, ok := targets[target]; !ok {
			return fmt.Errorf("%w: friend has no defensive team", account.ErrProgression)
		}
	}
	if err = s.Store.Update(p.Account, func(*sql.Tx) error {
		if target == 0 {
			p.Account.Network.Friends = targets
		} else {
			maps.Copy(p.Account.Network.Friends, targets)
		}
		return nil
	}); err != nil {
		return err
	}
	if err = s.syncNetwork(p); err != nil {
		return err
	}
	if target != 0 {
		return p.Send("onGetSpecifyFriendPvpInfo", target)
	}
	return p.Send("onGetFriendsPvpInfoEnd")
}
