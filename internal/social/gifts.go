package social

import (
	"context"
	"database/sql"

	"mmm-revived/internal/database"
	"mmm-revived/internal/model"
)

func (r *Repository) ClaimGifts(tx *sql.Tx, uid model.UserID, peers []model.UserID) (int, error) {
	day := today()
	q, ctx := r.db.Queries.WithTx(tx), context.Background()
	received, err := q.ReceivedGiftCount(ctx, database.ReceivedGiftCountParams{Day: day, Receiver: uid})
	if err != nil {
		return 0, err
	}
	count := 0
	for _, peer := range peers {
		if received+int64(count) >= int64(r.Limits.ReceivedGifts) {
			break
		}
		eligible, err := q.ClaimableGift(ctx, database.ClaimableGiftParams{Day: day, Receiver: uid, Sender: peer})
		if err != nil {
			return 0, err
		}
		if eligible == 0 {
			continue
		}
		if err = q.MarkGiftClaimed(ctx, database.MarkGiftClaimedParams{Day: day, Receiver: uid, Sender: peer}); err != nil {
			return 0, err
		}
		count++
	}
	return count, nil
}
