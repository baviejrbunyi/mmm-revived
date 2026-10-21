package account

import (
	"context"
	"database/sql"
	"errors"
	"maps"
	"math"
	"slices"
	"time"

	"mmm-revived/internal/database"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
)

const MailboxLimit = 500

func decodeMail(row database.Mailbox) (model.MailInfo, map[model.ItemID]int64, error) {
	result := model.MailInfo{Status: row.Status, Expires: row.Expires, Sent: row.Created, Template: row.Template, From: 1, Reward: row.RewardID, Items: []model.MailAttachment{}}
	quantities := map[model.ItemID]int64{}
	for _, part := range []struct {
		data   []byte
		target any
	}{{row.Title, &result.Title}, {row.Content, &result.Content}, {row.Sender, &result.Sender}, {row.Items, &quantities}} {
		if err := json.Unmarshal(part.data, part.target); err != nil {
			return result, nil, err
		}
	}
	result.Items = make([]model.MailAttachment, 0, len(quantities))
	for _, id := range slices.Sorted(maps.Keys(quantities)) {
		result.Items = append(result.Items, model.MailAttachment{Item: id, Count: quantities[id], Attributes: "{}"})
	}
	return result, quantities, nil
}

func mailbox(q *database.Queries, uid model.UserID, now int64) (map[model.MailID]model.MailInfo, error) {
	rows, err := q.ListMailbox(context.Background(), database.ListMailboxParams{Owner: uid, Now: now, MaxRows: MailboxLimit})
	if err != nil {
		return nil, err
	}
	result := make(map[model.MailID]model.MailInfo, len(rows))
	for _, row := range rows {
		mail, _, err := decodeMail(row)
		if err != nil {
			return nil, err
		}
		result[row.ID] = mail
	}
	return result, nil
}

func (s *Store) Mailbox(uid model.UserID) (map[model.MailID]model.MailInfo, error) {
	return mailbox(s.Database.Queries, uid, time.Now().Unix())
}

func addQuantity(total map[model.ItemID]int64, item model.ItemID, amount int64) bool {
	if total[item] > math.MaxInt64-amount {
		return false
	}
	total[item] += amount
	return true
}

func (s *Store) ClaimMail(tx *sql.Tx, a *Account, ids []model.MailID) (map[model.ItemID]int64, error) {
	q, ctx := s.Database.Queries.WithTx(tx), context.Background()
	now := time.Now().Unix()
	quantities := map[model.ItemID]int64{}
	if ids == nil {
		rows, err := q.ListMailbox(ctx, database.ListMailboxParams{Owner: a.UID, Now: now, MaxRows: MailboxLimit})
		if err != nil {
			return nil, err
		}
		ids = make([]model.MailID, 0, len(rows))
		for _, row := range rows {
			if row.Status < model.MailRewarded {
				ids = append(ids, row.ID)
			}
		}
	}
	for _, id := range ids {
		row, err := q.GetMail(ctx, database.GetMailParams{Owner: a.UID, ID: id})
		if errors.Is(err, sql.ErrNoRows) {
			return nil, rejected("mail does not exist")
		}
		if err != nil {
			return nil, err
		}
		if row.Expires > 0 && row.Expires <= now {
			return nil, rejected("mail is expired")
		}
		if row.Status >= model.MailRewarded {
			continue
		}
		_, items, err := decodeMail(row)
		if err != nil {
			return nil, err
		}
		if row.RewardID != 0 {
			reward, err := rewardItems(s.Rules, row.RewardID, 1)
			if err != nil {
				return nil, err
			}
			for item, amount := range reward {
				if !addQuantity(items, item, amount) {
					return nil, rejected("mail reward overflow")
				}
			}
		}
		for item, amount := range items {
			if amount < 0 || !addQuantity(quantities, item, amount) {
				return nil, rejected("mail reward overflow")
			}
		}
		affected, err := q.ClaimMail(ctx, database.ClaimMailParams{Owner: a.UID, ID: id, Now: now})
		if err != nil {
			return nil, err
		}
		if affected != 1 {
			return nil, rejected("mail claim conflicted")
		}
	}
	if len(quantities) > 0 {
		if err := a.GrantItems(s.Rules, quantities); err != nil {
			return nil, err
		}
	}
	return quantities, nil
}
