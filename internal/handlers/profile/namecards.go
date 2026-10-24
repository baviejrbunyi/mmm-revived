package profile

import (
	"fmt"
	"math/bits"
	"slices"
	"strconv"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
	"mmm-revived/internal/wire"
)

func (s service) nameCards(p *server.Player, r NameCards) error {
	includeHeroes := slices.Contains(r.Keys, "show") || slices.Contains(r.Keys, "supportHeroes")
	profiles, err := s.Store.PublicProfiles(r.Players, includeHeroes)
	if err != nil {
		return err
	}
	badges := make(map[int]int, len(s.Store.Catalog.Cosmetics[1035]))
	for _, id := range s.Store.Catalog.Cosmetics[1035] {
		badges[int(id)] = 1
	}
	cards := make([]map[string]string, 0, len(r.Players))
	for _, id := range r.Players {
		entry, found := profiles[id]
		if !found {
			cards = append(cards, map[string]string{"deleted": "1"})
			continue
		}
		a := entry.Account
		profile := a.Profile
		online, offline := entry.Online, entry.Offline
		if s.Online(id) != nil {
			online = time.Now().Unix()
		} else if online >= offline {
			offline = max(a.LastLogin, online+1)
		}
		card := model.NameCard{Name: a.TeamName, Head: decimal(profile.Head), Frame: decimal(profile.Frame), Level: decimal(a.Level), Deleted: "0", Signature: profile.Signature, Story: decimal(storyMission(a.Passed)), Offline: decimal(offline), Online: decimal(online), SeasonLevel: decimal(a.Events.Season.Level), PVP: decimal(pvpScore(&a)), Heroes: decimal(entry.Heroes), Badge: decimal(profile.Badge), Background: decimal(profile.Background), Host: "1", Points: "0", Created: decimal(a.CreatedAt), Skins: decimal(len(a.OwnedSkins(s.Store.Catalog))), Server: "Local", GuildID: decimal(entry.Guild), GuildTitle: string(entry.GuildRole)}
		support := map[model.HeroID]model.Hero{}
		for _, hero := range profile.SupportHeroes {
			if value, ok := a.Heroes[hero]; ok {
				support[hero] = value
			}
		}
		shown := []model.ShownHero{}
		for _, id := range profile.ShownHeroes {
			value, ok := a.Heroes[id]
			if !ok {
				id, value.Skin = 0, 0
			}
			shown = append(shown, model.ShownHero{ID: id, Skin: value.Skin})
		}
		for _, field := range []struct {
			name        string
			destination *string
			value       any
		}{
			{"supportHeroes", &card.Support, support}, {"show", &card.Show, shown}, {"ownAchieveBadgeInfo", &card.Badges, badges}, {"showTags", &card.Tags, profile.Tags}, {"season", &card.Season, s.seasonHonor(&a)},
		} {
			if !slices.Contains(r.Keys, field.name) {
				continue
			}
			*field.destination, err = server.EncodePayload(field.value)
			if err != nil {
				return err
			}
		}
		fields, err := wire.StructFields(card)
		if err != nil {
			return err
		}
		selected := map[string]string{"deleted": "0"}
		for _, key := range r.Keys {
			value, ok := fields[key].(string)
			if !ok {
				return fmt.Errorf("unknown name card field %q", key)
			}
			selected[key] = value
		}
		if slices.Contains(r.Keys, "showTags") {
			for _, key := range honorScores {
				selected[strconv.Itoa(key)] = decimal(a.Manual.Bests[key])
			}
		}
		cards = append(cards, selected)
	}
	return p.Send("onReqNameCard", r.Players, cards)
}

var honorScores = [...]int{1004102, 1004111, 1004121, 1004131, 2000000}

func storyMission(passed map[model.MissionID]model.MissionProgress) model.MissionID {
	for _, id := range storyMissions {
		if _, ok := passed[id]; !ok {
			return id
		}
	}
	return storyMissions[len(storyMissions)-1]
}

func pvpScore(a *account.Account) int {
	if !a.Network.Initialized {
		return 1000
	}
	return a.Network.Score
}

func (s service) seasonHonor(a *account.Account) map[string]int {
	stars := 0
	if rules := s.Store.Rules; rules != nil {
		for _, group := range rules.Journey.SpiralGroups {
			for _, mission := range group.Missions {
				stars += bits.OnesCount(uint(a.Journey.Spiral[mission] & 7))
			}
		}
	}
	return map[string]int{"season": a.Events.Season.ID, "stars": stars, "passed": len(a.Journey.Spiral)}
}

func decimal[T ~int | ~int32 | ~int64](value T) string { return strconv.FormatInt(int64(value), 10) }
