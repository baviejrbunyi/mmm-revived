package server

import (
	"errors"
	"time"

	"mmm-revived/internal/model"
)

const (
	duelTick      = 50 * time.Millisecond
	duelFrame     = time.Second / 30
	duelLead      = 2 * time.Second
	duelAhead     = 8
	duelSpacing   = 50 * time.Millisecond
	duelPlayCard  = 1
	duelArguments = 8
)

var ErrDuelOperation = errors.New("invalid manual duel operation")

type DuelClock struct {
	Start   time.Time
	Horizon int64
	Last    int64
	Sent    map[model.UserID]time.Time
}

func (r *Room) BeginDuel(now time.Time) {
	if r.Manual {
		r.Duel = DuelClock{Start: now.Add(duelLead), Horizon: -1, Sent: map[model.UserID]time.Time{}}
	}
}

func (r *Room) duelActive() bool {
	return r.Manual && r.Started && !r.Finished && !r.Duel.Start.IsZero()
}

func (s *Service) advanceDuels(now time.Time) {
	for _, room := range s.rooms {
		if !room.duelActive() {
			continue
		}
		frame := int64(now.Sub(room.Duel.Start)/duelFrame) + duelAhead
		horizon := max(frame, room.Duel.Last, room.Duel.Horizon)
		if horizon == room.Duel.Horizon {
			continue
		}
		room.Duel.Horizon = horizon
		for _, member := range room.Players {
			if err := member.Send("onDuelHorizon", horizon); err != nil {
				member.Session.Logger.Error("send duel horizon", "room", room.ID, "error", err)
			}
		}
	}
}

func (s *Service) RelayDuelOperation(p *Player, operation int64, arguments map[any]any) error {
	room := p.Room
	if room == nil || !room.duelActive() || operation != duelPlayCard || len(arguments) > duelArguments {
		return ErrDuelOperation
	}
	now := time.Now()
	if last, ok := room.Duel.Sent[p.Account.UID]; ok && now.Sub(last) < duelSpacing {
		return nil
	}
	room.Duel.Sent[p.Account.UID] = now
	frame := room.Duel.Horizon + 1
	room.Duel.Last = frame
	for _, member := range room.Players {
		if err := member.Send("onDuelOperation", frame, p.Account.UID, operation, arguments); err != nil {
			member.Session.Logger.Error("send duel operation", "room", room.ID, "error", err)
		}
	}
	return nil
}
