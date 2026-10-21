package duel

import (
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("duelFriend", rpc.Bind(DecodeChallenge, s.challenge))
	r.Handle("duelFriendReply", rpc.Bind(DecodeReply, s.reply))
}
