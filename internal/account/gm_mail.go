package account

import (
	"context"
	"database/sql"

	"mmm-revived/internal/database"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
)

func (s *Store) DeliverGMMail(tx *sql.Tx, uid model.UserID, template int, now int64) (model.MailID, error) {
	rule, known := s.Rules.GM.Mail[template]
	if !known {
		rule = GMMailRule{Title: "GM mail", Content: "Local GM message", Sender: "Local GM"}
	}
	values := map[model.ItemID]int64{}
	var err error
	switch {
	case rule.Reward != 0:
		if values, err = rewardItems(s.Rules, rule.Reward, 1); err != nil {
			return 0, err
		}
	case !known:
		values[1] = 1000
	}
	ctx := context.Background()
	q := s.Database.Queries.WithTx(tx)
	count, err := q.MailboxCount(ctx, uid)
	if err != nil {
		return 0, err
	}
	if count >= MailboxLimit {
		return 0, rejected("mailbox is full")
	}
	var payloads [4][]byte
	for index, value := range [...]any{model.LocalizedText{"cn": rule.Title, "en": rule.Title}, model.LocalizedText{"cn": rule.Content, "en": rule.Content}, model.LocalizedText{"cn": rule.Sender, "en": rule.Sender}, values} {
		if payloads[index], err = json.Marshal(value); err != nil {
			return 0, err
		}
	}
	hasReward := int64(0)
	if len(values) > 0 {
		hasReward = 1
	}
	return q.DeliverMail(ctx, database.DeliverMailParams{Owner: uid, Title: payloads[0], Content: payloads[1], Sender: payloads[2], Template: int64(template), Created: now, Expires: now + 30*86400, RewardID: rule.Reward, Items: payloads[3], HasReward: hasReward})
}
