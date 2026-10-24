package account

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	"mmm-revived/internal/database"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
)

func (s *Store) LoadGuildWar(tx *sql.Tx, uid model.UserID, war int, now int64) (model.GuildID, model.GuildWarState, error) {
	q, ctx := s.queries(tx), context.Background()
	member, err := q.GetGuildMembership(ctx, uid)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, model.GuildWarState{}, rejected("guild membership required")
	}
	if err != nil {
		return 0, model.GuildWarState{}, err
	}
	rule, known := s.Rules.Network.Wars[war]
	if !known || len(rule.Areas) != 4 {
		return 0, model.GuildWarState{}, rejected("unknown guild war")
	}
	var state model.GuildWarState
	encoded, err := q.GetGuildWar(ctx, database.GetGuildWarParams{GuildID: member.GuildID, War: int64(war)})
	if err == nil {
		if err = json.Unmarshal(encoded, &state); err != nil {
			return 0, state, err
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, state, err
	}
	migrateGuildWarDays(&state, now)
	if state.Endurance == nil {
		state.Areas = slices.Clone(rule.Areas[:3])
		state.Blocked = rule.Areas[3]
		state.Endurance = map[int]int{0: rule.BossHP}
		for _, area := range rule.Areas {
			state.Endurance[area] = s.Rules.Network.Areas[area].HP
		}
	}
	if state.Members == nil {
		state.Members = map[model.UserID]model.GuildWarMember{}
	}
	members, err := q.ListGuildMembers(ctx, member.GuildID)
	if err != nil {
		return 0, state, err
	}
	day := LogicDay(now)
	for _, entry := range members {
		value, ok := state.Members[entry.UID]
		if !ok {
			value = model.GuildWarMember{Tickets: 3, Day: day, Finished: map[int][]int{}, Unlocked: map[int]int{}}
		}
		if value.Day != day {
			value.Day = day
			value.Tickets = 3
		}
		if value.Finished == nil {
			value.Finished = map[int][]int{}
		}
		if value.Unlocked == nil {
			value.Unlocked = map[int]int{}
		}
		state.Members[entry.UID] = value
	}
	return member.GuildID, state, nil
}

func (s *Store) SaveGuildWar(tx *sql.Tx, guild model.GuildID, war int, state model.GuildWarState) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return s.queries(tx).SaveGuildWar(context.Background(), database.SaveGuildWarParams{GuildID: guild, War: int64(war), State: encoded})
}

func (s *Store) GuildWarLevelAllowed(a *Account, war, area, level int, state model.GuildWarState) (GuildWarLevelRule, error) {
	r, known := s.Rules.Network.WarLevels[level]
	set, valid := s.Rules.Network.WarSet(war, area, r.Group)
	if !known || !valid || a.Level < r.MinimumLevel || a.Level > r.MaximumLevel {
		return r, rejected("guild war difficulty unavailable")
	}
	m := state.Members[a.UID]
	if m.Tickets <= 0 {
		return r, rejected("no guild war attempts remain")
	}
	if area == 0 {
		if state.Endurance[state.Areas[0]] > 0 {
			return r, rejected("guild war boss is locked")
		}
		if len(m.Finished[0]) > 0 {
			return r, rejected("guild war boss already fought")
		}
		return r, nil
	}
	index := slices.Index(state.Areas, area)
	if index < 0 && (area != state.Blocked || state.Killed == 0) {
		return r, rejected("guild war area blocked")
	}
	if state.Killed == 0 {
		if index > 0 && state.Endurance[state.Areas[index-1]] > 0 {
			return r, rejected("guild war area locked")
		}
		if state.Endurance[area] == 0 {
			return r, rejected("guild war area cleared")
		}
		if set > max(1, m.Unlocked[area]) {
			return r, rejected("guild war group locked")
		}
	}
	for _, group := range s.Rules.Network.Areas[area].Sets[set-1] {
		if slices.Contains(m.Finished[area], group) {
			return r, rejected("guild war group set cleared")
		}
	}
	return r, nil
}

func (s *Store) GuildWarHistory(guild model.GuildID, war, area, group int) ([]model.GuildWarHistory, error) {
	rows, err := s.Database.Queries.ListGuildWarHistory(context.Background(), database.ListGuildWarHistoryParams{GuildID: guild, War: int64(war), Area: int64(area), GroupID: int64(group)})
	if err != nil {
		return nil, err
	}
	result := make([]model.GuildWarHistory, 0, len(rows))
	for i, row := range rows {
		var value model.GuildWarHistory
		if err = json.Unmarshal(row, &value); err != nil {
			return nil, err
		}
		value.Best = i == 0
		result = append(result, value)
	}
	return result, nil
}

type GuildWarProgress struct {
	Score      map[int]int64
	Assist     map[int]int
	UpdateTime map[int]int64
}

func (s *Store) RecordGuildWarProgress(tx *sql.Tx, guild model.GuildID, war int, uid model.UserID, area int, score int64, assist int, now int64) error {
	params := database.AddGuildWarProgressParams{GuildID: guild, War: int64(war), UID: uid, Area: int64(area), Score: score, Assist: int64(assist)}
	if score != 0 {
		params.ScoreTime = now
	}
	if assist != 0 {
		params.AssistTime = now
	}
	return s.queries(tx).AddGuildWarProgress(context.Background(), params)
}

func (s *Store) LoadGuildWarProgress(guild model.GuildID, war int) (map[model.UserID]GuildWarProgress, error) {
	rows, err := s.Database.Queries.ListGuildWarProgress(context.Background(), database.ListGuildWarProgressParams{GuildID: guild, War: int64(war)})
	if err != nil {
		return nil, err
	}
	result := map[model.UserID]GuildWarProgress{}
	for _, row := range rows {
		entry, ok := result[row.UID]
		if !ok {
			entry = GuildWarProgress{Score: map[int]int64{}, Assist: map[int]int{}, UpdateTime: map[int]int64{}}
			result[row.UID] = entry
		}
		area := int(row.Area)
		if row.Score != 0 {
			entry.Score[area] += row.Score
		}
		if row.Assist != 0 {
			entry.Assist[area] += int(row.Assist)
		}
		if updated := max(row.ScoreTime, row.AssistTime); updated > 0 {
			entry.UpdateTime[area] = max(entry.UpdateTime[area], updated)
		}
		if row.ScoreTime > 0 {
			entry.UpdateTime[-1] = max(entry.UpdateTime[-1], row.ScoreTime)
		}
		if row.AssistTime > 0 {
			entry.UpdateTime[-2] = max(entry.UpdateTime[-2], row.AssistTime)
		}
	}
	return result, nil
}
