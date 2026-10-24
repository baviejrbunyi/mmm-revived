package server

import (
	"mmm-revived/internal/calendar"
	"mmm-revived/internal/model"
)

func (s *Service) SettleGuildWeek(id model.GuildID, now int64) error {
	mailed, err := s.Store.Social.SettleGuildWeek(id, now)
	if err != nil {
		return err
	}
	s.Store.NoteMail(mailed...)
	return nil
}

func (s *Service) guildWeek(p *Player, guild model.GuildComponent, now int64) error {
	if guild.ID == 0 {
		return nil
	}
	if err := s.SettleGuildWeek(guild.ID, now); err != nil || calendar.LogicWeekday(now) != 1 {
		return err
	}
	return p.Send("onGuildTaskRefresh")
}
