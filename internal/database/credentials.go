package database

import (
	"context"

	"mmm-revived/internal/model"
)

const GetCredential = `SELECT uid,version,memory_kib,iterations,parallelism,salt,digest,created FROM account_credentials WHERE uid=?1`

type Credential struct {
	UID         model.UserID
	Version     int64
	MemoryKiB   int64
	Iterations  int64
	Parallelism int64
	Salt        []byte
	Digest      []byte
	Created     int64
}

func (q *Queries) GetCredential(ctx context.Context, uid model.UserID) (Credential, error) {
	var result Credential
	err := q.db.QueryRowContext(ctx, GetCredential, uid).Scan(&result.UID, &result.Version, &result.MemoryKiB, &result.Iterations, &result.Parallelism, &result.Salt, &result.Digest, &result.Created)
	return result, err
}

const ClaimCredential = `INSERT INTO account_credentials(uid,version,memory_kib,iterations,parallelism,salt,digest,created) VALUES (?1,?2,?3,?4,?5,?6,?7,?8) ON CONFLICT(uid) DO NOTHING`

func (q *Queries) ClaimCredential(ctx context.Context, credential Credential) (int64, error) {
	result, err := q.db.ExecContext(ctx, ClaimCredential, credential.UID, credential.Version, credential.MemoryKiB, credential.Iterations, credential.Parallelism, credential.Salt, credential.Digest, credential.Created)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
