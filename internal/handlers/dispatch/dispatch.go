package dispatch

import (
	"database/sql"
	"fmt"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/server"
)

func (s service) dispatch(p *server.Player, action func(*account.Rules, int64) (string, []any, error)) error {
	if s.Store.Rules == nil || p.State != server.Lobby {
		return fmt.Errorf("%w: dispatch is unavailable", account.ErrProgression)
	}
	var method string
	var args []any
	err := s.Store.Update(p.Account, func(*sql.Tx) error {
		var err error
		method, args, err = action(s.Store.Rules, time.Now().Unix())
		return err
	})
	if err != nil {
		return err
	}
	if err = s.Sync(p, "dispatchComp", "heroComp", "itemComp", "taskComp"); err != nil {
		return err
	}
	return p.Send(method, args...)
}
