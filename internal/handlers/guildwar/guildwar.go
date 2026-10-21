package guildwar

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"math"
	"slices"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/calendar"
	"mmm-revived/internal/database"
	"mmm-revived/internal/handlers/battle"
	"mmm-revived/internal/handlers/guild"
	"mmm-revived/internal/handlers/network"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
	"mmm-revived/internal/social"
)

type warMember struct {
	Day        int64         `wire:"day"`
	Tickets    int           `wire:"ticket"`
	Score      int64         `wire:"score"`
	Assists    int           `wire:"assist"`
	Finished   map[int][]int `wire:"finishAreaGroup"`
	Unlocked   map[int]int   `wire:"areaUnlockSet"`
	UpdateTime map[int]int64 `wire:"updateTime"`
	AreaScore  map[int]int64 `wire:"areaScore"`
	AreaAssist map[int]int   `wire:"areaAssist"`
}

type warHistory struct {
	UID            model.UserID            `wire:"uid"`
	TeamLevel      int                     `wire:"teamLevel"`
	Duration       int64                   `wire:"useTime"`
	Timestamp      int64                   `wire:"timestamp"`
	Heroes         map[model.HeroID]string `wire:"heroData"`
	Best           bool                    `wire:"best"`
	Assist         model.HeroID            `wire:"assistHero"`
	AssistUID      model.UserID            `wire:"assistUid"`
	AssistData     string                  `wire:"assistHeroData"`
	CompositeScore int64                   `wire:"compositeScore"`
}

type warGroupInfo struct {
	Best *warHistory
}

func (g warGroupInfo) WireValue() any {
	if g.Best == nil {
		return map[string]any{"bestRecord": model.Empty{}}
	}
	return map[string]any{"bestRecord": g.Best}
}

func historyWire(value model.GuildWarHistory) warHistory {
	return warHistory{UID: value.UID, TeamLevel: value.TeamLevel, Duration: value.Duration, Timestamp: value.Timestamp, Heroes: value.Heroes, Best: value.Best, Assist: value.Assist, AssistUID: value.AssistUID, AssistData: value.AssistData, CompositeScore: max(1, math.MaxInt32-value.Duration)}
}

func histories(values []model.GuildWarHistory) []warHistory {
	result := make([]warHistory, len(values))
	for i, value := range values {
		result[i] = historyWire(value)
	}
	return result
}

func warGeneral(state model.GuildWarState, uid model.UserID) model.GuildWarGeneral {
	var score int64
	for _, member := range state.Members {
		score += member.Score
	}
	return model.GuildWarGeneral{Areas: state.Areas, Blocked: state.Blocked, Endurance: state.Endurance, Killed: state.Killed, Member: state.Members[uid], Score: score}
}

type warContext struct {
	war   int
	phase int
	guild model.GuildID
	state model.GuildWarState
}

func (s service) currentMembers(id model.GuildID) ([]model.UserID, error) {
	rows, err := s.Store.Database.Queries.ListGuildMembers(context.Background(), id)
	if err != nil {
		return nil, err
	}
	result := make([]model.UserID, len(rows))
	for i, row := range rows {
		result[i] = row.UID
	}
	return result, nil
}

func (s service) warMembers(id model.GuildID, war int, members map[model.UserID]model.GuildWarMember) (map[model.UserID]warMember, error) {
	current, err := s.currentMembers(id)
	if err != nil {
		return nil, err
	}
	progress, err := s.Store.LoadGuildWarProgress(id, war)
	if err != nil {
		return nil, err
	}
	result := make(map[model.UserID]warMember, len(current))
	for _, uid := range current {
		member, ok := members[uid]
		if !ok {
			continue
		}
		entry := progress[uid]
		value := warMember{Day: member.Day, Tickets: member.Tickets, Score: member.Score, Assists: member.Assists, Finished: member.Finished, Unlocked: member.Unlocked, UpdateTime: map[int]int64{}, AreaScore: map[int]int64{}, AreaAssist: map[int]int{}}
		for area, updated := range entry.UpdateTime {
			value.UpdateTime[area] = calendar.ClientTime(updated)
		}
		maps.Copy(value.AreaScore, entry.Score)
		maps.Copy(value.AreaAssist, entry.Assist)
		result[uid] = value
	}
	return result, nil
}

func (s service) warGroup(respond func(*server.Player, WarLevel, *warContext) error) rpc.Endpoint[*server.Player] {
	return network.Bind(s.Service, DecodeWarLevel, func(p *server.Player, level WarLevel, now int64) error {
		return s.guildWar(p, now, func(_ *sql.Tx, w *warContext) error {
			if _, ok := s.Store.Rules.Network.WarSet(w.war, level.Area, level.Level); !ok {
				return fmt.Errorf("%w: invalid guild war group", account.ErrProgression)
			}
			return nil
		}, func(w *warContext) error { return respond(p, level, w) })
	})
}

func (s service) warAssist(p *server.Player, level WarLevel, w *warContext) error {
	current, err := s.currentMembers(w.guild)
	if err != nil {
		return err
	}
	ids := make([]model.UserID, 0, len(current))
	for _, uid := range current {
		if uid != p.Account.UID {
			ids = append(ids, uid)
		}
	}
	profiles, err := network.Profiles(s.Service, ids)
	if err != nil {
		return err
	}
	result := []model.GuildWarAssist{}
	for uid, profile := range profiles {
		for _, id := range profile.Account.Profile.SupportHeroes {
			if _, owned := profile.Account.Heroes[id]; !owned {
				continue
			}
			data, err := network.SupportHeroPayload(&profile.Account, id)
			if err != nil {
				return err
			}
			result = append(result, model.GuildWarAssist{Member: uid, Hero: id, Data: data, Score: w.state.Members[uid].Score})
		}
	}
	return p.Send("onGetGuildWarAssistList", level.Level, result)
}

func (s service) publishPhase(p *server.Player, war, phase int) error {
	if err := p.Update([]string{"guildComp", "warId"}, war); err != nil {
		return err
	}
	return p.Update([]string{"guildComp", "warPhase"}, phase)
}

func (s service) guildWar(p *server.Player, now int64, mutate func(*sql.Tx, *warContext) error, respond func(*warContext) error) error {
	w := &warContext{}
	w.war, w.phase = social.CurrentWar(now)
	if w.war == 0 {
		if err := s.publishPhase(p, 0, social.WarEmpty); err != nil {
			return err
		}
		return fmt.Errorf("%w: guild war is not active", account.ErrProgression)
	}
	err := s.Store.Update(p.Account, func(tx *sql.Tx) error {
		var err error
		if w.guild, w.state, err = s.Store.LoadGuildWar(tx, p.Account.UID, w.war, now); err != nil {
			return err
		}
		if mutate != nil {
			if err = mutate(tx, w); err != nil {
				return err
			}
		}
		return s.Store.SaveGuildWar(tx, w.guild, w.war, w.state)
	})
	if err != nil {
		return err
	}
	if err = s.publishPhase(p, w.war, w.phase); err != nil {
		return err
	}
	return respond(w)
}

func (s service) finishGuildWarBattle(p *server.Player, r battle.VerifyBattle) (bool, error) {
	pending := p.Account.Network.Pending
	if pending == nil || pending.Battle != r.Battle || pending.Guild == 0 {
		return false, nil
	}
	now := time.Now().Unix()
	before := *p.Account
	support := p.Account.Network.Support
	var assistant *account.PublicProfile
	if r.Win && r.Result.Extra.UseAssist && support.Guild && support.Owner != 0 && support.Owner != p.Account.UID {
		profiles, err := network.Profiles(s.Service, []model.UserID{support.Owner})
		if err != nil {
			return true, err
		}
		if profile, ok := profiles[support.Owner]; ok && profile.Guild == pending.Guild {
			assistant = &profile
		}
	}
	var state model.GuildWarState
	var quantities map[model.ItemID]int64
	killed := false
	err := s.Store.Update(p.Account, func(tx *sql.Tx) error {
		guildID, current, err := s.Store.LoadGuildWar(tx, p.Account.UID, pending.War, now)
		if err != nil {
			return err
		}
		if guildID != pending.Guild || pending.Started == 0 {
			return fmt.Errorf("%w: guild war battle ownership mismatch", account.ErrProgression)
		}
		state = current
		if r.Win {
			level := s.Store.Rules.Network.WarLevels[pending.Level]
			member := state.Members[p.Account.UID]
			if slices.Contains(member.Finished[pending.Area], level.Group) {
				return fmt.Errorf("%w: guild war group already completed", account.ErrProgression)
			}
			member.Finished[pending.Area] = append(member.Finished[pending.Area], level.Group)
			if pending.Area != 0 {
				set, _ := s.Store.Rules.Network.WarSet(pending.War, pending.Area, level.Group)
				member.Unlocked[pending.Area] = max(member.Unlocked[pending.Area], set+1)
			}
			member.Score += level.Medals
			state.Members[p.Account.UID] = member
			if err = s.Store.RecordGuildWarProgress(tx, guildID, pending.War, p.Account.UID, pending.Area, level.Medals, 0, now); err != nil {
				return err
			}
			previous := state.Endurance[pending.Area]
			state.Endurance[pending.Area] = max(0, previous-level.Reduction)
			if pending.Area == 0 && state.Endurance[0] == 0 && state.Killed == 0 {
				state.Killed = now
				killed = true
			}
			quantities = map[model.ItemID]int64{}
			for _, entry := range s.Store.Rules.Rewards[level.Reward] {
				quantities[model.ItemID(entry[0])] += entry[1]
			}
			if err = p.Account.GrantItems(s.Store.Rules, quantities); err != nil {
				return err
			}
			heroes := map[model.HeroID]string{}
			for _, id := range p.SelectedHeroes {
				if _, owned := p.Account.Heroes[id]; !owned {
					continue
				}
				data, err := network.SupportHeroPayload(p.Account, id)
				if err != nil {
					return err
				}
				heroes[id] = data
			}
			history := model.GuildWarHistory{UID: p.Account.UID, TeamLevel: p.Account.Level, Duration: r.Result.Extra.UseTime, Timestamp: now, Heroes: heroes}
			if assistant != nil {
				if owner, ok := state.Members[support.Owner]; ok {
					owner.Assists += level.Assist
					state.Members[support.Owner] = owner
					if err = s.Store.RecordGuildWarProgress(tx, guildID, pending.War, support.Owner, pending.Area, 0, level.Assist, now); err != nil {
						return err
					}
				}
				history.Assist, history.AssistUID = support.Hero, support.Owner
				if history.AssistData, err = network.SupportHeroPayload(&assistant.Account, support.Hero); err != nil {
					return err
				}
			}
			encoded, err := json.Marshal(history)
			if err != nil {
				return err
			}
			if err = s.Store.Database.Queries.WithTx(tx).SaveGuildWarHistory(context.Background(), database.SaveGuildWarHistoryParams{GuildID: guildID, War: int64(pending.War), Area: int64(pending.Area), GroupID: int64(level.Group), UID: p.Account.UID, Duration: r.Result.Extra.UseTime, Record: encoded}); err != nil {
				return err
			}
			if previous > 0 && state.Endurance[pending.Area] == 0 {
				reward := s.Store.Rules.Network.Wars[pending.War].AreaRewards[pending.Area]
				if pending.Area == 0 {
					reward = s.Store.Rules.Network.Wars[pending.War].BossReward
				}
				if err = s.Store.DeliverGuildWarClear(tx, guildID, reward, now); err != nil {
					return err
				}
			}
			if err = s.Store.RecordRank(tx, p.Account.UID, model.RankGuildWar, pending.War, member.Score, now); err != nil {
				return err
			}
		}
		p.Account.Network.Pending = nil
		return s.Store.SaveGuildWar(tx, pending.Guild, pending.War, state)
	})
	if err != nil {
		return true, err
	}
	p.State, p.Nonce, p.LastBattleWon = server.Lobby, "", r.Win
	if err = s.Sync(p, "itemComp", "heroComp", "taskComp"); err != nil {
		return true, err
	}
	if err = p.Send("confirmBattleVerify"); err != nil {
		return true, err
	}
	if err = p.Send("onGetGuildWarGeneralInfo", warGeneral(state, p.Account.UID)); err != nil {
		return true, err
	}
	if len(quantities) > 0 {
		if err = p.Send("showReward", []model.RewardID{}, s.ItemRewards(before, p.Account, quantities), 1); err != nil {
			return true, err
		}
	}
	if err = p.Send("onBattleEnd", r.Battle, r.Result); err != nil {
		return true, err
	}
	if killed {
		return true, guild.Sync(s.Service, pending.Guild)
	}
	return true, nil
}
