package social

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"mmm-revived/internal/database"
	"mmm-revived/internal/model"
)

const localWorld = "local-1"

func (r *Repository) JoinLocalWorld(uid model.UserID, channel string) (model.LocalWorldChannel, error) {
	if len(channel) > 64 || strings.ContainsAny(channel, "\x00\r\n") || !utf8.ValidString(channel) {
		return model.LocalWorldChannel{}, reject(CodeCommon, "invalid world channel")
	}
	ctx := context.Background()
	if err := r.db.Queries.SetWorldMembership(ctx, database.SetWorldMembershipParams{UID: uid, Channel: localWorld, Joined: time.Now().Unix()}); err != nil {
		return model.LocalWorldChannel{}, err
	}
	count, err := r.db.Queries.CountWorldMembers(ctx, localWorld)
	return model.LocalWorldChannel{ID: localWorld, SubID: localWorld, Name: "Local", Members: count}, err
}

type WorldMessage struct {
	ID      int64
	Sender  model.UserID
	Kind    int
	Text    string
	Created int64
}

func (r *Repository) SendWorldChat(uid model.UserID, kind int, text string, now int64) (WorldMessage, error) {
	message := WorldMessage{Sender: uid, Kind: kind, Text: text, Created: now}
	if err := r.validChat(kind, text); err != nil {
		return message, err
	}
	err := r.transaction(func(tx *sql.Tx) error {
		q, ctx := r.db.Queries.WithTx(tx), context.Background()
		var err error
		if message.ID, err = q.SaveWorldMessage(ctx, database.SaveWorldMessageParams{Channel: localWorld, Sender: uid, Kind: int64(kind), Content: text, Created: now}); err != nil {
			return err
		}
		return q.PruneWorldMessages(ctx, localWorld)
	})
	return message, err
}

func (r *Repository) WorldHistory() ([]WorldMessage, error) {
	rows, err := r.db.Queries.ListWorldMessages(context.Background(), localWorld)
	if err != nil {
		return nil, err
	}
	result := make([]WorldMessage, len(rows))
	for i, row := range rows {
		result[i] = WorldMessage{ID: row.ID, Sender: row.Sender, Kind: int(row.Kind), Text: row.Content, Created: row.Created}
	}
	slices.Reverse(result)
	return result, nil
}

func (r *Repository) WorldMembers() ([]model.UserID, error) {
	return r.db.Queries.ListWorldMembers(context.Background(), localWorld)
}
