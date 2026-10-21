package server

import (
	"time"

	"mmm-revived/internal/matchmaking"
	"mmm-revived/internal/model"
)

type Room struct {
	ID          string
	Players     [2]*Player
	Round       int
	Ready       map[model.UserID]bool
	Choices     map[model.UserID][]model.Selection
	Positions   map[model.UserID]Placement
	Confirmed   map[model.UserID]bool
	Rows        map[model.UserID][]any
	Reports     map[model.UserID]model.MatchReport
	Heroes      map[model.UserID]map[model.HeroID]model.Hero
	Deadline    time.Time
	Seed        int64
	Started     bool
	Announced   bool
	Finished    bool
	Claim       model.UserID
	Mode        model.MatchMode
	Manual      bool
	Duel        DuelClock
	Activity    model.ActivityID
	Battle      int
	Environment int
	Expire      func(*Room) error
}

func (s *Service) MatchStatus(p *Player, kind int) error {
	status := model.MatchComponent{Type: kind, Estimate: 10}
	if kind != 0 {
		status.StartedAt = time.Now().Unix()
	}
	return p.Update([]string{"matchComp"}, status)
}

func (s *Service) FinishMatch(match *Room, winner model.UserID) error {
	if match.Finished {
		return nil
	}
	match.Finished = true
	delete(s.rooms, match.ID)
	if match.Claim != 0 {
		winner = match.Claim
	}
	var firstError error
	for _, member := range match.Players {
		member.Room, member.State = nil, Lobby
		if err := s.settleMatch(match, member, winner); err != nil && firstError == nil {
			firstError = err
		}
		if match.Announced {
			continue
		}
		if err := member.Send("pvpBattleFinish", winner == member.Account.UID); err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}

func (s *Service) AnnounceMatch(match *Room, winner model.UserID, confirm time.Duration) error {
	if match.Finished || match.Announced {
		return nil
	}
	match.Announced, match.Claim, match.Deadline = true, winner, time.Now().Add(confirm)
	var firstError error
	for _, member := range match.Players {
		if err := member.Send("pvpBattleFinish", winner == member.Account.UID); err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}

func (s *Service) settleMatch(match *Room, member *Player, winner model.UserID) error {
	activity := member.Account.Activities[match.Activity]
	rule := s.Store.Rules.PVP[match.Activity]
	draft := match.Mode.UsesDraft()
	var firstError error
	if winner != 0 {
		won := winner == member.Account.UID
		reward := rule.LossReward
		if won {
			reward = rule.WinReward
		}
		if draft {
			outcome := 2
			if won {
				outcome = 1
				activity.Wins++
			}
			activity.History = append(activity.History, outcome)
		}
		before := member.Account.Items[28]
		firstError = member.Account.Grant(s.Store.Rules, reward)
		activity.Points += member.Account.Items[28] - before
		for _, id := range rule.PointTasks {
			member.Account.UpdateTask(id, int(activity.Points), s.Store.Rules.Tasks[id].Target)
		}
		for _, id := range rule.WinTasks {
			member.Account.UpdateTask(id, max(activity.Wins, activity.MaxWins), s.Store.Rules.Tasks[id].Target)
		}
	}
	switch {
	case draft && winner != 0:
		if err := member.Account.BeginPVPEnhancement(s.Store.Rules, match.Activity, activity); err != nil && firstError == nil {
			firstError = err
		}
	case draft:
		activity.Status = model.PVPMatching
		member.Account.Activities[match.Activity] = activity
	case winner != 0:
		member.Account.Activities[match.Activity] = activity
	}
	if err := s.SaveAndSync(member, "activityComp", "itemComp", "taskComp"); err != nil && firstError == nil {
		firstError = err
	}
	return firstError
}

func NotifyDuel(p *Player, peer model.UserID, result model.DuelResult) error {
	return p.Send("onFriendDuelResult", peer, int(result))
}

func (s *Service) CloseInvites(invites []matchmaking.Invite, inviter, invitee model.DuelResult) {
	for _, invite := range invites {
		for _, side := range [...]struct {
			uid, peer model.UserID
			result    model.DuelResult
		}{{invite.From, invite.To, inviter}, {invite.To, invite.From, invitee}} {
			p := s.byUID[side.uid]
			if p == nil {
				continue
			}
			if err := NotifyDuel(p, side.peer, side.result); err != nil {
				p.Session.Logger.Error("notify duel invitation", "peer", side.peer, "error", err)
			}
		}
	}
}

func (s *Service) expireInvites(now time.Time) {
	invites := s.queue.Invites()
	s.CloseInvites(invites.Remove(func(invite matchmaking.Invite) bool { return !now.Before(invite.Expires) }), model.DuelExpired, model.DuelExpired)
	s.CloseInvites(invites.Remove(func(invite matchmaking.Invite) bool { return s.byUID[invite.To] == nil }), model.DuelOffline, model.DuelCancelled)
	s.CloseInvites(invites.Remove(func(invite matchmaking.Invite) bool { return s.byUID[invite.From] == nil }), model.DuelOffline, model.DuelCancelled)
}

type Placement struct {
	Round  int
	Heroes map[model.HeroID]int
}
