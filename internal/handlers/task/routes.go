package task

import (
	"database/sql"
	"fmt"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("getTaskRwd", rpc.Bind(DecodeTasks, s.claimTasks))
	r.Handle("getLvReward", rpc.Bind(DecodeLevelRewards, s.claimLevels))
	r.Handle("doShareOnce", rpc.Bind(DecodeShareOnce, func(p *server.Player, _ struct{}) error {
		if p.State != server.Lobby {
			return fmt.Errorf("%w: share is unavailable during battle", account.ErrProgression)
		}
		var shared bool
		if err := s.Store.Update(p.Account, func(*sql.Tx) (err error) {
			if shared, err = p.Account.ShareOnce(s.Store.Rules, time.Now().Unix()); err == nil {
				p.Account.RecordJourneyTaskEvent(s.Store.Rules, 59, 1, 0)
			}
			return err
		}); err != nil {
			return err
		}
		if err := s.Sync(p, "taskComp", "itemComp"); err != nil || shared {
			return err
		}
		return server.Acknowledge(p)
	}))
}
