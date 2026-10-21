package account

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"mmm-revived/internal/database"
)

const MaximumPasswordBytes = 256

const passwordMemory = 64 * 1024
const passwordIterations = 3
const passwordParallelism = 4

var ErrAuthentication = errors.New("invalid username or password")
var ErrAuthenticationBusy = errors.New("authentication is busy")
var errCredentialChanged = errors.New("credential changed during authentication")
var passwordSlots = make(chan struct{}, 2)

func validPassword(password string) bool {
	return len(password) > 0 && len(password) <= MaximumPasswordBytes && utf8.ValidString(password) && !strings.ContainsRune(password, 0)
}

func passwordCredential(password string) (database.Credential, error) {
	credential := database.Credential{
		Version: argon2.Version, MemoryKiB: passwordMemory, Iterations: passwordIterations,
		Parallelism: passwordParallelism, Salt: make([]byte, 16), Created: time.Now().Unix(),
	}
	rand.Read(credential.Salt)
	credential.Digest = argon2.IDKey([]byte(password), credential.Salt, passwordIterations, passwordMemory, passwordParallelism, 32)
	return credential, nil
}

func verifyPassword(password string, credential database.Credential) bool {
	if credential.Version != argon2.Version || credential.MemoryKiB != passwordMemory || credential.Iterations != passwordIterations || credential.Parallelism != passwordParallelism || len(credential.Salt) != 16 || len(credential.Digest) != 32 {
		return false
	}
	digest := argon2.IDKey([]byte(password), credential.Salt, uint32(credential.Iterations), uint32(credential.MemoryKiB), uint8(credential.Parallelism), 32)
	return subtle.ConstantTimeCompare(digest, credential.Digest) == 1
}

func sameCredential(a, b database.Credential) bool {
	return a.UID == b.UID && a.Version == b.Version && a.MemoryKiB == b.MemoryKiB && a.Iterations == b.Iterations && a.Parallelism == b.Parallelism && subtle.ConstantTimeCompare(a.Salt, b.Salt) == 1 && subtle.ConstantTimeCompare(a.Digest, b.Digest) == 1
}

func (s *Store) Authenticate(name, password string) (*Account, error) {
	name = strings.TrimSpace(name)
	if len(name) == 0 || len(name) > 64 || !utf8.ValidString(name) || strings.ContainsAny(name, "\r\n\x00") || !validPassword(password) {
		return nil, ErrAuthentication
	}
	select {
	case passwordSlots <- struct{}{}:
		defer func() { <-passwordSlots }()
	default:
		return nil, ErrAuthenticationBusy
	}
	ctx := context.Background()
	var candidate *database.Credential
	for range 4 {
		row, err := s.Database.Queries.GetAccount(ctx, name)
		var verified *database.Credential
		if err == nil {
			credential, lookupErr := s.Database.Queries.GetCredential(ctx, row.UID)
			if lookupErr == nil {
				if !verifyPassword(password, credential) {
					return nil, ErrAuthentication
				}
				verified = &credential
			} else if !errors.Is(lookupErr, sql.ErrNoRows) {
				return nil, lookupErr
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if verified == nil && candidate == nil {
			value, err := passwordCredential(password)
			if err != nil {
				return nil, err
			}
			candidate = &value
		}
		s.mu.Lock()
		a, err := s.authenticateLocked(name, verified, candidate)
		s.mu.Unlock()
		if err == nil {
			return a, nil
		}
		if !errors.Is(err, errCredentialChanged) && !retryAuthentication(err) {
			return nil, err
		}
	}
	return nil, ErrAuthenticationBusy
}

func retryAuthentication(err error) bool {
	sqliteError, ok := errors.AsType[*sqlite.Error](err)
	if !ok {
		return false
	}
	code := sqliteError.Code()
	return code&255 == sqlite3.SQLITE_BUSY || code&255 == sqlite3.SQLITE_LOCKED || code == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY || code == sqlite3.SQLITE_CONSTRAINT_UNIQUE
}

func (s *Store) authenticateLocked(name string, verified, candidate *database.Credential) (*Account, error) {
	ctx := context.Background()
	row, err := s.Database.Queries.GetAccount(ctx, name)
	if errors.Is(err, sql.ErrNoRows) {
		if verified != nil || candidate == nil {
			return nil, errCredentialChanged
		}
		return s.createAccountWithCredential(name, candidate)
	}
	if err != nil {
		return nil, err
	}
	credential, err := s.Database.Queries.GetCredential(ctx, row.UID)
	if err == nil {
		if verified == nil || !sameCredential(credential, *verified) {
			return nil, errCredentialChanged
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	} else if verified != nil || candidate == nil {
		return nil, errCredentialChanged
	}
	a, err := s.restore(row.UID, row.Profile)
	if err != nil {
		return nil, err
	}
	tx, err := s.Database.SQL.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	q := s.Database.Queries.WithTx(tx)
	if verified != nil {
		current, err := q.GetCredential(ctx, a.UID)
		if err != nil || !sameCredential(current, *verified) {
			return nil, errCredentialChanged
		}
	} else {
		value := *candidate
		value.UID = a.UID
		count, err := q.ClaimCredential(ctx, value)
		if err != nil {
			return nil, err
		}
		if count != 1 {
			return nil, errCredentialChanged
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return a, nil
}
