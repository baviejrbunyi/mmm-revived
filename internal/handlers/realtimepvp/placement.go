package realtimepvp

import (
	"fmt"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

func (s service) placementRoom(p *server.Player, r PVPHeroPlacement) (*server.Room, error) {
	match := p.Room
	if match == nil || match.Finished || match.Started || p.State != server.PVPDeployment || match.Round != r.Round || match.Confirmed[p.Account.UID] {
		return nil, nil
	}
	for _, member := range match.Players {
		if !match.Ready[member.Account.UID] {
			return nil, fmt.Errorf("%w: match is still loading", account.ErrProgression)
		}
	}
	if _, ok := s.matchHeroes(p)[r.Hero]; !ok {
		return nil, fmt.Errorf("%w: hero is outside the draft", account.ErrProgression)
	}
	for _, previous := range match.Choices[p.Account.UID] {
		if previous.Hero == r.Hero {
			return nil, nil
		}
	}
	return match, nil
}

func (s service) movePVPHero(p *server.Player, r PVPHeroPlacement) error {
	match, err := s.placementRoom(p, r)
	if match == nil || err != nil {
		return err
	}
	for _, previous := range match.Choices[p.Account.UID] {
		if previous.Born == r.Born {
			return fmt.Errorf("%w: position was already confirmed", account.ErrProgression)
		}
	}
	if match.Positions == nil {
		match.Positions = map[model.UserID]server.Placement{}
	}
	placement := match.Positions[p.Account.UID]
	if placement.Round != r.Round {
		placement = server.Placement{Round: r.Round, Heroes: map[model.HeroID]int{}}
	}
	if _, exists := placement.Heroes[r.Hero]; !exists && len(placement.Heroes)+len(match.Choices[p.Account.UID]) >= match.Round*roundActors {
		return fmt.Errorf("%w: round deployment is full", account.ErrProgression)
	}
	placement.Heroes[r.Hero] = r.Born
	match.Positions[p.Account.UID] = placement
	return nil
}

func (s service) leavePVPHero(p *server.Player, r PVPHeroPlacement) error {
	match, err := s.placementRoom(p, r)
	if match == nil || err != nil {
		return err
	}
	placement := match.Positions[p.Account.UID]
	if placement.Round != r.Round {
		return nil
	}
	if _, exists := placement.Heroes[r.Hero]; !exists {
		return nil
	}
	delete(placement.Heroes, r.Hero)
	match.Positions[p.Account.UID] = placement
	return nil
}

func (s service) validatePVPPlacement(p *server.Player, r ConfirmSelection) error {
	placement := p.Room.Positions[p.Account.UID]
	if placement.Round != r.Round || len(placement.Heroes) == 0 {
		return nil
	}
	if len(placement.Heroes) != len(r.Heroes) {
		return fmt.Errorf("%w: confirmed deployment does not match placements", account.ErrProgression)
	}
	for _, hero := range r.Heroes {
		if born, exists := placement.Heroes[hero.Hero]; !exists || born != hero.Born {
			return fmt.Errorf("%w: confirmed hero position changed", account.ErrProgression)
		}
	}
	return nil
}
