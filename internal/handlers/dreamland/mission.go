package dreamland

import (
	"fmt"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

func (s service) missionStep(p *server.Player, mission model.MissionID, step func(*account.Rules) error) error {
	if s.Store.Rules == nil {
		return server.ErrProgressionRules
	}
	if p.State != server.Lobby {
		return fmt.Errorf("%w: battle is active", account.ErrProgression)
	}
	if err := step(s.Store.Rules); err != nil {
		return err
	}
	return s.syncMission(p, mission)
}

func (s service) claimMission(p *server.Player, mission model.MissionID) error {
	rules := s.Store.Rules
	if rules == nil {
		return server.ErrProgressionRules
	}
	before := *p.Account
	rewarded := p.Account.Passed[mission].Rewarded
	if err := p.Account.ClaimMission(rules, mission); err != nil {
		return err
	}
	if err := s.SaveAndSync(p, "dreamlandComp", "itemComp", "taskComp", "heroComp"); err != nil {
		return err
	}
	reward := rules.Missions[mission].Reward
	granted := map[model.ItemID]int64{}
	if rewarded == 0 && reward != 0 {
		granted = account.RewardQuantities(rules, reward)
	}
	if items := s.ItemRewards(before, p.Account, granted); len(items) > 0 {
		if err := p.Send("showReward", []model.RewardID{reward}, items, 3); err != nil {
			return err
		}
	} else if err := p.Send("onInitNode", mission, p.Account.Missions[mission].Node.ID); err != nil {
		return err
	}
	return p.Send("passMission", mission, p.Account.Passed[mission].Stars)
}

func (s service) failMission(p *server.Player, r MissionFail) error {
	rules := s.Store.Rules
	if rules == nil {
		return server.ErrProgressionRules
	}
	mission, ok := p.Account.Missions[r.Mission]
	if !ok || !p.Account.MissionActive(rules, r.Mission) {
		return fmt.Errorf("%w: mission is not active", account.ErrProgression)
	}
	mission.Fail = 1
	p.Account.Missions[r.Mission] = mission
	if p.Account.CurrentStory == r.Mission {
		p.Account.CurrentStory = 0
	}
	if p.Account.CurrentMemento == r.Mission {
		p.Account.CurrentMemento = 0
	}
	if p.Account.CurrentMission == r.Mission {
		p.Account.CurrentMission = 0
	}
	p.State = server.Lobby
	if err := s.SaveAndSync(p, "dreamlandComp"); err != nil {
		return err
	}
	return p.Send("failMission", r.Mission, r.Reason)
}

func (s service) claimChapter(p *server.Player, r ChapterRewards) error {
	rules := s.Store.Rules
	if rules == nil {
		return server.ErrProgressionRules
	}
	before := *p.Account
	if err := p.Account.ClaimChapter(rules, r.Chapter, r.Stars); err != nil {
		return err
	}
	if err := s.SaveAndSync(p, "dreamlandComp", "itemComp", "taskComp", "heroComp"); err != nil {
		return err
	}
	ids := make([]model.RewardID, 0, len(r.Stars))
	for _, star := range r.Stars {
		ids = append(ids, rules.Chapters[r.Chapter].Rewards[star])
	}
	if err := p.ShowRewardsIfAny(s.ItemRewards(before, p.Account, account.RewardQuantities(rules, ids...))); err != nil {
		return err
	}
	return p.Send("onDlStarReward", r.Chapter, r.Stars)
}

func (s service) syncMission(p *server.Player, id model.MissionID) error {
	if err := s.SaveAndSync(p, "dreamlandComp", "itemComp"); err != nil {
		return err
	}
	return p.Send("onInitNode", id, p.Account.Missions[id].Node.ID)
}
