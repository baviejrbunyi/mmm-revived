package invite

import (
	"database/sql"
	"fmt"

	"mmm-revived/internal/account"
	"mmm-revived/internal/handlers/journey"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

func (s service) claimInviteList(p *server.Player, claim InviteListClaim) error {
	if err := s.inviteOpen(p); err != nil {
		return err
	}
	rows, err := s.Store.InviteProgress(p.Account.UID)
	if err != nil {
		return err
	}
	for _, proof := range rows {
		if proof.UID == claim.Target {
			return s.journeyInvite(p, func(a *account.Account, rules *account.Rules) (map[model.ItemID]int64, error) {
				return a.ClaimJourneyInviteList(rules, claim.Target, claim.Indices, proof)
			})
		}
	}
	return errInviteUnavailable
}

var errInviteUnavailable = fmt.Errorf("%w: invitation action is unavailable", account.ErrProgression)

func (s service) inviteOpen(p *server.Player) error {
	if s.Store.Rules == nil || p.State != server.Lobby {
		return errInviteUnavailable
	}
	return nil
}

func (s service) bindInvite(p *server.Player, code string) error {
	if err := s.inviteOpen(p); err != nil {
		return err
	}
	inviter, err := account.ParseJourneyInviteCode(code)
	if err != nil {
		return p.Send("onBindTargetFailed")
	}
	if err = s.Store.BindInvite(inviter, p.Account.UID); err != nil {
		return p.Send("onBindTargetFailed")
	}
	if err = s.Sync(p, "inviteComp", "taskComp"); err != nil {
		return err
	}
	if other := s.Online(inviter); other != nil {
		return s.Sync(other, "inviteComp", "taskComp")
	}
	return nil
}

func (s service) claimInvite(p *server.Player, r journey.JourneyPair) error {
	if err := s.inviteOpen(p); err != nil {
		return err
	}
	rows, err := s.Store.InviteProgress(p.Account.UID)
	if err != nil {
		return err
	}
	target := model.UserID(r.ID)
	var proof model.InviteProgress
	for _, row := range rows {
		if row.UID == target {
			proof = row
			break
		}
	}
	if proof.UID == 0 {
		return fmt.Errorf("%w: player is not bound to this inviter", account.ErrProgression)
	}
	return s.journeyInvite(p, func(a *account.Account, rules *account.Rules) (map[model.ItemID]int64, error) {
		return a.ClaimJourneyInvite(rules, target, r.Node, proof)
	})
}

func (s service) journeyInvite(p *server.Player, action func(*account.Account, *account.Rules) (map[model.ItemID]int64, error)) error {
	if err := s.inviteOpen(p); err != nil {
		return err
	}
	var rewards []model.ItemReward
	err := s.Store.Update(p.Account, func(*sql.Tx) error {
		before := *p.Account
		quantities, err := action(p.Account, s.Store.Rules)
		if err != nil {
			return err
		}
		rewards = s.ItemRewards(before, p.Account, quantities)
		return nil
	})
	if err != nil {
		return err
	}
	if err = s.Sync(p, "inviteComp", "taskComp", "itemComp", "heroComp"); err != nil {
		return err
	}
	return journey.RewardsOrAcknowledge(p)(rewards)
}
