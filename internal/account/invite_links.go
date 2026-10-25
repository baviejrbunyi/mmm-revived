package account

import (
	"context"
	"database/sql"
	"errors"
	"maps"
	"slices"
	"strconv"
	"time"

	"mmm-revived/internal/database"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
)

func (s *Store) BindInvite(inviter, invitee model.UserID) error {
	if inviter <= 0 || invitee <= 0 || inviter == invitee {
		return rejected("invalid invitation binding")
	}
	ctx := context.Background()
	tx, err := s.Database.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.Database.Queries.WithTx(tx)
	for _, uid := range []model.UserID{inviter, invitee} {
		count, err := q.AccountExists(ctx, uid)
		if err != nil {
			return err
		}
		if count != 1 {
			return rejected("invitation account does not exist")
		}
	}
	if existing, err := q.GetInviter(ctx, invitee); err == nil {
		if existing.Inviter == inviter {
			return nil
		}
		return rejected("invitation already bound")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	inviterLevel, err := q.GetAccountLevel(ctx, inviter)
	if err != nil {
		return err
	}
	inviteeLevel, err := q.GetAccountLevel(ctx, invitee)
	if err != nil {
		return err
	}
	_, inviterRule, inviterKnown := s.Rules.ActivityOfType(inviterActivityType)
	_, inviteeRule, inviteeKnown := s.Rules.ActivityOfType(inviteeActivityType)
	if !inviterKnown || !inviteeKnown || inviterLevel < int64(inviterRule.Level) || inviteeLevel < int64(inviteeRule.Level) || inviteeLevel >= int64(s.Rules.Constants.Constants.InviteCloseBindLevel) {
		return rejected("invitation level requirements not met")
	}
	children, err := q.ListInvitees(ctx, inviter)
	if err != nil {
		return err
	}
	if len(children) >= s.Rules.RetailActivity.InviteMaximum {
		return rejected("invitation capacity reached")
	}
	seen := map[model.UserID]bool{invitee: true}
	for uid := inviter; uid != 0; {
		if seen[uid] || len(seen) > 1000 {
			return rejected("invitation cycle or excessive chain")
		}
		seen[uid] = true
		parent, err := q.GetInviter(ctx, uid)
		if errors.Is(err, sql.ErrNoRows) {
			break
		}
		if err != nil {
			return err
		}
		uid = parent.Inviter
	}
	if err = q.InsertInvite(ctx, database.InsertInviteParams{Invitee: invitee, Inviter: inviter, Bound: time.Now().Unix()}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Invitees(inviter model.UserID) ([]model.UserID, error) {
	return s.Database.Queries.ListInvitees(context.Background(), inviter)
}

func (s *Store) Inviter(invitee model.UserID) (model.UserID, int64, error) {
	row, err := s.Database.Queries.GetInviter(context.Background(), invitee)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, nil
	}
	return row.Inviter, row.Bound, err
}

func (s *Store) InviteProgress(inviter model.UserID) ([]model.InviteProgress, error) {
	rows, err := s.Database.Queries.GetInviteProgress(context.Background(), inviter)
	if err != nil {
		return nil, err
	}
	result := make([]model.InviteProgress, 0, len(rows))
	for _, row := range rows {
		progress := model.InviteProgress{UID: row.Invitee, Bound: row.Bound, Level: int(row.Level), Tasks: map[model.TaskID]int{}}
		if s.Rules != nil {
			records, err := s.Database.Queries.ListRecords(context.Background(), row.Invitee)
			if err != nil {
				return nil, err
			}
			for _, record := range records {
				id, err := strconv.ParseInt(record.ID, 10, 32)
				if record.Category != model.RecordTask || err != nil || !slices.Contains(slices.Collect(maps.Values(s.Rules.Journey.InviteTasks)), model.TaskID(id)) {
					continue
				}
				var task model.Task
				if err = json.Unmarshal(record.Payload, &task); err != nil {
					return nil, err
				}
				progress.Tasks[model.TaskID(id)] = task.Status
			}
		}
		result = append(result, progress)
	}
	return result, nil
}
