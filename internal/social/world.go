package social

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"mmm-revived/internal/database"
	"mmm-revived/internal/model"
)

const localWorld = "local-1"

func (r *Repository) JoinLocalWorld(uid model.UserID, channel string) (model.LocalWorldChannel, error) {
	if channel == "" {
		channel = localWorld
	}
	if len(channel) > 64 || strings.ContainsAny(channel, "\x00\r\n") || !utf8.ValidString(channel) {
		return model.LocalWorldChannel{}, reject(CodeCommon, "invalid world channel")
	}
	if channel != localWorld && channel != "1" {
		return model.LocalWorldChannel{}, reject(CodeCommon, "unknown local world channel")
	}
	ctx := context.Background()
	if err := r.db.Queries.SetWorldMembership(ctx, database.SetWorldMembershipParams{UID: uid, Channel: localWorld, Joined: time.Now().Unix()}); err != nil {
		return model.LocalWorldChannel{}, err
	}
	count, err := r.db.Queries.CountWorldMembers(ctx, localWorld)
	return model.LocalWorldChannel{ID: localWorld, SubID: localWorld, Name: "Local", Members: count}, err
}
