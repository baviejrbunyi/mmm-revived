package draw

import (
	"database/sql"
	"errors"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

func (s service) draw(p *server.Player, r Draw) error {
	if s.Store.Rules == nil || p.State != server.Lobby {
		return p.Send("onTryDraw", 5)
	}
	rewards := make([]model.ItemReward, 0, r.Count)
	err := s.Store.Update(p.Account, func(*sql.Tx) error {
		for range r.Count {
			before := *p.Account
			item, err := p.Account.DrawOne(s.Store.Rules, r.Pool, time.Now().Unix())
			if err != nil {
				return err
			}
			rewards = append(rewards, s.ItemRewards(before, p.Account, map[model.ItemID]int64{item: 1})...)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, account.ErrProgression) {
			p.Session.Logger.Warn("draw rejected", "pool", r.Pool, "count", r.Count, "error", err)
			return p.Send("onTryDraw", 5)
		}
		return err
	}
	if err = s.Sync(p, "drawComp", "heroComp", "itemComp", "taskComp"); err != nil {
		return err
	}
	if err = p.Send("onTryDraw", 0); err != nil {
		return err
	}
	return p.Send("showDraw", r.Pool, rewards)
}
