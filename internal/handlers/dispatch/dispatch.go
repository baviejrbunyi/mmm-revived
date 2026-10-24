package dispatch

import (
	"database/sql"
	"fmt"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/server"
)

type message struct {
	method string
	args   []any
}

func (s service) dispatch(p *server.Player, action func(*account.Rules, int64) ([]message, error)) error {
	if s.Store.Rules == nil || p.State != server.Lobby {
		return fmt.Errorf("%w: dispatch is unavailable", account.ErrProgression)
	}
	var messages []message
	err := s.Store.Update(p.Account, func(*sql.Tx) error {
		var err error
		messages, err = action(s.Store.Rules, time.Now().Unix())
		return err
	})
	if err != nil {
		return err
	}
	if err = s.Sync(p, "dispatchComp", "heroComp", "itemComp", "taskComp", "manualComp"); err != nil {
		return err
	}
	for _, m := range messages {
		if err = p.Send(m.method, m.args...); err != nil {
			return err
		}
	}
	return nil
}

func (s service) unlockMonsters(p *server.Player, enemies []int) error {
	if s.Store.Rules == nil || len(enemies) == 0 {
		return nil
	}
	changed := false
	err := s.Store.Update(p.Account, func(*sql.Tx) error {
		changed = p.Account.UnlockMonsterManual(s.Store.Rules, enemies, time.Now().Unix())
		return nil
	})
	if err != nil || !changed {
		return err
	}
	return s.Sync(p, "manualComp")
}
