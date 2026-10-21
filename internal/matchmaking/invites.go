package matchmaking

import (
	"maps"
	"time"

	"mmm-revived/internal/model"
)

type Invite struct {
	From    model.UserID
	To      model.UserID
	Mode    model.MatchMode
	Manual  bool
	Expires time.Time
}

func (i Invite) Involves(uid model.UserID) bool { return i.From == uid || i.To == uid }

type Invites struct{ pending map[model.UserID]Invite }

func (q *Queue) Invites() *Invites { return &q.invites }

func (i *Invites) Add(invite Invite) {
	if i.pending == nil {
		i.pending = map[model.UserID]Invite{}
	}
	i.pending[invite.To] = invite
}

func (i *Invites) Incoming(to model.UserID) (Invite, bool) {
	invite, ok := i.pending[to]
	return invite, ok
}

func (i *Invites) Take(from, to model.UserID) (Invite, bool) {
	invite, ok := i.pending[to]
	if !ok || invite.From != from {
		return Invite{}, false
	}
	delete(i.pending, to)
	return invite, true
}

func (i *Invites) Remove(match func(Invite) bool) []Invite {
	var removed []Invite
	maps.DeleteFunc(i.pending, func(_ model.UserID, invite Invite) bool {
		if !match(invite) {
			return false
		}
		removed = append(removed, invite)
		return true
	})
	return removed
}
