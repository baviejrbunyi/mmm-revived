package server

import (
	"maps"

	"mmm-revived/internal/model"
)

func (s *Service) SendMailbox(p *Player) error {
	mails, err := s.Store.Mailbox(p.Account.UID)
	if err != nil {
		return err
	}
	if !p.mailReady {
		p.mailReady = true
		for id := range mails {
			p.mailSeen = max(p.mailSeen, id)
		}
	}
	maps.DeleteFunc(mails, func(id model.MailID, _ model.MailInfo) bool { return id > p.mailSeen })
	return p.Send("upMails", mails, 1)
}

func (s *Service) NotifyNewMail(owners ...model.UserID) {
	for _, uid := range owners {
		p := s.byUID[uid]
		if p == nil || !p.mailReady {
			continue
		}
		mails, err := s.Store.Mailbox(uid)
		if err == nil {
			maps.DeleteFunc(mails, func(id model.MailID, _ model.MailInfo) bool { return id <= p.mailSeen })
			if len(mails) == 0 {
				continue
			}
			for id := range mails {
				p.mailSeen = max(p.mailSeen, id)
			}
			err = p.Send("notifyNewMail", mails, 1)
		}
		if err != nil {
			p.Session.Logger.Warn("new mail notification failed", "uid", uid, "error", err)
		}
	}
}

func (s *Service) flushLive() {
	s.NotifyNewMail(s.Store.TakeMailed()...)
	s.pushGuildProgress()
}
