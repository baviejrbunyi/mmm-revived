package multipvp

import (
	"mmm-revived/internal/handlers/battle"
	"mmm-revived/internal/handlers/journey"
	"mmm-revived/internal/model"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("multiPVPAttachSelfGround", rpc.Bind(journey.DecodeJourneyActivity, func(p *server.Player, id model.ActivityID) error {
		return s.updateActivity(p, func(now int64) error { return p.Account.RefreshRetailPVPBox(s.Store.Rules, id, now) }, "onMultiPVPAttachSelfGround")
	}))
	r.Handle("multiPVPSelectMine", rpc.Bind(DecodeMineSelection, func(p *server.Player, request MineSelection) error {
		return s.lobbyActivity(p, func(now int64) error {
			return p.Account.SelectRetailPVPMine(s.Store.Rules, request.Activity, request.Index, request.Mine, now)
		}, "onMultiPVPAttachSelfGround")
	}))
	r.Handle("multiPVPUpdatePVPInfo", rpc.Bind(DecodeDefense, func(p *server.Player, request Defense) error {
		return s.lobbyActivity(p, func(now int64) error {
			return p.Account.SetRetailPVPDefense(s.Store.Rules, request.Activity, request.Index, request.Heroes, now)
		}, "onMultiPVPUpdatePVPInfo")
	}))
	r.Handle("multiPVPAttachEnemyGround", rpc.Bind(journey.DecodeJourneyActivity, func(p *server.Player, id model.ActivityID) error { return s.refreshEnemies(p, id, false) }))
	r.Handle("multiPVPRefreshAllEnemy", rpc.Bind(journey.DecodeJourneyActivity, func(p *server.Player, id model.ActivityID) error { return s.refreshEnemies(p, id, true) }))
	r.Handle("multiPVPAttachRevengeGround", rpc.Bind(DecodeRevenge, s.attachRevenge))
	r.Handle("multiPVPReapBox", rpc.Bind(journey.DecodeJourneyActivity, func(p *server.Player, id model.ActivityID) error {
		var coin, point int64
		if err := s.lobbyActivity(p, func(now int64) error {
			var err error
			coin, point, err = p.Account.ReapRetailPVPBox(s.Store.Rules, id, now)
			return err
		}, ""); err != nil {
			return err
		}
		return p.Send("onMultiPVPReapBox", coin, point)
	}))
}

func PrepareBattle(srv *server.Service, p *server.Player, r battle.CreateBattle) (bool, error) {
	return service{srv}.prepareBattle(p, r)
}

func LockDeployment(srv *server.Service, p *server.Player, heroes []model.HeroID) error {
	return service{srv}.lockDeployment(p, heroes)
}

func FinishBattle(srv *server.Service, p *server.Player, r battle.VerifyBattle) (bool, error) {
	return service{srv}.finishBattle(p, r)
}
