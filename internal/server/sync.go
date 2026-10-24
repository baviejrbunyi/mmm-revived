package server

import (
	"errors"
	"fmt"
	"slices"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
)

var automaticComponents = []string{"taskComp", "itemComp", "heroComp", "battlepassComp", "monthlyPassComp", "activityComp", "basicComp", "energyComp"}

var itemDependents = []string{"seasonComp", "battlepassComp", "monthlyPassComp", "basicComp", "energyComp", "manualComp", "benefitsSummaryComp", "makeupComp"}

type component func(*Service, *Player, *model.Avatar) (any, error)

func field[T any](get func(*model.Avatar) T) component {
	return func(_ *Service, _ *Player, profile *model.Avatar) (any, error) { return get(profile), nil }
}

var components = map[string]component{
	"historyComp": func(s *Service, p *Player, _ *model.Avatar) (any, error) { return s.Store.HistorySnapshot(p.Account) },
	"inviteComp": func(s *Service, p *Player, _ *model.Avatar) (any, error) {
		return s.Store.JourneyInviteSnapshot(p.Account)
	},
	"battleNewComp":   field(func(a *model.Avatar) model.BattleComponent { return a.Battle }),
	"challengeComp":   field(func(a *model.Avatar) model.ChallengeComponent { return a.Challenges }),
	"spiralComp":      field(func(a *model.Avatar) model.SpiralComponent { return a.Spiral }),
	"seasonComp":      field(func(a *model.Avatar) model.SeasonComponent { return a.Season }),
	"bossComp":        field(func(a *model.Avatar) model.BossComponent { return a.Boss }),
	"noviceComp":      field(func(a *model.Avatar) model.NoviceComponent { return a.Novice }),
	"battlepassComp":  field(func(a *model.Avatar) model.BattlePassComponent { return a.BattlePass }),
	"monthlyPassComp": field(func(a *model.Avatar) model.MonthlyPassComponent { return a.MonthlyPass }),
	"energyComp":      field(func(a *model.Avatar) model.EnergyComponent { return a.Energy }),
	"drawComp":        field(func(a *model.Avatar) model.DrawComponent { return a.Draw }),
	"dispatchComp":    field(func(a *model.Avatar) model.DispatchComponent { return a.Dispatch }),
	"shopComp":        field(func(a *model.Avatar) model.ShopComponent { return a.Shop }),
	"basicComp":       field(func(a *model.Avatar) model.BasicComponent { return a.Basic }),
	"makeupComp":      field(func(a *model.Avatar) model.AppearanceComponent { return a.Appearance }),
	"lobbyComp":       field(func(a *model.Avatar) model.LobbyComponent { return a.Lobby }),
	"lineupComp":      field(func(a *model.Avatar) model.LineupComponent { return a.Lineups }),
	"heroComp":        field(func(a *model.Avatar) model.HeroComponent { return a.Heroes }),
	"itemComp":        field(func(a *model.Avatar) model.ItemComponent { return a.Items }),
	"taskComp":        field(func(a *model.Avatar) model.TaskComponent { return a.Tasks }),
	"dreamlandComp":   field(func(a *model.Avatar) model.DreamlandComponent { return a.Dreamland }),
	"matchComp":       field(func(a *model.Avatar) model.MatchComponent { return a.Match }),
	"activityComp":    field(func(a *model.Avatar) model.ActivityComponent { return a.Activities }),
	"achieveComp":     field(func(a *model.Avatar) model.AchievementComponent { return a.Achievement }),
}

var ErrProgressionRules = errors.New("progression rules are unavailable")

func (s *Service) SaveAndSync(p *Player, names ...string) error {
	if err := s.Store.Save(p.Account); err != nil {
		return err
	}
	return s.Sync(p, names...)
}

func (s *Service) Sync(p *Player, names ...string) error {
	automatic := p.Account.Journey.AutomaticPending
	if automatic {
		names = slices.Clip(names)
		for _, name := range automaticComponents {
			if !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
	}
	profile := s.clientSnapshot(p.Account)
	for _, name := range names {
		if name == "itemComp" {
			for _, dependent := range itemDependents {
				value, _ := components[dependent](s, p, &profile)
				if err := p.Update([]string{dependent}, value); err != nil {
					return err
				}
			}
		}
		read, ok := components[name]
		if !ok {
			return fmt.Errorf("unknown component %s", name)
		}
		value, err := read(s, p, &profile)
		if err != nil {
			return err
		}
		if err = p.Update([]string{name}, value); err != nil {
			return err
		}
	}
	if automatic {
		p.Account.Journey.AutomaticPending = false
	}
	if err := s.notifyAchievements(p, names, &profile); err != nil {
		return err
	}
	return s.notifyLevel(p)
}

func (s *Service) notifyLevel(p *Player) error {
	previous, current := p.Level, p.Account.Level
	p.Level = current
	if !p.Account.Retail || previous <= 0 || current <= previous || s.Store.Rules == nil {
		return nil
	}
	energy := p.Account.Items[6] - account.LevelEnergyRecovery(s.Store.Rules, previous, current)
	if err := p.Send("upLv", previous, current, max(0, energy)); err != nil {
		return err
	}
	return p.Update([]string{"activityComp"}, s.clientSnapshot(p.Account).Activities)
}
