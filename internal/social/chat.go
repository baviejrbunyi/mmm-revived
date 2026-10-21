package social

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"mmm-revived/internal/database"
	"mmm-revived/internal/model"
)

func privateRoom(first, second model.UserID) string {
	return fmt.Sprintf("private:%d:%d", min(first, second), max(first, second))
}

func (r *Repository) Pending(uid model.UserID) (map[string][]model.ChatMessage, error) {
	rows, err := r.db.Queries.ListPendingMessages(context.Background(), database.ListPendingMessagesParams{Receiver: uid, MaxRows: int64(r.Limits.History)})
	if err != nil {
		return nil, err
	}
	result := map[string][]model.ChatMessage{}
	for _, row := range rows {
		key := strconv.FormatUint(uint64(row.Sender), 10)
		result[key] = append(result[key], model.ChatMessage{ID: strconv.FormatInt(row.ID, 10), Sender: row.Sender, Receiver: row.Receiver, Kind: int(row.Kind), Text: row.Content, Time: row.Created})
	}
	for _, messages := range result {
		slices.Reverse(messages)
	}
	return result, nil
}

func (r *Repository) Acknowledge(uid, peer model.UserID) error {
	ctx := context.Background()
	if peer == 0 {
		return r.db.Queries.AcknowledgeAllMessages(ctx, uid)
	}
	return r.db.Queries.AcknowledgePeerMessages(ctx, database.AcknowledgePeerMessagesParams{Receiver: uid, Sender: peer})
}
