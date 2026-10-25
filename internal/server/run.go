package server

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/calendar"
)

func (s *Service) Run(ctx context.Context) {
	if s.Store.Social != nil {
		if err := s.Store.Social.ClosePresence(time.Now().Unix()); err != nil {
			slog.Error("close stale presence", "error", err)
		}
	}
	ticker := time.NewTicker(s.config.TickInterval)
	defer ticker.Stop()
	duels := time.NewTicker(duelTick)
	defer duels.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-duels.C:
			s.mu.Lock()
			s.advanceDuels(now)
			s.flushLive()
			s.mu.Unlock()
		case now := <-ticker.C:
			s.mu.Lock()
			s.refreshCalendar(now.Unix())
			s.refreshDispatches(now.Unix())
			s.expireInvites(now)
			s.refreshRetail(now.Unix())
			s.expireCosmetics(now.Unix())
			if err := s.Store.SettleBossRanks(now.Unix()); err != nil {
				slog.Error("settle boss ranks", "error", err)
			}
			for _, entry := range s.queue.Expire(now.Add(-s.config.QueueTimeout)) {
				if p := s.byUID[entry.Player]; p != nil {
					p.State = Lobby
					if err := s.MatchStatus(p, 0); err != nil {
						p.Session.Logger.Error("expire matchmaking status", "error", err)
					}
					if err := p.Send("onRealtimePvpMatchFailed"); err != nil {
						p.Session.Logger.Error("expire matchmaking notification", "error", err)
					}
				}
			}
			for _, match := range s.rooms {
				if match.Finished || !now.After(match.Deadline) {
					continue
				}
				if match.Expire != nil && !match.Started && len(match.Ready) == 2 {
					err := match.Expire(match)
					if err == nil {
						continue
					}
					match.Players[0].Session.Logger.Error("expire match round", "room", match.ID, "error", err)
				}
				if err := s.FinishMatch(match, 0); err != nil {
					match.Players[0].Session.Logger.Error("finish expired match", "room", match.ID, "error", err)
				}
			}
			s.flushLive()
			s.mu.Unlock()
		}
	}
}

func (s *Service) refreshCalendar(now int64) {
	cycle, err := s.Store.AdvanceCalendar(now)
	if err != nil {
		slog.Error("advance content calendar", "cycle", cycle, "error", err)
	}
	if s.Store.Rules == nil {
		return
	}
	for _, p := range s.byUID {
		if !p.Account.CalendarStale(s.Store.Rules, now) {
			continue
		}
		wrapped := p.Account.Calendar.Cycle != cycle
		err := s.Store.Update(p.Account, func(*sql.Tx) error { return p.Account.RollDay(s.Store.Rules, now) })
		if err == nil {
			err = s.Sync(p, "taskComp", "itemComp", "activityComp", "challengeComp", "bossComp", "seasonComp", "shopComp", "battlepassComp")
		}
		if err == nil && wrapped {
			err = SyncTime(p)
		}
		if err != nil {
			p.Session.Logger.Error("advance account calendar", "uid", p.Account.UID, "cycle", cycle, "error", err)
		}
	}
}

func (s *Service) refreshDispatches(now int64) {
	if s.Store.Rules == nil || len(s.Store.Rules.DispatchMissions) == 0 {
		return
	}
	for _, p := range s.byUID {
		if !p.Account.DispatchUnlocked(s.Store.Rules) || p.Account.Dispatch.Initialized && p.Account.Dispatch.Day == account.LogicDay(now) {
			continue
		}
		err := s.Store.Update(p.Account, func(*sql.Tx) error { return p.Account.PrepareDispatch(s.Store.Rules, now) })
		if err == nil {
			err = s.Sync(p, "dispatchComp")
		}
		if err == nil && p.State == Lobby {
			err = p.Send("onDispatchRefreshFinish")
		}
		if err != nil {
			p.Session.Logger.Error("refresh dispatches", "uid", p.Account.UID, "error", err)
		}
	}
}

func (s *Service) refreshRetail(now int64) {
	if s.Store.Rules == nil {
		return
	}
	day := account.LogicDay(now)
	for _, p := range s.byUID {
		var err error
		if p.Day != day {
			p.Day = day
			err = s.rollDay(p, now)
		} else if p.Account.RegenerateEnergy(s.Store.Rules, now) {
			err = s.Store.Save(p.Account)
			if err == nil {
				err = s.Sync(p, "energyComp")
			}
		}
		if err != nil {
			p.Session.Logger.Error("refresh account day", "uid", p.Account.UID, "error", err)
		}
	}
}

func (s *Service) rollDay(p *Player, now int64) error {
	a := p.Account
	if err := s.Store.Update(a, func(*sql.Tx) error {
		a.RegenerateEnergy(s.Store.Rules, now)
		return a.RollDay(s.Store.Rules, now)
	}); err != nil {
		return err
	}
	if err := s.Sync(p, "taskComp", "itemComp", "activityComp", "challengeComp", "bossComp", "seasonComp", "energyComp", "shopComp", "dispatchComp", "battlepassComp", "monthlyPassComp"); err != nil {
		return err
	}
	if err := p.Update([]string{"rogueComp"}, a.RogueComponent(false)); err != nil {
		return err
	}
	if err := p.Update([]string{"roguePveComp"}, a.RogueComponent(true)); err != nil {
		return err
	}
	friends, err := s.Store.Social.Snapshot(a.UID)
	if err != nil {
		return err
	}
	if err = p.Update([]string{"friendComp"}, friends); err != nil {
		return err
	}
	guild, err := s.Store.Social.GuildSnapshot(a.UID)
	if err != nil {
		return err
	}
	if err = p.Update([]string{"guildComp"}, guild); err != nil {
		return err
	}
	if err = s.guildWeek(p, guild, now); err != nil {
		return err
	}
	if err = p.Send("dailyTimer", calendar.RefreshSeconds); err != nil {
		return err
	}
	if err = p.Send("resetDailyTasks"); err != nil {
		return err
	}
	if a.Journey.Monthly.Expires > now {
		return p.Send("onMonthlyPassDailyRefresh")
	}
	return nil
}
