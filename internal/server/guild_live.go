package server

import (
	"context"
	"log/slog"
)

func (s *Service) pushGuildProgress() {
	if s.Store.Social == nil {
		return
	}
	for _, change := range s.Store.Social.TakeGuildProgress() {
		members, err := s.Store.Database.Queries.ListGuildMembers(context.Background(), change.Guild)
		if err != nil {
			slog.Error("list guild members for progress", "guild", change.Guild, "error", err)
			continue
		}
		for _, member := range members {
			p := s.byUID[member.UID]
			if p == nil || p.properties == nil {
				continue
			}
			snapshot, err := s.Store.Social.GuildSnapshot(member.UID)
			if err == nil {
				err = p.Update([]string{"guildComp"}, snapshot)
			}
			for _, news := range change.News {
				if err == nil {
					err = p.Send("onGuildNewsUpdate", news)
				}
			}
			if err != nil {
				p.Session.Logger.Warn("guild progress update failed", "uid", member.UID, "error", err)
			}
		}
	}
}
