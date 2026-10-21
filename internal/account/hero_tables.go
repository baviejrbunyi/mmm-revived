package account

import "mmm-revived/internal/model"

var heroTalentSets = map[model.HeroID]model.HeroID{
	114: 113, 116: 107, 118: 107, 120: 107, 124: 127, 140: 107, 141: 107, 142: 107,
	143: 107, 147: 107, 149: 107, 198: 101, 199: 101, 901: 103, 903: 127,
}

var heroStarRewards = map[model.HeroID]model.RewardID{
	101: 60000201, 102: 60000202, 103: 60000203, 104: 60000204, 105: 60000205, 106: 60000206,
	107: 60000207, 108: 60000208, 109: 60000209, 110: 60000210, 111: 60000211, 112: 60000212,
	113: 60000213, 114: 60000214, 115: 60000215, 116: 60000216, 117: 60000217, 119: 60000219,
	122: 60000222, 123: 60000223, 124: 60000224, 125: 60000225, 126: 60000226, 127: 60000227,
	128: 60000228, 130: 60000230, 131: 60000231, 132: 60000232, 133: 60000233, 134: 60000234,
	135: 60000235, 136: 60000236, 137: 60000237, 138: 60000238, 139: 60000239, 144: 60000244,
	145: 60000245, 146: 60000246, 148: 60000247,
}

var heroAwakeSubstitutes = map[int]model.ItemID{4: 41, 5: 42}

const trialHeroMaxTalent = 3

func HeroStarReward(id model.HeroID) model.RewardID { return heroStarRewards[id] }

func (r *Rules) HeroTalent(id model.HeroID, slot, level int) (TalentRule, bool) {
	if set, mapped := heroTalentSets[id]; mapped {
		id = set
	}
	rule, known := r.HeroTalents[talentRuleKey(id, slot, level)]
	return rule, known
}
