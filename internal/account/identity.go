package account

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"mmm-revived/internal/database"
	"mmm-revived/internal/model"
)

const NativeUIDMigration = "native-uid32-v1"

func (s *Store) migrateNativeUIDs() error {
	ctx := context.Background()
	if _, err := s.Database.Queries.GetMigration(ctx, NativeUIDMigration); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	tx, err := s.Database.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.Database.Queries.WithTx(tx)
	ids, err := q.ListAccountUIDs(ctx)
	if err != nil {
		return err
	}
	next := model.UserID(100000)
	for _, id := range ids {
		if id <= math.MaxInt32 {
			next = max(next, id)
		}
	}
	for _, id := range ids {
		if id <= math.MaxInt32 {
			continue
		}
		next++
		if next > math.MaxInt32 {
			return errors.New("native user ID range is exhausted")
		}
		if err = q.MigrateAccountUID(ctx, database.MigrateAccountUIDParams{NewUID: next, OldUID: id}); err != nil {
			return err
		}
	}
	if err = q.SaveMigration(ctx, NativeUIDMigration); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	s.saved = map[model.UserID]map[recordKey][]byte{}
	return nil
}

func (s *Store) createAccount(name string) (*Account, error) {
	return s.createAccountWithCredential(name, nil)
}

func (s *Store) createAccountWithCredential(name string, credential *database.Credential) (*Account, error) {
	ctx := context.Background()
	tx, err := s.Database.SQL.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	q := s.Database.Queries.WithTx(tx)
	next, err := q.NextAccountUID(ctx)
	if err != nil {
		return nil, err
	}
	if next < 1 || next > math.MaxInt32 {
		return nil, errors.New("native user ID range is exhausted")
	}
	created := time.Now().Unix()
	var a Account
	if s.Fresh && s.Rules != nil {
		a = NewRetailAccount(model.UserID(next), name, created, s.Rules, s.Catalog)
	} else {
		a = NewAccount(model.UserID(next), name, created, s.Catalog)
	}
	a.LastLogin = a.CreatedAt
	a.Initialize(s.Catalog, s.Rules)
	if err = a.PrepareDispatch(s.Rules, a.LastLogin); err != nil {
		return nil, err
	}
	if a.Retail {
		err = a.BeginRetail(s.Rules)
	} else {
		err = a.ApplyStart(s.Rules, s.Catalog, s.Start, a.CreatedAt)
	}
	if err != nil {
		return nil, err
	}
	s.saved[a.UID] = map[recordKey][]byte{}
	records, err := s.write(&a, tx)
	if err != nil {
		delete(s.saved, a.UID)
		return nil, err
	}
	if credential != nil {
		credential.UID = a.UID
		count, claimErr := q.ClaimCredential(ctx, *credential)
		if claimErr != nil || count != 1 {
			delete(s.saved, a.UID)
			if claimErr != nil {
				return nil, claimErr
			}
			return nil, errCredentialChanged
		}
	}
	if err = tx.Commit(); err != nil {
		delete(s.saved, a.UID)
		return nil, err
	}
	s.saved[a.UID] = records
	return &a, nil
}
