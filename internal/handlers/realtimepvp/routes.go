package realtimepvp

import (
	"mmm-revived/internal/account"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("realtimePVPStart", rpc.Bind(protocol.DecodeActivity, s.startDraft))
	r.Handle("realtimePVPReadyToMatch", rpc.Bind(protocol.DecodeActivity, s.readyDraft))
	r.Handle("realtimePVPRechooseHero", s.pvpChoice(func(a *account.Account, rules *account.Rules, r PVPChoice) error {
		return a.RerollPVP(rules, r.Activity, r.Index)
	}))
	r.Handle("realtimePVPChooseNewHero", s.pvpChoice(func(a *account.Account, rules *account.Rules, r PVPChoice) error {
		return a.ChoosePVPHero(rules, r.Activity, r.Index)
	}))
	r.Handle("realtimePVPUpHero", s.pvpChoice(func(a *account.Account, rules *account.Rules, r PVPChoice) error {
		return a.UpgradePVPHero(rules, r.Activity, r.Index, false)
	}))
	r.Handle("realtimePVPUpHeroMax", s.pvpChoice(func(a *account.Account, rules *account.Rules, r PVPChoice) error {
		return a.UpgradePVPHero(rules, r.Activity, r.Index, true)
	}))
	r.Handle("realtimePVPSettle", s.pvpSettle(false))
	r.Handle("realtimePVPGiveup", s.pvpSettle(true))
	r.Handle("realtimePVPMatch", rpc.Bind(DecodeMatchStart, s.joinMatch))
	r.Handle("clientReqCancelMatch", rpc.Bare(s.cancelMatch))
	r.Handle("loadFinish", rpc.Bare(s.loadMatch))
	r.Handle("confirmRealTimePvPHero", rpc.Bind(DecodeSelection, s.confirmSelection))
	r.Handle("reportRealTimeResult", rpc.Bind(DecodeMatchReport, s.reportMatch))
	r.Handle("duelOperation", rpc.Bind(DecodeDuelOperation, func(p *server.Player, r DuelOperation) error {
		return s.RelayDuelOperation(p, r.Operation, r.Arguments)
	}))
	r.Handle("surrend", rpc.Bare(func(p *server.Player) error {
		if p.Room == nil {
			return nil
		}
		return s.FinishMatch(p.Room, p.Opponent().Account.UID)
	}))
	r.Handle("gainRealtimePvpReward", rpc.Bind(DecodeActivityTasks, s.claimActivityTasks))
	r.Handle("heroLeaveMap", rpc.Bind(DecodePVPHeroLeave, s.leavePVPHero))
	r.Handle("heroChangePos", rpc.Bind(DecodePVPHeroPosition, s.movePVPHero))
}
