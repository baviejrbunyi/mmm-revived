package account

import (
	"maps"
	"slices"

	"mmm-revived/internal/model"
)

type RogueEventResult struct {
	Kind   string
	Token  int64
	Buffs  []int
	Heroes []model.HeroID
	Health map[model.HeroID]int64
}

func emptyRogueEventInfo() model.RoguePVEEventInfo {
	return model.RoguePVEEventInfo{BuffOptions: []int{}, ShopOptions: [][]int{}, EventOptions: [][]int{}}
}

func rogueEventOptions(ids []int) [][]int {
	options := make([][]int, 0, len(ids))
	for _, id := range ids {
		options = append(options, []int{id, 1})
	}
	return options
}

func (a *Account) TryRoguePVE(r *Rules, id int, now int64) error {
	a.InitializeRogue(r, now)
	rule, exists := r.Rogue.PVE[id]
	if !exists || a.Level < rule.Level || rule.Difficulty > a.RogueProgress.MaxDifficulty+1 || a.RogueProgress.Run.Status != model.RogueIdle || a.RogueProgress.PVE.Status != model.RoguePVEIdle {
		return rejected("rogue pve difficulty is unavailable")
	}
	run := model.RoguePVERun{ID: id, Mana: -1, Status: model.RoguePVEChooseHero, Started: now, Battle: model.RoguePVEBattleInfo{Heroes: []model.HeroID{}, HeroStatus: map[model.HeroID]model.RogueHeroStatus{}, Buffs: []int{}}}
	run.Battles = make([]model.BattleID, 0, len(rule.Battles))
	for _, pool := range rule.Battles {
		battle, err := randomChoice(pool)
		if err != nil {
			return err
		}
		if _, known := r.BattleModes[battle]; !known {
			return rejected("rogue pve battle is unavailable")
		}
		run.Battles = append(run.Battles, battle)
	}
	run.EventGroups = make([]int, 0, len(rule.EventGroups))
	for _, pool := range rule.EventGroups {
		group, err := randomChoice(pool)
		if err != nil {
			return err
		}
		run.EventGroups = append(run.EventGroups, group)
	}
	if len(run.Battles) == 0 || len(run.EventGroups) != len(run.Battles) {
		return rejected("rogue pve sequence is invalid")
	}
	a.RogueProgress.PVE = run
	return nil
}

func (a *Account) ChooseRoguePVEHeroes(r *Rules, heroes []model.HeroID) error {
	run := &a.RogueProgress.PVE
	if run.Status != model.RoguePVEChooseHero || len(heroes) < 1 || len(heroes) > 3 {
		return rejected("invalid rogue pve team")
	}
	seen := make(map[model.HeroID]bool, len(heroes))
	for _, id := range heroes {
		hero, owned := a.Heroes[id]
		if !owned || hero.Trial != 0 || seen[id] {
			return rejected("hero is unavailable for rogue pve")
		}
		seen[id] = true
	}
	run.Battle.Heroes = slices.Clone(heroes)
	run.Battle.HeroStatus = make(map[model.HeroID]model.RogueHeroStatus, len(heroes))
	for _, id := range heroes {
		run.Battle.HeroStatus[id] = model.RogueHeroStatus{HP: 10000, MaximumHP: 10000}
	}
	run.Status = model.RoguePVEEvent
	return a.prepareRogueEvent(r)
}

func (a *Account) rogueEvent(r *Rules) (int, RogueEventRule, error) {
	run := &a.RogueProgress.PVE
	if run.Status != model.RoguePVEEvent || run.Round >= len(run.EventGroups) {
		return 0, RogueEventRule{}, rejected("no rogue pve event is active")
	}
	events := r.Rogue.EventGroups[run.EventGroups[run.Round]]
	if run.EventIndex < 0 || run.EventIndex >= len(events) {
		return 0, RogueEventRule{}, rejected("invalid rogue pve event position")
	}
	id := events[run.EventIndex]
	rule, exists := r.Rogue.Events[id]
	if !exists {
		return 0, rule, rejected("rogue pve event is unavailable")
	}
	return id, rule, nil
}

func (a *Account) ActiveRoguePVEEvent(r *Rules) (int, bool) {
	id, _, err := a.rogueEvent(r)
	return id, err == nil
}

func (a *Account) rogueBuffOptions(r *Rules, param []int) ([]int, error) {
	if len(param) == 0 {
		return []int{}, nil
	}
	run := &a.RogueProgress.PVE
	rule := r.Rogue.PVE[run.ID]
	result := []int{}
	used := make(map[int]bool, len(run.Battle.Buffs))
	for _, id := range run.Battle.Buffs {
		used[id] = true
	}
	count := min(param[0], len(rule.Buffs))
	if count < 0 || count > 100 {
		return nil, rejected("invalid rogue buff count")
	}
	var weights []int
	total := 0
	for j := 2; j < len(param); j += 2 {
		weight := max(0, param[j])
		weights = append(weights, weight)
		total += weight
	}
	available := func(tier int) []int {
		var pool []int
		for _, id := range rule.Buffs {
			if buff, known := r.Rogue.Buffs[id]; known && !used[id] && (tier == 0 || buff.Tier == tier) {
				pool = append(pool, id)
			}
		}
		return pool
	}
	for range count {
		tier := 0
		if total > 0 {
			index, err := weightedIndex(weights, total)
			if err != nil {
				return nil, err
			}
			if index >= 0 {
				tier = param[2*index+1]
			}
		}
		pool := available(tier)
		if len(pool) == 0 {
			pool = available(0)
		}
		if len(pool) == 0 {
			break
		}
		id, err := randomChoice(pool)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
		used[id] = true
	}
	return result, nil
}

func (a *Account) prepareRogueEvent(r *Rules) error {
	_, event, err := a.rogueEvent(r)
	if err != nil {
		return err
	}
	info := emptyRogueEventInfo()
	switch event.Action1 {
	case 1:
		if len(event.Param1) < 2 {
			return rejected("invalid rogue coin event")
		}
		var amount int
		amount, err = rogueRoll(event.Param1[0], event.Param1[1])
		info.Token = int64(amount)
	case 2:
		info.BuffOptions = slices.Clone(event.Param1)
	case 3:
		info.BuffOptions, err = a.rogueBuffOptions(r, event.Param1)
	case 4:
		for _, id := range event.Param1 {
			if _, exists := r.Rogue.Buffs[id]; exists {
				info.ShopOptions = append(info.ShopOptions, []int{id, 1})
			}
		}
	case 5:
		var ids []int
		ids, err = a.rogueBuffOptions(r, event.Param1)
		for _, id := range ids {
			info.ShopOptions = append(info.ShopOptions, []int{id, 1})
		}
	case 8:
		info.EventOptions = rogueEventOptions(event.Param1)
	}
	if err != nil {
		return err
	}
	a.RogueProgress.PVE.Battle.Event = info
	return nil
}

func (a *Account) advanceRogueEvent(r *Rules) error {
	run := &a.RogueProgress.PVE
	run.EventIndex++
	if run.EventIndex >= len(r.Rogue.EventGroups[run.EventGroups[run.Round]]) {
		run.Status = model.RoguePVEPrepare
		run.Battle.Event = emptyRogueEventInfo()
		return nil
	}
	return a.prepareRogueEvent(r)
}

func rogueRoll(minimum, maximum int) (int, error) {
	if maximum < minimum || maximum-minimum > 10000000 {
		return 0, rejected("invalid rogue event range")
	}
	roll, err := secureIntN(int64(maximum - minimum + 1))
	if err != nil {
		return 0, err
	}
	return minimum + int(roll), nil
}

func (a *Account) executeRogueAction(r *Rules, action int, param []int) ([]RogueEventResult, error) {
	run := &a.RogueProgress.PVE
	switch action {
	case 0:
		return nil, nil
	case 1:
		if len(param) < 2 {
			return nil, rejected("invalid rogue coin event")
		}
		roll, err := rogueRoll(param[0], param[1])
		if err != nil {
			return nil, err
		}
		amount := int64(roll)
		run.Battle.Token += amount
		return []RogueEventResult{{Kind: "token", Token: amount}}, nil
	case 6, 7, 10:
		if len(param) < 1 {
			return nil, rejected("invalid rogue health event")
		}
		resurrect := action == 7
		eligible := []model.HeroID{}
		for _, id := range run.Battle.Heroes {
			if hp := run.Battle.HeroStatus[id].HP; resurrect && hp == 0 || !resurrect && hp > 0 {
				eligible = append(eligible, id)
			}
		}
		heroes, err := sample(eligible, min(param[0], len(eligible)))
		if err != nil {
			return nil, err
		}
		percent := 100
		if len(param) > 2 {
			percent = param[2]
		}
		health := map[model.HeroID]int64{}
		for _, id := range heroes {
			status := run.Battle.HeroStatus[id]
			if resurrect {
				status.HP = status.MaximumHP
			} else {
				change := status.MaximumHP * int64(percent) / 100
				if action == 10 {
					change = -change
				}
				previous := status.HP
				status.HP = max(0, min(status.MaximumHP, status.HP+change))
				health[id] = status.HP - previous
			}
			run.Battle.HeroStatus[id] = status
		}
		kind := "heal"
		if resurrect {
			kind = "resurrect"
		}
		return []RogueEventResult{{Kind: kind, Heroes: heroes, Health: health}}, nil
	case 9:
		ids, err := a.rogueBuffOptions(r, param)
		if err != nil {
			return nil, err
		}
		run.Battle.Buffs = append(run.Battle.Buffs, ids...)
		return []RogueEventResult{{Kind: "buff", Buffs: ids}}, nil
	case 11:
		gained := []int{}
		for _, id := range param {
			if _, known := r.Rogue.Buffs[id]; !known {
				return nil, rejected("unknown event buff")
			}
			if !slices.Contains(run.Battle.Buffs, id) {
				run.Battle.Buffs = append(run.Battle.Buffs, id)
				gained = append(gained, id)
			}
		}
		return []RogueEventResult{{Kind: "buff", Buffs: gained}}, nil
	case 12:
		if len(param) < 1 {
			return nil, rejected("invalid lose buff event")
		}
		pool := []int{}
		for _, id := range run.Battle.Buffs {
			if len(param) < 2 || r.Rogue.Buffs[id].Tier == param[1] {
				pool = append(pool, id)
			}
		}
		lost, err := sample(pool, min(param[0], len(pool)))
		if err != nil {
			return nil, err
		}
		run.Battle.Buffs = slices.DeleteFunc(run.Battle.Buffs, func(id int) bool { return slices.Contains(lost, id) })
		return []RogueEventResult{{Kind: "lose_buff", Buffs: lost}}, nil
	case 13:
		if len(param) < 1 || param[0] < 0 || run.Battle.Token < int64(param[0]) {
			return nil, rejected("insufficient rogue pve tokens")
		}
		run.Battle.Token -= int64(param[0])
		return []RogueEventResult{{Kind: "token", Token: -int64(param[0])}}, nil
	case 14:
		if len(param) < 3 {
			return nil, rejected("invalid probabilistic token event")
		}
		roll, err := rogueRoll(1, 100)
		if err != nil {
			return nil, err
		}
		if roll <= param[0] {
			return a.executeRogueAction(r, 1, param[1:])
		}
		return []RogueEventResult{{Kind: "token"}}, nil
	case 15:
		if len(param) < 2 {
			return nil, rejected("invalid probabilistic buff event")
		}
		roll, err := rogueRoll(1, 100)
		if err != nil {
			return nil, err
		}
		if roll <= param[0] {
			return a.executeRogueAction(r, 9, param[1:])
		}
		return []RogueEventResult{{Kind: "buff", Buffs: []int{}}}, nil
	default:
		return nil, rejected("rogue event requires a choice")
	}
}

func (a *Account) ExecuteRogueEvent(r *Rules, selected int) ([]RogueEventResult, error) {
	_, event, err := a.rogueEvent(r)
	if err != nil {
		return nil, err
	}
	run := &a.RogueProgress.PVE
	if selected != 0 {
		options := run.Battle.Event.EventOptions
		if len(options) == 0 && event.Action1 == 8 {
			options = rogueEventOptions(event.Param1)
		}
		if event.Action1 != 8 || !slices.ContainsFunc(options, func(option []int) bool { return len(option) == 2 && option[0] == selected && option[1] == 1 }) {
			return nil, rejected("invalid rogue event choice")
		}
		var exists bool
		if event, exists = r.Rogue.Events[selected]; !exists {
			return nil, rejected("selected event is unavailable")
		}
	} else if event.Action1 == 2 || event.Action1 == 3 || event.Action1 == 4 || event.Action1 == 5 || event.Action1 == 8 {
		return nil, rejected("rogue event requires a selection")
	}
	results := []RogueEventResult{}
	for index, step := range [...]struct {
		action int
		param  []int
	}{{event.Action1, event.Param1}, {event.Action2, event.Param2}, {event.Action3, event.Param3}} {
		if index == 0 && selected == 0 && step.action == 1 && run.Battle.Event.Token > 0 {
			run.Battle.Token += run.Battle.Event.Token
			results = append(results, RogueEventResult{Kind: "token", Token: run.Battle.Event.Token})
			continue
		}
		next, err := a.executeRogueAction(r, step.action, step.param)
		if err != nil {
			return nil, err
		}
		results = append(results, next...)
	}
	if err := a.advanceRogueEvent(r); err != nil {
		return nil, err
	}
	return results, nil
}

func (a *Account) SelectRogueBuff(r *Rules, id int) error {
	_, event, err := a.rogueEvent(r)
	if err != nil {
		return err
	}
	run := &a.RogueProgress.PVE
	if (event.Action1 != 2 && event.Action1 != 3) || !slices.Contains(run.Battle.Event.BuffOptions, id) || slices.Contains(run.Battle.Buffs, id) {
		return rejected("invalid rogue buff selection")
	}
	if _, known := r.Rogue.Buffs[id]; !known {
		return rejected("rogue buff is unavailable")
	}
	run.Battle.Buffs = append(run.Battle.Buffs, id)
	return a.advanceRogueEvent(r)
}

func (a *Account) BuyRogueBuffs(r *Rules, ids []int) error {
	_, event, err := a.rogueEvent(r)
	if err != nil {
		return err
	}
	run := &a.RogueProgress.PVE
	if (event.Action1 != 4 && event.Action1 != 5) || len(ids) < 1 || len(ids) > 20 {
		return rejected("rogue buff shop is unavailable")
	}
	cost := int64(0)
	indices := make([]int, 0, len(ids))
	seen := make(map[int]bool, len(ids))
	for _, id := range ids {
		if seen[id] || slices.Contains(run.Battle.Buffs, id) {
			return rejected("duplicate rogue buff purchase")
		}
		seen[id] = true
		found := slices.IndexFunc(run.Battle.Event.ShopOptions, func(entry []int) bool {
			return len(entry) == 2 && entry[0] == id && entry[1] == 1
		})
		buff, known := r.Rogue.Buffs[id]
		if found < 0 || !known || buff.Cost < 0 {
			return rejected("rogue buff is unavailable")
		}
		cost += buff.Cost
		indices = append(indices, found)
	}
	if cost > run.Battle.Token {
		return rejected("insufficient rogue pve tokens")
	}
	run.Battle.Token -= cost
	run.Battle.Buffs = append(run.Battle.Buffs, ids...)
	for _, index := range indices {
		run.Battle.Event.ShopOptions[index][1] = 0
	}
	return nil
}

func (a *Account) FinishRogueBuffShopping(r *Rules) error {
	_, event, err := a.rogueEvent(r)
	if err != nil {
		return err
	}
	if event.Action1 != 4 && event.Action1 != 5 {
		return rejected("rogue buff shop is unavailable")
	}
	return a.advanceRogueEvent(r)
}

func (a *Account) SettleRoguePVE() error {
	run := &a.RogueProgress.PVE
	if run.Status == model.RoguePVEIdle {
		return rejected("rogue pve has not started")
	}
	if run.Status != model.RoguePVESettle && run.Status != model.RoguePVEChooseHero {
		a.RogueProgress.PVERecords = appendRecent(a.RogueProgress.PVERecords, a.roguePVERecord(), 10)
	}
	run.Status = model.RoguePVESettle
	return nil
}

func (a *Account) ApplyRogueHeroStatus(values map[model.HeroID]model.RogueHeroStatus) error {
	run := &a.RogueProgress.PVE
	for id, value := range values {
		if !slices.Contains(run.Battle.Heroes, id) || value.MaximumHP <= 0 || value.MaximumHP > 1000000000 || value.HP < 0 || value.HP > value.MaximumHP {
			return rejected("invalid rogue hero health")
		}
	}
	maps.Copy(run.Battle.HeroStatus, values)
	return nil
}
