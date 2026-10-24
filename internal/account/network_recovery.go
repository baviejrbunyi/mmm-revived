package account

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"mmm-revived/internal/database"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
)

func (s *Store) RecoverNetworkBattles() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx := context.Background()
	tx, err := s.Database.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.Database.Queries.WithTx(tx)
	rows, err := q.ListPendingNetworkAccounts(ctx)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	today := LogicDay(now)
	for _, row := range rows {
		var network model.NetworkState
		if err = json.Unmarshal(row.Network, &network); err != nil {
			return err
		}
		pending := network.Pending
		if pending == nil {
			continue
		}
		if pending.Guild == 0 {
			if !pending.Friend {
				network.Attempts++
			}
		} else {
			encoded, err := q.GetGuildWar(ctx, database.GetGuildWarParams{GuildID: pending.Guild, War: int64(pending.War)})
			switch {
			case err == nil:
				var state model.GuildWarState
				if err = json.Unmarshal(encoded, &state); err != nil {
					return err
				}
				migrateGuildWarDays(&state, now)
				if member, exists := state.Members[row.UID]; exists && member.Day == today {
					member.Tickets = min(3, member.Tickets+1)
					state.Members[row.UID] = member
					if err = s.SaveGuildWar(tx, pending.Guild, pending.War, state); err != nil {
						return err
					}
				}
			case !errors.Is(err, sql.ErrNoRows):
				return err
			}
		}
		network.Pending = nil
		encoded, err := json.Marshal(network)
		if err != nil {
			return err
		}
		if err = q.SaveRecoveredNetwork(ctx, database.SaveRecoveredNetworkParams{UID: row.UID, Network: string(encoded)}); err != nil {
			return err
		}
	}
	return tx.Commit()
}
