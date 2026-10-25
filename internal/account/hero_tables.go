package account

import "mmm-revived/internal/model"

func (r *Rules) HeroStarReward(id model.HeroID) model.RewardID {
	if r == nil {
		return 0
	}
	return r.Constants.HeroStarRewards[id]
}

func (r *Rules) HeroTalent(id model.HeroID, slot, level int) (TalentRule, bool) {
	set, mapped := r.HeroTalentSets[id]
	rule, known := r.HeroTalents[talentRuleKey(set, slot, level)]
	return rule, mapped && known
}
