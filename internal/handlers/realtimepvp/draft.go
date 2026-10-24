package realtimepvp

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

func (s service) pvpRule(p *server.Player, activity model.ActivityID) (account.PVPRule, error) {
	if p.State == server.Lobby {
		if err := s.season(p); err != nil {
			return account.PVPRule{}, err
		}
	}
	rule, ok := s.Store.Rules.PVP[activity]
	if p.State != server.Lobby || !ok || activity != p.Account.RealtimeActivity() {
		return rule, fmt.Errorf("%w: activity is not open", account.ErrProgression)
	}
	return rule, nil
}

func (s service) readyDraft(p *server.Player, id model.ActivityID) error {
	rule, err := s.pvpRule(p, id)
	if err != nil {
		return err
	}
	activity := p.Account.Activities[id]
	if activity.Status != model.PVPRedraft || len(activity.Lineup) != rule.InitialHeroes {
		return fmt.Errorf("%w: draft is incomplete", account.ErrProgression)
	}
	activity.Status = model.PVPMatching
	p.Account.Activities[id] = activity
	return s.SaveAndSync(p, "activityComp")
}

func (s service) startDraft(p *server.Player, id model.ActivityID) error {
	rule, err := s.pvpRule(p, id)
	if err != nil {
		return err
	}
	activity := p.Account.Activities[id]
	if activity.Status != model.PVPInactive || activity.Chance <= 0 {
		return fmt.Errorf("%w: run is already active or no attempts remain", account.ErrProgression)
	}
	pool := slices.Clone(rule.Heroes)
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	if len(pool) < rule.InitialHeroes {
		return errors.New("invalid draft pool")
	}
	activity.Lineup = nil
	activity.History = nil
	activity.Wins = 0
	activity.Rerolled = false
	activity.Round, activity.Upgraded = 0, 0
	activity.Chance--
	for _, hero := range pool[:rule.InitialHeroes] {
		activity.Lineup = append(activity.Lineup, model.PVPHero{ID: hero, Level: 1, Star: rule.InitialStar})
	}
	activity.Status = model.PVPRedraft
	p.Account.Activities[id] = activity
	return s.SaveAndSync(p, "activityComp")
}

func (s service) pvpChoice(choose func(*account.Account, *account.Rules, PVPChoice) error) rpc.Endpoint[*server.Player] {
	return rpc.Bind(DecodePVPChoice, func(p *server.Player, r PVPChoice) error {
		return s.pvpAction(p, func() error { return choose(p.Account, s.Store.Rules, r) })
	})
}

func (s service) pvpSettle(giveUp bool) rpc.Endpoint[*server.Player] {
	return rpc.Bind(protocol.DecodeActivity, func(p *server.Player, activity model.ActivityID) error {
		return s.pvpAction(p, func() error { return p.Account.SettlePVP(activity, giveUp) })
	})
}

func (s service) pvpAction(p *server.Player, action func() error) error {
	if p.State != server.Lobby {
		return fmt.Errorf("%w: player is not in lobby", account.ErrProgression)
	}
	if err := s.season(p); err != nil {
		return err
	}
	if err := action(); err != nil {
		return err
	}
	return s.SaveAndSync(p, "activityComp")
}

func (s service) claimActivityTasks(p *server.Player, r ActivityTasks) error {
	if p.State == server.Lobby {
		if err := s.season(p); err != nil {
			return err
		}
	}
	rule, ok := s.Store.Rules.PVP[r.Activity]
	if !ok || r.Activity != p.Account.RealtimeActivity() {
		return fmt.Errorf("%w: activity is not open", account.ErrProgression)
	}
	rewards := make([]model.RewardID, 0, len(r.Tasks))
	for _, id := range r.Tasks {
		if !slices.Contains(rule.PointTasks, id) && !slices.Contains(rule.WinTasks, id) {
			return fmt.Errorf("%w: task does not belong to activity", account.ErrProgression)
		}
		rewards = append(rewards, s.Store.Rules.Tasks[id].Reward)
	}
	before := *p.Account
	if err := p.Account.ClaimTasks(s.Store.Rules, r.Tasks); err != nil {
		return err
	}
	if err := s.SaveAndSync(p, "taskComp", "itemComp", "heroComp"); err != nil {
		return err
	}
	return p.ShowRewardsIfAny(s.ItemRewards(before, p.Account, account.RewardQuantities(s.Store.Rules, rewards...)))
}
