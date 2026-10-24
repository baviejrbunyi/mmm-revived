package dispatch

import (
	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("dispatchDoRefresh", rpc.Bare(func(p *server.Player) error {
		return s.dispatch(p, func(rules *account.Rules, now int64) ([]message, error) {
			return []message{{"onDispatchRefreshFinish", nil}}, p.Account.RefreshDispatch(rules, now)
		})
	}))
	r.Handle("dispatchDoDispatch", rpc.Bind(DecodeDispatchStart, func(p *server.Player, r DispatchStart) error {
		return s.dispatch(p, func(rules *account.Rules, now int64) ([]message, error) {
			return []message{{"onStartDispatch", []any{r.Mission}}}, p.Account.StartDispatch(rules, r.Mission, r.Heroes, now)
		})
	}))
	r.Handle("dispatchCancel", rpc.Bind(DecodeDispatch, func(p *server.Player, mission model.DispatchID) error {
		return s.dispatch(p, func(_ *account.Rules, now int64) ([]message, error) {
			return []message{{"onCancelDispatchFinish", []any{mission}}}, p.Account.CancelDispatch(mission, now)
		})
	}))
	r.Handle("dispatchGetReward", rpc.Bind(DecodeDispatches, func(p *server.Player, missions []model.DispatchID) error {
		return s.dispatch(p, func(rules *account.Rules, now int64) ([]message, error) {
			before := *p.Account
			quantities, egg, err := p.Account.ClaimDispatch(rules, missions, now)
			if err != nil {
				return nil, err
			}
			var messages []message
			if egg != 0 {
				messages = append(messages, message{"onGetDispatchEgg", []any{int(egg)}})
			}
			return append(messages, message{"onGetDispatchReward", []any{s.ItemRewards(before, p.Account, quantities)}}), nil
		})
	}))
	r.Handle("unlockMonsterManual", rpc.Bind(DecodeMonsters, s.unlockMonsters))
}
