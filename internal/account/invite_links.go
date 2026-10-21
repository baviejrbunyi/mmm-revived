package account

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"mmm-revived/internal/database"
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
	if inviterLevel < 15 || inviteeLevel < 8 || inviteeLevel > 16 {
		return rejected("invitation level requirements not met")
	}
	children, err := q.ListInvitees(ctx, inviter)
	if err != nil {
		return err
	}
	maximum := 10
	if s.Rules != nil && s.Rules.RetailActivity.InviteMaximum > 0 {
		maximum = s.Rules.RetailActivity.InviteMaximum
	}
	if len(children) >= maximum {
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
		result = append(result, model.InviteProgress{UID: row.Invitee, Bound: row.Bound, Level: int(row.Level), Story: row.Story != 0, Purchases: map[int]bool{3711: row.Purchase1 >= 3, 3712: row.Purchase2 >= 3, 3713: row.Purchase3 >= 3}})
	}
	return result, nil
}
