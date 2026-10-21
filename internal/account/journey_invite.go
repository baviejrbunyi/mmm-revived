package account

import (
	"strconv"
	"strings"

	"mmm-revived/internal/model"
)

func JourneyInviteCode(uid model.UserID) string {
	return "MMM" + strings.ToUpper(strconv.FormatInt(int64(uid), 36))
}

func ParseJourneyInviteCode(code string) (model.UserID, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	digits, found := strings.CutPrefix(code, "MMM")
	if !found || len(code) < 4 || len(code) > 10 {
		return 0, rejected("invalid invitation code")
	}
	value, err := strconv.ParseInt(digits, 36, 32)
	if err != nil || value <= 0 || JourneyInviteCode(model.UserID(value)) != code {
		return 0, rejected("invalid invitation code")
	}
	return model.UserID(value), nil
}

func journeyInviteEligible(index int, progress model.InviteProgress) bool {
	switch index {
	case 1011:
		return progress.Story
	case 1012:
		return progress.Level >= 15
	case 1013:
		return progress.Level >= 30
	case 1014:
		return progress.Level >= 40
	case 1015:
		return progress.Level >= 50
	case 1111, 1112, 1113:
		return progress.Purchases[index+2600]
	default:
		return false
	}
}

func (s *Store) JourneyInviteSnapshot(a *Account) (model.InviteComponent, error) {
	result := model.InviteComponent{RewardCounts: a.InviteRewardCounts(), Code: JourneyInviteCode(a.UID), BoundAt: map[model.UserID]int64{}, Tasks: map[model.UserID]map[int]int{}}
	if s.Rules == nil {
		return result, nil
	}
	rewards := s.Rules.Journey.InviteRewards
	parent, _, err := s.Inviter(a.UID)
	if err != nil {
		return result, err
	}
	if parent != 0 {
		result.BoundCode = JourneyInviteCode(parent)
	}
	rows, err := s.InviteProgress(a.UID)
	if err != nil {
		return result, err
	}
	for _, row := range rows {
		result.BoundAt[row.UID] = row.Bound
		claimed := a.Journey.InviteRewards[row.UID]
		tasks := make(map[int]int, len(rewards))
		for index := range rewards {
			switch {
			case claimed[index]:
				tasks[index] = 4
			case journeyInviteEligible(index, row):
				tasks[index] = 3
			default:
				tasks[index] = 2
			}
		}
		result.Tasks[row.UID] = tasks
	}
	for count, taskID := range s.Rules.Journey.InviteCountTasks {
		a.UpdateTask(taskID, len(rows), count)
	}
	return result, nil
}

func (a *Account) ClaimJourneyInvite(r *Rules, target model.UserID, index int, progress model.InviteProgress) (map[model.ItemID]int64, error) {
	return a.ClaimJourneyInviteList(r, target, []int{index}, progress)
}

func (a *Account) ShareJourneyInvite(r *Rules) error {
	if a.Level < 15 {
		return nil
	}
	a.Journey.SharedInvite = true
	a.UpdateTask(3701, 1, 1)
	return nil
}
