package account

import (
	"cmp"
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

const (
	realtimeActivityType  = 2
	asyncActivityType     = 14
	defaultRealtimeChance = 3
	maximumRealtimeChance = 999
)

func (r *Rules) datedActivity(kind int, now int64, known func(model.ActivityID) bool) model.ActivityID {
	if r == nil {
		return 0
	}
	moment := now + r.Journey.CalendarOffset
	var open, latest, first model.ActivityID
	later := func(id model.ActivityID, start int64, best model.ActivityID) bool {
		current := r.Journey.Activities[best].Start
		return best == 0 || start > current || start == current && id > best
	}
	for id, rule := range r.Journey.Activities {
		if rule.Type != kind || !known(id) {
			continue
		}
		switch {
		case rule.Start > moment:
			if first == 0 || rule.Start < r.Journey.Activities[first].Start || rule.Start == r.Journey.Activities[first].Start && id < first {
				first = id
			}
		case rule.End <= 0 || moment <= rule.End:
			if later(id, rule.Start, open) {
				open = id
			}
		default:
			if later(id, rule.Start, latest) {
				latest = id
			}
		}
	}
	return cmp.Or(open, latest, first)
}

func (r *Rules) RealtimeSeason(now int64) model.ActivityID {
	return r.datedActivity(realtimeActivityType, now, func(id model.ActivityID) bool {
		_, known := r.PVP[id]
		return known
	})
}

func (a *Account) RealtimeActivity() model.ActivityID {
	best, at := DefaultPVPActivity, int64(-1)
	for id, activity := range a.Activities {
		if activity.InitializedAt > at || activity.InitializedAt == at && id > best {
			best, at = id, activity.InitializedAt
		}
	}
	return best
}

func (a *Account) PrepareRealtimePVP(r *Rules, now int64) {
	season := r.RealtimeSeason(now)
	if season == 0 {
		return
	}
	if a.Activities == nil {
		a.Activities = map[model.ActivityID]model.PVPActivity{}
	}
	if a.Tasks == nil {
		a.Tasks = map[model.TaskID]model.Task{}
	}
	rule, day := r.PVP[season], LogicDay(now)
	daily := cmp.Or(rule.DayRecover, defaultRealtimeChance)
	current, exists := a.Activities[season]
	if exists && current.InitializedAt > 0 {
		for id := range a.Activities {
			if id != season {
				delete(a.Activities, id)
			}
		}
		if last := a.Network.RealtimeDay; last < day {
			if last > 0 {
				current.Chance = min(maximumRealtimeChance, current.Chance+daily*int(min(day-last, maximumRealtimeChance)))
			}
			a.Network.RealtimeDay = day
			a.Activities[season] = current
		}
		return
	}
	previousID := a.RealtimeActivity()
	previous := a.Activities[previousID]
	a.settleRealtimeSeason(r, r.PVP[previousID], rule)
	next := model.PVPActivity{InitializedAt: now, Chance: daily}
	if previous.Status != model.PVPInactive {
		next.Round, next.Upgraded, next.UpgradeType, next.Rerolled, next.Status = previous.Round, previous.Upgraded, previous.UpgradeType, previous.Rerolled, previous.Status
		next.Lineup, next.Wins, next.History, next.NewHeroes, next.Upgrades = previous.Lineup, previous.Wins, previous.History, previous.NewHeroes, previous.Upgrades
	}
	a.Activities = map[model.ActivityID]model.PVPActivity{season: next}
	a.Network.RealtimeDay = day
}

func (a *Account) settleRealtimeSeason(r *Rules, previous, next PVPRule) {
	items := map[model.ItemID]int64{}
	for _, id := range slices.Concat(previous.PointTasks, previous.WinTasks) {
		if a.Tasks[id].Status != 3 {
			continue
		}
		if quantities, err := rewardItems(r, r.Tasks[id].Reward, 1); err == nil {
			addItems(items, quantities)
		}
	}
	for _, id := range slices.Concat(previous.PointTasks, previous.WinTasks, next.PointTasks, next.WinTasks) {
		a.Tasks[id] = model.Task{Status: 2}
	}
	a.queueSeasonMail(model.SeasonMail{Title: "Mystic Clash Season Rewards", Content: "The last Mystic Clash season has ended. Its unclaimed point and win rewards have been sent to you.", Items: items})
}

func (a *Account) queueSeasonMail(mail model.SeasonMail) {
	if len(mail.Items) == 0 && (mail.Reward == 0 || mail.Mode == "") {
		return
	}
	a.Network.SeasonMail = append(a.Network.SeasonMail, mail)
}

type BotPick struct {
	Hero model.PVPHero
	Born int
}

var (
	botRows = [...][]int{{1, 2, 3, 4}, {5, 6, 7}, {8, 9, 10}}
	botRole = map[int][]int{1: {0, 1}, 2: {0, 1}, 3: {2, 1}, 4: {1, 2}}
	botPlan = [...][]int{{1, 2}, {2, 1}, {3, 4}}
)

func (r *Rules) RealtimeBotPlan(id model.ActivityID, player model.PVPActivity, deploy int) ([]BotPick, error) {
	rule, known := r.PVP[id]
	if !known || deploy <= 0 {
		return nil, rejected("realtime bot rules are unavailable")
	}
	battle := min(max(1, len(player.History)+1), max(1, rule.Total))
	stars := slices.Clone(r.PVPRounds[battle].AIStars)
	for len(stars) < deploy {
		stars = append(stars, cmp.Or(slices.Min(append([]int{5}, stars...)), 1))
	}
	stars = stars[:deploy]
	playerStars := make([]int, 0, len(player.Lineup))
	for _, hero := range player.Lineup {
		playerStars = append(playerStars, hero.Star)
	}
	slices.SortFunc(playerStars, func(a, b int) int { return b - a })
	strength := 0
	for _, star := range playerStars[:min(deploy, len(playerStars))] {
		strength += star
	}
	scaleBotStars(stars, strength-2, strength+1)
	pool := slices.DeleteFunc(slices.Clone(rule.Heroes), func(hero model.HeroID) bool {
		return slices.ContainsFunc(player.Lineup, func(owned model.PVPHero) bool { return owned.ID == hero })
	})
	if len(pool) < deploy {
		pool = slices.Clone(rule.Heroes)
	}
	pool, err := sample(pool, len(pool))
	if err != nil {
		return nil, err
	}
	taken := map[int]bool{}
	picks := make([]BotPick, 0, deploy)
	for slot := range deploy {
		careers := botPlan[slot%len(botPlan)]
		index := max(slices.IndexFunc(pool, func(hero model.HeroID) bool { return slices.Contains(careers, r.HeroTraits[hero].Career) }), 0)
		if index >= len(pool) {
			return nil, rejected("realtime bot hero pool is empty")
		}
		hero := pool[index]
		pool = slices.Delete(pool, index, index+1)
		born, err := botBorn(r.HeroTraits[hero].Career, taken)
		if err != nil {
			return nil, err
		}
		taken[born] = true
		picks = append(picks, BotPick{Hero: model.PVPHero{ID: hero, Level: 1, Star: stars[slot]}, Born: born})
	}
	return picks, nil
}

func scaleBotStars(stars []int, minimum, maximum int) {
	total := 0
	for _, star := range stars {
		total += star
	}
	for total > max(maximum, len(stars)) {
		index := slices.Index(stars, slices.Max(stars))
		stars[index]--
		total--
	}
	for total < min(minimum, 5*len(stars)) {
		index := slices.Index(stars, slices.Min(stars))
		stars[index]++
		total++
	}
}

func botBorn(career int, taken map[int]bool) (int, error) {
	for _, row := range append(slices.Clone(botRole[career]), 0, 1, 2) {
		free := slices.DeleteFunc(slices.Clone(botRows[row]), func(born int) bool { return taken[born] })
		if len(free) == 0 {
			continue
		}
		choice, err := randomChoice(free)
		return choice, err
	}
	return 0, rejected("realtime bot has no free position")
}
