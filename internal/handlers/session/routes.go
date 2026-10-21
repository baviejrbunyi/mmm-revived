package session

import (
	"mmm-revived/internal/calendar"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Public("login", rpc.Bind(DecodeLogin, func(p *server.Player, login Login) error { return s.Login(p, login.Name, login.Password) }))
	for _, name := range []string{"mobileLogin", "cdkeyContinueLogin"} {
		r.Public(name, rpc.Bind(protocol.DecodeIgnore, loginUnavailable))
	}
	r.Handle("syncTime", rpc.Bare(server.SyncTime))
	r.Handle("pulse", rpc.Bare(func(p *server.Player) error { return p.Send("onPulse") }))
	r.Handle("queryLoginTime", rpc.Bare(func(p *server.Player) error {
		return p.Send("onQueryLoginTime", calendar.ClientTime(p.Account.LastLogin), calendar.ClientTime(p.Account.PreviousLogin))
	}))
	r.Handle("getMostUseHero", rpc.Bind(DecodeBattleID, func(p *server.Player, battle model.BattleID) error {
		return p.Send("onGetMostUseHero", battle, s.mostUsedHeroes(p))
	}))
	r.Handle("setGuide", rpc.Bind(DecodeGuide, s.setGuide))
	r.HandleAll(rpc.Bare(func(*server.Player) error { return nil }), "noviceStartSaLog", "noviceEndSaLog", "unlockMonsterManual", "recordPlotFinish", "report", "reachNewbie", "acsdkCheatCheck", "acsdkHttpProxy")
	r.HandleAll(offline, "afterBindMobile", "checkGuestBind", "getBindMobileReward", "getSurveyReward", "ngpApplyPrime", "ngpApplyPrimeConfirm", "ngpLoginFromGame", "ngpPrimeEmailAuthCodePreCheck", "ngpPrimeEmailPreCheckConfirm", "pullHotfix", "refreshResource")
	r.Handle("checkDiscordBind", rpc.Bind(protocol.DecodeIgnore, func(p *server.Player, _ struct{}) error { return p.Send("onCheckDiscordBind", false) }))
	r.Handle("getUnimatrixToken", rpc.Bare(func(p *server.Player) error {
		p.Session.Logger.Debug("offline RPC", "method", "getUnimatrixToken")
		return p.Send("onGetUnimatrixToken", "", "")
	}))
}
