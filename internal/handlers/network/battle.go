package network

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/handlers/battle"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

type AsyncBattle struct {
	Battle   model.BattleID
	Player   model.UserID
	Opponent int64
	Token    string
}

type AsyncOutcome struct {
	Result     battle.BattleResult `json:"result"`
	ScoreDelta int                 `json:"score_delta"`
	Replay     string              `json:"replay"`
	Packet     string              `json:"packet"`
}

func (s service) botLevel(p *server.Player) int {
	score, _ := p.Account.NetworkScores(s.Store.Rules)
	_, rank := s.Store.Rules.Network.Rank(score)
	return max(1, rank.AILevel)
}

func (s service) botAccount(id int, level int) (account.Account, error) {
	rule, ok := s.Store.Rules.Network.Bots[id]
	if !ok {
		return account.Account{}, fmt.Errorf("%w: unknown asynchronous opponent", account.ErrProgression)
	}
	a := account.NewAccount(model.UserID(-id), "local-ai", time.Now().Unix(), s.Store.Catalog)
	a.PrepareNetwork(s.Store.Rules.Network, time.Now().Unix())
	a.Network.Defense = map[model.HeroID]int{}
	for id, slot := range rule.Heroes {
		hero, exists := a.Heroes[id]
		if !exists {
			continue
		}
		hero.Level = max(1, level)
		hero.Break = s.Store.Rules.Levels[hero.Level].Break
		a.Heroes[id] = hero
		a.Network.Defense[id] = slot
		a.Network.DefenseOrder = append(a.Network.DefenseOrder, id)
	}
	if len(a.Network.Defense) == 0 {
		return a, fmt.Errorf("%w: asynchronous bot has no valid heroes", account.ErrProgression)
	}
	slices.Sort(a.Network.DefenseOrder)
	a.Network.Environment, a.Network.Camp = rule.Environment, rule.Camp
	return a, nil
}

func (s service) refreshAsyncTargets(p *server.Player, refresh bool, now int64) error {
	ids, err := s.Store.Database.Queries.ListAsyncCandidates(context.Background(), p.Account.UID)
	if err != nil {
		return err
	}
	profiles, err := s.networkProfiles(ids)
	if err != nil {
		return err
	}
	targets := []model.AsyncTarget{}
	for _, uid := range ids {
		a := profiles[uid].Account
		if len(a.Network.Defense) == 0 {
			continue
		}
		encoded, err := server.EncodePayload(a.NetworkDefense(s.Store.Rules))
		if err != nil {
			return err
		}
		targets = append(targets, model.AsyncTarget{UID: int64(uid), Score: max(s.Store.Rules.Network.Initial, a.Network.Score), Level: a.Level, Defense: encoded, Shown: encoded})
	}
	bots := make([]int, 0, len(s.Store.Rules.Network.Bots))
	for id := range s.Store.Rules.Network.Bots {
		bots = append(bots, id)
	}
	slices.Sort(bots)
	level := s.botLevel(p)
	for _, id := range bots {
		a, err := s.botAccount(id, level)
		if err != nil {
			continue
		}
		encoded, err := server.EncodePayload(a.NetworkDefense(s.Store.Rules))
		if err != nil {
			return err
		}
		targets = append(targets, model.AsyncTarget{UID: int64(a.UID), Score: p.Account.Network.Score, Level: level, Defense: encoded, Shown: encoded, AI: id, AILevel: level})
		if len(targets) >= 6 {
			break
		}
	}
	if len(targets) > 3 {
		offset := max(0, p.Account.Network.Refreshed-1) % len(targets)
		if refresh {
			offset = (offset + 1) % len(targets)
		}
		rotated := append(slices.Clone(targets[offset:]), targets[:offset]...)
		targets = rotated[:3]
	}
	err = s.Store.Update(p.Account, func(tx *sql.Tx) error {
		rule := s.Store.Rules.Network.Activities[p.Account.Network.Activity]
		if refresh {
			if p.Account.Network.Refreshed > rule.Refreshes {
				return fmt.Errorf("%w: daily opponent refresh limit reached", account.ErrProgression)
			}
			if now-p.Account.Network.LastRefresh < 2 {
				return fmt.Errorf("%w: opponent refresh cooldown", account.ErrProgression)
			}
			p.Account.Network.Refreshed++
			p.Account.Network.LastRefresh = now
		}
		p.Account.Network.Targets, p.Account.Network.LastRequest = targets, now
		return s.Store.RecordRank(tx, p.Account.UID, model.RankAsync, int(p.Account.Network.Activity), int64(p.Account.Network.MaxScore), now)
	})
	if err != nil {
		return err
	}
	return s.syncNetwork(p)
}

func (s service) prepareNetworkBattle(p *server.Player, r battle.CreateBattle) (bool, error) {
	if s.Store.Rules == nil {
		return false, nil
	}
	if p.State == server.Lobby {
		p.Account.Network.Support.Owner, p.Account.Network.Support.Hero, p.Account.Network.Support.Guild = 0, 0, false
	}
	mode := s.Store.Rules.BattleModes[r.Battle]
	if mode != 6 && mode != 25 {
		return false, nil
	}
	if p.State != server.Lobby {
		return true, fmt.Errorf("%w: a battle is already active", account.ErrProgression)
	}
	now := time.Now().Unix()
	p.Account.PrepareNetwork(s.Store.Rules.Network, now)
	var extra any = model.BattleExtra{TeamLevel: p.Account.Level}
	var pending *model.NetworkBattle
	if mode == 25 {
		pending = p.Account.Network.Pending
		if pending == nil || pending.Guild == 0 || pending.Battle != r.Battle || pending.Token != r.Extra.BattleToken || pending.Area != r.Extra.AreaID || pending.Level != r.Extra.LevelID || pending.Started != 0 {
			return true, fmt.Errorf("%w: guild war challenge token mismatch", account.ErrProgression)
		}
	} else {
		rule, ok := s.Store.Rules.Network.Activities[r.Extra.Activity]
		if !ok || r.Extra.Activity != p.Account.Network.Activity || !slices.Contains(rule.Battles, r.Battle) || len(p.Account.Network.Defense) == 0 {
			return true, fmt.Errorf("%w: asynchronous battle activity or defense unavailable", account.ErrProgression)
		}
		var uid int64
		friend := r.Extra.FriendUID != 0
		if friend {
			if _, ok := p.Account.Network.Friends[r.Extra.FriendUID]; !ok {
				return true, fmt.Errorf("%w: friend target was not requested", account.ErrProgression)
			}
			uid = int64(r.Extra.FriendUID)
		} else {
			if r.Extra.Target == nil {
				return true, fmt.Errorf("%w: asynchronous opponent missing", account.ErrProgression)
			}
			for index, target := range p.Account.Network.Targets {
				if target.UID == r.Extra.Target.UID {
					r.Extra.Target = &target
					r.Extra.Mode = index + 1
					uid = target.UID
					break
				}
			}
			if uid == 0 || p.Account.Network.Attempts <= 0 {
				return true, fmt.Errorf("%w: asynchronous opponent or attempts unavailable", account.ErrProgression)
			}
		}
		defender, err := s.asyncOpponent(p, uid)
		if err != nil {
			return true, err
		}
		extra = model.AsyncBattleInfo{TeamLevel: p.Account.Level, SkinCollection: p.Account.SkinCollectionLevel(s.Store.Rules), Attacker: p.Account.NetworkDefense(s.Store.Rules), Defender: defender, Bonds: p.Account.BondAttributes(s.Store.Rules)}
		pending = &model.NetworkBattle{Battle: r.Battle, Opponent: uid, Friend: friend, Activity: r.Extra.Activity, Target: r.Extra.Target, Mode: r.Extra.Mode, Defender: defender}
	}
	token := server.RandomToken()
	if err := s.Store.Update(p.Account, func(tx *sql.Tx) error {
		p.Account.Network.Pending = pending
		if mode == 6 {
			p.Account.Network.Pending.Token = token
		}
		if mode == 6 && !pending.Friend {
			p.Account.Network.Attempts--
		}
		if mode == 25 {
			p.Account.Network.Pending.Started = now
		}
		return nil
	}); err != nil {
		return true, err
	}
	p.RogueBattle, p.BossMission, p.Limited = false, 0, nil
	p.Challenge, p.Spiral, p.CampActivity, p.CampNode = 0, 0, 0, 0
	p.ActivityBattle, p.ActivityMission, p.ActivityNode, p.Brawl = 0, 0, 0, 0
	p.Deployment = nil
	p.Nonce, p.BattleID, p.State, p.Mission = token, r.Battle, server.Deployment, 0
	p.Chip, p.SelectedHeroes, p.LastBattleWon = nil, nil, false
	artifacts, highlights := battle.GearInfo(p.Account)
	return true, p.Send("createBattle", r.Battle, 0, mode, p.Account.Heroes, artifacts, highlights, model.Empty{}, model.Empty{}, extra, r.Extra, p.Nonce)
}

func (s service) asyncOpponent(p *server.Player, uid int64) (model.AsyncDefense, error) {
	if uid < 0 {
		a, err := s.botAccount(int(-uid), s.botLevel(p))
		if err != nil {
			return model.AsyncDefense{}, err
		}
		return a.NetworkDefense(s.Store.Rules), nil
	}
	profiles, err := s.networkProfiles([]model.UserID{model.UserID(uid)})
	if err != nil {
		return model.AsyncDefense{}, err
	}
	a, ok := profiles[model.UserID(uid)]
	if !ok || len(a.Account.Network.Defense) == 0 {
		return model.AsyncDefense{}, fmt.Errorf("%w: asynchronous opponent defense unavailable", account.ErrProgression)
	}
	return a.Account.NetworkDefense(s.Store.Rules), nil
}
