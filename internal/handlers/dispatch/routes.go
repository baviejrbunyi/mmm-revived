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
		return s.dispatch(p, func(rules *account.Rules, now int64) (string, []any, error) {
			return "onDispatchRefreshFinish", nil, p.Account.RefreshDispatch(rules, now)
		})
	}))
	r.Handle("dispatchDoDispatch", rpc.Bind(DecodeDispatchStart, func(p *server.Player, r DispatchStart) error {
		return s.dispatch(p, func(rules *account.Rules, now int64) (string, []any, error) {
			return "onStartDispatch", []any{r.Mission}, p.Account.StartDispatch(rules, r.Mission, r.Heroes, now)
		})
	}))
	r.Handle("dispatchCancel", rpc.Bind(DecodeDispatch, func(p *server.Player, mission model.DispatchID) error {
		return s.dispatch(p, func(_ *account.Rules, now int64) (string, []any, error) {
			return "onCancelDispatchFinish", []any{mission}, p.Account.CancelDispatch(mission, now)
		})
	}))
	r.Handle("dispatchGetReward", rpc.Bind(DecodeDispatches, func(p *server.Player, missions []model.DispatchID) error {
		return s.dispatch(p, func(rules *account.Rules, now int64) (string, []any, error) {
			before := *p.Account
			quantities, err := p.Account.ClaimDispatch(rules, missions, now)
			if err != nil {
				return "", nil, err
			}
			return "onGetDispatchReward", []any{s.ItemRewards(before, p.Account, quantities)}, nil
		})
	}))
}
