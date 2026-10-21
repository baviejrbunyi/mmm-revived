package multipvp

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/handlers/journey"
	"mmm-revived/internal/handlers/network"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

func (s service) lobbyActivity(p *server.Player, action func(int64) error, callback string, args ...any) error {
	if p.State != server.Lobby {
		return journey.ErrUnavailable
	}
	return s.updateActivity(p, action, callback, args...)
}

func (s service) updateActivity(p *server.Player, action func(int64) error, callback string, args ...any) error {
	if s.Store.Rules == nil {
		return journey.ErrUnavailable
	}
	rankChanged := false
	if err := s.Store.Update(p.Account, func(tx *sql.Tx) error {
		previous := map[model.ActivityID]int64{}
		for id, activity := range p.Account.Journey.Activities {
			if activity.RetailPVP != nil {
				previous[id] = activity.RetailPVP.Point
			}
		}
		now := time.Now().Unix()
		p.Account.InitializeRetailPVP(s.Store.Rules, now)
		if err := action(now); err != nil {
			return err
		}
		for id, activity := range p.Account.Journey.Activities {
			if activity.RetailPVP != nil && previous[id] != activity.RetailPVP.Point {
				if err := s.Store.RecordRank(tx, p.Account.UID, model.RankRetailPVP, int(id), activity.RetailPVP.Point, now); err != nil {
					return err
				}
				rankChanged = true
			}
		}
		return nil
	}); err != nil {
		return err
	}
	names := []string{"activityComp", "itemComp", "basicComp", "taskComp"}
	if rankChanged {
		names = append(names, "historyComp")
	}
	if err := s.Sync(p, names...); err != nil {
		return err
	}
	if callback != "" {
		return p.Send(callback, args...)
	}
	return nil
}

func (s service) mineTarget(a *account.Account, id model.ActivityID, index string) (model.RetailPVPTarget, bool, error) {
	activity := a.Journey.Activities[id]
	if _, _, err := a.RetailPVPActivity(s.Store.Rules, id, time.Now().Unix()); err != nil {
		return model.RetailPVPTarget{}, false, nil
	}
	if activity.RetailPVP == nil || !activity.Open {
		return model.RetailPVPTarget{}, false, nil
	}
	mine, exists := activity.RetailPVP.Mines[index]
	if !exists {
		return model.RetailPVPTarget{}, false, nil
	}
	defense, okay := a.RetailPVPMineDefense(id, index, s.Store.Rules)
	if !okay {
		return model.RetailPVPTarget{}, false, nil
	}
	encoded, err := server.EncodePayload(defense)
	return model.RetailPVPTarget{UID: int64(a.UID), Mine: mine.Mine, MineIndex: index, Efficiency: mine.Efficiency, Level: a.Level, Defense: encoded}, true, err
}

func (s service) botTarget(id int, level int) (model.RetailPVPTarget, model.AsyncDefense, error) {
	rule := s.Store.Rules.RetailPVP.AI[id]
	mine := s.Store.Rules.RetailPVP.Mines[rule.Mine]
	info := model.AsyncHeroInfo{Heroes: map[model.HeroID]model.Hero{}, EquippedGear: map[model.HeroID]map[int]model.Artifact{}, Equipped: map[model.HeroID]model.Highlight{}, Bonds: map[string]int64{}}
	positions := map[model.HeroID]model.AsyncPosition{}
	order := []model.HeroID{}
	for hero, born := range rule.Lineup {
		info.Heroes[hero] = model.Hero{ID: hero, Level: level, Star: 3, Talents: map[int]int{1: 1, 2: 1, 3: 1, 4: 1, 5: 1}, AwakeIDs: []int{}, Artifacts: map[int]model.InstanceID{}, Highlights: map[int]model.InstanceID{}}
		info.EquippedGear[hero] = map[int]model.Artifact{}
		positions[hero] = model.AsyncPosition{Born: born}
		order = append(order, hero)
	}
	slices.Sort(order)
	defense := model.AsyncDefense{UID: int64(-id), PVPLevel: level, WinFix: 1, LoseFix: 1, Positions: positions, Order: order, Interactions: map[int]model.AsyncInteraction{}, HeroInfo: info}
	encoded, err := server.EncodePayload(defense)
	return model.RetailPVPTarget{UID: int64(-id), Mine: rule.Mine, MineIndex: "0", Efficiency: mine.InitialEfficiency, Level: level, AIDefense: encoded}, defense, err
}

func (s service) refreshEnemies(p *server.Player, id model.ActivityID, refresh bool) error {
	if refresh && p.State != server.Lobby || s.Store.Rules == nil {
		return journey.ErrUnavailable
	}
	now := time.Now().Unix()
	state, _, err := p.Account.RetailPVPActivity(s.Store.Rules, id, now)
	if err != nil {
		return err
	}
	var targets []model.RetailPVPTarget
	if refresh || state.Day != now/86400 || len(state.Enemies) == 0 {
		if targets, err = s.enemyTargets(p, id, state); err != nil {
			return err
		}
	}
	return s.updateActivity(p, func(now int64) error {
		state, main, err := p.Account.RetailPVPActivity(s.Store.Rules, id, now)
		if err != nil {
			return err
		}
		if refresh {
			if state.Refreshed >= main.DailyRefresh {
				return fmt.Errorf("%w: Halloween daily refresh limit reached", account.ErrProgression)
			}
			state.Refreshed++
		} else if len(state.Enemies) > 0 || targets == nil {
			return nil
		}
		state.Enemies = map[int]model.RetailPVPTarget{}
		if len(targets) > 0 {
			offset := state.Refreshed % len(targets)
			for index := range min(3, len(targets)) {
				state.Enemies[index] = targets[(index+offset)%len(targets)]
			}
		}
		return nil
	}, "onMultiPVPAttachEnemyGround")
}

func (s service) enemyTargets(p *server.Player, id model.ActivityID, state *model.RetailPVPState) ([]model.RetailPVPTarget, error) {
	point, okay := s.Store.Rules.RetailPVP.PointRule(state.Point)
	if !okay {
		return nil, fmt.Errorf("%w: Halloween point rule is unavailable", account.ErrProgression)
	}
	ids, err := s.Store.Database.Queries.ListAsyncCandidates(context.Background(), p.Account.UID)
	if err != nil {
		return nil, err
	}
	profiles, err := network.Profiles(s.Service, ids)
	if err != nil {
		return nil, err
	}
	targets := []model.RetailPVPTarget{}
	for _, uid := range ids {
		a := profiles[uid].Account
		for index := range 3 {
			target, okay, err := s.mineTarget(&a, id, strconv.Itoa(index))
			if err != nil {
				return nil, err
			}
			if okay {
				targets = append(targets, target)
				break
			}
		}
	}
	bots := []int{}
	for bot, row := range s.Store.Rules.RetailPVP.AI {
		if row.Type == point.AIType && row.Version == s.Store.Rules.RetailPVP.Activities[id].Parameter {
			bots = append(bots, bot)
		}
	}
	slices.Sort(bots)
	for _, bot := range bots {
		target, _, err := s.botTarget(bot, max(1, p.Account.Level+point.AILevel))
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func (s service) attachRevenge(p *server.Player, request Revenge) error {
	if p.State != server.Lobby || s.Store.Rules == nil {
		return journey.ErrUnavailable
	}
	now := time.Now().Unix()
	state, _, err := p.Account.RetailPVPActivity(s.Store.Rules, request.Activity, now)
	if err != nil {
		return err
	}
	if request.Index < 0 || request.Index >= len(state.Records) {
		return fmt.Errorf("%w: Halloween revenge record is unavailable", account.ErrProgression)
	}
	record := state.Records[request.Index]
	if record.Attack || record.UID <= 0 || record.UID == int64(p.Account.UID) || record.Revenged != 0 {
		return fmt.Errorf("%w: Halloween revenge target is invalid", account.ErrProgression)
	}
	profiles, err := network.Profiles(s.Service, []model.UserID{model.UserID(record.UID)})
	if err != nil {
		return err
	}
	profile, exists := profiles[model.UserID(record.UID)]
	if !exists {
		return fmt.Errorf("%w: Halloween revenge player is unavailable", account.ErrProgression)
	}
	peer := profile.Account
	var targets [3]*model.RetailPVPTarget
	for index := range targets {
		target, okay, err := s.mineTarget(&peer, request.Activity, strconv.Itoa(index))
		if err != nil {
			return err
		}
		if okay {
			targets[index] = &target
		}
	}
	return s.updateActivity(p, func(now int64) error {
		state, _, err := p.Account.RetailPVPActivity(s.Store.Rules, request.Activity, now)
		if err != nil {
			return err
		}
		uid := model.UserID(record.UID)
		current := state.Revenges[uid]
		if current.DailyAttacks >= s.Store.Rules.RetailPVP.MaximumRevenge {
			return fmt.Errorf("%w: Halloween revenge daily limit reached", account.ErrProgression)
		}
		current.Refreshed = now
		current.First, current.Second, current.Third = targets[0], targets[1], targets[2]
		state.Revenges[uid] = current
		return nil
	}, "onMultiPVPAttachRevengeGround", request.Index)
}
