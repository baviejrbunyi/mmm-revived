package account

import (
	"context"
	"database/sql"
	"errors"
	"maps"

	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
)

func (s *Store) AccountByUID(uid model.UserID) (*Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	encoded, err := encodeProfileIDs([]model.UserID{uid})
	if err != nil {
		return nil, err
	}
	rows, err := s.Database.Queries.GetPublicProfiles(context.Background(), encoded)
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		return nil, sql.ErrNoRows
	}
	return s.restore(uid, rows[0].Profile)
}

func (s *Store) UpdatePlayers(players []*Account, operation func(*sql.Tx) error) error {
	if len(players) == 0 || len(players) > 2 {
		return errors.New("invalid account transaction size")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	backup := make(map[model.UserID][]byte, len(players))
	for _, a := range players {
		if a == nil || backup[a.UID] != nil {
			return errors.New("invalid account transaction participant")
		}
		data, err := json.Marshal(a)
		if err != nil {
			return err
		}
		backup[a.UID] = data
		if s.saved[a.UID] == nil {
			records, err := s.readRecords(a.UID)
			if err != nil {
				return err
			}
			s.saved[a.UID] = records
		}
	}
	tx, err := s.Database.SQL.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		tx.Rollback()
		if committed {
			return
		}
		for _, a := range players {
			var restored Account
			if json.Unmarshal(backup[a.UID], &restored) == nil {
				*a = restored
			}
		}
	}()
	if err = operation(tx); err != nil {
		return err
	}
	updated := make(map[model.UserID]map[recordKey][]byte, len(players))
	for _, a := range players {
		records, err := s.write(a, tx)
		if err != nil {
			return err
		}
		updated[a.UID] = records
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	maps.Copy(s.saved, updated)
	return nil
}
