package account

import (
	"maps"
	"slices"

	"mmm-revived/internal/model"
)

func attributeTotals(growth map[string][]int64) map[string]int64 {
	result := make(map[string]int64, len(growth))
	for attribute, rolls := range growth {
		for _, value := range rolls {
			result[attribute] += value
		}
	}
	return result
}

func cloneGrowth(growth map[string][]int64) map[string][]int64 {
	result := make(map[string][]int64, len(growth))
	for attribute, rolls := range growth {
		result[attribute] = slices.Clone(rolls)
	}
	return result
}

func growthFromSub(sub map[string]int64) map[string][]int64 {
	growth := make(map[string][]int64, len(sub))
	for name, value := range sub {
		growth[name] = []int64{value}
	}
	return growth
}

func randomRange(low, high int64) (int64, error) {
	if low < 0 || high < low || high > 1000000000 {
		return 0, rejected("invalid attribute range")
	}
	n, err := secureIntN(high - low + 1)
	return low + n, err
}

func rollAttribute(template ArtifactTemplate, excluded map[string][]int64) (string, int64, error) {
	attributes := make([]string, 0, len(template.Attributes))
	var total int64
	for name, limits := range template.Attributes {
		weight := template.Weights[name]
		if len(excluded[name]) == 0 && weight > 0 && weight <= 1000000 && limits[0] <= limits[1] {
			attributes = append(attributes, name)
			total += weight
		}
	}
	if total == 0 {
		return "", 0, rejected("artifact attribute pool is empty")
	}
	slices.Sort(attributes)
	roll, err := randomRange(0, total-1)
	if err != nil {
		return "", 0, err
	}
	for _, name := range attributes {
		roll -= template.Weights[name]
		if roll < 0 {
			limits := template.Attributes[name]
			value, err := randomRange(limits[0], limits[1])
			return name, value, err
		}
	}
	return "", 0, rejected("invalid artifact attribute weights")
}

func initializeArtifactGrowth(r *Rules, artifact *model.Artifact) error {
	if len(artifact.Growth) > 0 {
		return nil
	}
	artifact.Growth = growthFromSub(artifact.Sub)
	if len(artifact.Growth) > 0 {
		return nil
	}
	template := r.ArtifactTemplates[artifact.Template]
	for range min(3, max(1, r.Artifacts[artifact.ID].Star-2)) {
		name, value, err := rollAttribute(template, artifact.Growth)
		if err != nil {
			return err
		}
		artifact.Growth[name] = []int64{value}
	}
	artifact.Sub = attributeTotals(artifact.Growth)
	return nil
}

func (a *Account) RefineArtifact(r *Rules, id model.InstanceID, locked string) error {
	artifact, owned := a.Artifacts[id]
	if !owned || !r.Artifacts[artifact.ID].Refinable {
		return rejected("artifact cannot be refined")
	}
	original := artifact
	if err := initializeArtifactGrowth(r, &artifact); err != nil {
		return err
	}
	if locked != "" && len(artifact.Growth[locked]) == 0 {
		return rejected("locked attribute does not belong to artifact")
	}
	template := r.ArtifactTemplates[artifact.Template]
	pending := map[string][]int64{}
	if locked != "" {
		pending[locked] = slices.Clone(artifact.Growth[locked])
	}
	counts := make([]int, 0, len(artifact.Growth))
	for name, rolls := range artifact.Growth {
		if name != locked {
			counts = append(counts, len(rolls))
		}
	}
	slices.Sort(counts)
	for _, count := range counts {
		name, value, err := rollAttribute(template, pending)
		if err != nil {
			return err
		}
		rolls := make([]int64, 1, max(1, count))
		rolls[0] = value
		limits := template.Attributes[name]
		for range count - 1 {
			value, err := randomRange(limits[0], limits[1])
			if err != nil {
				return err
			}
			rolls = append(rolls, value)
		}
		pending[name] = rolls
	}
	cost := map[model.ItemID]int64{1: 30000, 30: 5}
	if locked != "" {
		cost[39] = 1
	}
	if err := a.Spend(cost); err != nil {
		return err
	}
	original.PendingGrowth = pending
	a.Artifacts[id] = original
	return nil
}

func (a *Account) ConfirmArtifactRefine(id model.InstanceID) error {
	artifact, owned := a.Artifacts[id]
	if !owned || len(artifact.PendingGrowth) == 0 {
		return rejected("artifact has no pending refinement")
	}
	artifact.Growth = artifact.PendingGrowth
	artifact.Sub = attributeTotals(artifact.Growth)
	artifact.PendingGrowth = map[string][]int64{}
	a.Artifacts[id] = artifact
	return nil
}

func (a *Account) SalvageGear(r *Rules, ids []model.InstanceID, highlights bool) (map[model.ItemID]int64, error) {
	if len(ids) == 0 || len(ids) > 5000 {
		return nil, rejected("invalid salvage selection")
	}
	reward := map[model.ItemID]int64{}
	seen := make(map[model.InstanceID]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return nil, rejected("duplicate salvage selection")
		}
		seen[id] = true
		if highlights {
			entry, owned := a.Highlights[id]
			rarity := r.HighlightTemplates[entry.ID].Rarity
			level, known := r.HighlightSalvage[rarity][entry.Level]
			awake, awakeningKnown := r.HighlightAwakeSalvage[rarity][entry.Awake]
			if !owned || entry.Locked != 0 || entry.Install != 0 || !known || !awakeningKnown {
				return nil, rejected("highlight cannot be salvaged")
			}
			addItems(reward, level)
			addItems(reward, awake)
		} else {
			entry, owned := a.Artifacts[id]
			count, known := r.ArtifactSalvage[entry.Level][r.Artifacts[entry.ID].Star]
			if !owned || entry.Locked != 0 || entry.Install != 0 || !known {
				return nil, rejected("artifact cannot be salvaged")
			}
			reward[12] += count
		}
	}
	if err := a.GrantItems(r, reward); err != nil {
		return nil, err
	}
	if highlights {
		for _, id := range ids {
			delete(a.Highlights, id)
		}
	} else {
		for _, id := range ids {
			delete(a.Artifacts, id)
		}
	}
	return reward, nil
}

func (a *Account) AwakeHighlight(r *Rules, id model.InstanceID, consumed []model.InstanceID, items map[model.ItemID]int64) error {
	target, owned := a.Highlights[id]
	template, known := r.HighlightTemplates[target.ID]
	if !owned || !known || target.Awake >= 5 || len(consumed) > 30 {
		return rejected("highlight cannot be awakened")
	}
	levels := 0
	seen := map[model.InstanceID]bool{id: true}
	for _, instance := range consumed {
		entry, owned := a.Highlights[instance]
		if seen[instance] || !owned || entry.ID != target.ID || entry.Install != 0 || entry.Locked != 0 {
			return rejected("invalid awakening equipment")
		}
		seen[instance] = true
		levels += 1 + entry.Awake
	}
	var common model.ItemID
	switch template.Rarity {
	case 4:
		common = 100011
	case 5:
		common = 100012
	}
	for item, count := range items {
		if item == 0 || item != template.AwakeItem && item != common || count <= 0 || count > 30 {
			return rejected("invalid awakening material")
		}
		levels += int(count)
	}
	if levels == 0 || levels > 30 {
		return rejected("invalid awakening quantity")
	}
	next := *a
	next.Items = maps.Clone(a.Items)
	if err := next.Spend(items); err != nil {
		return err
	}
	if overflow := target.Awake + levels - 5; overflow > 0 {
		if err := next.GrantItems(r, map[model.ItemID]int64{template.AwakeItem: int64(overflow)}); err != nil {
			return err
		}
	}
	target.Awake = min(5, target.Awake+levels)
	next.Highlights = maps.Clone(next.Highlights)
	for _, instance := range consumed {
		delete(next.Highlights, instance)
	}
	next.Highlights[id] = target
	*a = next
	return nil
}

func (a *Account) ClaimProficiency(r *Rules, heroes []model.HeroID, level int) (map[model.ItemID]int64, error) {
	if len(heroes) == 0 || len(heroes) > 100 {
		return nil, rejected("invalid proficiency selection")
	}
	seen := make(map[model.HeroID]bool, len(heroes))
	reward := map[model.ItemID]int64{}
	updated := maps.Clone(a.Heroes)
	for _, id := range heroes {
		hero, owned := updated[id]
		maximum := hero.ProficiencyLevel
		if level > 0 {
			maximum = level
		}
		if !owned || seen[id] || maximum > hero.ProficiencyLevel || maximum > len(r.ProficiencyLevels) {
			return nil, rejected("proficiency reward is not available")
		}
		seen[id] = true
		for tier := hero.ProficiencyReward + 1; tier <= maximum; tier++ {
			if rewardID := r.ProficiencyRewards[id][tier]; rewardID != 0 {
				items, err := rewardItems(r, rewardID, 1)
				if err != nil {
					return nil, err
				}
				addItems(reward, items)
			}
		}
		hero.ProficiencyReward = max(hero.ProficiencyReward, maximum)
		updated[id] = hero
	}
	a.Heroes = updated
	if err := a.GrantItems(r, reward); err != nil {
		return nil, err
	}
	return reward, nil
}
