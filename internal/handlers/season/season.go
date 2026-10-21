package season

import (
	"database/sql"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

type eventAction func(tx *sql.Tx, rules *account.Rules, now int64) (map[model.ItemID]int64, error)

func (s service) eventsOpen(p *server.Player) error {
	if p.State != server.Lobby || s.Store.Rules == nil {
		return account.ErrProgression
	}
	return nil
}

func (s service) recordBossRank(tx *sql.Tx, p *server.Player, now int64) error {
	return s.Store.RecordRank(tx, p.Account.UID, model.RankBoss, p.Account.Events.Boss.Open, p.Account.BossBest(), now)
}

func rewardsOrAcknowledge(p *server.Player) func([]model.ItemReward) error {
	return func(rewards []model.ItemReward) error {
		if len(rewards) == 0 {
			return server.Acknowledge(p)
		}
		return p.ShowRewards(rewards)
	}
}

func (s service) event(p *server.Player, action eventAction, finish func([]model.ItemReward) error) error {
	if err := s.eventsOpen(p); err != nil {
		return err
	}
	before := *p.Account
	var quantities map[model.ItemID]int64
	err := s.Store.Update(p.Account, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		p.Account.InitializeEvents(s.Store.Rules, now)
		var err error
		quantities, err = action(tx, s.Store.Rules, now)
		return err
	})
	if err != nil {
		return err
	}
	if err = s.Sync(p, "seasonComp", "bossComp", "historyComp", "heroComp", "itemComp", "taskComp"); err != nil {
		return err
	}
	return finish(s.ItemRewards(before, p.Account, quantities))
}
