package account

import (
	"context"
	"database/sql"
	"errors"

	"mmm-revived/internal/calendar"
)

func (s *Store) ConfigureCalendar(requested *int64) (int64, error) {
	ctx := context.Background()
	offset, err := s.Database.Queries.ReadClockOffset(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if requested != nil {
		offset = *requested
	}
	if err = calendar.SetOffset(offset); err != nil {
		return 0, err
	}
	if err = s.Database.Queries.SaveClockOffset(ctx, offset); err != nil {
		return 0, err
	}
	if s.Rules != nil {
		s.Rules.Draw.CalendarOffset = offset
		s.Rules.Journey.CalendarOffset = offset
		s.Rules.Events.CalendarOffset = offset
	}
	return offset, nil
}
