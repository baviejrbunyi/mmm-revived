package guild

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
	"mmm-revived/internal/social"
)

func Reject(p *server.Player, err error) error {
	rejection, ok := social.Rejected(err)
	if !ok {
		return err
	}
	return p.Send("showErr", rejection.Code, append([]int64{}, rejection.Args...))
}

func respond(p *server.Player, err error, method string, args ...any) error {
	rejection, ok := social.Rejected(err)
	if !ok || len(rejection.Args) != 0 {
		return Reject(p, err)
	}
	return p.Send(method, append(args, rejection.Code)...)
}

func sendResult[T any](p *server.Player, method string, value T, err error) error {
	if err != nil {
		return Reject(p, err)
	}
	return p.Send(method, value)
}

func (s service) guildAudience(id model.GuildID) ([]model.UserID, error) {
	if id == 0 {
		return nil, nil
	}
	rows, err := s.Store.Database.Queries.ListGuildMembers(context.Background(), id)
	if err != nil {
		return nil, err
	}
	result := make([]model.UserID, len(rows))
	for i, row := range rows {
		result[i] = row.UID
	}
	return result, nil
}

func (s service) syncGuild(id model.GuildID, affected ...model.UserID) error {
	members, err := s.guildAudience(id)
	if err != nil {
		return err
	}
	affected = append(affected, members...)
	seen := make(map[model.UserID]bool, len(affected))
	for _, uid := range affected {
		if seen[uid] {
			continue
		}
		seen[uid] = true
		p := s.Online(uid)
		if p == nil {
			continue
		}
		snapshot, err := s.Store.Social.GuildSnapshot(uid)
		if err != nil {
			return err
		}
		if err = p.Update([]string{"guildComp"}, snapshot); err != nil {
			p.Session.Logger.Warn("guild update failed", "error", err)
			p.Session.Conn.Close()
		}
	}
	return nil
}

func (s service) guildBroadcast(id model.GuildID, method string, args ...any) error {
	members, err := s.guildAudience(id)
	if err != nil {
		return err
	}
	for _, uid := range members {
		s.notify(uid, method, args...)
	}
	return nil
}

func (s service) broadcastNews(id model.GuildID, news ...model.GuildNews) error {
	for _, entry := range news {
		if err := s.guildBroadcast(id, "onGuildNewsUpdate", entry); err != nil {
			return err
		}
	}
	return nil
}

func (s service) notify(uid model.UserID, method string, args ...any) {
	if target := s.Online(uid); target != nil {
		if err := target.Send(method, args...); err != nil {
			target.Session.Logger.Warn("guild notification failed", "error", err)
			target.Session.Conn.Close()
		}
	}
}

func (s service) createGuild(p *server.Player, settings model.GuildSettings) error {
	repo := s.Store.Social
	var id model.GuildID
	err := s.Store.Update(p.Account, func(tx *sql.Tx) error {
		var err error
		if id, err = repo.CreateGuild(tx, p.Account.UID, settings); err != nil {
			return err
		}
		return p.Account.Spend(map[model.ItemID]int64{2: repo.GuildLimits.CreationCost})
	})
	if errors.Is(err, account.ErrProgression) {
		return p.Send("onCreateGuildResult", social.CodeCreationCost)
	}
	if err != nil {
		return respond(p, err, "onCreateGuildResult")
	}
	if err = s.Sync(p, "itemComp"); err != nil {
		return err
	}
	if err = s.syncGuild(id, p.Account.UID); err != nil {
		return err
	}
	if err = p.Send("onCreateGuildResult", 0); err != nil {
		return err
	}
	return p.Send("onGuildInitFinish")
}

func (s service) updateGuildInfo(p *server.Player, settings model.GuildSettings) error {
	info, news, err := s.Store.Social.UpdateGuild(p.Account.UID, settings, true)
	if err != nil {
		return respond(p, err, "onUpdateGuildInfoResult")
	}
	if err = s.syncGuild(info.ID); err != nil {
		return err
	}
	if err = s.guildBroadcast(info.ID, "onGuildInitInfoChange", info); err != nil {
		return err
	}
	if err = s.broadcastNews(info.ID, news...); err != nil {
		return err
	}
	return p.Send("onUpdateGuildInfoResult", 0)
}

func (s service) updateAnnouncement(p *server.Player, text string) error {
	info, news, err := s.Store.Social.UpdateGuild(p.Account.UID, model.GuildSettings{Announcement: &text}, false)
	if err != nil {
		return respond(p, err, "onResponseUpdateGuildAnnounce")
	}
	if err = s.guildBroadcast(info.ID, "onGuildAnnounceChange", 0, info.Announcement); err != nil {
		return err
	}
	if err = s.broadcastNews(info.ID, news...); err != nil {
		return err
	}
	return p.Send("onResponseUpdateGuildAnnounce", 0)
}

func (s service) findGuilds(p *server.Player, id model.GuildID, name string, recommend bool) error {
	ids, err := s.Store.Social.FindGuilds(p.Account.UID, id, name, p.Account.Level, recommend)
	if err != nil {
		return err
	}
	if err = s.syncGuild(0, p.Account.UID); err != nil {
		return err
	}
	if recommend {
		return p.Send("onRecommendGuilds", 1, ids)
	}
	return p.Send("onGuildSearchResult", ids)
}

func (s service) joinGuild(p *server.Player, id model.GuildID) error {
	news, applied, err := s.Store.Social.JoinGuild(p.Account.UID, id, p.Account.Level)
	if err != nil {
		return Reject(p, err)
	}
	if news.ID == 0 {
		if err = s.syncGuild(0, p.Account.UID); err != nil {
			return err
		}
		if !applied {
			return server.Acknowledge(p)
		}
		return s.notifyApplication(id, p.Account.UID)
	}
	if err = s.syncGuild(id, p.Account.UID); err != nil {
		return err
	}
	if err = s.broadcastNews(id, news); err != nil {
		return err
	}
	return p.Send("onGuildInitFinish")
}

func (s service) cancelApplication(p *server.Player, id model.GuildID) error {
	if err := s.Store.Social.CancelGuildApplication(p.Account.UID, id); err != nil {
		return err
	}
	if err := s.syncGuild(0, p.Account.UID); err != nil {
		return err
	}
	return server.Acknowledge(p)
}

func (s service) notifyApplication(id model.GuildID, applicant model.UserID) error {
	rows, err := s.Store.Database.Queries.ListGuildMembers(context.Background(), id)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	for _, row := range rows {
		if row.Role == model.GuildBoss || row.Role == model.GuildManager {
			s.notify(row.UID, "onGuildNewJoinApply", applicant, now)
		}
	}
	return nil
}

func (s service) inviteToGuild(p *server.Player, target model.UserID) error {
	if _, err := s.Store.Social.InviteToGuild(p.Account.UID, target); err != nil {
		return respond(p, err, "onGuildSendInviteResult", target)
	}
	if err := s.syncGuild(0, target); err != nil {
		return err
	}
	return p.Send("onGuildSendInviteResult", target, 0)
}

func (s service) decideGuildInvite(p *server.Player, r Decision[model.GuildID]) error {
	news, err := s.Store.Social.DecideGuildInvite(p.Account.UID, r.Target, r.Accept, p.Account.Level)
	if err != nil {
		return respond(p, err, "onGuildAcceptInviteResult")
	}
	if !r.Accept {
		return s.syncGuild(0, p.Account.UID)
	}
	if err = s.syncGuild(r.Target, p.Account.UID); err != nil {
		return err
	}
	if err = s.broadcastNews(r.Target, news); err != nil {
		return err
	}
	if err = p.Send("onGuildInitFinish"); err != nil {
		return err
	}
	return p.Send("onGuildAcceptInviteResult", 0)
}

func (s service) decideGuildApplication(p *server.Player, r Decision[model.UserID]) error {
	method := "onGuildJoinApplyRejectResult"
	if r.Accept {
		method = "onGuildJoinApplyAcceptResult"
	}
	id, targets, news, err := s.Store.Social.DecideGuildApplications(p.Account.UID, r.Target, r.Accept)
	if err != nil {
		return respond(p, err, method, r.Target)
	}
	if !r.Accept {
		if err = s.syncGuild(0, targets...); err != nil {
			return err
		}
		return p.Send(method, r.Target, 0)
	}
	if err = s.syncGuild(id, targets...); err != nil {
		return err
	}
	for _, uid := range targets {
		s.notify(uid, "onGuildInitFinish")
	}
	if err = s.broadcastNews(id, news...); err != nil {
		return err
	}
	return p.Send(method, r.Target, 0)
}

func (s service) leaveGuild(p *server.Player) error {
	departure, err := s.Store.Social.LeaveGuild(p.Account.UID)
	if err != nil {
		return respond(p, err, "onResponseLeaveGuildResult")
	}
	if err = s.syncGuild(departure.ID, append(departure.Pending, p.Account.UID)...); err != nil {
		return err
	}
	if !departure.Disbands {
		if err = s.guildBroadcast(departure.ID, "onGuildMemberLeave", p.Account.UID); err != nil {
			return err
		}
		if err = s.broadcastNews(departure.ID, departure.News); err != nil {
			return err
		}
	}
	if err = p.Send("onRemoveFromGuild", 0); err != nil {
		return err
	}
	return p.Send("onResponseLeaveGuildResult", 0)
}

func (s service) kickGuildMember(p *server.Player, target model.UserID) error {
	id, news, err := s.Store.Social.KickGuildMember(p.Account.UID, target)
	if err != nil {
		return respond(p, err, "onResponseGuildKickMemberResult", target)
	}
	if err = s.syncGuild(id, target); err != nil {
		return err
	}
	s.notify(target, "onRemoveFromGuild", 1)
	if err = s.guildBroadcast(id, "onGuildMemberLeave", target); err != nil {
		return err
	}
	if err = s.broadcastNews(id, news); err != nil {
		return err
	}
	return p.Send("onResponseGuildKickMemberResult", target, 0)
}

func (s service) setGuildAdmin(p *server.Player, r Decision[model.UserID]) error {
	method := "onResponseUnSetAdmin"
	if r.Accept {
		method = "onResponseSetAdmin"
	}
	id, news, err := s.Store.Social.SetGuildAdmin(p.Account.UID, r.Target, r.Accept)
	if err != nil {
		return respond(p, err, method, r.Target)
	}
	if err = s.syncGuild(id); err != nil {
		return err
	}
	if err = s.guildBroadcast(id, "onGuildAdminChanged", r.Target, r.Accept); err != nil {
		return err
	}
	if err = s.broadcastNews(id, news); err != nil {
		return err
	}
	return p.Send(method, r.Target, 0)
}

func (s service) transferGuild(p *server.Player, target model.UserID) error {
	id, news, err := s.Store.Social.TransferGuild(p.Account.UID, target)
	if err != nil {
		return respond(p, err, "onResponseTransferBoss", target)
	}
	if err = s.syncGuild(id); err != nil {
		return err
	}
	if err = s.guildBroadcast(id, "onGuildAdminChanged", target, false); err != nil {
		return err
	}
	if err = s.guildBroadcast(id, "onGuildBossChanged", target); err != nil {
		return err
	}
	if err = s.broadcastNews(id, news); err != nil {
		return err
	}
	return p.Send("onResponseTransferBoss", target, 0)
}

func (s service) guildMembers(p *server.Player) error {
	now := time.Now().Unix()
	if snapshot, err := s.Store.Social.GuildSnapshot(p.Account.UID); err != nil {
		return err
	} else if snapshot.ID != 0 {
		if err = s.SettleGuildWeek(snapshot.ID, now); err != nil {
			return err
		}
		if err = p.Update([]string{"guildComp"}, snapshot); err != nil {
			return err
		}
	}
	members, err := s.Store.Social.GuildMembers(p.Account.UID)
	return sendResult(p, "onResponseGuildMembersInfo", members, err)
}

func (s service) reportGuild(p *server.Player, r GuildReport) error {
	if err := s.Store.Social.ReportGuild(p.Account.UID, r.Guild, r.Reason, r.Text); err != nil {
		return err
	}
	return server.Acknowledge(p)
}

func Sync(srv *server.Service, id model.GuildID, affected ...model.UserID) error {
	return service{srv}.syncGuild(id, affected...)
}

func SendChat(srv *server.Service, p *server.Player, kind int, text string) error {
	return service{srv}.sendChat(p, kind, text)
}

func (s service) sendChat(p *server.Player, kind int, text string) error {
	id, message, err := s.Store.Social.SendGuildChat(p.Account.UID, kind, text)
	if _, rejected := social.Rejected(err); rejected {
		return p.Send("onSendGuildChatErr")
	}
	if err != nil {
		return err
	}
	if err = s.Store.Update(p.Account, func(*sql.Tx) error {
		p.Account.RecordJourneyTaskEvent(s.Store.Rules, 49, 1, 0)
		return nil
	}); err != nil {
		return err
	}
	if err = s.Sync(p, "taskComp"); err != nil {
		return err
	}
	return s.guildBroadcast(id, "onGuildNewChat", message)
}
