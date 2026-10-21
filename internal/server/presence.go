package server

import "time"

func (s *Service) announcePresence(p *Player, online bool) {
	state, err := s.Store.Social.Snapshot(p.Account.UID)
	if err != nil {
		p.Session.Logger.Error("load friends for presence", "error", err)
		return
	}
	now := time.Now().Unix()
	since, until := now, int64(0)
	if !online {
		since, until = min(p.Account.LastLogin, now-1), now
	}
	for uid := range state.Friends {
		friend := s.byUID[uid]
		if friend == nil || friend == p {
			continue
		}
		if err := friend.Send("onFriendPresence", p.Account.UID, since, until); err != nil {
			friend.Session.Logger.Error("send friend presence", "peer", p.Account.UID, "error", err)
		}
	}
}
