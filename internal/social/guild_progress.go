package social

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"maps"
	"slices"
	"time"

	"mmm-revived/internal/calendar"
	"mmm-revived/internal/database"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
)

const (
	newsLevelUp            int64 = 1
	guildExperienceRate    int64 = 5
	guildSettlementMail    int64 = 8011
	guildSettlementExpires int64 = 30 * daySeconds
)

type guildLevel struct {
	Experience  int64
	Members     int
	RewardItem  model.ItemID
	Minimum     int64
	PerReward   int64
	RewardLimit int64
}

var guildLevels = []guildLevel{
	{},
	{Experience: 2500, Members: 30, RewardItem: 800003, Minimum: 200, PerReward: 120, RewardLimit: 4},
	{Experience: 5000, Members: 32, RewardItem: 800003, Minimum: 200, PerReward: 120, RewardLimit: 4},
	{Experience: 8000, Members: 34, RewardItem: 800003, Minimum: 200, PerReward: 120, RewardLimit: 4},
	{Experience: 14000, Members: 36, RewardItem: 800003, Minimum: 200, PerReward: 120, RewardLimit: 5},
	{Experience: 18000, Members: 38, RewardItem: 800004, Minimum: 200, PerReward: 150, RewardLimit: 5},
	{Experience: 21000, Members: 40, RewardItem: 800004, Minimum: 200, PerReward: 150, RewardLimit: 5},
	{Experience: 27000, Members: 42, RewardItem: 800004, Minimum: 200, PerReward: 150, RewardLimit: 6},
	{Experience: 30000, Members: 44, RewardItem: 800004, Minimum: 200, PerReward: 150, RewardLimit: 6},
	{Experience: 36000, Members: 46, RewardItem: 800005, Minimum: 200, PerReward: 200, RewardLimit: 6},
	{Members: 50, RewardItem: 800005, Minimum: 200, PerReward: 200, RewardLimit: 7},
}

func levelRule(level int64) guildLevel {
	return guildLevels[min(max(level, 1), int64(len(guildLevels)-1))]
}

func (r *Repository) memberLimit(level int64) int {
	return max(r.GuildLimits.Members, levelRule(level).Members)
}

func (r *Repository) Contribute(tx *sql.Tx, uid model.UserID, amount, now int64) error {
	if amount <= 0 {
		return nil
	}
	q, ctx := r.db.Queries.WithTx(tx), context.Background()
	member, err := q.GetGuildMembership(ctx, uid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err = q.AddGuildContribution(ctx, database.AddGuildContributionParams{GuildID: member.GuildID, UID: uid, Week: calendar.LogicWeek(now), Score: amount}); err != nil {
		return err
	}
	if err = q.AddGuildHistoryScore(ctx, database.AddGuildHistoryScoreParams{Score: amount, UID: uid}); err != nil {
		return err
	}
	guild, err := q.GetGuild(ctx, member.GuildID)
	if err != nil {
		return err
	}
	level, experience := max(guild.Level, 1), guild.Experience+amount*guildExperienceRate
	var news []model.GuildNews
	for level < int64(len(guildLevels)-1) && experience >= guildLevels[level].Experience {
		experience -= guildLevels[level].Experience
		level++
		id, err := q.InsertGuildNews(ctx, database.InsertGuildNewsParams{GuildID: guild.ID, Kind: newsLevelUp, Actor: uid, Value: level, Created: now})
		if err != nil {
			return err
		}
		news = append(news, model.GuildNews{ID: id, Kind: int(newsLevelUp), Time: now, Parameters: []model.UserID{model.UserID(level)}})
	}
	if level == int64(len(guildLevels)-1) {
		experience = 0
	}
	if err = q.SetGuildProgress(ctx, database.SetGuildProgressParams{Level: level, Experience: experience, ID: guild.ID}); err != nil {
		return err
	}
	r.progress.note(guild.ID, news)
	return nil
}

func (r *Repository) weeklyScores(q *database.Queries, id model.GuildID, now int64) (map[model.UserID]int64, error) {
	week := calendar.LogicWeek(now)
	rows, err := q.ListGuildContributions(context.Background(), database.ListGuildContributionsParams{GuildID: id, FromWeek: week, ToWeek: week + 1})
	if err != nil {
		return nil, err
	}
	result := make(map[model.UserID]int64, len(rows))
	for _, row := range rows {
		result[row.UID] += row.Score
	}
	return result, nil
}

func (r *Repository) SettleGuildWeek(id model.GuildID, now int64) ([]model.UserID, error) {
	var mailed []model.UserID
	if id == 0 {
		return nil, nil
	}
	err := r.transaction(func(tx *sql.Tx) error {
		q, ctx := r.db.Queries.WithTx(tx), context.Background()
		week := calendar.LogicWeek(now)
		rows, err := q.ListGuildContributions(ctx, database.ListGuildContributionsParams{GuildID: id, FromWeek: 0, ToWeek: week})
		if err != nil || len(rows) == 0 {
			return err
		}
		guild, err := loadGuild(q, id)
		if err != nil {
			return err
		}
		rule := levelRule(guild.Level)
		var texts [3][]byte
		for index, text := range [...]string{"Alliance Rewards Settlement", "Thank you for contributing to your alliance last week! Here are some Dream Orbs as a reward.", "Official"} {
			if texts[index], err = json.Marshal(model.LocalizedText{"cn": text, "en": text}); err != nil {
				return err
			}
		}
		scores := map[[2]int64]int64{}
		for _, row := range rows {
			scores[[2]int64{row.Week, int64(row.UID)}] += row.Score
		}
		for _, key := range slices.SortedFunc(maps.Keys(scores), func(a, b [2]int64) int { return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1])) }) {
			score := scores[key]
			if score < rule.Minimum || rule.PerReward <= 0 {
				continue
			}
			rewards := min(score/rule.PerReward, rule.RewardLimit)
			if rewards <= 0 {
				continue
			}
			items, err := json.Marshal(map[model.ItemID]int64{rule.RewardItem: rewards})
			if err != nil {
				return err
			}
			owner := model.UserID(key[1])
			if _, err = q.DeliverMail(ctx, database.DeliverMailParams{Owner: owner, Title: texts[0], Content: texts[1], Sender: texts[2], Template: guildSettlementMail, Created: now, Expires: now + guildSettlementExpires, Items: items, HasReward: 1}); err != nil {
				return err
			}
			mailed = append(mailed, owner)
		}
		return q.DeleteGuildContributions(ctx, database.DeleteGuildContributionsParams{GuildID: id, Week: week})
	})
	return mailed, err
}

func (r *Repository) SettleMemberGuild(uid model.UserID) (model.GuildID, []model.UserID, error) {
	member, err := r.db.Queries.GetGuildMembership(context.Background(), uid)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, nil
	}
	if err != nil {
		return 0, nil, err
	}
	mailed, err := r.SettleGuildWeek(member.GuildID, time.Now().Unix())
	return member.GuildID, mailed, err
}
