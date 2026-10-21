package server

import (
	"context"
	"database/sql"
	"time"

	"mmm-revived/internal/account"
)

func (s *Service) Run(ctx context.Context) {
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
			s.mu.Unlock()
		case now := <-ticker.C:
			s.mu.Lock()
			s.refreshDispatches(now.Unix())
			s.expireInvites(now)
			s.refreshRetail(now.Unix())
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
			s.mu.Unlock()
		}
	}
}

func (s *Service) refreshDispatches(now int64) {
	if s.Store.Rules == nil || len(s.Store.Rules.DispatchMissions) == 0 {
		return
	}
	for _, p := range s.byUID {
		if p.Account.Level < 12 || p.Account.Dispatch.Initialized && p.Account.Dispatch.Day == now/86400 {
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
		if !p.Account.Retail {
			continue
		}
		rolled := p.Day != day
		var err error
		if rolled {
			p.Day = day
			err = s.Store.Update(p.Account, func(*sql.Tx) error { p.Account.RollRetailDay(s.Store.Rules, now); return nil })
			if err == nil {
				err = s.Sync(p, "taskComp", "itemComp", "activityComp", "bossComp")
			}
			if err == nil {
				err = p.Send("dailyTimer", account.DailyRefreshSeconds)
			}
			if err == nil {
				err = p.Send("resetDailyTasks")
			}
		} else if p.Account.RegenerateEnergy(s.Store.Rules, now) {
			err = s.Store.Save(p.Account)
			if err == nil {
				err = s.Sync(p, "energyComp")
			}
		}
		if err != nil {
			p.Session.Logger.Error("refresh retail account", "uid", p.Account.UID, "error", err)
		}
	}
}
