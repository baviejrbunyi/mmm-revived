package matchmaking

import (
	"cmp"
	"errors"
	"slices"
	"time"

	"mmm-revived/internal/model"
)

const DefaultCapacity = 2048

var (
	errInvalidEntry = errors.New("invalid matchmaking entry")
	errQueued       = errors.New("player is already queued")
	errFull         = errors.New("matchmaking queue is full")
)

type Entry struct {
	Player   model.UserID
	Activity model.ActivityID
	Version  int
	Joined   time.Time
}

type Pair struct{ First, Second Entry }

type Queue struct {
	Capacity int
	entries  []Entry
	players  map[model.UserID]bool
	invites  Invites
}

func (q *Queue) Join(entry Entry) (*Pair, error) {
	if entry.Player == 0 || entry.Activity <= 0 || entry.Version < 0 {
		return nil, errInvalidEntry
	}
	if q.players[entry.Player] {
		return nil, errQueued
	}
	if i := slices.IndexFunc(q.entries, func(candidate Entry) bool {
		return candidate.Activity == entry.Activity && candidate.Version == entry.Version
	}); i >= 0 {
		candidate := q.entries[i]
		q.entries = slices.Delete(q.entries, i, i+1)
		delete(q.players, candidate.Player)
		return &Pair{First: candidate, Second: entry}, nil
	}
	if len(q.entries) >= cmp.Or(q.Capacity, DefaultCapacity) {
		return nil, errFull
	}
	if q.players == nil {
		q.players = map[model.UserID]bool{}
	}
	q.entries = append(q.entries, entry)
	q.players[entry.Player] = true
	return nil, nil
}

func (q *Queue) Cancel(player model.UserID) bool {
	if !q.players[player] {
		return false
	}
	i := slices.IndexFunc(q.entries, func(entry Entry) bool { return entry.Player == player })
	if i < 0 {
		return false
	}
	q.entries = slices.Delete(q.entries, i, i+1)
	delete(q.players, player)
	return true
}

func (q *Queue) Expire(before time.Time) []Entry {
	var expired []Entry
	q.entries = slices.DeleteFunc(q.entries, func(entry Entry) bool {
		if !entry.Joined.Before(before) {
			return false
		}
		delete(q.players, entry.Player)
		expired = append(expired, entry)
		return true
	})
	return expired
}

func (q *Queue) Entry(player model.UserID) (Entry, bool) {
	if !q.players[player] {
		return Entry{}, false
	}
	i := slices.IndexFunc(q.entries, func(entry Entry) bool { return entry.Player == player })
	if i < 0 {
		return Entry{}, false
	}
	return q.entries[i], true
}
