package account

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func (s *Store) refreshJourneyRelationalTasks(tx *sql.Tx, a *Account) error {
	if s.Rules == nil {
		return nil
	}
	queries := s.Database.Queries.WithTx(tx)
	ctx := context.Background()
	friends, err := queries.FriendCount(ctx, a.UID)
	if err != nil {
		return err
	}
	member, err := queries.GetGuildMembership(ctx, a.UID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	a.Journey.GuildMember = err == nil
	now := time.Now().Unix()
	a.refreshGuildTasks(s.Rules, now)
	if err == nil {
		if a.Journey.GuildJoined != member.Joined {
			a.Journey.GuildJoined = member.Joined
			a.Journey.GuildLoginDays = 0
			a.Journey.GuildLoginDay = 0
		}
		if a.Journey.GuildLoginDay != LogicDay(now) {
			a.Journey.GuildLoginDay = LogicDay(now)
			a.Journey.GuildLoginDays++
		}
	}
	invitees, err := queries.ListInvitees(ctx, a.UID)
	if err != nil {
		return err
	}
	for id, rule := range s.Rules.Tasks {
		switch rule.Type {
		case 33:
			a.UpdateTask(id, int(friends), rule.Target)
		case 48:
			if a.Journey.GuildMember {
				a.UpdateTask(id, 1, rule.Target)
			}
		case 86:
			a.UpdateTask(id, len(invitees), rule.Target)
		}
	}
	return nil
}
