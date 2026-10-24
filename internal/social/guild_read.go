package social

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"mmm-revived/internal/calendar"
	"mmm-revived/internal/database"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
)

func weekStart(now int64) int64 {
	return (calendar.LogicWeek(now)*7-3)*daySeconds + calendar.RefreshSeconds
}

func (r *Repository) GuildSnapshot(uid model.UserID) (model.GuildComponent, error) {
	result := model.GuildComponent{Invites: map[model.GuildID]int64{}, Applications: map[model.GuildID]int64{}, Members: []model.UserID{}}
	ctx, q, now := context.Background(), r.db.Queries, time.Now().Unix()
	member, err := q.GetGuildMembership(ctx, uid)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	if err == nil {
		guild, err := q.GetGuild(ctx, member.GuildID)
		if err != nil {
			return result, err
		}
		result.ID, result.Title, result.Name = guild.ID, member.Role, guild.Name
		result.Level, result.Experience, result.NameUpdated = int(guild.Level), guild.Experience, guild.NameUpdated
		members, err := q.ListGuildMembers(ctx, guild.ID)
		if err != nil {
			return result, err
		}
		result.Members = make([]model.UserID, len(members))
		for i, entry := range members {
			result.Members[i] = entry.UID
		}
		result.Head, result.Frame = int(guild.Head), int(guild.Frame)
		result.TaskRefreshed = weekStart(now)
		result.WarID, result.WarPhase = CurrentWar(now)
		if result.WarID != 0 {
			encoded, err := q.GetGuildWar(ctx, database.GetGuildWarParams{GuildID: guild.ID, War: int64(result.WarID)})
			if err == nil {
				var state model.GuildWarState
				if err = json.Unmarshal(encoded, &state); err != nil {
					return result, err
				}
				result.BossKilled = state.Killed
			} else if !errors.Is(err, sql.ErrNoRows) {
				return result, err
			}
		}
	}
	after := now - r.GuildLimits.RequestLifetime
	applications, err := q.ListPlayerGuildApplications(ctx, database.ListPlayerGuildApplicationsParams{Applicant: uid, After: after})
	if err != nil {
		return result, err
	}
	for _, entry := range applications {
		result.Applications[entry.GuildID] = entry.Created
	}
	invites, err := q.ListPlayerGuildInvites(ctx, database.ListPlayerGuildInvitesParams{Player: uid, After: after})
	if err != nil {
		return result, err
	}
	for _, entry := range invites {
		result.Invites[entry.GuildID] = entry.Created
	}
	state, err := q.GetGuildPlayerState(ctx, uid)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	result.LastLeft, result.LastSearch = state.LastLeft, state.LastSearch
	return result, err
}

func (r *Repository) GuildInfos(ids []model.GuildID) (map[model.GuildID]any, error) {
	encoded, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Queries.GetGuilds(context.Background(), string(encoded))
	if err != nil {
		return nil, err
	}
	result := make(map[model.GuildID]any, len(ids))
	for _, id := range ids {
		result[id] = model.Empty{}
	}
	for _, row := range rows {
		result[row.ID] = guildInfo(row)
	}
	return result, nil
}

func (r *Repository) GuildMembers(uid model.UserID) (model.GuildMembersInfo, error) {
	q := r.db.Queries
	result := model.GuildMembersInfo{Members: map[model.UserID]model.GuildMemberInfo{}, Admins: []model.UserID{}}
	member, err := membership(q, uid, CodeNotInGuild)
	if err != nil {
		return result, err
	}
	rows, err := q.ListGuildMembers(context.Background(), member.GuildID)
	if err != nil {
		return result, err
	}
	scores, err := r.weeklyScores(q, member.GuildID, time.Now().Unix())
	if err != nil {
		return result, err
	}
	for _, entry := range rows {
		result.Members[entry.UID] = model.GuildMemberInfo{Joined: entry.Joined, Online: entry.Online, Offline: entry.Offline, Score: scores[entry.UID], TotalScore: entry.HistoryScore}
		switch entry.Role {
		case model.GuildBoss:
			result.Boss = entry.UID
		case model.GuildManager:
			result.Admins = append(result.Admins, entry.UID)
		}
	}
	return result, nil
}

func (r *Repository) GuildApplications(uid model.UserID) (map[model.UserID]int64, error) {
	q := r.db.Queries
	member, err := guildAuthority(q, uid, false)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListGuildApplications(context.Background(), database.ListGuildApplicationsParams{GuildID: member.GuildID, After: time.Now().Unix() - r.GuildLimits.RequestLifetime})
	if err != nil {
		return nil, err
	}
	result := make(map[model.UserID]int64, len(rows))
	for _, entry := range rows {
		result[entry.Applicant] = entry.Created
	}
	return result, nil
}

func (r *Repository) GuildInvitations(uid model.UserID) (map[model.UserID]int64, error) {
	q := r.db.Queries
	member, err := guildAuthority(q, uid, false)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListGuildInvites(context.Background(), database.ListGuildInvitesParams{GuildID: member.GuildID, After: time.Now().Unix() - r.GuildLimits.RequestLifetime})
	if err != nil {
		return nil, err
	}
	result := make(map[model.UserID]int64, len(rows))
	for _, entry := range rows {
		result[entry.Player] = entry.Created
	}
	return result, nil
}

func (r *Repository) FindGuilds(uid model.UserID, id model.GuildID, name string, level int, recommend bool) ([]model.GuildID, error) {
	ctx, q := context.Background(), r.db.Queries
	var ids []model.GuildID
	var err error
	switch {
	case recommend:
		ids, err = q.RecommendGuilds(ctx, database.RecommendGuildsParams{Level: int64(level), Capacity: int64(r.GuildLimits.Members)})
	case id != 0:
		var guild database.Guild
		guild, err = q.GetGuild(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			ids, err = []model.GuildID{}, nil
		} else if err == nil {
			ids = []model.GuildID{guild.ID}
		}
	case name == "":
		ids = []model.GuildID{}
	default:
		ids, err = q.FindGuildsByName(ctx, searchPattern(name))
	}
	if err != nil {
		return nil, err
	}
	if !recommend {
		err = q.SetGuildSearch(ctx, database.SetGuildSearchParams{UID: uid, Time: time.Now().Unix()})
	}
	return ids, err
}

func (r *Repository) GuildNews(uid model.UserID) ([]model.GuildNews, error) {
	q := r.db.Queries
	member, err := membership(q, uid, CodeNotInGuild)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListGuildNews(context.Background(), member.GuildID)
	if err != nil {
		return nil, err
	}
	result := make([]model.GuildNews, len(rows))
	for i, row := range rows {
		parameters := []model.UserID{row.Actor}
		switch {
		case row.Kind == newsLevelUp:
			parameters = []model.UserID{model.UserID(row.Value)}
		case row.Subject != 0:
			parameters = append(parameters, row.Subject)
		}
		result[i] = model.GuildNews{ID: row.ID, Kind: int(row.Kind), Time: row.Created, Parameters: parameters}
	}
	return result, nil
}

func (r *Repository) SendGuildChat(uid model.UserID, kind int, text string) (model.GuildID, model.GuildChatMessage, error) {
	var guild model.GuildID
	now := time.Now()
	message := model.GuildChatMessage{Sender: uid, Kind: kind, Text: text, Time: now.Unix(), Timestamp: calendar.ClientTime(now.Unix())*1000 + int64(now.Nanosecond()/1e6), Poster: strconv.FormatUint(uint64(uid), 10)}
	if err := r.validChat(kind, text); err != nil {
		return guild, message, err
	}
	err := r.transaction(func(tx *sql.Tx) error {
		q := r.db.Queries.WithTx(tx)
		member, err := membership(q, uid, CodeNotInGuild)
		if err != nil {
			return err
		}
		guild = member.GuildID
		message.ID, err = q.InsertGuildMessage(context.Background(), database.InsertGuildMessageParams{GuildID: guild, Sender: uid, Kind: int64(kind), Content: text, Created: message.Time})
		return err
	})
	return guild, message, err
}

func (r *Repository) GuildChatHistory(uid model.UserID) ([]model.GuildChatMessage, error) {
	q := r.db.Queries
	member, err := membership(q, uid, CodeNotInGuild)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListGuildMessages(context.Background(), database.ListGuildMessagesParams{GuildID: member.GuildID, MaxRows: int64(r.Limits.History)})
	if err != nil {
		return nil, err
	}
	result := make([]model.GuildChatMessage, len(rows))
	for index, row := range rows {
		result[len(rows)-1-index] = model.GuildChatMessage{ID: row.ID, Sender: row.Sender, Kind: int(row.Kind), Text: row.Content, Time: row.Created, Timestamp: calendar.ClientTime(row.Created) * 1000, Poster: strconv.FormatUint(uint64(row.Sender), 10)}
	}
	return result, nil
}

func (r *Repository) ReportGuild(uid model.UserID, id model.GuildID, reason, content string) error {
	return r.db.Queries.InsertGuildReport(context.Background(), database.InsertGuildReportParams{Reporter: uid, GuildID: id, Reason: reason, Content: content, Created: time.Now().Unix()})
}
