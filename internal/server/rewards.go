package server

import (
	"cmp"
	"slices"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
)

func (s *Service) ItemRewards(before account.Account, after *account.Account, quantities map[model.ItemID]int64) []model.ItemReward {
	rules := s.Store.Rules
	type entry struct {
		id   model.ItemID
		rank int
	}
	entries := make([]entry, 0, len(quantities))
	for id := range quantities {
		rule := rules.ItemRules[id]
		_, exists := rules.Artifacts[model.ArtifactID(int64(rule.Rarity)*1000+rule.Secondary)]
		rank := 2
		switch {
		case (rule.Type == 1013 || rule.Type == 1014) && rule.Secondary != 0 && exists:
			rank = 0
		case rule.Type == 1029:
			rank = 1
		case rule.Type == 1036:
			rank = 3
		}
		entries = append(entries, entry{id: id, rank: rank})
	}
	slices.SortFunc(entries, func(a, b entry) int {
		return cmp.Or(cmp.Compare(a.rank, b.rank), cmp.Compare(a.id, b.id))
	})
	var artifacts, highlights []model.InstanceID
	for id := range after.Artifacts {
		if _, existed := before.Artifacts[id]; !existed {
			artifacts = append(artifacts, id)
		}
	}
	for id := range after.Highlights {
		if _, existed := before.Highlights[id]; !existed {
			highlights = append(highlights, id)
		}
	}
	slices.Sort(artifacts)
	slices.Sort(highlights)
	owned := map[model.HighlightID]bool{}
	if len(highlights) > 0 {
		for _, highlight := range before.Highlights {
			owned[highlight.ID] = true
		}
	}
	result := make([]model.ItemReward, 0, len(entries))
	usedArtifacts, usedHighlights := map[model.InstanceID]bool{}, map[model.InstanceID]bool{}
	for _, e := range entries {
		id := e.id
		amount := quantities[id]
		if amount <= 0 {
			continue
		}
		rule := rules.ItemRules[id]
		switch rule.Type {
		case 1013, 1014, 1029:
			exact := model.ArtifactID(int64(rule.Rarity)*1000 + rule.Secondary)
			_, exactExists := rules.Artifacts[exact]
			for range amount {
				var instance model.InstanceID
				for _, candidate := range artifacts {
					entry := after.Artifacts[candidate]
					matches := rules.Artifacts[entry.ID].Star == rule.Rarity
					switch {
					case rule.Type == 1029:
						matches = slices.Contains(rules.Artifacts[entry.ID].Pools, id)
					case rule.Secondary != 0 && exactExists:
						matches = entry.ID == exact
					}
					if !usedArtifacts[candidate] && matches {
						instance = candidate
						break
					}
				}
				if instance == "" {
					break
				}
				usedArtifacts[instance] = true
				artifact := after.Artifacts[instance]
				details := model.RewardDetails{New: true, Instance: instance, Unique: instance, ID: int32(artifact.ID), Level: artifact.Level, Template: artifact.Template, SubAttributes: artifact.Sub}
				result = append(result, model.ItemReward{Item: id, Count: 1, Details: details})
			}
		case 1015, 1036:
			for range amount {
				var instance model.InstanceID
				for _, candidate := range highlights {
					value := after.Highlights[candidate].ID
					matches := value == model.HighlightID(rule.Parameter)
					if rule.Type == 1036 {
						matches = int64(rules.HighlightTemplates[value].Rarity) == rule.Parameter
					}
					if !usedHighlights[candidate] && matches {
						instance = candidate
						break
					}
				}
				if instance == "" {
					break
				}
				usedHighlights[instance] = true
				highlight := after.Highlights[instance]
				result = append(result, model.ItemReward{Item: id, Count: 1, Details: model.RewardDetails{New: !owned[highlight.ID], Instance: instance, Unique: instance, ID: int32(highlight.ID), Level: highlight.Level}})
				owned[highlight.ID] = true
			}
		default:
			details := model.RewardDetails{}
			if rule.Type == 1002 {
				hero := model.HeroID(rule.Parameter)
				previous, existed := before.Heroes[hero]
				details.New = !existed
				if existed && previous.Trial != 0 && after.Heroes[hero].Trial == 0 {
					details.TrialConverted = 1
				}
				converted := amount
				if !existed || details.TrialConverted == 1 {
					converted--
				}
				if converted > 0 && len(rule.Refund) > 0 {
					details.Conversion = make(map[model.ItemID]int64, len(rule.Refund))
					for item, count := range rule.Refund {
						details.Conversion[item] = count * converted
					}
				}
			}
			result = append(result, model.ItemReward{Item: id, Count: amount, Details: details})
		}
	}
	return result
}
