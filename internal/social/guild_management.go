package social

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	"mmm-revived/internal/database"
	"mmm-revived/internal/model"
)

func (r *Repository) InviteToGuild(uid, target model.UserID) (model.GuildID, error) {
	return r.guildAction(uid, false, func(q *database.Queries, actor database.GuildMember) error {
		ctx, now := context.Background(), time.Now().Unix()
		exists, err := q.AccountExists(ctx, target)
		if err != nil {
			return err
		}
		if exists == 0 {
			return reject(CodeInvalidPlayer, "player does not exist")
		}
		if _, err = q.GetGuildMembership(ctx, target); err == nil {
			return reject(CodeTargetInOtherGuild, "target already belongs to a guild")
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		invites, err := q.ListPlayerGuildInvites(ctx, database.ListPlayerGuildInvitesParams{Player: target, After: now - r.GuildLimits.RequestLifetime})
		if err != nil {
			return err
		}
		if !slices.ContainsFunc(invites, func(invite database.ListPlayerGuildInvitesRow) bool { return invite.GuildID == actor.GuildID }) && len(invites) >= r.GuildLimits.Invites {
			return reject(CodeInviteLimit, "target guild invitation limit reached")
		}
		return q.InsertGuildInvite(ctx, database.InsertGuildInviteParams{GuildID: actor.GuildID, Player: target, Inviter: uid, Created: now})
	})
}

func (r *Repository) DecideGuildInvite(uid model.UserID, id model.GuildID, accept bool, level int) (model.GuildNews, error) {
	var news model.GuildNews
	err := r.transaction(func(tx *sql.Tx) error {
		q, ctx := r.db.Queries.WithTx(tx), context.Background()
		created, err := q.GetGuildInvite(ctx, database.GetGuildInviteParams{GuildID: id, Player: uid})
		if errors.Is(err, sql.ErrNoRows) || err == nil && created+r.GuildLimits.RequestLifetime <= time.Now().Unix() {
			return reject(CodeInviteInvalid, "guild invitation is missing or expired")
		}
		if err != nil {
			return err
		}
		if !accept {
			return q.DeleteGuildInvite(ctx, database.DeleteGuildInviteParams{GuildID: id, Player: uid})
		}
		guild, err := loadGuild(q, id)
		if err != nil {
			return err
		}
		news, err = r.joinGuild(q, guild, uid, level, CodeInAnotherGuild)
		return err
	})
	return news, err
}

func guildTarget(q *database.Queries, guild model.GuildID, uid, target model.UserID) (database.GuildMember, error) {
	member, err := membership(q, target, CodeTargetNotInGuild)
	if err == nil && (member.GuildID != guild || target == uid) {
		err = reject(CodeTargetNotInGuild, "target is not a member of this guild")
	}
	return member, err
}

func (r *Repository) SetGuildAdmin(uid, target model.UserID, admin bool) (model.GuildID, model.GuildNews, error) {
	var news model.GuildNews
	id, err := r.guildAction(uid, true, func(q *database.Queries, actor database.GuildMember) error {
		member, err := guildTarget(q, actor.GuildID, uid, target)
		if err != nil {
			return err
		}
		ctx := context.Background()
		role, kind := model.GuildMember, newsDemoted
		switch {
		case admin && member.Role == model.GuildManager:
			return reject(CodeTargetIsAdmin, "target is already a manager")
		case !admin && member.Role != model.GuildManager:
			return reject(CodeTargetNotAdmin, "target is not a manager")
		case admin:
			count, err := q.GuildAdminCount(ctx, actor.GuildID)
			if err != nil {
				return err
			}
			if count >= int64(r.GuildLimits.Admins) {
				return reject(CodeAdminLimit, "guild manager limit reached")
			}
			role, kind = model.GuildManager, newsPromoted
		}
		if err = q.SetGuildRole(ctx, database.SetGuildRoleParams{UID: target, Role: role}); err != nil {
			return err
		}
		news, err = insertNews(q, actor.GuildID, kind, uid, target, time.Now().Unix())
		return err
	})
	return id, news, err
}

func (r *Repository) TransferGuild(uid, target model.UserID) (model.GuildID, model.GuildNews, error) {
	var news model.GuildNews
	id, err := r.guildAction(uid, true, func(q *database.Queries, actor database.GuildMember) error {
		if _, err := guildTarget(q, actor.GuildID, uid, target); err != nil {
			return err
		}
		ctx := context.Background()
		if err := q.SetGuildRole(ctx, database.SetGuildRoleParams{UID: uid, Role: model.GuildMember}); err != nil {
			return err
		}
		if err := q.SetGuildRole(ctx, database.SetGuildRoleParams{UID: target, Role: model.GuildBoss}); err != nil {
			return err
		}
		if err := q.SetGuildBoss(ctx, database.SetGuildBossParams{ID: actor.GuildID, Boss: target}); err != nil {
			return err
		}
		var err error
		news, err = insertNews(q, actor.GuildID, newsTransferred, uid, target, time.Now().Unix())
		return err
	})
	return id, news, err
}

type GuildDeparture struct {
	ID       model.GuildID
	News     model.GuildNews
	Disbands bool
	Pending  []model.UserID
}

func (r *Repository) LeaveGuild(uid model.UserID) (GuildDeparture, error) {
	var result GuildDeparture
	err := r.transaction(func(tx *sql.Tx) error {
		q, ctx := r.db.Queries.WithTx(tx), context.Background()
		member, err := membership(q, uid, CodeAlreadyLeft)
		if err != nil {
			return err
		}
		result.ID = member.GuildID
		guild, err := q.GetGuild(ctx, result.ID)
		if err != nil {
			return err
		}
		result.Disbands = member.Role == model.GuildBoss
		if result.Disbands && guild.MemberCount > 1 {
			return reject(CodeTransferFirst, "transfer leadership before leaving")
		}
		if err = q.DeleteGuildMember(ctx, uid); err != nil {
			return err
		}
		now := time.Now().Unix()
		if err = q.SetGuildLeft(ctx, database.SetGuildLeftParams{UID: uid, Time: now}); err != nil {
			return err
		}
		if !result.Disbands {
			result.News, err = insertNews(q, result.ID, newsLeft, uid, 0, now)
			return err
		}
		applications, err := q.ListGuildApplications(ctx, database.ListGuildApplicationsParams{GuildID: result.ID})
		if err != nil {
			return err
		}
		invites, err := q.ListGuildInvites(ctx, database.ListGuildInvitesParams{GuildID: result.ID})
		if err != nil {
			return err
		}
		for _, application := range applications {
			result.Pending = append(result.Pending, application.Applicant)
		}
		for _, invite := range invites {
			result.Pending = append(result.Pending, invite.Player)
		}
		return q.DeleteGuild(ctx, result.ID)
	})
	return result, err
}

func (r *Repository) KickGuildMember(uid, target model.UserID) (model.GuildID, model.GuildNews, error) {
	var news model.GuildNews
	id, err := r.guildAction(uid, false, func(q *database.Queries, actor database.GuildMember) error {
		member, err := guildTarget(q, actor.GuildID, uid, target)
		if err != nil {
			return err
		}
		if member.Role == model.GuildBoss || actor.Role == model.GuildManager && member.Role != model.GuildMember {
			return reject(CodeNotAdmin, "cannot remove this guild member")
		}
		ctx, now := context.Background(), time.Now().Unix()
		if err = q.DeleteGuildMember(ctx, target); err != nil {
			return err
		}
		if err = q.SetGuildLeft(ctx, database.SetGuildLeftParams{UID: target, Time: now}); err != nil {
			return err
		}
		news, err = insertNews(q, actor.GuildID, newsKicked, uid, target, now)
		return err
	})
	return id, news, err
}

func (r *Repository) UpdateGuild(uid model.UserID, edit model.GuildSettings, bossOnly bool) (model.GuildInfo, []model.GuildNews, error) {
	var info model.GuildInfo
	var news []model.GuildNews
	_, err := r.guildAction(uid, bossOnly, func(q *database.Queries, actor database.GuildMember) error {
		ctx, now := context.Background(), time.Now().Unix()
		guild, err := q.GetGuild(ctx, actor.GuildID)
		if err != nil {
			return err
		}
		if info, err = r.mergeGuildSettings(guildInfo(guild), edit); err != nil {
			return err
		}
		if info.Name != guild.Name {
			if actor.Role != model.GuildBoss || guild.NameUpdated+r.GuildLimits.RenameCooldown > now {
				return reject(CodeCommon, "guild rename is unavailable")
			}
			if !strings.EqualFold(info.Name, guild.Name) {
				count, err := q.GuildNameExists(ctx, info.Name)
				if err != nil {
					return err
				}
				if count != 0 {
					return reject(CodeGuildNameTaken, "guild name is already taken")
				}
			}
			info.NameUpdated = now
		}
		if err = q.UpdateGuildSettings(ctx, database.UpdateGuildSettingsParams{ID: info.ID, Name: info.Name, SearchName: strings.ToLower(info.Name), Signature: info.Signature, Announcement: info.Announcement, Approval: int64(info.Approval), MinimumLevel: int64(info.MinimumLevel), Head: int64(info.Head), Frame: int64(info.Frame), NameUpdated: info.NameUpdated}); err != nil {
			return err
		}
		if info.Announcement == guild.Announcement {
			return nil
		}
		entry, err := insertNews(q, info.ID, newsAnnounced, uid, 0, now)
		news = append(news, entry)
		return err
	})
	return info, news, err
}
