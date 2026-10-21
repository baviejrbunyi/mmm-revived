package account

import (
	"math"
	"slices"
	"strconv"

	"mmm-revived/internal/model"
)

func (a *Account) InitializeRetailPVP(r *Rules, now int64) {
	if a.Journey.Activities == nil {
		a.Journey.Activities = map[model.ActivityID]model.JourneyActivity{}
	}
	for id, rule := range r.RetailPVP.Activities {
		if rule.Type != 31 || !a.journeyActivityAvailable(rule, now, r.Journey.CalendarOffset) {
			continue
		}
		if _, ok := r.RetailPVP.Main[rule.Parameter]; !ok {
			continue
		}
		activity := a.Journey.Activities[id]
		if activity.RetailPVP == nil {
			activity.Initialized = now
			activity.RetailPVP = &model.RetailPVPState{Initialized: now, LastBoxRequest: now}
		}
		activity.Open = true
		state := activity.RetailPVP
		if state.Day != now/86400 {
			state.Day = now / 86400
			state.Refreshed = 0
			state.DailyDefenses = 0
			state.Enemies = map[int]model.RetailPVPTarget{}
			state.Attacked = []int64{}
			state.DailyUsed = map[model.HeroID]int{}
			for uid, revenge := range state.Revenges {
				revenge.DailyAttacks = 0
				state.Revenges[uid] = revenge
			}
		}
		if state.Mines == nil {
			state.Mines = map[string]model.RetailPVPMine{}
		}
		if state.Enemies == nil {
			state.Enemies = map[int]model.RetailPVPTarget{}
		}
		if state.Revenges == nil {
			state.Revenges = map[model.UserID]model.RetailPVPRevenge{}
		}
		if state.Attacked == nil {
			state.Attacked = []int64{}
		}
		if state.DailyUsed == nil {
			state.DailyUsed = map[model.HeroID]int{}
		}
		if state.Records == nil {
			state.Records = []model.RetailPVPRecord{}
		}
		a.Journey.Activities[id] = activity
	}
}

func (a *Account) RetailPVPActivity(r *Rules, id model.ActivityID, now int64) (*model.RetailPVPState, RetailPVPMainRule, error) {
	rule, known := r.RetailPVP.Activities[id]
	main, configured := r.RetailPVP.Main[rule.Parameter]
	activity := a.Journey.Activities[id]
	if !known || !configured || rule.Type != 31 || !a.journeyActivityAvailable(rule, now, r.Journey.CalendarOffset) || !activity.Open || activity.RetailPVP == nil || activity.RetailPVP.SettleStatus != 0 {
		return nil, main, rejected("Halloween activity is unavailable")
	}
	return activity.RetailPVP, main, nil
}

func (a *Account) RefreshRetailPVPBox(r *Rules, id model.ActivityID, now int64) error {
	state, _, err := a.RetailPVPActivity(r, id, now)
	if err != nil {
		return err
	}
	point, ok := r.RetailPVP.PointRule(state.Point)
	if !ok || point.MineLimit < 0 || point.PointLimit < 0 {
		return rejected("Halloween point configuration is invalid")
	}
	if now <= state.LastBoxRequest {
		return nil
	}
	elapsed := min(now-state.LastBoxRequest, int64(366*86400))
	var mineRate, pointRate int64
	for _, mine := range state.Mines {
		if len(mine.Defense) == 0 {
			continue
		}
		row, known := r.RetailPVP.Mines[mine.Mine]
		if !known || mine.Efficiency < 0 || mine.Efficiency > row.MaximumEfficiency || row.BaseMine < 0 || row.BaseMine > 1000000 || row.BasePoint < 0 || row.BasePoint > 1000000 {
			return rejected("Halloween mine configuration is invalid")
		}
		mineRate += row.BaseMine * int64(mine.Efficiency)
		pointRate += row.BasePoint * int64(mine.Efficiency)
	}
	const denominator = int64(360000)
	mineValue := state.MineRemainder + elapsed*mineRate
	pointValue := state.PointRemainder + elapsed*pointRate
	state.BoxMineCoin = min(point.MineLimit, state.BoxMineCoin+mineValue/denominator)
	state.BoxPoint = min(point.PointLimit, state.BoxPoint+pointValue/denominator)
	state.MineRemainder, state.PointRemainder = mineValue%denominator, pointValue%denominator
	if state.BoxMineCoin == point.MineLimit {
		state.MineRemainder = 0
	}
	if state.BoxPoint == point.PointLimit {
		state.PointRemainder = 0
	}
	state.LastBoxRequest = now
	return nil
}

func (a *Account) SelectRetailPVPMine(r *Rules, id model.ActivityID, index, mineID int, now int64) error {
	state, main, err := a.RetailPVPActivity(r, id, now)
	if err != nil {
		return err
	}
	point, known := r.RetailPVP.PointRule(state.Point)
	row, configured := r.RetailPVP.Mines[mineID]
	if !known || !configured || index < 0 || index >= min(3, point.Mines) || !slices.Contains(main.Mines, mineID) || row.InitialEfficiency < row.MinimumEfficiency || row.InitialEfficiency > row.MaximumEfficiency {
		return rejected("Halloween mine is locked or unknown")
	}
	key := strconv.Itoa(index)
	if previous, exists := state.Mines[key]; exists {
		if previous.Mine == mineID {
			return nil
		}
		return rejected("Halloween mine is already selected")
	}
	for _, previous := range state.Mines {
		if previous.Mine == mineID {
			return rejected("Halloween mine is already used")
		}
	}
	if err = a.RefreshRetailPVPBox(r, id, now); err != nil {
		return err
	}
	state.Mines[key] = model.RetailPVPMine{Mine: mineID, Efficiency: row.InitialEfficiency, Defense: map[model.HeroID]model.RetailPVPPosition{}}
	return nil
}

func (a *Account) SetRetailPVPDefense(r *Rules, id model.ActivityID, index int, heroes map[model.HeroID]int, now int64) error {
	state, _, err := a.RetailPVPActivity(r, id, now)
	if err != nil {
		return err
	}
	key := strconv.Itoa(index)
	mine, selected := state.Mines[key]
	if !selected || index < 0 || index > 2 || len(heroes) == 0 || len(heroes) > 3 {
		return rejected("Halloween defense is unavailable")
	}
	positions := map[int]bool{}
	defense := make(map[model.HeroID]model.RetailPVPPosition, len(heroes))
	for id, born := range heroes {
		if _, owned := a.Heroes[id]; !owned || born < 1 || positions[born] {
			return rejected("invalid Halloween defense hero or position")
		}
		for otherKey, otherMine := range state.Mines {
			if _, used := otherMine.Defense[id]; otherKey != key && used {
				return rejected("Halloween defense hero is assigned to another mine")
			}
		}
		positions[born] = true
		defense[id] = model.RetailPVPPosition{Born: born}
	}
	if err = a.RefreshRetailPVPBox(r, id, now); err != nil {
		return err
	}
	mine.Defense = defense
	state.Mines[key] = mine
	return nil
}

func (a *Account) ReapRetailPVPBox(r *Rules, id model.ActivityID, now int64) (int64, int64, error) {
	if err := a.RefreshRetailPVPBox(r, id, now); err != nil {
		return 0, 0, err
	}
	state, _, err := a.RetailPVPActivity(r, id, now)
	if err != nil {
		return 0, 0, err
	}
	coin, point := state.BoxMineCoin, state.BoxPoint
	if coin < 0 || point < 0 || state.Point > math.MaxInt64-point {
		return 0, 0, rejected("invalid Halloween mining balance")
	}
	if err = a.grantItems(r, map[model.ItemID]int64{71: coin}, 0); err != nil {
		return 0, 0, err
	}
	state.Point += point
	state.BoxMineCoin, state.BoxPoint = 0, 0
	return coin, point, nil
}

func (a *Account) RetailPVPMineDefense(activityID model.ActivityID, index string, r *Rules) (model.AsyncDefense, bool) {
	activity := a.Journey.Activities[activityID]
	if activity.RetailPVP == nil {
		return model.AsyncDefense{}, false
	}
	mine, exists := activity.RetailPVP.Mines[index]
	if !exists || len(mine.Defense) == 0 {
		return model.AsyncDefense{}, false
	}
	copy := *a
	copy.Network.Defense = map[model.HeroID]int{}
	copy.Network.DefenseOrder = []model.HeroID{}
	copy.Network.Interactions = map[int][]int{}
	copy.Network.Camp, copy.Network.Environment = 0, 0
	for id, position := range mine.Defense {
		copy.Network.Defense[id] = position.Born
		copy.Network.DefenseOrder = append(copy.Network.DefenseOrder, id)
	}
	slices.Sort(copy.Network.DefenseOrder)
	return copy.NetworkDefense(r), true
}
