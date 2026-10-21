package account

import (
	"slices"

	"mmm-revived/internal/model"
)

func sample[T any](values []T, count int) ([]T, error) {
	result := slices.Clone(values)
	for i := len(result) - 1; i > 0; i-- {
		j, err := secureIntN(int64(i + 1))
		if err != nil {
			return nil, err
		}
		result[i], result[j] = result[j], result[i]
	}
	return result[:min(max(count, 0), len(result))], nil
}

func (a *Account) RerollPVP(r *Rules, id model.ActivityID, index int) error {
	activity := a.Activities[id]
	if activity.Status != model.PVPRedraft || activity.Rerolled || index < 0 || index >= len(activity.Lineup) {
		return rejected("invalid draft reroll")
	}
	rule := r.PVP[id]
	choice, err := sample(availablePVPHeroes(rule, activity), 1)
	if err != nil {
		return err
	}
	if len(choice) != 1 {
		return rejected("no available draft replacement")
	}
	activity.Lineup = slices.Clone(activity.Lineup)
	activity.Lineup[index] = model.PVPHero{ID: choice[0], Level: 1, Star: rule.InitialStar}
	activity.Rerolled = true
	a.Activities[id] = activity
	return nil
}

func availablePVPHeroes(rule PVPRule, activity model.PVPActivity) []model.HeroID {
	pool := make([]model.HeroID, 0, len(rule.Heroes))
	for _, id := range rule.Heroes {
		if !slices.ContainsFunc(activity.Lineup, func(hero model.PVPHero) bool { return hero.ID == id }) {
			pool = append(pool, id)
		}
	}
	return pool
}

func (a *Account) BeginPVPEnhancement(r *Rules, id model.ActivityID, activity model.PVPActivity) error {
	activity.Round = len(activity.History)
	activity.Upgraded = 0
	activity.NewHeroes = nil
	activity.Upgrades = nil
	round := r.PVPRounds[activity.Round]
	switch {
	case activity.Round >= r.PVP[id].Total:
		activity.Status = model.PVPSettle
	case round.NewHero > 0:
		pool, err := sample(availablePVPHeroes(r.PVP[id], activity), round.NewPool)
		if err != nil {
			return err
		}
		activity.NewHeroes, activity.Status = pool, model.PVPChooseHero
	default:
		activity.Status = model.PVPUpgradeHero
		if err := advancePVPUpgrade(r, &activity); err != nil {
			return err
		}
	}
	a.Activities[id] = activity
	return nil
}

func advancePVPUpgrade(r *Rules, activity *model.PVPActivity) error {
	round := r.PVPRounds[activity.Round]
	if activity.Status == model.PVPUpgradeHero && activity.Upgraded >= round.Upgrades {
		activity.Status, activity.Upgraded = model.PVPUpgradeMaximum, 0
	}
	if activity.Status == model.PVPUpgradeMaximum && activity.Upgraded >= round.MaximumUpgrades {
		activity.Status, activity.Upgrades = model.PVPMatching, nil
		return nil
	}
	careers := make([]int, 0, len(activity.Lineup))
	for _, hero := range activity.Lineup {
		if career := r.HeroTraits[hero.ID].Career; career > 0 {
			careers = append(careers, career)
		}
	}
	slices.Sort(careers)
	careers = slices.Compact(careers)
	count := round.UpgradePool
	if activity.Status == model.PVPUpgradeMaximum {
		count = round.MaximumPool
	}
	pool, err := sample(careers, count)
	if err != nil {
		return err
	}
	if len(pool) == 0 {
		return rejected("empty upgrade pool")
	}
	activity.UpgradeType, activity.Upgrades = "career", pool
	return nil
}

func (a *Account) ChoosePVPHero(r *Rules, id model.ActivityID, index int) error {
	activity := a.Activities[id]
	if activity.Status != model.PVPChooseHero || index < 0 || index >= len(activity.NewHeroes) {
		return rejected("invalid new hero choice")
	}
	activity.Lineup = append(slices.Clone(activity.Lineup), model.PVPHero{ID: activity.NewHeroes[index], Level: 1, Star: r.PVPRounds[activity.Round].NewStar})
	activity.NewHeroes = nil
	activity.Status, activity.Upgraded = model.PVPUpgradeHero, 0
	if err := advancePVPUpgrade(r, &activity); err != nil {
		return err
	}
	a.Activities[id] = activity
	return nil
}

func (a *Account) UpgradePVPHero(r *Rules, id model.ActivityID, index int, maximum bool) error {
	activity := a.Activities[id]
	want := model.PVPUpgradeHero
	if maximum {
		want = model.PVPUpgradeMaximum
	}
	if activity.Status != want || index < 0 || index >= len(activity.Upgrades) {
		return rejected("invalid hero upgrade choice")
	}
	career := activity.Upgrades[index]
	maxStar := 1
	for _, hero := range activity.Lineup {
		maxStar = max(maxStar, hero.Star)
	}
	activity.Lineup = slices.Clone(activity.Lineup)
	for i, hero := range activity.Lineup {
		if r.HeroTraits[hero.ID].Career != career {
			continue
		}
		if maximum {
			hero.Star = maxStar
		} else {
			hero.Star = min(5, hero.Star+1)
		}
		activity.Lineup[i] = hero
	}
	activity.Upgraded++
	if err := advancePVPUpgrade(r, &activity); err != nil {
		return err
	}
	a.Activities[id] = activity
	return nil
}

func (a *Account) SettlePVP(id model.ActivityID, giveup bool) error {
	activity, ok := a.Activities[id]
	if !ok || activity.Status == model.PVPInactive || !giveup && activity.Status != model.PVPSettle {
		return rejected("run cannot be settled")
	}
	if giveup {
		activity.Status = model.PVPSettle
	} else {
		activity.Status, activity.MaxWins = model.PVPInactive, max(activity.MaxWins, activity.Wins)
	}
	a.Activities[id] = activity
	return nil
}
