package social

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"mmm-revived/internal/database"
	"mmm-revived/internal/model"
)

type GuildLimits struct {
	CreationCost       int64
	Members            int
	Admins             int
	Applications       int
	PlayerApplications int
	Invites            int
	LeaveCooldown      int64
	RequestLifetime    int64
	RenameCooldown     int64
	MaximumLevel       int
}

var DefaultGuildLimits = GuildLimits{CreationCost: 1000, Members: 30, Admins: 5, Applications: 99, PlayerApplications: 5, Invites: 5, LeaveCooldown: 21600, RequestLifetime: 86400, RenameCooldown: 604800, MaximumLevel: 80}

const (
	newsCreated int64 = iota
	_
	newsJoined
	newsLeft
	newsKicked
	newsPromoted
	newsDemoted
	newsTransferred
	newsAnnounced
)

func validGuildText(text string, maximum int) bool {
	return utf8.ValidString(text) && !strings.ContainsRune(text, 0) && utf8.RuneCountInString(text) <= maximum
}

func validGuildName(name string) bool {
	return validGuildText(name, 14) && utf8.RuneCountInString(name) >= 2 && strings.ContainsFunc(name, func(r rune) bool { return !unicode.IsDigit(r) })
}

func (r *Repository) mergeGuildSettings(info model.GuildInfo, edit model.GuildSettings) (model.GuildInfo, error) {
	if edit.Name != nil {
		info.Name = strings.TrimSpace(*edit.Name)
	}
	if edit.Signature != nil {
		info.Signature = *edit.Signature
	}
	if edit.Announcement != nil {
		info.Announcement = *edit.Announcement
	}
	if edit.Approval != nil {
		info.Approval = *edit.Approval
	}
	if edit.MinimumLevel != nil {
		info.MinimumLevel = *edit.MinimumLevel
	}
	if edit.Head != nil {
		info.Head = *edit.Head
	}
	if edit.Frame != nil {
		info.Frame = *edit.Frame
	}
	switch {
	case !validGuildName(info.Name):
		return info, reject(CodeInvalidGuildName, "invalid guild name")
	case !validGuildText(info.Signature, 100):
		return info, reject(CodeInvalidSignature, "invalid guild signature")
	case !validGuildText(info.Announcement, 500):
		return info, reject(CodeInvalidAnnounce, "invalid guild announcement")
	case info.Approval < 0 || info.Approval > 2 || info.MinimumLevel < 0 || info.MinimumLevel > r.GuildLimits.MaximumLevel || info.Head < 0 || info.Frame < 0:
		return info, reject(CodeCommon, "invalid guild settings")
	}
	return info, nil
}

func guildInfo(row database.Guild) model.GuildInfo {
	return model.GuildInfo{ID: row.ID, Name: row.Name, Signature: row.Signature, Announcement: row.Announcement, Approval: int(row.Approval), MinimumLevel: int(row.MinimumLevel), Head: int(row.Head), Frame: int(row.Frame), Boss: row.Boss, Level: int(row.Level), Experience: row.Experience, Members: int(row.MemberCount), Created: row.Created, NameUpdated: row.NameUpdated, Server: 1}
}

func membership(q *database.Queries, uid model.UserID, code int) (database.GuildMember, error) {
	member, err := q.GetGuildMembership(context.Background(), uid)
	if errors.Is(err, sql.ErrNoRows) {
		return member, reject(code, "player does not belong to a guild")
	}
	return member, err
}

func guildAuthority(q *database.Queries, uid model.UserID, bossOnly bool) (database.GuildMember, error) {
	member, err := membership(q, uid, CodeNotInGuild)
	if err != nil {
		return member, err
	}
	switch {
	case member.Role == model.GuildBoss:
	case bossOnly:
		return member, reject(CodeNotBoss, "guild leader permission required")
	case member.Role != model.GuildManager:
		return member, reject(CodeNotAdmin, "guild manager permission required")
	}
	return member, nil
}

func loadGuild(q *database.Queries, id model.GuildID) (database.Guild, error) {
	guild, err := q.GetGuild(context.Background(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return guild, reject(CodeGuildDeleted, "guild does not exist")
	}
	return guild, err
}

func (r *Repository) guildAction(uid model.UserID, bossOnly bool, action func(*database.Queries, database.GuildMember) error) (model.GuildID, error) {
	var id model.GuildID
	err := r.transaction(func(tx *sql.Tx) error {
		q := r.db.Queries.WithTx(tx)
		actor, err := guildAuthority(q, uid, bossOnly)
		if err != nil {
			return err
		}
		id = actor.GuildID
		return action(q, actor)
	})
	return id, err
}

func insertNews(q *database.Queries, id model.GuildID, kind int64, actor, subject model.UserID, now int64) (model.GuildNews, error) {
	news := model.GuildNews{Kind: int(kind), Time: now, Parameters: []model.UserID{actor}}
	if subject != 0 {
		news.Parameters = append(news.Parameters, subject)
	}
	var err error
	news.ID, err = q.InsertGuildNews(context.Background(), database.InsertGuildNewsParams{GuildID: id, Kind: kind, Actor: actor, Subject: subject, Created: now})
	return news, err
}

func (r *Repository) ensureFreeGuild(q *database.Queries, uid model.UserID, code int) error {
	ctx := context.Background()
	if _, err := q.GetGuildMembership(ctx, uid); err == nil {
		return reject(code, "player already belongs to a guild")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	state, err := q.GetGuildPlayerState(ctx, uid)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if remaining := state.LastLeft + r.GuildLimits.LeaveCooldown - time.Now().Unix(); remaining > 0 {
		return reject(CodeJoinCooldown, "guild leave cooldown is active", remaining/3600, remaining%3600/60)
	}
	return nil
}

func (r *Repository) checkGuildEntry(guild database.Guild, level int) error {
	if guild.MemberCount >= int64(r.memberLimit(guild.Level)) {
		return reject(CodeGuildFull, "guild is full")
	}
	if int64(level) < guild.MinimumLevel {
		return reject(CodeLevelTooLow, "guild level requirement is not met")
	}
	return nil
}

func addGuildMember(q *database.Queries, id model.GuildID, uid model.UserID, role model.GuildRole, kind, now int64) (model.GuildNews, error) {
	ctx := context.Background()
	if err := q.InsertGuildMember(ctx, database.InsertGuildMemberParams{UID: uid, GuildID: id, Role: role, Joined: now}); err != nil {
		return model.GuildNews{}, err
	}
	if err := q.DeletePlayerGuildApplications(ctx, uid); err != nil {
		return model.GuildNews{}, err
	}
	if err := q.DeletePlayerGuildInvites(ctx, uid); err != nil {
		return model.GuildNews{}, err
	}
	return insertNews(q, id, kind, uid, 0, now)
}

func (r *Repository) CreateGuild(tx *sql.Tx, uid model.UserID, edit model.GuildSettings) (model.GuildID, error) {
	q, ctx := r.db.Queries.WithTx(tx), context.Background()
	if err := r.ensureFreeGuild(q, uid, CodeAlreadyInGuild); err != nil {
		return 0, err
	}
	info, err := r.mergeGuildSettings(model.GuildInfo{}, edit)
	if err != nil {
		return 0, err
	}
	count, err := q.GuildNameExists(ctx, info.Name)
	if err != nil {
		return 0, err
	}
	if count != 0 {
		return 0, reject(CodeGuildNameTaken, "guild name is already taken")
	}
	now := time.Now().Unix()
	id, err := q.CreateGuild(ctx, database.CreateGuildParams{Name: info.Name, SearchName: strings.ToLower(info.Name), Signature: info.Signature, Announcement: info.Announcement, Approval: int64(info.Approval), MinimumLevel: int64(info.MinimumLevel), Head: int64(info.Head), Frame: int64(info.Frame), Boss: uid, Created: now})
	if err != nil {
		return 0, err
	}
	_, err = addGuildMember(q, id, uid, model.GuildBoss, newsCreated, now)
	return id, err
}

func (r *Repository) joinGuild(q *database.Queries, guild database.Guild, uid model.UserID, level, code int) (model.GuildNews, error) {
	if err := r.ensureFreeGuild(q, uid, code); err != nil {
		return model.GuildNews{}, err
	}
	if err := r.checkGuildEntry(guild, level); err != nil {
		return model.GuildNews{}, err
	}
	return addGuildMember(q, guild.ID, uid, model.GuildMember, newsJoined, time.Now().Unix())
}

func (r *Repository) JoinGuild(uid model.UserID, id model.GuildID, level int) (model.GuildNews, bool, error) {
	var news model.GuildNews
	applied := false
	err := r.transaction(func(tx *sql.Tx) error {
		q, ctx := r.db.Queries.WithTx(tx), context.Background()
		if err := r.ensureFreeGuild(q, uid, CodeAlreadyInGuild); err != nil {
			return err
		}
		guild, err := loadGuild(q, id)
		if err != nil {
			return err
		}
		if guild.Approval == 2 {
			return reject(CodeInviteOnly, "guild requires an invitation")
		}
		if err = r.checkGuildEntry(guild, level); err != nil {
			return err
		}
		now := time.Now().Unix()
		if guild.Approval == 0 {
			news, err = addGuildMember(q, id, uid, model.GuildMember, newsJoined, now)
			return err
		}
		after := now - r.GuildLimits.RequestLifetime
		mine, err := q.ListPlayerGuildApplications(ctx, database.ListPlayerGuildApplicationsParams{Applicant: uid, After: after})
		if err != nil {
			return err
		}
		if slices.ContainsFunc(mine, func(application database.ListPlayerGuildApplicationsRow) bool { return application.GuildID == id }) {
			return nil
		}
		applications, err := q.ListGuildApplications(ctx, database.ListGuildApplicationsParams{GuildID: id, After: after})
		if err != nil {
			return err
		}
		if len(applications) >= r.GuildLimits.Applications || len(mine) >= r.GuildLimits.PlayerApplications {
			return reject(CodeApplicationLimit, "guild application limit reached")
		}
		applied = true
		return q.InsertGuildApplication(ctx, database.InsertGuildApplicationParams{GuildID: id, Applicant: uid, Created: now})
	})
	return news, applied, err
}

func (r *Repository) CancelGuildApplication(uid model.UserID, id model.GuildID) error {
	return r.db.Queries.DeleteGuildApplication(context.Background(), database.DeleteGuildApplicationParams{GuildID: id, Applicant: uid})
}

func (r *Repository) decideGuildApplication(q *database.Queries, id model.GuildID, target model.UserID, accept bool) (model.GuildNews, error) {
	ctx := context.Background()
	created, err := q.GetGuildApplication(ctx, database.GetGuildApplicationParams{GuildID: id, Applicant: target})
	if errors.Is(err, sql.ErrNoRows) || err == nil && created+r.GuildLimits.RequestLifetime <= time.Now().Unix() {
		return model.GuildNews{}, reject(CodeApplicationMissing, "guild application is missing or expired")
	}
	if err != nil {
		return model.GuildNews{}, err
	}
	if !accept {
		return model.GuildNews{}, q.DeleteGuildApplication(ctx, database.DeleteGuildApplicationParams{GuildID: id, Applicant: target})
	}
	guild, err := loadGuild(q, id)
	if err != nil {
		return model.GuildNews{}, err
	}
	level, err := q.PlayerLevel(ctx, target)
	if err != nil {
		return model.GuildNews{}, err
	}
	return r.joinGuild(q, guild, target, int(level), CodeTargetInOtherGuild)
}

func (r *Repository) DecideGuildApplications(uid, target model.UserID, accept bool) (model.GuildID, []model.UserID, []model.GuildNews, error) {
	var decided []model.UserID
	var news []model.GuildNews
	id, err := r.guildAction(uid, false, func(q *database.Queries, actor database.GuildMember) error {
		if target != 0 {
			entry, err := r.decideGuildApplication(q, actor.GuildID, target, accept)
			if err != nil {
				return err
			}
			decided = []model.UserID{target}
			if accept {
				news = append(news, entry)
			}
			return nil
		}
		applications, err := q.ListGuildApplications(context.Background(), database.ListGuildApplicationsParams{GuildID: actor.GuildID, After: time.Now().Unix() - r.GuildLimits.RequestLifetime})
		if err != nil {
			return err
		}
		for _, application := range applications {
			entry, err := r.decideGuildApplication(q, actor.GuildID, application.Applicant, accept)
			if errors.Is(err, ErrRejected) {
				continue
			}
			if err != nil {
				return err
			}
			decided = append(decided, application.Applicant)
			if accept {
				news = append(news, entry)
			}
		}
		return nil
	})
	return id, decided, news, err
}
