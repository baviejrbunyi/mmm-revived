package task

import (
	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

func (s service) claimTasks(p *server.Player, tasks []model.TaskID) error {
	rules := s.Store.Rules
	if rules == nil {
		return server.ErrProgressionRules
	}
	if len(tasks) == 0 {
		return server.Acknowledge(p)
	}
	before := *p.Account
	if err := p.Account.ClaimTasks(rules, tasks); err != nil {
		return err
	}
	if err := s.SaveAndSync(p, "taskComp", "itemComp", "heroComp", "achieveComp"); err != nil {
		return err
	}
	ids := make([]model.RewardID, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, rules.Tasks[task].Reward)
	}
	rewards := s.ItemRewards(before, p.Account, account.RewardQuantities(rules, ids...))
	if len(tasks) == 1 {
		for index := range rewards {
			rewards[index].Details.Task = tasks[0]
		}
	}
	return p.ShowRewardsIfAny(rewards)
}

func (s service) claimLevels(p *server.Player, levels []int) error {
	rules := s.Store.Rules
	if rules == nil {
		return server.ErrProgressionRules
	}
	before := *p.Account
	if err := p.Account.ClaimLevels(rules, levels); err != nil {
		return err
	}
	if err := s.SaveAndSync(p, "itemComp", "taskComp", "heroComp"); err != nil {
		return err
	}
	ids := make([]model.RewardID, 0, len(levels))
	for _, level := range levels {
		ids = append(ids, rules.TeamLevels[level].Reward)
	}
	if err := p.ShowRewardsIfAny(s.ItemRewards(before, p.Account, account.RewardQuantities(rules, ids...))); err != nil {
		return err
	}
	return p.Send("onGetLvReward", levels)
}
