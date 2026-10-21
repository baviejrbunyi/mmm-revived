package duel

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/handlers/realtimepvp"
	"mmm-revived/internal/matchmaking"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

const (
	inviteLifetime = 60 * time.Second
	minimumHeroes  = 3
	randomHeroStar = 3
)

func (s service) challenge(p *server.Player, r Challenge) error {
	result, err := s.offer(p, r)
	if err != nil {
		return err
	}
	return server.NotifyDuel(p, r.Target, result)
}

func (s service) offer(p *server.Player, r Challenge) (model.DuelResult, error) {
	if friends, err := s.friends(p.Account.UID, r.Target); err != nil || !friends {
		return model.DuelNotFriend, err
	}
	target := s.Online(r.Target)
	if target == nil {
		return model.DuelOffline, nil
	}
	invites := s.Queue().Invites()
	if incoming, ok := invites.Incoming(r.Target); ok && incoming.From != p.Account.UID || !available(p) || !available(target) {
		return model.DuelBusy, nil
	}
	if _, result := s.pools(r.Mode, p, target); result != model.DuelSent {
		return result, nil
	}
	for _, previous := range invites.Remove(func(invite matchmaking.Invite) bool { return invite.From == p.Account.UID }) {
		if invitee := s.Online(previous.To); invitee != nil {
			if err := server.NotifyDuel(invitee, p.Account.UID, model.DuelCancelled); err != nil {
				invitee.Session.Logger.Error("notify withdrawn duel", "peer", p.Account.UID, "error", err)
			}
		}
	}
	invites.Add(matchmaking.Invite{From: p.Account.UID, To: r.Target, Mode: r.Mode, Manual: r.Manual, Expires: time.Now().Add(inviteLifetime)})
	if err := target.Send("onFriendDuelInvite", p.Account.UID, cmp.Or(p.Account.TeamName, p.Account.Name), int(r.Mode), r.Manual); err != nil {
		target.Session.Logger.Error("send duel invitation", "peer", p.Account.UID, "error", err)
		invites.Take(p.Account.UID, r.Target)
		return model.DuelOffline, nil
	}
	return model.DuelSent, nil
}

func (s service) reply(p *server.Player, r Reply) error {
	invite, ok := s.Queue().Invites().Take(r.Target, p.Account.UID)
	if !ok {
		return server.NotifyDuel(p, r.Target, model.DuelExpired)
	}
	inviter := s.Online(invite.From)
	if inviter == nil {
		return server.NotifyDuel(p, invite.From, model.DuelCancelled)
	}
	if !r.Accept {
		if err := server.NotifyDuel(inviter, p.Account.UID, model.DuelDeclined); err != nil {
			inviter.Session.Logger.Error("notify declined duel", "peer", p.Account.UID, "error", err)
		}
		return server.Acknowledge(p)
	}
	if !available(inviter) || !available(p) {
		return server.NotifyDuel(p, invite.From, model.DuelBusy)
	}
	friends, err := s.friends(invite.From, p.Account.UID)
	if err != nil {
		return err
	}
	heroes, result := s.pools(invite.Mode, inviter, p)
	if !friends {
		result = model.DuelNotFriend
	}
	if result != model.DuelSent {
		s.CloseInvites([]matchmaking.Invite{invite}, result, result)
		return nil
	}
	return realtimepvp.StartMatch(s.Service, [2]*server.Player{inviter, p}, account.DefaultPVPActivity, invite.Mode, invite.Manual, heroes)
}

func (s service) friends(uid, peer model.UserID) (bool, error) {
	if uid == peer {
		return false, nil
	}
	for _, pair := range [...][2]model.UserID{{uid, peer}, {peer, uid}} {
		snapshot, err := s.Store.Social.Snapshot(pair[0])
		if err != nil {
			return false, err
		}
		if _, ok := snapshot.Friends[pair[1]]; !ok || slices.Contains(snapshot.Blocked, pair[1]) {
			return false, nil
		}
	}
	return true, nil
}

func available(p *server.Player) bool {
	return p.Room == nil && (p.State == server.Lobby || p.State == server.Queued)
}

func (s service) pools(mode model.MatchMode, players ...*server.Player) (map[model.UserID]map[model.HeroID]model.Hero, model.DuelResult) {
	heroes := make(map[model.UserID]map[model.HeroID]model.Hero, len(players))
	for _, p := range players {
		pool, result := s.pool(mode, p.Account)
		if result != model.DuelSent {
			return nil, result
		}
		heroes[p.Account.UID] = pool
	}
	return heroes, model.DuelSent
}

func (s service) pool(mode model.MatchMode, a *account.Account) (map[model.HeroID]model.Hero, model.DuelResult) {
	rule := s.Store.Rules.PVP[account.DefaultPVPActivity]
	heroes := map[model.HeroID]model.Hero{}
	switch mode {
	case model.MatchDraft:
		if a.Activities[account.DefaultPVPActivity].Status != model.PVPMatching {
			return nil, model.DuelNoDraft
		}
		heroes = realtimepvp.DraftHeroes(a, account.DefaultPVPActivity)
		if len(heroes) == 0 {
			return nil, model.DuelNoDraft
		}
		return heroes, model.DuelSent
	case model.MatchOwnHeroes:
		for id, hero := range a.Heroes {
			if _, known := s.Store.Catalog.Heroes[id]; known && hero.Trial == 0 {
				heroes[id] = realtimepvp.BattleHero(hero)
			}
		}
	case model.MatchRandomHeroes:
		pool := slices.Clone(rule.Heroes)
		rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
		for _, id := range pool[:min(len(pool), max(rule.InitialHeroes, minimumHeroes))] {
			heroes[id] = realtimepvp.BattleHero(model.Hero{ID: id, Level: 1, Star: randomHeroStar})
		}
	}
	if len(heroes) < minimumHeroes {
		return nil, model.DuelNoHeroes
	}
	return heroes, model.DuelSent
}
