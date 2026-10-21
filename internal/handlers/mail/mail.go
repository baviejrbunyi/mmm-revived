package mail

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/database"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

type mailAction func(tx *sql.Tx, q *database.Queries, owner model.UserID, now int64) (map[model.ItemID]int64, error)

func (s service) claimMail(p *server.Player, ids []model.MailID) mailAction {
	return func(tx *sql.Tx, _ *database.Queries, _ model.UserID, _ int64) (map[model.ItemID]int64, error) {
		return s.Store.ClaimMail(tx, p.Account, ids)
	}
}

func readMail(id model.MailID) mailAction {
	return func(_ *sql.Tx, q *database.Queries, owner model.UserID, now int64) (map[model.ItemID]int64, error) {
		ctx := context.Background()
		row, err := q.GetMail(ctx, database.GetMailParams{Owner: owner, ID: id})
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: mail does not exist", account.ErrProgression)
		}
		if err != nil {
			return nil, err
		}
		if row.Expires > 0 && row.Expires <= now {
			return nil, fmt.Errorf("%w: mail expired", account.ErrProgression)
		}
		if row.HasReward == 0 {
			_, err = q.ClaimMail(ctx, database.ClaimMailParams{Owner: owner, ID: id, Now: now})
			return nil, err
		}
		return nil, q.OpenMail(ctx, database.OpenMailParams{Owner: owner, ID: id, Now: now})
	}
}

func removeMail(id model.MailID) mailAction {
	return func(_ *sql.Tx, q *database.Queries, owner model.UserID, now int64) (map[model.ItemID]int64, error) {
		ctx := context.Background()
		count, err := q.DeleteClaimedMail(ctx, database.DeleteClaimedMailParams{Owner: owner, ID: id, Now: now})
		if err != nil || count > 0 {
			return nil, err
		}
		if _, err = q.GetMail(ctx, database.GetMailParams{Owner: owner, ID: id}); errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		} else if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w: claim mail rewards before deleting", account.ErrProgression)
	}
}

func removeAllMail(_ *sql.Tx, q *database.Queries, owner model.UserID, now int64) (map[model.ItemID]int64, error) {
	return nil, q.DeleteAllClaimedMail(context.Background(), database.DeleteAllClaimedMailParams{Owner: owner, Now: now})
}

func (s service) mail(p *server.Player, action mailAction) error {
	before := *p.Account
	var quantities map[model.ItemID]int64
	err := s.Store.Update(p.Account, func(tx *sql.Tx) error {
		var err error
		quantities, err = action(tx, s.Store.Database.Queries.WithTx(tx), p.Account.UID, time.Now().Unix())
		return err
	})
	if err != nil {
		return err
	}
	if len(quantities) > 0 {
		if err = s.Sync(p, "itemComp", "heroComp", "taskComp"); err != nil {
			return err
		}
	}
	mails, err := s.Store.Mailbox(p.Account.UID)
	if err != nil {
		return err
	}
	if err = p.Send("upMails", mails, 1); err != nil {
		return err
	}
	if len(quantities) > 0 {
		return p.ShowRewards(s.ItemRewards(before, p.Account, quantities))
	}
	return nil
}
