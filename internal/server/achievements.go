package server

import (
	"slices"

	"mmm-revived/internal/model"
)

func (s *Service) notifyAchievements(p *Player, names []string, profile *model.Avatar) error {
	if !slices.Contains(names, "taskComp") || !p.Account.AchievementsChanged(s.Store.Rules) {
		return nil
	}
	if !slices.Contains(names, "achieveComp") {
		if err := p.Update([]string{"achieveComp"}, profile.Achievement); err != nil {
			return err
		}
	}
	return p.Send("onAchieveUp")
}
