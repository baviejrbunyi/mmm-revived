package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"

	"mmm-revived/internal/calendar"
	"mmm-revived/internal/database"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
)

const (
	bossRankMail      = 8001
	bossScoreMail     = 8002
	bossRankMigration = "boss-rank-%d-%d"
)

var bossRankTexts = [3]model.LocalizedText{
	{"cn": "Nightfall Ranking Reward", "en": "Nightfall Ranking Reward"},
	{"cn": "The Nightfall event has ended. Here is your ranking reward.", "en": "The Nightfall event has ended. Here is your ranking reward."},
	{"cn": "MARVEL Mystic Mayhem Team", "en": "MARVEL Mystic Mayhem Team"},
}

func (a *Account) settleBossScores(r *Rules) {
	boss := a.Events.Boss
	items := map[model.ItemID]int64{}
	for _, score := range slices.Sorted(maps.Keys(r.Events.BossScores)) {
		if int64(score) > boss.TotalScore || slices.Contains(boss.Rewards, score) {
			continue
		}
		if quantities, err := rewardItems(r, r.Events.BossScores[score], 1); err == nil {
			addItems(items, quantities)
		}
	}
	if len(items) > 0 {
		a.Calendar.Mail = append(a.Calendar.Mail, CalendarMail{Template: bossScoreMail, Items: items})
	}
}

func (s *Store) SettleBossRanks(now int64) error {
	if s.Rules == nil {
		return nil
	}
	moment, cycle := now+s.Rules.Events.CalendarOffset, calendar.Cycle()
	for _, id := range slices.Sorted(maps.Keys(s.Rules.Events.BossOpens)) {
		if s.Rules.Events.BossOpens[id].End > moment {
			continue
		}
		name := fmt.Sprintf(bossRankMigration, cycle, id)
		if s.bossDone[name] {
			continue
		}
		if err := s.settleBossOpen(name, id, now); err != nil {
			return err
		}
		if s.bossDone == nil {
			s.bossDone = map[string]bool{}
		}
		s.bossDone[name] = true
	}
	return nil
}

func (s *Store) settleBossOpen(name string, open int, now int64) error {
	ctx := context.Background()
	if _, err := s.Database.Queries.GetMigration(ctx, name); !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	tx, err := s.Database.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = s.mailBossRank(tx, open, now); err != nil {
		return err
	}
	if err = s.Database.Queries.WithTx(tx).SaveMigration(ctx, name); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) settleCutBossRanks(tx *sql.Tx, now int64) error {
	start, period := calendar.Loop()
	if s.Rules == nil || period <= 0 {
		return nil
	}
	end := start + period
	for _, id := range slices.Sorted(maps.Keys(s.Rules.Events.BossOpens)) {
		if open := s.Rules.Events.BossOpens[id]; open.Start < end && open.End > end {
			if err := s.mailBossRank(tx, id, now); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) mailBossRank(tx *sql.Tx, open int, now int64) error {
	ctx := context.Background()
	rows, err := tx.QueryContext(ctx, "SELECT uid,score FROM rank_entries WHERE mode=? AND rank_id=? ORDER BY score DESC,achieved,uid", string(model.RankBoss), open)
	if err != nil {
		return err
	}
	type entry struct {
		uid   model.UserID
		score int64
	}
	var board []entry
	for rows.Next() {
		var row entry
		if err = rows.Scan(&row.uid, &row.score); err != nil {
			rows.Close()
			return err
		}
		board = append(board, row)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	var texts [3][]byte
	for index, text := range bossRankTexts {
		if texts[index], err = json.Marshal(text); err != nil {
			return err
		}
	}
	q, total := s.queries(tx), int64(len(board))
	for index, row := range board {
		if row.score <= 0 {
			continue
		}
		claimed, err := bossRankClaimed(tx, row.uid, open)
		if err != nil {
			return err
		}
		if claimed {
			continue
		}
		position, reward := int64(index+1), model.RewardID(0)
		for _, id := range slices.Sorted(maps.Keys(s.Rules.Events.BossRanks)) {
			if rule := s.Rules.Events.BossRanks[id]; rule.Matches(position, total) {
				reward = rule.Reward
				break
			}
		}
		if reward == 0 {
			continue
		}
		items, err := rewardItems(s.Rules, reward, 1)
		if err != nil {
			return err
		}
		encoded, err := json.Marshal(items)
		if err != nil {
			return err
		}
		if _, err = s.deliverMail(q, ctx, database.DeliverMailParams{Owner: row.uid, Title: texts[0], Content: texts[1], Sender: texts[2], Template: bossRankMail, Created: now, Expires: now + 30*DaySeconds, RewardID: reward, Items: encoded, HasReward: 1}); err != nil {
			return err
		}
	}
	return nil
}

func bossRankClaimed(tx *sql.Tx, uid model.UserID, open int) (bool, error) {
	var profile []byte
	err := tx.QueryRowContext(context.Background(), "SELECT profile FROM accounts WHERE uid=?", uid).Scan(&profile)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	var state struct {
		Events model.EventProgress `json:"events"`
	}
	if err = json.Unmarshal(profile, &state); err != nil {
		return false, err
	}
	return slices.Contains(state.Events.Boss.RankClaimed, open), nil
}
