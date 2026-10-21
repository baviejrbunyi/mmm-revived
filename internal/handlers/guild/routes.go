package guild

import (
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("clientReqCreateGuild", rpc.Bind(DecodeGuildSettings, s.createGuild))
	r.Handle("clientReqUpdateGuildInitInfo", rpc.Bind(DecodeGuildSettings, s.updateGuildInfo))
	r.Handle("clientReqUpdateAnnounce", rpc.Bind(DecodeGuildText, s.updateAnnouncement))
	r.Handle("clientReqGuildInfos", rpc.Bind(DecodeGuildIDs, func(p *server.Player, ids []model.GuildID) error {
		infos, err := s.Store.Social.GuildInfos(ids)
		if err != nil {
			return err
		}
		return p.Send("onReqGuildInfo", 1, infos)
	}))
	r.Handle("clientReqGuildInfoAboutMembers", rpc.Bare(func(p *server.Player) error {
		members, err := s.Store.Social.GuildMembers(p.Account.UID)
		return sendResult(p, "onResponseGuildMembersInfo", members, err)
	}))
	r.Handle("clientReqGuildInfoAboutApply", rpc.Bare(func(p *server.Player) error {
		applications, err := s.Store.Social.GuildApplications(p.Account.UID)
		return sendResult(p, "onResponseGuildApplyInfo", applications, err)
	}))
	r.Handle("clientReqGuildInfoAboutInvites", rpc.Bare(func(p *server.Player) error {
		invitations, err := s.Store.Social.GuildInvitations(p.Account.UID)
		return sendResult(p, "onGuildGetInviteInfo", invitations, err)
	}))
	r.Handle("clientReqGuildInfoAboutChat", rpc.Bare(func(p *server.Player) error {
		history, err := s.Store.Social.GuildChatHistory(p.Account.UID)
		return sendResult(p, "onResponseGetAllChat", history, err)
	}))
	r.Handle("clientReqGuildInfoAboutNews", rpc.Bare(func(p *server.Player) error {
		news, err := s.Store.Social.GuildNews(p.Account.UID)
		return sendResult(p, "onResponseGetAllNews", news, err)
	}))
	r.Handle("clientReqRecommendGuild", rpc.Bare(func(p *server.Player) error { return s.findGuilds(p, 0, "", true) }))
	r.Handle("clientReqSearchGuildById", rpc.Bind(DecodeGuildSearchID, func(p *server.Player, id model.GuildID) error {
		if id == 0 {
			return p.Send("onGuildSearchResult", []model.GuildID{})
		}
		return s.findGuilds(p, id, "", false)
	}))
	r.Handle("clientReqSearchGuildByName", rpc.Bind(DecodeGuildSearch, func(p *server.Player, name string) error {
		return s.findGuilds(p, 0, name, false)
	}))
	r.Handle("clientReqJoinGuild", rpc.Bind(DecodeGuildID, s.joinGuild))
	r.Handle("clientReqCancelJoinGuild", rpc.Bind(DecodeGuildID, s.cancelApplication))
	r.Handle("clientReqSendGuildInvite", rpc.Bind(protocol.DecodePlayer, s.inviteToGuild))
	r.Handle("clientReqHandleGuildInvite", rpc.Bind(DecodeInviteDecision, s.decideGuildInvite))
	r.Handle("clientReqHandleGuildJoinApply", rpc.Bind(DecodeApplicationDecision, s.decideGuildApplication))
	r.Handle("clientReqLeaveGuild", rpc.Bare(s.leaveGuild))
	r.Handle("clientReqKickGuildMember", rpc.Bind(protocol.DecodePlayer, s.kickGuildMember))
	r.Handle("clientSetGuildAdmin", rpc.Bind(DecodeAdminDecision, s.setGuildAdmin))
	r.Handle("clientReqTransferBoss", rpc.Bind(protocol.DecodePlayer, s.transferGuild))
	r.Handle("clientReqReportGuild", rpc.Bind(DecodeGuildReport, s.reportGuild))
	r.Handle("reqGuildLimitedHeads", rpc.Bare(func(p *server.Player) error {
		return p.Send("onGetGuildLimitedHeads", model.Empty{}, model.Empty{})
	}))
}
