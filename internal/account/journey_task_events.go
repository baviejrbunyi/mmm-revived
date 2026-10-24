package account

import (
	"hash/fnv"
	"maps"
	"math"
	"slices"
	"time"

	"mmm-revived/internal/calendar"
	"mmm-revived/internal/model"
)

type battleAchievement struct {
	kind   int
	battle model.BattleID
}

var battleAchievements = map[string]battleAchievement{
	"ESCAPE_NVWU_FAZHEN":       {100, 110},
	"YUEGUANG_0_SCORE":         {101, 208},
	"CONTROL_BLACK_QISHI":      {102, 303},
	"NOT_KILL_MEDIC":           {103, 504},
	"USE_ONE_HERO_IN_208":      {104, 208},
	"EACAPE_JIUTOUSHE_JIGUANG": {105, 506},
}

const packetLimit = 1_000_000_000_000

func packetNumber(value any) int64 {
	var result int64
	switch n := value.(type) {
	case int64:
		result = n
	case int:
		result = int64(n)
	case uint64:
		result = int64(min(n, packetLimit))
	case float64:
		if math.IsNaN(n) || n < 0 {
			return 0
		}
		result = int64(min(n, packetLimit))
	case bool:
		if n {
			result = 1
		}
	}
	return min(max(result, 0), packetLimit)
}

func packetMap(value any) map[any]any {
	result, _ := value.(map[any]any)
	return result
}

func packetString(value any) string {
	switch text := value.(type) {
	case string:
		return text
	case []byte:
		return string(text)
	}
	return ""
}

func packetValues(value any) []any {
	switch values := value.(type) {
	case []any:
		return values
	case map[any]any:
		return slices.Collect(maps.Values(values))
	}
	return nil
}

func taskAmount(value int64) int {
	return int(min(max(value, 0), math.MaxInt32))
}

func pvpRuleTask(r *Rules, id model.TaskID) bool {
	for _, rule := range r.PVP {
		if slices.Contains(rule.PointTasks, id) || slices.Contains(rule.WinTasks, id) {
			return true
		}
	}
	return false
}

func (a *Account) journeyConditionTaskOpen(r *Rules, id model.TaskID, now int64) bool {
	for activityID, state := range a.Journey.Activities {
		rule, known := r.Journey.Activities[activityID]
		if !known || !state.Open || !a.journeyActivityAvailable(rule, now, r.Journey.CalendarOffset) {
			continue
		}
		if slices.Contains(rule.Tasks, id) || slices.ContainsFunc(rule.Conditions, func(condition JourneyConditionRule) bool { return slices.Contains(condition.Tasks, id) }) {
			return true
		}
	}
	return false
}

func journeyActivityTaskType(kind int) bool {
	switch kind {
	case 116, 117, 124, 125, 126, 128, 129, 130:
		return true
	}
	return false
}

func (a *Account) advanceJourneyTasks(r *Rules, kind int, amount int64, best bool, accept func(model.TaskID, TaskRule) bool) {
	if r == nil || amount <= 0 {
		return
	}
	now := time.Now().Unix()
	for id, rule := range r.Tasks {
		if rule.Type != kind || !a.journeyTaskEnabled(r, id, rule, now) || journeyActivityTaskType(kind) && !a.journeyConditionTaskOpen(r, id, now) {
			continue
		}
		task := a.Tasks[id]
		if task.Status >= 4 || accept != nil && !accept(id, rule) {
			continue
		}
		value := amount
		if !best {
			value += int64(task.Value)
		}
		a.UpdateTask(id, taskAmount(value), rule.Target)
	}
}

type battleHeroStatistics struct {
	damage, heal, ultimates, cards, activeKills int64
}

func (a *Account) RecordJourneyBattleResult(r *Rules, battle model.BattleID, win bool, packet map[any]any) {
	if r == nil || packet == nil {
		return
	}
	group := r.BattleModes[battle]
	extra := packetMap(packet["extra"])
	heroes := map[model.HeroID]battleHeroStatistics{}
	for _, value := range packetValues(packet["heroIds"]) {
		if id := packetNumber(value); id > 0 && id <= math.MaxInt32 {
			heroes[model.HeroID(id)] = battleHeroStatistics{}
		}
	}
	var damage, heal, ultimates, strongest int64
	careers := map[int]int64{}
	record := func(id model.HeroID, values map[any]any) {
		hero := battleHeroStatistics{damage: packetNumber(values["hurt"]), heal: max(0, packetNumber(values["heal"])-packetNumber(values["recover"])), ultimates: min(packetNumber(values["use_active_skill_card_time"]), 1000), activeKills: min(packetNumber(values["kill_by_active"]), 100000)}
		hero.cards = min(packetNumber(values["use_card_time"]), 1000) + hero.ultimates
		damage, heal, ultimates = min(damage+hero.damage, packetLimit), min(heal+hero.heal, packetLimit), ultimates+hero.ultimates
		strongest = max(strongest, hero.damage/max(1, hero.cards))
		if id != 0 {
			heroes[id] = hero
			if traits, known := r.HeroTraits[id]; known && hero.activeKills > 0 {
				careers[traits.Career] += hero.activeKills
			}
		}
	}
	switch values := extra["statistics"].(type) {
	case map[any]any:
		for key, value := range values {
			id := packetNumber(key)
			if id > math.MaxInt32 {
				id = 0
			}
			record(model.HeroID(id), packetMap(value))
		}
	case []any:
		for _, value := range values {
			record(0, packetMap(value))
		}
	}
	a.advanceJourneyTasks(r, 54, strongest, true, nil)
	a.advanceJourneyTasks(r, 56, damage, false, nil)
	a.advanceJourneyTasks(r, 57, heal, false, nil)
	for career, kills := range careers {
		a.advanceJourneyTasks(r, 58, kills, false, func(_ model.TaskID, rule TaskRule) bool { return taskNumber(rule, 0, 0) == career })
	}
	for name, value := range packetMap(extra["achieve"]) {
		achievement, known := battleAchievements[packetString(name)]
		flags := packetMap(value)
		if !known || achievement.battle != battle || packetNumber(flags["ret"]) == 0 || !win && (packetNumber(flags["needwin"]) != 0 || packetNumber(flags["needWin"]) != 0) {
			continue
		}
		a.advanceJourneyTasks(r, achievement.kind, 1, true, nil)
	}
	a.advanceJourneyTasks(r, 112, 1, false, func(_ model.TaskID, rule TaskRule) bool { return slices.Contains(rule.Targets, int(battle)) })
	withHero := func(rule TaskRule) bool {
		_, used := heroes[model.HeroID(taskNumber(rule, 0, 0))]
		return used
	}
	inGroup := func(rule TaskRule) bool { return taskNumber(rule, 1, 0) == group && group != 0 }
	a.advanceJourneyTasks(r, 125, 1, false, func(_ model.TaskID, rule TaskRule) bool { return inGroup(rule) })
	a.advanceJourneyTasks(r, 124, 1, false, func(_ model.TaskID, rule TaskRule) bool { return inGroup(rule) && withHero(rule) })
	if win {
		a.advanceJourneyTasks(r, 117, 1, false, func(_ model.TaskID, rule TaskRule) bool { return inGroup(rule) })
		a.advanceJourneyTasks(r, 116, 1, false, func(_ model.TaskID, rule TaskRule) bool { return inGroup(rule) && withHero(rule) })
	}
	a.advanceJourneyTasks(r, 126, ultimates, false, func(id model.TaskID, rule TaskRule) bool {
		if heroes[model.HeroID(taskNumber(rule, 0, 0))].ultimates > 0 {
			if a.Journey.TaskStats.Doubled == nil {
				a.Journey.TaskStats.Doubled = map[model.TaskID]bool{}
			}
			a.Journey.TaskStats.Doubled[id] = true
		}
		return true
	})
	if group == 28 && packetNumber(extra["isRevenge"]) != 0 {
		a.advanceJourneyTasks(r, 129, 1, false, nil)
	}
	mode := 0
	switch group {
	case rogueRunMode:
		mode = 1
	case roguePVEMode:
		mode = 2
	}
	if mode != 0 {
		kills := packetNumber(packet["killnum"])
		if kills == 0 {
			kills = packetNumber(packetMap(extra["roguePveStat"])["killNum"])
		}
		a.advanceJourneyTasks(r, 89, min(kills, 100000), false, func(_ model.TaskID, rule TaskRule) bool { return taskNumber(rule, 0, 0) == mode })
	}
}

func (a *Account) RecordJourneyRogueFinish(r *Rules, mode, wins int) {
	a.advanceJourneyTasks(r, 27, 1, false, func(_ model.TaskID, rule TaskRule) bool {
		kind := taskNumber(rule, 0, 0)
		return (kind == 0 || kind == mode) && wins >= taskNumber(rule, 1, 1)
	})
	run := &a.RogueProgress.PVE
	if mode != 2 || len(run.Battles) == 0 || run.Round < len(run.Battles) {
		return
	}
	for _, status := range run.Battle.HeroStatus {
		if status.MaximumHP > 0 && status.HP <= 0 {
			return
		}
	}
	a.advanceJourneyTasks(r, 90, 1, false, nil)
}

func (a *Account) journeyTaskBackfill(rule TaskRule) int64 {
	p := &a.RogueProgress
	switch rule.Type {
	case 27:
		var count int64
		kind, wins := taskNumber(rule, 0, 0), taskNumber(rule, 1, 1)
		if kind == 0 || kind == 1 {
			for _, record := range p.Records {
				if record.Wins >= wins {
					count++
				}
			}
		}
		if kind == 0 || kind == 2 {
			for _, record := range p.PVERecords {
				if record.Round >= wins {
					count++
				}
			}
		}
		return count
	case 59:
		if a.System.LastShare > 0 {
			return 1
		}
	case 83:
		return int64(p.Run.SaleCount)
	case 84:
		return int64(p.Run.RefreshCount)
	case 85:
		return int64(p.Run.MergeCount)
	}
	return 0
}

func (a *Account) rogueLevelValue(minimum int) int64 {
	var best int64
	for _, record := range a.RogueProgress.Records {
		var count int64
		for _, hero := range record.Team {
			if hero.Level+hero.ExtraLevel >= minimum {
				count++
			}
		}
		best = max(best, count)
	}
	run := &a.RogueProgress.Run
	if run.Status == model.RogueIdle {
		return best
	}
	var count int64
	for _, instance := range run.Heroes {
		hero := run.Objects[instance]
		if hero.Level+run.Objects[hero.Relic].ExtraLevel >= minimum {
			count++
		}
	}
	return max(best, count)
}

func (a *Account) rogueFavorValue(targets []int) int64 {
	count := func(buffs []int) int64 {
		var result int64
		for _, id := range buffs {
			if slices.Contains(targets, id) {
				result++
			}
		}
		return result
	}
	best := count(a.RogueProgress.PVE.Battle.Buffs)
	for _, record := range a.RogueProgress.PVERecords {
		best = max(best, count(record.Team.Buffs))
	}
	return best
}

func (a *Account) journeyDrawValue(r *Rules, rule TaskRule) int64 {
	ids := slices.Sorted(maps.Keys(a.Draw.History))
	rarity := func(record model.DrawRecord) int { return r.ItemRules[record.Item].Rarity }
	switch rule.Type {
	case 61, 62:
		minimum, pools := taskNumber(rule, 0, 5), rule.Lists[1]
		var streak, best int64
		for _, id := range ids {
			record := a.Draw.History[id]
			if len(pools) > 0 && !slices.Contains(pools, int(record.Pool)) {
				continue
			}
			if (rarity(record) >= minimum) == (rule.Type == 61) {
				streak++
				best = max(best, streak)
			} else {
				streak = 0
			}
		}
		return best
	case 93:
		needed, quality, kind := taskNumber(rule, 0, 1), taskNumber(rule, 1, 0), taskNumber(rule, 3, 0)
		for start := 0; start < len(ids); {
			first := a.Draw.History[ids[start]]
			end := start + 1
			for end < len(ids) && a.Draw.History[ids[end]].Pool == first.Pool && a.Draw.History[ids[end]].Time == first.Time {
				end++
			}
			if (end-start)%5 == 0 {
				for chunk := start; chunk < end; chunk += 5 {
					matched := 0
					for _, id := range ids[chunk : chunk+5] {
						record := a.Draw.History[id]
						item := r.ItemRules[record.Item]
						if (quality == 0 || item.Rarity == quality) && (kind == 0 || item.Type == kind) {
							matched++
						}
					}
					if matched >= needed {
						return 1
					}
				}
			}
			start = end
		}
	}
	return 0
}

func (a *Account) journeyTaskReward(id model.TaskID, rule TaskRule) model.RewardID {
	if rule.Type == 126 && a.Journey.TaskStats.Doubled[id] && len(rule.Params) > 1 && rule.Params[1] > 0 {
		return model.RewardID(rule.Params[1])
	}
	return rule.Reward
}

func (a *Account) recordAchievementTimes(r *Rules, now int64) {
	for id := range a.Journey.TaskStats.Doubled {
		if task := a.Tasks[id]; task.Value == 0 || task.Status >= 4 {
			delete(a.Journey.TaskStats.Doubled, id)
		}
	}
	for id, rule := range r.Tasks {
		if rule.Achievement == 0 || a.Tasks[id].Status < 3 || a.Journey.TaskStats.Achieved[id] != 0 {
			continue
		}
		if a.Journey.TaskStats.Achieved == nil {
			a.Journey.TaskStats.Achieved = map[model.TaskID]int64{}
		}
		a.Journey.TaskStats.Achieved[id] = calendar.ClientTime(now)
	}
}

func (a *Account) AchievementSnapshot(r *Rules) model.AchievementComponent {
	result := model.AchievementComponent{Achievements: map[int]model.AchievementRecord{}}
	if r == nil {
		return result
	}
	for id, rule := range r.Tasks {
		if rule.Achievement == 0 {
			continue
		}
		if at := a.Journey.TaskStats.Achieved[id]; at != 0 && a.Tasks[id].Status >= 3 {
			result.Achievements[rule.Achievement] = model.AchievementRecord{Time: at}
		}
		if a.Tasks[id].Status == 4 {
			result.Points += rule.Points
		}
	}
	return result
}

func (a *Account) AchievementsChanged(r *Rules) bool {
	if r == nil {
		return false
	}
	hash := fnv.New64a()
	var buffer [16]byte
	for _, id := range slices.Sorted(maps.Keys(r.Tasks)) {
		if r.Tasks[id].Achievement == 0 {
			continue
		}
		task := a.Tasks[id]
		value := uint64(uint32(task.Value))<<32 | uint64(uint32(task.Status))
		for index := range 8 {
			buffer[index] = byte(uint64(id) >> (8 * index))
			buffer[8+index] = byte(value >> (8 * index))
		}
		hash.Write(buffer[:])
	}
	digest := hash.Sum64()
	previous := a.Journey.TaskStats.Digest
	a.Journey.TaskStats.Digest = digest
	return previous != 0 && previous != digest
}
