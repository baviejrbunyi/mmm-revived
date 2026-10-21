package account

import (
	"context"
	"database/sql"

	"mmm-revived/internal/database"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
)

func (s *Store) DeliverGuildWarClear(tx *sql.Tx, guild model.GuildID, reward model.RewardID, now int64) error {
	if reward == 0 {
		return nil
	}
	items, err := rewardItems(s.Rules, reward, 1)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		return err
	}
	var texts [3][]byte
	for index, text := range [...]string{"Guild war area cleared", "Your guild cleared a battle area. Claim your contribution rewards.", "Local server"} {
		if texts[index], err = json.Marshal(model.LocalizedText{"cn": text, "en": text}); err != nil {
			return err
		}
	}
	q, ctx := s.Database.Queries.WithTx(tx), context.Background()
	members, err := q.ListGuildMembers(ctx, guild)
	if err != nil {
		return err
	}
	for _, member := range members {
		if _, err = q.DeliverMail(ctx, database.DeliverMailParams{Owner: member.UID, Title: texts[0], Content: texts[1], Sender: texts[2], Created: now, RewardID: reward, Items: encoded, HasReward: 1}); err != nil {
			return err
		}
	}
	return nil
}
