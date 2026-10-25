package account

import (
	"database/sql"
	"time"
)

const guildScoreItemType = 1033

func (s *Store) creditGuildContribution(tx *sql.Tx, a *Account) error {
	if s.Social == nil || s.Rules == nil {
		return nil
	}
	var amount int64
	var held bool
	for id, count := range a.Items {
		if s.Rules.ItemRules[id].Type == guildScoreItemType {
			amount, held = amount+count, true
			delete(a.Items, id)
		}
	}
	if !held {
		return nil
	}
	return s.Social.Contribute(tx, a.UID, amount, time.Now().Unix())
}
