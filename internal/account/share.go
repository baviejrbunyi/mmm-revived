package account

import (
	"maps"

	"mmm-revived/internal/model"
)

const DailyShareTask model.TaskID = 30001

func (a *Account) InitializeRetailShare(r *Rules, now int64) {
	if r == nil || now <= 0 {
		return
	}
	rule, known := r.Tasks[DailyShareTask]
	if !known || rule.Type != 59 {
		return
	}
	task := a.Tasks[DailyShareTask]
	if a.System.LastShare == 0 && task.Status >= 4 {
		a.System.LastShare = now
		return
	}
	if a.System.LastShare > 0 && a.System.LastShare/86400 < now/86400 && task.Status >= 3 {
		a.Tasks[DailyShareTask] = model.Task{Status: 2}
	}
}

func (a *Account) ShareOnce(r *Rules, now int64) (bool, error) {
	if r == nil || now <= 0 {
		return false, rejected("share rules are unavailable")
	}
	rule, known := r.Tasks[DailyShareTask]
	if !known || rule.Type != 59 || rule.AutoClaim != 1 || rule.Target < 1 || rule.Reward <= 0 {
		return false, rejected("daily share task is unavailable")
	}
	if a.System.LastShare > now || a.System.LastShare > 0 && a.System.LastShare/86400 == now/86400 {
		return false, nil
	}
	next := *a
	next.Tasks = maps.Clone(a.Tasks)
	if next.Tasks == nil {
		next.Tasks = map[model.TaskID]model.Task{}
	}
	next.InitializeRetailShare(r, now)
	if next.System.LastShare > 0 && next.System.LastShare/86400 == now/86400 {
		*a = next
		return false, nil
	}
	next.UpdateTask(DailyShareTask, rule.Target, rule.Target)
	if err := next.ProcessAutomaticTasks(r, now); err != nil {
		return false, err
	}
	if next.Tasks[DailyShareTask].Status != 4 {
		return false, rejected("daily share reward was not granted")
	}
	next.System.LastShare = now
	*a = next
	return true, nil
}
