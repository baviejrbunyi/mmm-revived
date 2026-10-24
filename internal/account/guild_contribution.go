package account

import (
	"database/sql"
	"time"

	"mmm-revived/internal/model"
)

const guildContribution model.ItemID = 32

func (s *Store) creditGuildContribution(tx *sql.Tx, a *Account) error {
	amount, held := a.Items[guildContribution]
	if !held || s.Social == nil {
		return nil
	}
	delete(a.Items, guildContribution)
	return s.Social.Contribute(tx, a.UID, amount, time.Now().Unix())
}
