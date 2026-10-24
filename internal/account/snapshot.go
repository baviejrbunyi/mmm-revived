package account

import (
	"time"

	"mmm-revived/internal/model"
)

func (a *Account) Snapshot(c Catalog) model.Avatar {
	now := time.Now().Unix()
	day := LogicDay(now)
	energyBought := 0
	if a.Energy.Day == LogicDay(now) {
		energyBought = a.Energy.Bought
	}
	profile := model.Avatar{
		GM:        model.GMComponent{History: a.GM.History},
		UID:       a.UID,
		AccountID: a.Name,
		Username:  a.Name,
		Host:      1,
		CreatedAt: a.CreatedAt,
		LastLogin: a.LastLogin,
		Basic: model.BasicComponent{
			Signature:      a.Profile.Signature,
			Experience:     a.Experience,
			RewardedLevels: a.RewardedLevels,
			Break:          a.TeamBreak,
			TeamName:       a.TeamName,
			Level:          a.Level,
			Gold:           a.Items[1],
			Diamond:        a.Items[2],
			Yuanbao:        a.Items[3],
			MobileReward:   true,
		},
		Heroes: model.HeroComponent{
			Heroes:          a.Heroes,
			History:         make(map[model.HeroID]model.HeroHistory, len(a.Heroes)),
			Artifacts:       a.Artifacts,
			ArtifactCounts:  map[model.ArtifactID]int{},
			Highlights:      a.Highlights,
			HighlightCounts: map[model.HighlightID]int{},
			Skins:           a.OwnedSkins(c),
			ShowHero:        a.Profile.ShownHeroes,
		},
		Items: model.ItemComponent{Normal: a.Items, Limited: limitedItems(a.LimitedHeroes), Crafts: a.CraftRecipes},
		Lobby: model.LobbyComponent{
			Heroes:    map[int]model.LobbyHero{1: a.Profile.Lobby},
			Guides:    a.Guides,
			HiddenIDs: a.Profile.HiddenIDs,
			RedDots:   a.Profile.RedDots,
		},
		Energy: model.EnergyComponent{
			Energy:      a.Items[6],
			Bank:        a.Energy.Bank,
			Purchases:   energyBought,
			RefreshedAt: a.Energy.Refreshed,
		},
		Season: model.SeasonComponent{Level: 1},
		Appearance: model.AppearanceComponent{
			Head:        a.Profile.Head,
			Frame:       a.Profile.Frame,
			Background:  a.Profile.Background,
			Heads:       cosmetics(c, 1022, DefaultHead),
			Frames:      cosmetics(c, 1023, DefaultHeadFrame),
			Backgrounds: cosmetics(c, 1045, DefaultBackground),
			Support:     a.Profile.SupportHeroes,
			Tags:        a.Profile.Tags,
			Badge:       a.Profile.Badge,
			BadgeInfo:   map[int]int{},
		},
		Lineups:    model.LineupComponent{Lineups: a.Lineups},
		BattlePass: model.BattlePassComponent{Level: 1},
		PVP:        model.PVPComponent{Score: 1000, MaxScore: 1000},
		Rogue:      a.RogueComponent(false),
		RoguePVE:   a.RogueComponent(true),
		Dreamland:  model.DreamlandComponent{Chapters: a.Chapters, Missions: a.Missions, Passed: a.Passed, Story: a.CurrentStory, Memento: a.CurrentMemento},
		Tasks:      model.TaskComponent{Tasks: a.Tasks},
	}
	profile.Dispatch = model.DispatchComponent{Acceptable: map[model.DispatchID]model.DispatchMission{}, Active: map[model.DispatchID]model.DispatchMission{}}
	if a.Dispatch.Day == day {
		profile.Dispatch.Count, profile.Dispatch.Refreshes = a.Dispatch.Started, a.Dispatch.Refreshes
	}
	for id, mission := range a.Dispatch.Missions {
		if mission.Started == 0 {
			profile.Dispatch.Acceptable[id] = mission
		} else {
			profile.Dispatch.Active[id] = mission
		}
	}
	profile.Shop = model.ShopComponent{Bought: make(map[model.ShopID]int, len(a.ShopPurchases)), Counts: map[model.ShopID]int{}}
	for id, purchase := range a.ShopPurchases {
		profile.Shop.Bought[id] = purchase.Total
		if purchase.Period == shopPeriod(purchase.PeriodType, now) {
			profile.Shop.Counts[id] = purchase.Count
		}
	}
	profile.Activities = model.ActivityComponent{Info: make(map[model.ActivityID]model.ActivityInfo, len(a.Activities)+2), Status: map[model.ActivityID]int{a.RealtimeActivity(): 1}}
	for id, state := range a.Activities {
		profile.Activities.Info[id] = state
	}
	if a.ResetActivityOpen(now) {
		profile.Activities.Status[DefaultResetActivity] = 1
		profile.Activities.Info[DefaultResetActivity] = model.PVPActivity{InitializedAt: a.CreatedAt}
	}
	if a.Retail {
		profile.Appearance.Heads = a.retailCosmetics(c, 1022, DefaultHead)
		profile.Appearance.Frames = a.retailCosmetics(c, 1023, DefaultHeadFrame)
		profile.Appearance.Backgrounds = a.retailCosmetics(c, 1045, DefaultBackground)
	}
	a.timedCosmeticSnapshot(c, &profile.Appearance)
	for _, id := range c.Cosmetics[1035] {
		if !a.Retail || a.Items[id] > 0 {
			profile.Appearance.BadgeInfo[int(id)] = 1
		}
	}
	for id, hero := range a.Heroes {
		profile.Heroes.History[id] = model.HeroHistory{MaxLevel: hero.Level}
	}
	for _, artifact := range a.Artifacts {
		profile.Heroes.ArtifactCounts[artifact.ID]++
	}
	for _, highlight := range a.Highlights {
		profile.Heroes.HighlightCounts[highlight.ID]++
	}
	profile.Draw = a.DrawSnapshot()
	a.JourneySnapshot(&profile, now)
	a.EventsSnapshot(&profile)
	a.manualSnapshot(&profile)
	a.ApplySystemSnapshot(&profile, now)
	profile.PVP = a.NetworkSnapshot(now)
	profile.Battle.SupportUses = a.Network.Support.Used
	if a.Network.Activity != 0 {
		profile.Activities.Info[a.Network.Activity] = model.AsyncActivity{Score: a.Network.Points}
		profile.Activities.Status[a.Network.Activity] = 1
	}
	return profile
}

func cosmetics(c Catalog, kind int, fallback model.ItemID) []model.ItemID {
	if len(c.Cosmetics[kind]) > 0 {
		return c.Cosmetics[kind]
	}
	return []model.ItemID{fallback}
}
