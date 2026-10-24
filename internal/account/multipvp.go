package account

import (
	"maps"
	"math"
	"slices"
	"strconv"

	"mmm-revived/internal/calendar"
	"mmm-revived/internal/model"
)

const retailCurrency model.ItemID = 71

func (a *Account) InitializeRetailPVP(r *Rules, now int64) {
	if r == nil {
		return
	}
	a.PrepareRealtimePVP(r, now)
	a.PrepareAsyncPVP(r, now)
	if a.Journey.Activities == nil {
		a.Journey.Activities = map[model.ActivityID]model.JourneyActivity{}
	}
	for _, id := range slices.Sorted(maps.Keys(r.RetailPVP.Activities)) {
		rule := r.RetailPVP.Activities[id]
		main, configured := r.RetailPVP.Main[rule.Parameter]
		if rule.Type != 31 || !configured {
			continue
		}
		if !a.journeyActivityAvailable(rule, now, r.Journey.CalendarOffset) {
			a.closeRetailPVP(r, id, rule, main, now)
			continue
		}
		activity := a.Journey.Activities[id]
		if activity.RetailPVP == nil {
			activity.Initialized = now
			activity.RetailPVP = &model.RetailPVPState{Initialized: now, LastBoxRequest: now}
		}
		activity.Open = true
		state := activity.RetailPVP
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
		if moment, due := a.retailSettlement(r, id, rule, now); due {
			a.finishRetailPVP(r, id, main, moment)
		}
		if day := LogicDay(now); state.Day != day {
			if state.Day > 0 && state.Day < day && state.SettleStatus == 0 {
				a.growRetailMines(r, id, int(min(day-state.Day, 366)), now)
			}
			state.Day = day
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
	}
}

func (a *Account) retailSettlement(r *Rules, id model.ActivityID, rule JourneyActivityRule, now int64) (int64, bool) {
	days := r.RetailPVP.SettleDays[id]
	if days <= 0 {
		return 0, false
	}
	offset := r.Journey.CalendarOffset
	day := (rule.Start-calendar.RefreshSeconds)/DaySeconds + int64(days)
	if (now+offset-calendar.RefreshSeconds)/DaySeconds < day {
		return 0, false
	}
	return min(now, day*DaySeconds+calendar.RefreshSeconds-offset), true
}

func (a *Account) finishRetailPVP(r *Rules, id model.ActivityID, main RetailPVPMainRule, moment int64) {
	state := a.Journey.Activities[id].RetailPVP
	if state == nil || state.SettleStatus != 0 {
		return
	}
	if moment > state.LastBoxRequest {
		_ = a.RefreshRetailPVPBox(r, id, moment)
	}
	if state.BoxMineCoin > 0 && a.grantItems(r, map[model.ItemID]int64{retailCurrency: state.BoxMineCoin}, 0) == nil {
		state.BoxMineCoin = 0
	}
	state.Point += max(0, state.BoxPoint)
	state.BoxPoint, state.MineRemainder, state.PointRemainder = 0, 0, 0
	state.SettleStatus = 1
	if state.Point > 0 && main.RankingReward != 0 && main.RankingLimit > 0 {
		a.queueSeasonMail(model.SeasonMail{Title: "Mad Carnival Ranking Reward", Content: "Mad Carnival has ended. You finished at rank {1}. Here is your ranking reward.", Reward: main.RankingReward, Mode: model.RankRetailPVP, Rank: int(id), Score: state.Point, Limit: main.RankingLimit})
	}
}

func (a *Account) closeRetailPVP(r *Rules, id model.ActivityID, rule JourneyActivityRule, main RetailPVPMainRule, now int64) {
	activity, exists := a.Journey.Activities[id]
	if !exists || !activity.Open || activity.RetailPVP == nil {
		return
	}
	if activity.RetailPVP.SettleStatus == 0 {
		moment := now
		if rule.End > 0 {
			moment = min(now, rule.End-r.Journey.CalendarOffset)
		}
		if settle, due := a.retailSettlement(r, id, rule, now); due {
			moment = min(moment, settle)
		}
		a.finishRetailPVP(r, id, main, moment)
	}
	activity.Open = false
	a.Journey.Activities[id] = activity
	for _, other := range a.Journey.Activities {
		if other.Open && other.RetailPVP != nil {
			return
		}
	}
	a.Items[retailCurrency] = 0
}

func (a *Account) growRetailMines(r *Rules, id model.ActivityID, days int, now int64) {
	if days <= 0 || a.RefreshRetailPVPBox(r, id, now) != nil {
		return
	}
	state := a.Journey.Activities[id].RetailPVP
	for key, mine := range state.Mines {
		row, known := r.RetailPVP.Mines[mine.Mine]
		if !known || len(mine.Defense) == 0 || row.AutoGrow <= 0 || mine.Efficiency >= row.MaximumEfficiency {
			continue
		}
		mine.Efficiency = min(row.MaximumEfficiency, mine.Efficiency+row.AutoGrow*days)
		state.Mines[key] = mine
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
	if err = a.grantItems(r, map[model.ItemID]int64{retailCurrency: coin}, 0); err != nil {
		return 0, 0, err
	}
	state.Point += point
	state.BoxMineCoin, state.BoxPoint = 0, 0
	a.RecordJourneyTaskEvent(r, 128, 1, 0)
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

func (a *Account) settlePVPCycle(r *Rules) {
	if a.Network.Initialized && a.Network.Activity != 0 {
		a.settleAsyncSeason(r, a.Network.Activity, a.Network.Activity)
	}
	if id := a.RealtimeActivity(); a.Activities[id].InitializedAt > 0 {
		rule := r.PVP[id]
		a.settleRealtimeSeason(r, rule, rule)
		activity := a.Activities[id]
		activity.InitializedAt = 0
		a.Activities[id] = activity
	}
	closed := false
	for _, id := range slices.Sorted(maps.Keys(a.Journey.Activities)) {
		activity := a.Journey.Activities[id]
		rule, known := r.RetailPVP.Activities[id]
		if !known || !activity.Open || activity.RetailPVP == nil {
			continue
		}
		a.finishRetailPVP(r, id, r.RetailPVP.Main[rule.Parameter], activity.RetailPVP.LastBoxRequest)
		closed = true
	}
	if closed {
		a.Items[retailCurrency] = 0
	}
}
