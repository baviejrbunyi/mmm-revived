package account

import (
	"bytes"
	"cmp"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"maps"
	"math"
	"slices"

	"mmm-revived/internal/calendar"
	"mmm-revived/internal/model"
	"mmm-revived/internal/wire"
)

const (
	manualVersion      = 1
	bossRecordLimit    = 10
	benefitsLimit      = 300
	dailyTaskSystem    = 3
	taskStatusRewarded = 4
	bossScoreKey       = 1000000
	brawlScoreKey      = 2000000
)

type ManualRules struct {
	Monsters      map[int]int              `json:"monsters"`
	Battles       map[model.BattleID][]int `json:"battles"`
	ArtifactSuits map[model.ArtifactID]int `json:"artifact_suits"`
	Eggs          []model.ItemID           `json:"eggs"`
	EggChance     float64                  `json:"egg_chance"`
	EggNewChance  float64                  `json:"egg_new_chance"`
}

type benefitClaim struct {
	key    string
	reason string
	field  string
	param  int64
	reward model.RewardID
	weekly bool
}

func firstSeen[K comparable](seen map[K]int64, id K, now int64) bool {
	if _, known := seen[id]; known {
		return false
	}
	seen[id] = now
	return true
}

func (a *Account) ensureManual() *model.ManualState {
	m := &a.Manual
	if m.Monsters == nil {
		m.Monsters = map[int]int64{}
	}
	if m.Items == nil {
		m.Items = map[model.ItemID]int64{}
	}
	if m.Artifacts == nil {
		m.Artifacts = map[model.ArtifactID]int64{}
	}
	if m.Highlights == nil {
		m.Highlights = map[model.HighlightID]int64{}
	}
	if m.Suits == nil {
		m.Suits = map[int]int64{}
	}
	if m.Eggs == nil {
		m.Eggs = map[model.ItemID]int{}
	}
	if m.Bests == nil {
		m.Bests = map[int]int64{}
	}
	if m.Benefits.Observed == nil {
		m.Benefits.Observed = map[string]bool{}
	}
	return m
}

func (a *Account) unlockMonsters(ids []int, now int64) bool {
	changed := false
	for _, id := range ids {
		changed = firstSeen(a.Manual.Monsters, id, now) || changed
	}
	return changed
}

func (a *Account) UnlockMonsterManual(r *Rules, enemies []int, now int64) bool {
	if r == nil {
		return false
	}
	a.ensureManual()
	ids := make([]int, 0, len(enemies))
	for _, enemy := range enemies {
		if id, known := r.Manual.Monsters[enemy]; known {
			ids = append(ids, id)
		}
	}
	return a.unlockMonsters(ids, now)
}

func (a *Account) ObserveManual(r *Rules, now int64) {
	m := a.ensureManual()
	if m.Version < manualVersion {
		a.backfillManual(r, now)
		m.Version = manualVersion
	}
	for battle := range a.Journey.Novice.BattleWins {
		a.unlockMonsters(r.Manual.Battles[battle], now)
	}
	for id, count := range a.Items {
		if _, known := r.ItemRules[id]; known && count > 0 {
			firstSeen(m.Items, id, now)
		}
	}
	for _, artifact := range a.Artifacts {
		if _, known := r.Artifacts[artifact.ID]; known {
			firstSeen(m.Artifacts, artifact.ID, now)
		}
	}
	for _, highlight := range a.Highlights {
		if _, known := r.HighlightTemplates[highlight.ID]; known {
			firstSeen(m.Highlights, highlight.ID, now)
		}
	}
	for id, seen := range m.Artifacts {
		if suit, known := r.Manual.ArtifactSuits[id]; known {
			firstSeen(m.Suits, suit, seen)
		}
	}
	a.observeBests(r)
	a.observeBenefits(r, now)
}

func (a *Account) observeBests(r *Rules) {
	best := func(key int, score int64) {
		if score > a.Manual.Bests[key] {
			a.Manual.Bests[key] = score
		}
	}
	if monster := r.Events.Bosses[r.Events.BossOpens[a.Events.Boss.Open].Event].Monster; monster != 0 {
		for _, score := range a.Events.Boss.Best {
			best(bossScoreKey+int(monster), score)
		}
	}
	for id, activity := range a.Journey.Activities {
		if r.Journey.Activities[id].Type == 10 {
			best(brawlScoreKey, int64(activity.BestScore))
		}
	}
}

func (a *Account) backfillManual(r *Rules, now int64) {
	var battles []model.BattleID
	for id := range a.Passed {
		visited := map[model.NodeID]bool{}
		for node := r.Missions[id].FirstNode; node != 0 && !visited[node]; node = r.Nodes[node].Next {
			visited[node] = true
			if rule := r.Nodes[node]; rule.Type == 1 && rule.Battle != 0 {
				battles = append(battles, rule.Battle)
			}
		}
	}
	for _, mission := range a.Missions {
		tiles := r.MementoMaps[r.Nodes[mission.Node.ID].Map]
		for chip := range mission.Visited {
			if tile, known := tiles[chip]; known && (tile.Type == model.MementoBattle || tile.Type == model.MementoBoss) {
				battles = append(battles, model.BattleID(tile.Parameter))
			}
		}
	}
	for id, record := range a.Journey.Challenges {
		if record.Stars > 0 {
			battles = append(battles, r.Journey.Challenges[id].Battle)
		}
	}
	for id, stars := range a.Journey.Spiral {
		if stars&1 != 0 {
			battles = append(battles, r.Journey.SpiralMissions[id].Battle)
		}
	}
	for mission, score := range a.Events.Boss.Best {
		if score > 0 {
			battles = append(battles, r.Events.BossMissions[mission].Battle)
		}
	}
	for tag, rule := range r.Defaults.Retail.TeamBreaks {
		if tag <= a.TeamBreak {
			battles = append(battles, rule.Battle)
		}
	}
	for _, activity := range a.Journey.Activities {
		for _, node := range activity.BattleNodes {
			for _, battle := range []model.JourneyHyperBattle{node.First, node.Second} {
				if battle.Stars != 0 {
					battles = append(battles, battle.ID)
				}
			}
		}
	}
	for _, battle := range battles {
		a.unlockMonsters(r.Manual.Battles[battle], now)
	}
}

func (a *Account) benefitClaims(r *Rules) []benefitClaim {
	var claims []benefitClaim
	add := func(claim benefitClaim) {
		if claim.reward != 0 {
			claims = append(claims, claim)
		}
	}
	for _, id := range r.Defaults.Retail.Daily.PointTasks {
		if a.Tasks[id].Status == taskStatusRewarded {
			add(benefitClaim{fmt.Sprintf("daily:%d:%d", a.Journey.TaskDay, id), "task_reward", "system", dailyTaskSystem, r.Tasks[id].Reward, true})
		}
	}
	boss := a.Events.Boss
	for _, score := range boss.Rewards {
		add(benefitClaim{fmt.Sprintf("boss:%d:%d", boss.Open, score), "boss_score", "openId", int64(boss.Open), r.Events.BossScores[score], false})
	}
	pass := a.Journey.Pass
	for level := range pass.Free {
		add(benefitClaim{fmt.Sprintf("pass:%d:%d", pass.ID, level), "battle_pass_reward", "passId", int64(pass.ID), r.Journey.Passes[pass.ID].Rewards[level].Free, false})
	}
	for id, activity := range a.Journey.Activities {
		rule := r.Journey.Activities[id]
		switch rule.Type {
		case 10:
			for _, index := range activity.Rewards {
				add(benefitClaim{fmt.Sprintf("brawl:%d:%d", id, index), "infinite_mayhem", "activityId", int64(id), rule.Rewards[index], false})
			}
		case 24:
			for _, day := range activity.SignRewards {
				add(benefitClaim{fmt.Sprintf("sign:%d:%d", id, day), "sign_activity", "activity_id", int64(id), rule.Rewards[day], false})
			}
		case 22:
			for _, node := range activity.BattleNodes {
				for star, rewards := range r.Journey.HyperBattles[node.ID].Rewards {
					if node.Rewarded&(1<<star) != 0 && len(rewards) > 0 {
						add(benefitClaim{fmt.Sprintf("hyper:%d:%d:%d", id, node.ID, star), "hyper_flow", "activityId", int64(id), rewards[0], false})
					}
				}
			}
		}
	}
	slices.SortFunc(claims, func(x, y benefitClaim) int { return cmp.Compare(x.key, y.key) })
	return claims
}

func (a *Account) observeBenefits(r *Rules, now int64) {
	state := &a.Manual.Benefits
	if week := LogicWeek(now); state.Week != week {
		state.Week, state.Weekly = week, []string{}
	}
	claims := a.benefitClaims(r)
	current := make(map[string]bool, len(claims))
	for _, claim := range claims {
		current[claim.key] = true
		if state.Observed[claim.key] {
			continue
		}
		items, err := rewardItems(r, claim.reward, 1)
		if err != nil {
			continue
		}
		a.appendBenefits(claim.weekly, claim.reason, claim.field, claim.param, items)
	}
	state.Observed = current
}

func (a *Account) RecordBenefits(reason, field string, param int64, items map[model.ItemID]int64) {
	a.ensureManual()
	a.appendBenefits(false, reason, field, param, items)
}

func (a *Account) appendBenefits(weekly bool, reason, field string, param int64, items map[model.ItemID]int64) {
	state := &a.Manual.Benefits
	for _, item := range slices.Sorted(maps.Keys(items)) {
		if items[item] <= 0 {
			continue
		}
		entry, err := encodeClientData(map[string]any{"reason": reason, field: param, "itemId": int64(item), "itemNum": items[item]})
		if err != nil {
			continue
		}
		if weekly {
			state.Weekly = append(state.Weekly, entry)
		} else {
			state.Lasting = append(state.Lasting, entry)
			if len(state.Lasting) > benefitsLimit {
				state.Lasting = slices.Clone(state.Lasting[len(state.Lasting)-benefitsLimit:])
			}
		}
	}
}

func encodeClientData(value any) (string, error) {
	packed, err := wire.Pack(value)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	writer := zlib.NewWriter(&buffer)
	if _, err = writer.Write(packed); err != nil {
		return "", err
	}
	if err = writer.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

func (a *Account) RecordBossBattle(mission int, won bool, heroes []model.HeroID, now int64) {
	m := a.ensureManual()
	var score int64
	if pending := a.Events.Boss.Pending; won && pending != nil && pending.Mission == mission {
		score = pending.Score
	}
	lineup, err := encodeClientData(a.bossLineup(heroes))
	if err != nil {
		return
	}
	record := model.BossRecord{Open: a.Events.Boss.Open, Mission: mission, Score: score, Time: now, Lineup: lineup}
	records := []model.BossRecord{record}
	for _, previous := range m.BossRecords {
		if previous.Open == record.Open && len(records) < bossRecordLimit {
			records = append(records, previous)
		}
	}
	m.BossRecords = records
}

func (a *Account) bossLineup(heroes []model.HeroID) []any {
	lineup := make([]any, 0, len(heroes))
	for _, id := range heroes {
		hero, owned := a.Heroes[id]
		if !owned {
			continue
		}
		entry, err := wire.StructFields(hero)
		if err != nil {
			continue
		}
		entry["hid"] = int64(id)
		artifacts := map[int]model.Artifact{}
		for slot, instance := range hero.Artifacts {
			if artifact, exists := a.Artifacts[instance]; instance != "" && exists {
				artifacts[slot] = artifact.ForClient(instance, a.CreatedAt)
			}
		}
		if len(artifacts) > 0 {
			entry["artis"] = artifacts
		}
		if instance := hero.Highlights[1]; instance != "" {
			if highlight, exists := a.Highlights[instance]; exists {
				entry["highlight"] = highlight.ForClient(instance, a.CreatedAt)
			}
		}
		lineup = append(lineup, entry)
	}
	return lineup
}

func (a *Account) MergeCollectionCounts(heroes *model.HeroComponent) {
	for id := range a.Manual.Artifacts {
		if heroes.ArtifactCounts[id] == 0 {
			heroes.ArtifactCounts[id] = 1
		}
	}
	for id := range a.Manual.Highlights {
		if heroes.HighlightCounts[id] == 0 {
			heroes.HighlightCounts[id] = 1
		}
	}
}

func (a *Account) manualSnapshot(profile *model.Avatar) {
	m := a.Manual
	manual := model.ManualComponent{
		Monsters:   make(map[int]int64, len(m.Monsters)),
		Artifacts:  make(map[int]int64, len(m.Suits)),
		Highlights: map[int]int64{},
		Items:      make(map[int]int64, len(m.Items)),
		EasterEggs: make(map[int]int, len(m.Eggs)),
	}
	maps.Copy(manual.Monsters, m.Monsters)
	maps.Copy(manual.Artifacts, m.Suits)
	for id, seen := range m.Items {
		manual.Items[int(id)] = calendar.ClientTime(seen)
	}
	for id, count := range m.Eggs {
		manual.EasterEggs[int(id)] = count
	}
	profile.Manual = manual
	profile.Appearance.ActMaxScore = maps.Clone(m.Bests)
	if profile.Appearance.ActMaxScore == nil {
		profile.Appearance.ActMaxScore = map[int]int64{}
	}
	profile.Benefits = model.BenefitsComponent{Weekly: slices.Clone(m.Benefits.Weekly), Limited: slices.Clone(m.Benefits.Lasting)}
	if profile.Benefits.Weekly == nil {
		profile.Benefits.Weekly = []string{}
	}
	if profile.Benefits.Limited == nil {
		profile.Benefits.Limited = []string{}
	}
	profile.Boss.Records = []model.BossRecord{}
	for _, record := range m.BossRecords {
		if record.Open == profile.Boss.OpenID {
			profile.Boss.Records = append(profile.Boss.Records, record)
		}
	}
}

func (a *Account) rollDispatchEgg(r *Rules, missions int, now int64) (model.ItemID, error) {
	rule := r.Manual
	if len(rule.Eggs) == 0 || rule.EggChance <= 0 {
		return 0, nil
	}
	m := a.ensureManual()
	found := false
	for range missions {
		roll, err := secureIntN(10000)
		if err != nil {
			return 0, err
		}
		if roll < int64(math.Round(rule.EggChance*100)) {
			found = true
			break
		}
	}
	if !found {
		return 0, nil
	}
	pool := rule.Eggs
	if missing := slices.DeleteFunc(slices.Clone(rule.Eggs), func(id model.ItemID) bool { return m.Eggs[id] > 0 }); len(missing) > 0 {
		roll, err := secureIntN(100)
		if err != nil {
			return 0, err
		}
		if roll < int64(math.Round(rule.EggNewChance)) {
			pool = missing
		}
	}
	egg, err := randomChoice(pool)
	if err != nil {
		return 0, err
	}
	m.Eggs[egg]++
	firstSeen(m.Items, egg, now)
	return egg, nil
}
