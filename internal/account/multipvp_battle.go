package account

import (
	"math"
	"slices"
	"strconv"

	"mmm-revived/internal/model"
)

func (a *Account) BeginRetailPVPBattle(r *Rules, id model.ActivityID, battle model.BattleID, target model.RetailPVPTarget, enemy, revenge int, token string, now int64) error {
	state, main, err := a.RetailPVPActivity(r, id, now)
	if err != nil {
		return err
	}
	point, known := r.RetailPVP.PointRule(state.Point)
	mine, configured := r.RetailPVP.Mines[target.Mine]
	if !known || !configured || len(mine.Battles) < 2 || mine.Battles[0] != battle || !slices.Contains(main.Mines, target.Mine) || target.UID == 0 || target.UID == int64(a.UID) || len(token) < 16 || len(state.Attacked) >= point.Challenges {
		return rejected("Halloween battle target or challenge count is unavailable")
	}
	hasDefense := false
	for _, mine := range state.Mines {
		hasDefense = hasDefense || len(mine.Defense) > 0
	}
	if !hasDefense {
		return rejected("Halloween defense must be configured before attacking")
	}
	if revenge > 0 {
		if revenge > len(state.Records) {
			return rejected("Halloween revenge record is unavailable")
		}
		record := state.Records[revenge-1]
		if record.Attack || record.Revenged != 0 || record.UID != target.UID || state.Revenges[model.UserID(record.UID)].DailyAttacks >= r.RetailPVP.MaximumRevenge {
			return rejected("Halloween revenge is unavailable")
		}
	} else if current, exists := state.Enemies[enemy]; !exists || current.UID != target.UID || current.Mine != target.Mine || current.MineIndex != target.MineIndex {
		return rejected("Halloween enemy snapshot is unavailable")
	}
	state.Pending = &model.RetailPVPBattle{Battle: battle, Token: token, Started: now, Target: target, RevengeIndex: revenge, EnemyIndex: enemy}
	if revenge > 0 {
		state.Pending.RevengeRecord = state.Records[revenge-1].BattleInfo
	}
	return nil
}

func (a *Account) LockRetailPVPBattle(r *Rules, id model.ActivityID, token string, heroes []model.HeroID, now int64) error {
	state, _, err := a.RetailPVPActivity(r, id, now)
	if err != nil {
		return err
	}
	pending := state.Pending
	point, known := r.RetailPVP.PointRule(state.Point)
	if !known || pending == nil || pending.Token != token || pending.Locked || len(heroes) == 0 || len(heroes) > 3 || len(state.Attacked) >= point.Challenges {
		return rejected("Halloween battle deployment is unavailable")
	}
	seen := map[model.HeroID]bool{}
	for _, id := range heroes {
		if _, owned := a.Heroes[id]; !owned || seen[id] || state.DailyUsed[id] >= r.RetailPVP.DailyHeroAttacks {
			return rejected("Halloween attack hero is unavailable")
		}
		seen[id] = true
	}
	for _, id := range heroes {
		state.DailyUsed[id]++
	}
	state.Attacked = append(state.Attacked, pending.Target.UID)
	pending.Heroes = slices.Clone(heroes)
	pending.Locked = true
	if pending.RevengeIndex > 0 {
		uid := model.UserID(pending.Target.UID)
		revenge := state.Revenges[uid]
		if revenge.DailyAttacks >= r.RetailPVP.MaximumRevenge {
			return rejected("Halloween revenge daily limit reached")
		}
		revenge.DailyAttacks++
		state.Revenges[uid] = revenge
	}
	return nil
}

func (a *Account) FinishRetailPVPBattle(r *Rules, id model.ActivityID, token string, won bool, history string, now int64) (model.RetailPVPRecord, error) {
	state, main, err := a.RetailPVPActivity(r, id, now)
	if err != nil {
		return model.RetailPVPRecord{}, err
	}
	pending := state.Pending
	if pending == nil || pending.Token != token || !pending.Locked || len(history) == 0 || len(history) > 2<<20 {
		return model.RetailPVPRecord{}, rejected("Halloween battle result is unavailable")
	}
	var coin, point int64
	if won {
		factor := int64(pending.Target.Efficiency/2 + 50)
		if factor < 0 || factor > 10000 || main.WinMine < 0 || main.WinMine > 1000000 || main.WinPoint < 0 || main.WinPoint > 1000000 {
			return model.RetailPVPRecord{}, rejected("invalid Halloween battle reward")
		}
		coin, point = main.WinMine*factor/100, main.WinPoint*factor/100
		if state.Point > math.MaxInt64-point {
			return model.RetailPVPRecord{}, rejected("Halloween score exceeds limit")
		}
		if err = a.grantItems(r, map[model.ItemID]int64{r.Constants.Items.HalloweenCurrency: coin}, 0); err != nil {
			return model.RetailPVPRecord{}, err
		}
		state.Point += point
	}
	if pending.RevengeIndex > 0 {
		for index, record := range state.Records {
			if !record.Attack && record.BattleInfo == pending.RevengeRecord && record.UID == pending.Target.UID {
				record.Revenged = 1
				state.Records[index] = record
				break
			}
		}
	} else {
		delete(state.Enemies, pending.EnemyIndex)
	}
	win := 0
	if won {
		win = 1
	}
	record := model.RetailPVPRecord{UID: pending.Target.UID, Win: win, Mine: pending.Target.Mine, MineGain: coin, PointGain: point, AILevel: pending.Target.Level, BattleInfo: history, Attack: true}
	state.Records = append([]model.RetailPVPRecord{record}, state.Records...)
	if len(state.Records) > 50 {
		state.Records = state.Records[:50]
	}
	state.Pending = nil
	return record, nil
}

func (a *Account) RecordRetailPVPDefense(r *Rules, id model.ActivityID, mineIndex string, attacker model.UserID, attackerWon bool, record model.RetailPVPRecord, now int64) error {
	if err := a.RefreshRetailPVPBox(r, id, now); err != nil {
		return err
	}
	state, _, err := a.RetailPVPActivity(r, id, now)
	if err != nil {
		return err
	}
	mine, known := state.Mines[mineIndex]
	if !known || mine.Mine != record.Mine || len(mine.Defense) == 0 {
		return rejected("Halloween defender mine is unavailable")
	}
	rule := r.RetailPVP.Mines[mine.Mine]
	previous := mine.Efficiency
	if state.DailyDefenses < r.RetailPVP.DailyDefenses {
		if attackerWon {
			mine.Efficiency = max(rule.MinimumEfficiency, mine.Efficiency-rule.DefeatPenalty)
		} else {
			mine.Efficiency = min(rule.MaximumEfficiency, mine.Efficiency+rule.DefenseBonus)
		}
	}
	state.DailyDefenses++
	state.Mines[mineIndex] = mine
	record.UID, record.Attack, record.EfficiencyDelta = int64(attacker), false, mine.Efficiency-previous
	record.Win = 1 - record.Win
	state.Records = append([]model.RetailPVPRecord{record}, state.Records...)
	if len(state.Records) > 50 {
		state.Records = state.Records[:50]
	}
	return nil
}

func (a *Account) PendingRetailPVP(token string) (model.ActivityID, *model.RetailPVPBattle) {
	for id, activity := range a.Journey.Activities {
		if activity.RetailPVP != nil && activity.RetailPVP.Pending != nil && activity.RetailPVP.Pending.Token == token {
			return id, activity.RetailPVP.Pending
		}
	}
	return 0, nil
}

func (a *Account) RetailPVPBattleTarget(r *Rules, id model.ActivityID, enemy, revenge int, mineIndex string, now int64) (model.RetailPVPTarget, error) {
	state, _, err := a.RetailPVPActivity(r, id, now)
	if err != nil {
		return model.RetailPVPTarget{}, err
	}
	if revenge == 0 {
		target, known := state.Enemies[enemy]
		if !known || enemy < 0 || enemy > 2 {
			return target, rejected("Halloween enemy is unavailable")
		}
		return target, nil
	}
	if revenge < 1 || revenge > len(state.Records) {
		return model.RetailPVPTarget{}, rejected("Halloween revenge record is unavailable")
	}
	row := state.Records[revenge-1]
	if row.Attack || row.Revenged != 0 {
		return model.RetailPVPTarget{}, rejected("Halloween revenge record cannot be attacked")
	}
	index, err := strconv.Atoi(mineIndex)
	if err != nil || index < 0 || index > 2 {
		return model.RetailPVPTarget{}, rejected("Halloween revenge mine is invalid")
	}
	list := state.Revenges[model.UserID(row.UID)]
	if list.DailyAttacks >= r.RetailPVP.MaximumRevenge {
		return model.RetailPVPTarget{}, rejected("Halloween revenge daily limit reached")
	}
	var target *model.RetailPVPTarget
	switch index {
	case 0:
		target = list.First
	case 1:
		target = list.Second
	case 2:
		target = list.Third
	}
	if target == nil {
		return model.RetailPVPTarget{}, rejected("Halloween revenge mine is unavailable")
	}
	return *target, nil
}
