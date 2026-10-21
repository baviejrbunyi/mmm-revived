package mail

import (
	"mmm-revived/internal/model"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("readMail", rpc.Bind(DecodeMailID, func(p *server.Player, id model.MailID) error { return s.mail(p, readMail(id)) }))
	r.Handle("removeMail", rpc.Bind(DecodeMailID, func(p *server.Player, id model.MailID) error { return s.mail(p, removeMail(id)) }))
	r.Handle("removeAllMail", rpc.Bare(func(p *server.Player) error { return s.mail(p, removeAllMail) }))
	r.Handle("getMailReward", rpc.Bind(DecodeMailID, func(p *server.Player, id model.MailID) error {
		return s.mail(p, s.claimMail(p, []model.MailID{id}))
	}))
	r.Handle("getMailRewardAll", rpc.Bare(func(p *server.Player) error { return s.mail(p, s.claimMail(p, nil)) }))
}
