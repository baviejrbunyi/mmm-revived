package dreamland

import (
	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("dlDoMission", rpc.Bind(DecodeMissionStart, func(p *server.Player, r MissionStart) error {
		return s.missionStep(p, r.Mission, func(rules *account.Rules) error { return p.Account.BeginMission(rules, r.Chapter, r.Mission) })
	}))
	r.Handle("dlEndNode", rpc.Bind(DecodeMission, func(p *server.Player, mission model.MissionID) error {
		return s.missionStep(p, mission, func(rules *account.Rules) error { return p.Account.AdvanceMission(rules, mission) })
	}))
	r.Handle("dlGetReward", rpc.Bind(DecodeMission, s.claimMission))
	r.Handle("dlFailNode", rpc.Bind(DecodeMissionFail, s.failMission))
	r.Handle("dlGetStarReward", rpc.Bind(DecodeChapterRewards, s.claimChapter))
	r.Handle("dlVisitChip", rpc.Bind(DecodeMementoVisit, s.visitMemento))
	r.Handle("dlFailTeam", rpc.Bind(DecodeMission, s.failMementoTeam))
	r.Handle("downloadPassMessage", rpc.Bare(func(p *server.Player) error {
		return s.communityMessage(p, community{scope: model.BattleMessages})
	}))
	r.Handle("uploadPassMessage", rpc.Bind(DecodePassMessage, func(p *server.Player, text string) error {
		return s.communityMessage(p, community{scope: model.BattleMessages, upload: true, text: text})
	}))
	r.Handle("dlDownloadBoxMessage", rpc.Bind(DecodeMission, func(p *server.Player, mission model.MissionID) error {
		return s.communityMessage(p, community{scope: model.MementoMessages, mission: mission})
	}))
	r.Handle("dlUploadBoxMessage", rpc.Bind(DecodeBoxMessage, func(p *server.Player, r BoxMessage) error {
		return s.communityMessage(p, community{scope: model.MementoMessages, mission: r.Mission, upload: true, text: r.Text})
	}))
}
