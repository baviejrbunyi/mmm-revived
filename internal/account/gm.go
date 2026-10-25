package account

import (
	"errors"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"mmm-revived/internal/calendar"
	"mmm-revived/internal/model"
)

func (a *Account) InitializeGM() {
	if a.GM.Coverage.Methods == nil {
		a.GM.Coverage.Methods = map[string]int64{}
	}
	if a.GM.Coverage.Issues == nil {
		a.GM.Coverage.Issues = []string{}
	}
}

func (a *Account) RecordGMRPC(method string) {
	if !a.GM.Coverage.Enabled {
		return
	}
	a.InitializeGM()
	if _, known := a.GM.Coverage.Methods[method]; !known && len(a.GM.Coverage.Methods) >= 1024 {
		return
	}
	a.GM.Coverage.Methods[method] = min(math.MaxInt64-1, a.GM.Coverage.Methods[method]) + 1
}

func (a *Account) RecordGMError(method, message string, now int64) {
	a.GM.LastError = method + ": " + message
	if len(a.GM.LastError) > 16384 {
		a.GM.LastError = a.GM.LastError[:16384]
	}
	a.GM.LastErrorTime = now
}

func (a *Account) gmCompleteNovice(r *Rules) {
	for id, rule := range r.Journey.NoviceTasks {
		a.Journey.Novice.Completed[id] = true
		for _, node := range rule.Nodes {
			a.Journey.Novice.Nodes[node] = 1
		}
	}
	a.Journey.Novice.Task, a.Journey.Novice.Node = 0, 0
}

func (a *Account) gmDreamland(r *Rules, tag string, ordinal int) error {
	if tag == "clear" {
		a.Passed = map[model.MissionID]model.MissionProgress{}
		a.Missions = map[model.MissionID]model.Mission{}
		a.Chapters = map[model.ChapterID]model.ChapterProgress{}
		a.CurrentStory, a.CurrentMemento, a.CurrentMission = 0, 0, 0
		return nil
	}
	if tag != "all" && tag != "story" && tag != "memento_easy" {
		return rejected("unknown dreamland group")
	}
	var chapters []model.ChapterID
	for id, value := range r.GM.Chapters {
		if tag == "all" || value == tag {
			chapters = append(chapters, id)
		}
	}
	slices.Sort(chapters)
	if ordinal > 0 {
		if ordinal > len(chapters) {
			return rejected("unknown dreamland chapter")
		}
		chapters = []model.ChapterID{chapters[ordinal-1]}
	}
	for _, chapter := range chapters {
		for _, id := range r.Chapters[chapter].Missions {
			entry := a.Passed[id]
			entry.Stars = 7
			a.Passed[id] = entry
		}
	}
	return nil
}

func (a *Account) ExecuteGM(r *Rules, catalog Catalog, command model.GMCommand, now int64) (model.GMCommandResult, error) {
	a.InitializeGM()
	result := model.GMCommandResult{Items: map[model.ItemID]int64{}}
	switch command.Kind {
	case model.GMUnknown:
		return result, rejected("unknown command: " + command.Name)
	case model.GMList:
		result.Text = "gmlist"
	case model.GMGetAll:
		for _, id := range catalog.Items {
			a.Items[id] = max(a.Items[id], command.Value)
		}
		defaults := NewAccount(a.UID, a.Name, a.CreatedAt, catalog)
		for id, hero := range defaults.Heroes {
			if _, owned := a.Heroes[id]; !owned {
				a.Heroes[id] = hero
			}
		}
		if err := a.gmUnlockEquipment(r, catalog); err != nil {
			return result, err
		}
		for id := range r.Crafts {
			if !slices.Contains(a.CraftRecipes, id) {
				a.CraftRecipes = append(a.CraftRecipes, id)
			}
		}
		slices.Sort(a.CraftRecipes)
		result.Text = "Inventory and recipes unlocked"
	case model.GMAddItem:
		id := model.ItemID(command.ID)
		if _, known := r.ItemRules[id]; !known {
			return result, rejected("unknown item")
		}
		if id == r.Settings.Items.MonthlyPass && command.Value > int64(r.Journey.MonthlyMaximumDuration) {
			return result, rejected("monthly pass grant exceeds the maximum duration")
		}
		if (id == r.Settings.Items.PassActivation || id == r.Settings.Items.PremiumPassActivation) && command.Value != 1 {
			return result, rejected("pass activation requires one item")
		}
		result.Items[id] = command.Value
		if err := a.GrantItems(r, result.Items); err != nil {
			return result, err
		}
		result.Text = "Item granted"
	case model.GMSetItem:
		id := model.ItemID(command.ID)
		rule, known := r.ItemRules[id]
		if !known || rule.Type == 1002 || rule.Type == 1013 || rule.Type == 1014 || rule.Type == 1015 {
			return result, rejected("setitem requires a stackable item")
		}
		a.Items[id] = command.Value
		result.Text = "Item quantity updated"
	case model.GMTeamLevel:
		level := int(command.Value)
		if _, known := r.TeamLevels[level]; !known {
			return result, rejected("unknown team level")
		}
		a.Level, a.TeamBreak, a.Experience = level, TeamBreakAt(r, level), 0
		a.RefreshTasks(r)
		result.Text = "Team level updated"
	case model.GMHeroLevel:
		maximum := 0
		for level := range r.Levels {
			maximum = max(maximum, level)
		}
		if command.Value > int64(maximum) {
			return result, rejected("hero level is outside the build data")
		}
		changed := false
		for id, hero := range a.Heroes {
			if command.ID == 0 || int64(id) == command.ID {
				hero.Level = int(command.Value)
				hero.Experience = 0
				hero.Break = r.Levels[hero.Level].Break
				a.Heroes[id] = hero
				changed = true
			}
		}
		if !changed {
			return result, rejected("hero is not owned")
		}
		a.RefreshTasks(r)
		result.Text = "Hero levels updated"
	case model.GMHeroStar:
		id := model.HeroID(command.ID)
		hero, owned := a.Heroes[id]
		if !owned {
			return result, rejected("hero is not owned")
		}
		hero.Star = int(command.Value)
		a.Heroes[id] = hero
		result.Text = "Hero star updated"
	case model.GMAddHero:
		id := model.HeroID(command.ID)
		template, known := catalog.Heroes[id]
		if !known {
			return result, rejected("unknown hero")
		}
		if _, owned := a.Heroes[id]; !owned {
			a.Heroes[id] = model.Hero{ID: id, Level: 1, Star: template.Star, Talents: maps.Clone(template.Talents), AwakeIDs: []int{}, Artifacts: map[int]model.InstanceID{}, Highlights: map[int]model.InstanceID{}}
		}
		result.Text = "Hero unlocked"
	case model.GMGuideFinish:
		for _, key := range r.GM.Guides {
			a.Guides[key] = "1"
		}
		result.Text = "System guides completed"
	case model.GMNoviceFinish:
		a.gmCompleteNovice(r)
		result.Text = "Tutorial completed"
	case model.GMNoviceSet:
		if command.ID == 0 {
			a.gmCompleteNovice(r)
		} else {
			rule, known := r.Journey.NoviceTasks[model.TaskID(command.ID)]
			if !known || len(rule.Nodes) == 0 {
				return result, rejected("unknown novice task")
			}
			a.Journey.Novice.Task = model.TaskID(command.ID)
			a.Journey.Novice.Node = rule.Nodes[0]
			a.openNoviceNodes(rule.Nodes)
			a.Journey.Novice.Nodes[rule.Nodes[0]] = 2
		}
		result.Text = "Tutorial task updated"
	case model.GMDreamlandFinish:
		if err := a.gmDreamland(r, command.Text, int(command.ID)); err != nil {
			return result, err
		}
		result.Text = "Dreamland progress updated"
	case model.GMNextDay:
		a.Energy.Day, a.Energy.Bought = LogicDay(now), 0
		a.Dispatch.Day = -1
		if err := a.PrepareDispatch(r, now); err != nil {
			return result, err
		}
		a.Events.Boss.Day = -1
		a.InitializeEvents(r, now)
		a.Network.Day = -1
		a.PrepareNetwork(r.Network, now)
		a.Journey.Monthly.LastDaily = 0
		for id, purchase := range a.ShopPurchases {
			if purchase.PeriodType == 1 {
				purchase.Count = 0
				a.ShopPurchases[id] = purchase
			}
		}
		result.Text = "Daily limits reset"
	case model.GMGetTime:
		result.Text = time.Unix(calendar.ContentTime(now), 0).UTC().Format(time.RFC3339)
		if cycle := calendar.Cycle(); cycle > 0 {
			result.Text += " (calendar loop " + strconv.FormatInt(cycle, 10) + ", date " + time.Unix(calendar.Monotonic(now), 0).UTC().Format(time.RFC3339) + ")"
		}
	case model.GMSetTime:
		offset := int64(0)
		if command.Value != 0 {
			offset = command.Value - now
		}
		if offset < -315576000 || offset > 315576000 {
			return result, rejected("calendar offset exceeds ten years")
		}
		if delta := offset - calendar.BaseOffset(); delta > 0 {
			if a.Energy.Refreshed > 0 {
				a.Energy.Refreshed -= delta
			}
			for id, mission := range a.Dispatch.Missions {
				if mission.Started != 0 {
					mission.Started, mission.Ends = mission.Started-delta, mission.Ends-delta
					a.Dispatch.Missions[id] = mission
				}
			}
		}
		a.GM.ClockOffset = &offset
		result.Clock = &offset
		result.Text = "Content calendar updated"
	case model.GMMail:
		result.Mail = int(command.ID)
		result.Text = "Mail delivered"
	case model.GMSetEnergy:
		a.Items[r.Constants.Items.Energy] = command.Value
		result.Text = "Energy updated"
	case model.GMBossReset:
		a.Events.Boss.Count = 0
		a.Events.Boss.Daily = map[int]int{}
		a.Events.Boss.Pending = nil
		result.Text = "Boss daily attempts reset"
	case model.GMScienceReset:
		a.Events.Season.Nodes = []int{}
		a.Events.Season.Resets = 1
		result.Text = "Science nodes reset"
	case model.GMScienceLevel:
		maximum := 1
		for level := range r.Events.SeasonLevels[a.Events.Season.ID] {
			maximum = max(maximum, level)
		}
		if command.Value > int64(maximum) {
			return result, rejected("unknown season level")
		}
		a.Events.Season.Level = int(command.Value)
		a.Events.Season.Experience = 0
		result.Text = "Season level updated"
	case model.GMRogueInitialize:
		if a.RogueProgress.Run.Status == model.RogueIdle {
			a.Level = max(r.Rogue.Unlock[rogueSystem].Level, a.Level)
			if err := a.InitRogue(r, now); err != nil {
				return result, err
			}
		}
		result.Text = "Rogue initialized"
	case model.GMRoguePVEInitialize:
		id := r.RoguePVEOptions(now)[int(command.ID)]
		rule, known := r.Rogue.PVE[id]
		if !known {
			return result, rejected("unknown rogue difficulty")
		}
		a.Level = max(a.Level, rule.Level, r.Rogue.Unlock[pveSystem].Level)
		a.RogueProgress.MaxDifficulty = max(a.RogueProgress.MaxDifficulty, rule.Difficulty)
		if a.RogueProgress.PVE.Status == model.RoguePVEIdle {
			if err := a.TryRoguePVE(r, id, now); err != nil {
				return result, err
			}
		}
		result.Text = "Rogue PvE initialized"
	case model.GMSave:
		result.Text = "Saved"
	case model.GMCheckChapters:
		complete := 0
		for _, rule := range r.Chapters {
			if len(rule.Missions) > 0 && !slices.ContainsFunc(rule.Missions, func(id model.MissionID) bool { return a.Passed[id].Stars == 0 }) {
				complete++
			}
		}
		result.Text = strconv.Itoa(complete) + "/" + strconv.Itoa(len(r.Chapters)) + " chapters complete"
	case model.GMResetBattle:
		result.ResetBattle = true
		result.Text = "Battle reset"
	case model.GMGuild:
		result.Text = "Guild state updated"
	default:
		return result, errors.New("unknown GM command kind")
	}
	a.GM.LastCommand = command.Name
	a.GM.Commands++
	return result, nil
}

func artifactCopies(r *Rules, catalog Catalog) int {
	if len(catalog.Artifacts) == 0 {
		return 0
	}
	return min(len(catalog.Heroes), r.Constants.Constants.DepotArtifactLimit/len(catalog.Artifacts))
}

func (a *Account) StockArtifacts(r *Rules, catalog Catalog) error {
	if a.Retail || r == nil || a.ArtifactCopies >= artifactCopies(r, catalog) {
		return nil
	}
	return a.stockArtifacts(r, catalog)
}

func (a *Account) stockArtifacts(r *Rules, catalog Catalog) error {
	copies := artifactCopies(r, catalog)
	owned := map[model.ArtifactID]int{}
	for _, value := range a.Artifacts {
		owned[value.ID]++
	}
	for _, id := range catalog.Artifacts {
		if rule, known := r.Artifacts[id]; !known || len(rule.Templates) == 0 {
			continue
		}
		for range copies - owned[id] {
			if err := a.createArtifact(r, id); err != nil {
				return err
			}
		}
		owned[id] = max(owned[id], copies)
	}
	a.ArtifactCopies = max(a.ArtifactCopies, copies)
	return nil
}

func (a *Account) gmUnlockEquipment(r *Rules, catalog Catalog) error {
	highlights := map[model.HighlightID]bool{}
	for _, value := range a.Highlights {
		highlights[value.ID] = true
	}
	if err := a.stockArtifacts(r, catalog); err != nil {
		return err
	}
	for _, id := range catalog.Highlights {
		if highlights[id] {
			continue
		}
		if _, known := r.HighlightTemplates[id]; !known {
			continue
		}
		a.Highlights[a.instanceID()] = model.Highlight{ID: id, Level: 1, Obtained: time.Now().Unix()}
		highlights[id] = true
	}
	return nil
}

func (a *Account) RecordGMHistory(history string) error {
	entries := strings.Split(strings.TrimSuffix(history, "|"), "|")
	kept, size := []string{}, 0
	for _, entry := range slices.Backward(entries) {
		if entry == "" || !utf8.ValidString(entry) || strings.ContainsFunc(entry, unicode.IsControl) {
			continue
		}
		if len(kept) == 256 || size+len(entry)+1 > 16384 {
			break
		}
		kept, size = append(kept, entry), size+len(entry)+1
	}
	if len(kept) == 0 {
		a.GM.History = ""
		return nil
	}
	slices.Reverse(kept)
	a.GM.History = strings.Join(kept, "|") + "|"
	return nil
}
