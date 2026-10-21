package social

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"unicode/utf8"

	"mmm-revived/internal/database"
	"mmm-revived/internal/model"
)

func (r *Repository) validChat(kind int, text string) error {
	if kind != 0 && kind != 7 || text == "" || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return reject(CodeCommon, "invalid chat message")
	}
	if utf8.RuneCountInString(text) > r.Limits.Message {
		return reject(CodeMessageTooLong, "chat message is too long")
	}
	return nil
}

func insertPrivate(q *database.Queries, message model.ChatMessage) (model.ChatMessage, error) {
	id, err := q.InsertPrivateMessage(context.Background(), database.InsertPrivateMessageParams{Room: privateRoom(message.Sender, message.Receiver), Sender: message.Sender, Receiver: message.Receiver, Kind: int64(message.Kind), Content: message.Text, Created: message.Time})
	message.ID = strconv.FormatInt(id, 10)
	return message, err
}

func (r *Repository) SendPrivateTx(tx *sql.Tx, message model.ChatMessage) (model.ChatMessage, error) {
	if err := r.validChat(message.Kind, message.Text); err != nil {
		return message, err
	}
	q, ctx := r.db.Queries.WithTx(tx), context.Background()
	friend, err := q.FriendshipExists(ctx, database.FriendshipExistsParams{Owner: message.Sender, Peer: message.Receiver})
	if err != nil {
		return message, err
	}
	if friend == 0 {
		return message, reject(CodeNotFriend, "private chat requires a friend")
	}
	if err = checkBlocks(q, message.Sender, message.Receiver); err != nil {
		return message, err
	}
	return insertPrivate(q, message)
}

func (r *Repository) SendGiftsTx(tx *sql.Tx, uid model.UserID, peers []model.UserID) (int, error) {
	day := today()
	q, ctx := r.db.Queries.WithTx(tx), context.Background()
	sent, err := q.SentGiftCount(ctx, database.SentGiftCountParams{Day: day, Sender: uid})
	if err != nil {
		return 0, err
	}
	count := 0
	for _, peer := range peers {
		if sent >= int64(r.Limits.SentGifts) {
			break
		}
		friend, err := q.FriendshipExists(ctx, database.FriendshipExistsParams{Owner: uid, Peer: peer})
		if err != nil {
			return 0, err
		}
		exists, err := q.GiftExists(ctx, database.GiftExistsParams{Day: day, Sender: uid, Receiver: peer})
		if err != nil {
			return 0, err
		}
		if friend == 0 || exists != 0 {
			continue
		}
		if err = q.InsertGift(ctx, database.InsertGiftParams{Day: day, Sender: uid, Receiver: peer}); err != nil {
			return 0, err
		}
		sent++
		count++
	}
	return count, nil
}
