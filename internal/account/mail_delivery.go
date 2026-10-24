package account

import (
	"context"
	"maps"
	"slices"

	"mmm-revived/internal/database"
	"mmm-revived/internal/model"
)

func (s *Store) deliverMail(q *database.Queries, ctx context.Context, mail database.DeliverMailParams) (model.MailID, error) {
	id, err := q.DeliverMail(ctx, mail)
	if err == nil {
		s.NoteMail(mail.Owner)
	}
	return id, err
}

func (s *Store) NoteMail(owners ...model.UserID) {
	s.mailMu.Lock()
	defer s.mailMu.Unlock()
	if s.mailed == nil {
		s.mailed = map[model.UserID]struct{}{}
	}
	for _, uid := range owners {
		s.mailed[uid] = struct{}{}
	}
}

func (s *Store) TakeMailed() []model.UserID {
	s.mailMu.Lock()
	defer s.mailMu.Unlock()
	if len(s.mailed) == 0 {
		return nil
	}
	owners := slices.Sorted(maps.Keys(s.mailed))
	clear(s.mailed)
	return owners
}
