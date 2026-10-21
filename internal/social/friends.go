package social

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"mmm-revived/internal/database"
	"mmm-revived/internal/model"
)

const welcomeMessage = 1

func checkBlocks(q *database.Queries, uid, peer model.UserID) error {
	ctx := context.Background()
	blocked, err := q.BlockEither(ctx, database.BlockEitherParams{Owner: uid, Peer: peer})
	if err != nil || blocked == 0 {
		return err
	}
	mine, err := q.ListBlocks(ctx, uid)
	if err != nil {
		return err
	}
	if slices.Contains(mine, peer) {
		return reject(CodeTargetBlocked, "player is blacklisted")
	}
	return reject(CodeBlockedByTarget, "player blacklisted the sender")
}

func (r *Repository) checkCapacity(q *database.Queries, uid, peer model.UserID) error {
	for _, check := range []struct {
		uid  model.UserID
		code int
	}{{uid, CodeFriendsFull}, {peer, CodeTargetFriendsFull}} {
		count, err := q.FriendCount(context.Background(), check.uid)
		if err != nil {
			return err
		}
		if count >= int64(r.Limits.Friends) {
			return reject(check.code, "friend limit reached")
		}
	}
	return nil
}

func (r *Repository) Apply(sender, receiver model.UserID, source model.FriendSource) error {
	if sender == receiver {
		return reject(CodeInvalidPlayer, "cannot add yourself")
	}
	return r.transaction(func(tx *sql.Tx) error {
		q, ctx := r.db.Queries.WithTx(tx), context.Background()
		exists, err := q.AccountExists(ctx, receiver)
		if err != nil {
			return err
		}
		if exists == 0 {
			return reject(CodeInvalidPlayer, "player does not exist")
		}
		friend, err := q.FriendshipExists(ctx, database.FriendshipExistsParams{Owner: sender, Peer: receiver})
		if err != nil {
			return err
		}
		if friend != 0 {
			return reject(CodeAlreadyFriend, "players are already friends")
		}
		if err = checkBlocks(q, sender, receiver); err != nil {
			return err
		}
		pending, err := q.FriendRequestExists(ctx, database.FriendRequestExistsParams{Receiver: receiver, Sender: sender})
		if err != nil || pending != 0 {
			return err
		}
		if err = r.checkCapacity(q, sender, receiver); err != nil {
			return err
		}
		count, err := q.RequestCount(ctx, receiver)
		if err != nil {
			return err
		}
		if count >= int64(r.Limits.Applications) {
			return reject(CodeCommon, "application limit reached")
		}
		return q.InsertFriendRequest(ctx, database.InsertFriendRequestParams{Receiver: receiver, Sender: sender, Source: source, Created: time.Now().Unix()})
	})
}

func (r *Repository) Decide(receiver, sender model.UserID, accept bool) (model.ChatMessage, error) {
	var welcome model.ChatMessage
	err := r.transaction(func(tx *sql.Tx) error {
		var err error
		welcome, err = r.decide(tx, receiver, sender, accept)
		return err
	})
	return welcome, err
}

func (r *Repository) decide(tx *sql.Tx, receiver, sender model.UserID, accept bool) (model.ChatMessage, error) {
	q, ctx := r.db.Queries.WithTx(tx), context.Background()
	exists, err := q.FriendRequestExists(ctx, database.FriendRequestExistsParams{Receiver: receiver, Sender: sender})
	if err != nil {
		return model.ChatMessage{}, err
	}
	if exists == 0 {
		return model.ChatMessage{}, reject(CodeNoApplication, "friend request is missing")
	}
	if !accept {
		return model.ChatMessage{}, q.DeleteFriendRequest(ctx, database.DeleteFriendRequestParams{Receiver: receiver, Sender: sender})
	}
	if err = r.checkCapacity(q, receiver, sender); err != nil {
		return model.ChatMessage{}, err
	}
	now := time.Now().Unix()
	if err = q.InsertFriendPair(ctx, database.InsertFriendPairParams{Owner: receiver, Peer: sender, Added: now}); err != nil {
		return model.ChatMessage{}, err
	}
	if err = q.DeleteFriendRequestPair(ctx, database.DeleteFriendRequestPairParams{Receiver: receiver, Sender: sender}); err != nil {
		return model.ChatMessage{}, err
	}
	return insertPrivate(q, model.ChatMessage{Sender: receiver, Receiver: sender, Kind: welcomeMessage, Time: now})
}

func (r *Repository) RejectAll(uid model.UserID) error {
	return r.db.Queries.DeleteAllFriendRequests(context.Background(), uid)
}

func (r *Repository) Remove(uid model.UserID, peers []model.UserID) error {
	return r.transaction(func(tx *sql.Tx) error {
		q, ctx := r.db.Queries.WithTx(tx), context.Background()
		for _, peer := range peers {
			if err := q.DeleteFriendPair(ctx, database.DeleteFriendPairParams{Owner: uid, Peer: peer}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) Block(uid, peer model.UserID) error {
	if uid == peer {
		return reject(CodeInvalidPlayer, "cannot block yourself")
	}
	return r.transaction(func(tx *sql.Tx) error {
		q, ctx := r.db.Queries.WithTx(tx), context.Background()
		exists, err := q.AccountExists(ctx, peer)
		if err != nil {
			return err
		}
		if exists == 0 {
			return reject(CodeInvalidPlayer, "player does not exist")
		}
		blocked, err := q.ListBlocks(ctx, uid)
		if err != nil {
			return err
		}
		if slices.Contains(blocked, peer) {
			return nil
		}
		if len(blocked) >= r.Limits.Blocked {
			return reject(CodeCommon, "blacklist limit reached")
		}
		if err = q.InsertBlock(ctx, database.InsertBlockParams{Owner: uid, Peer: peer, Created: time.Now().Unix()}); err != nil {
			return err
		}
		if err = q.DeleteFriendPair(ctx, database.DeleteFriendPairParams{Owner: uid, Peer: peer}); err != nil {
			return err
		}
		if err = q.DeleteFriendRequestPair(ctx, database.DeleteFriendRequestPairParams{Receiver: uid, Sender: peer}); err != nil {
			return err
		}
		return q.AcknowledgePairMessages(ctx, database.AcknowledgePairMessagesParams{Sender: uid, Receiver: peer})
	})
}

func (r *Repository) Unblock(uid, peer model.UserID) error {
	return r.db.Queries.DeleteBlock(context.Background(), database.DeleteBlockParams{Owner: uid, Peer: peer})
}

func (r *Repository) Rename(uid, peer model.UserID, name string) error {
	count, err := r.db.Queries.SetFriendNickname(context.Background(), database.SetFriendNicknameParams{Owner: uid, Peer: peer, Nickname: name})
	if err == nil && count == 0 {
		return reject(CodeNotFriend, "player is not a friend")
	}
	return err
}

func (r *Repository) Find(uid, target model.UserID, name string, recommend bool) ([]model.UserID, error) {
	q, ctx := r.db.Queries, context.Background()
	var result []model.UserID
	var err error
	switch {
	case recommend:
		result, err = q.RecommendPlayers(ctx, uid)
	case target != 0:
		result, err = q.FindPlayerByID(ctx, database.FindPlayerByIDParams{Self: uid, Target: target})
	case name == "":
		result = []model.UserID{}
	default:
		result, err = q.FindPlayersByName(ctx, database.FindPlayersByNameParams{Self: uid, Pattern: searchPattern(name)})
	}
	if err != nil {
		return nil, err
	}
	if recommend {
		err = q.SetRecommendTime(ctx, database.SetRecommendTimeParams{UID: uid, Time: time.Now().Unix()})
	} else {
		err = q.SetSearchTime(ctx, database.SetSearchTimeParams{UID: uid, Time: time.Now().Unix()})
	}
	return result, err
}
