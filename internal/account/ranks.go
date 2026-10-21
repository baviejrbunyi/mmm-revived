package account

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"mmm-revived/internal/database"
	"mmm-revived/internal/model"
)

func (s *Store) queries(tx *sql.Tx) *database.Queries {
	if tx == nil {
		return s.Database.Queries
	}
	return s.Database.Queries.WithTx(tx)
}

func (s *Store) HistorySnapshot(a *Account) (model.HistoryComponent, error) {
	rows, err := s.Database.Queries.ListPlayerRanks(context.Background(), a.UID)
	result := model.HistoryComponent{Ranks: make(map[string]model.RankEntry, len(rows)), Cooldowns: a.Network.RankCooldowns}
	if err != nil {
		return result, err
	}
	for _, row := range rows {
		result.Ranks["rank:"+string(row.Mode)+":"+strconv.FormatInt(row.RankID, 10)] = model.RankEntry{Score: row.Score, Achieved: row.Achieved}
	}
	return result, nil
}

func (s *Store) RecordRank(tx *sql.Tx, uid model.UserID, mode model.RankMode, rank int, score, now int64) error {
	if uid <= 0 || rank < 0 || score < 0 {
		return rejected("invalid rank entry")
	}
	return s.queries(tx).SaveRankEntry(context.Background(), database.SaveRankEntryParams{UID: uid, Mode: mode, RankID: int64(rank), Score: score, Achieved: now})
}

func (s *Store) RankPosition(uid model.UserID, mode model.RankMode, rank int) (int64, int64, error) {
	q, ctx := s.Database.Queries, context.Background()
	entry, err := q.GetRankEntry(ctx, database.GetRankEntryParams{UID: uid, Mode: mode, RankID: int64(rank)})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	higher, err := q.RankHigherCount(ctx, database.RankHigherCountParams{Mode: mode, RankID: int64(rank), Score: entry.Score, Achieved: entry.Achieved, UID: uid})
	if err != nil {
		return 0, 0, err
	}
	total, err := q.RankBoardSize(ctx, database.RankBoardSizeParams{Mode: mode, RankID: int64(rank)})
	return higher + 1, total, err
}

func (s *Store) PersonalRanks(uid model.UserID, mode model.RankMode, rank int, guild bool) (map[model.UserID]model.RankEntry, error) {
	q, ctx := s.Database.Queries, context.Background()
	result := map[model.UserID]model.RankEntry{}
	if guild {
		rows, err := q.ListGuildRanks(ctx, database.ListGuildRanksParams{UID: uid, Mode: mode, RankID: int64(rank)})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if row.UID != uid {
				result[row.UID] = model.RankEntry{Score: row.Score, Achieved: row.Achieved}
			}
		}
	} else {
		rows, err := q.ListFriendRanks(ctx, database.ListFriendRanksParams{Owner: uid, Mode: mode, RankID: int64(rank)})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result[row.UID] = model.RankEntry{Score: row.Score, Achieved: row.Achieved}
		}
	}
	return result, nil
}
