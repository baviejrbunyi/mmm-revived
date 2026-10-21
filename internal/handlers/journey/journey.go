package journey

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type journeyAction func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error)

var ErrUnavailable = fmt.Errorf("%w: journey action is unavailable", account.ErrProgression)

func RewardsOrAcknowledge(p *server.Player) func([]model.ItemReward) error {
	return func(rewards []model.ItemReward) error {
		if len(rewards) == 0 {
			return server.Acknowledge(p)
		}
		return p.ShowRewards(rewards)
	}
}

func (s service) activityLevels(claim func(*account.Account, *account.Rules, ActivityLevels, int64) (map[model.ItemID]int64, error)) rpc.Endpoint[*server.Player] {
	return rpc.Bind(DecodeActivityLevels, func(p *server.Player, levels ActivityLevels) error {
		return s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return claim(a, rules, levels, now)
		}, RewardsOrAcknowledge(p))
	})
}

func (s service) activity(action func(*account.Account, *account.Rules, model.ActivityID, int64) (map[model.ItemID]int64, error)) rpc.Endpoint[*server.Player] {
	return rpc.Bind(DecodeJourneyActivity, func(p *server.Player, activity model.ActivityID) error {
		return s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return action(a, rules, activity, now)
		}, RewardsOrAcknowledge(p))
	})
}

func (s service) replyThenRewards(p *server.Player, rewards []model.ItemReward, method string, args ...any) error {
	if err := p.Send(method, args...); err != nil {
		return err
	}
	return p.ShowRewardsIfAny(rewards)
}

func passFailure(err error) int {
	switch {
	case errors.Is(err, account.ErrJourneyPassUnavailable):
		return 149
	case errors.Is(err, account.ErrJourneyPassLevelFull):
		return 150
	case errors.Is(err, account.ErrJourneyPassCurrency):
		return 151
	}
	return 5
}

func (s service) journeyPass(p *server.Player, method string, action journeyAction) error {
	if s.Store.Rules == nil || p.State != server.Lobby {
		return p.Send(method, false, passFailure(ErrUnavailable))
	}
	rewards, err := s.journeyUpdate(p, action)
	if errors.Is(err, account.ErrProgression) {
		return p.Send(method, false, passFailure(err))
	}
	if err != nil {
		return err
	}
	if err = s.journeySync(p); err != nil {
		return err
	}
	return s.replyThenRewards(p, rewards, method, true, 0)
}

func (s service) journey(p *server.Player, action journeyAction, respond func([]model.ItemReward) error) error {
	if p.State != server.Lobby {
		return ErrUnavailable
	}
	return s.anywhere(p, action, respond)
}

func (s service) anywhere(p *server.Player, action journeyAction, respond func([]model.ItemReward) error) error {
	if s.Store.Rules == nil {
		return ErrUnavailable
	}
	rewards, err := s.journeyUpdate(p, action)
	if err != nil {
		return err
	}
	if err = s.journeySync(p); err != nil {
		return err
	}
	return respond(rewards)
}

func (s service) journeyUpdate(p *server.Player, action journeyAction) ([]model.ItemReward, error) {
	var rewards []model.ItemReward
	err := s.Store.Update(p.Account, func(*sql.Tx) error {
		now := time.Now().Unix()
		p.Account.InitializeJourney(s.Store.Rules, now)
		before := *p.Account
		quantities, err := action(p.Account, s.Store.Rules, now)
		if err != nil {
			return err
		}
		rewards = s.ItemRewards(before, p.Account, quantities)
		return nil
	})
	return rewards, err
}

func (s service) journeySync(p *server.Player) error {
	return s.Sync(p, "noviceComp", "battlepassComp", "monthlyPassComp", "activityComp", "itemComp", "heroComp", "basicComp", "energyComp", "taskComp", "challengeComp", "spiralComp")
}
