package task

import (
	"errors"
	"math"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

func DecodeLevelRewards(c protocol.Call) ([]int, error) { return c.Integers(0, 200, 1, 1000) }

func DecodeTasks(c protocol.Call) ([]model.TaskID, error) {
	if len(c.Args) != 1 {
		return nil, errors.New("getTaskRwd requires a task list")
	}
	if protocol.EmptyTable(c.Args[0]) {
		return []model.TaskID{}, nil
	}
	list, ok := c.Args[0].([]any)
	if !ok || len(list) > 100 {
		return nil, errors.New("invalid task list")
	}
	tasks := make([]model.TaskID, 0, len(list))
	seen := make(map[model.TaskID]bool, len(list))
	for _, value := range list {
		n, err := protocol.Integer(value)
		if err != nil || n <= 0 || n > math.MaxInt32 || seen[model.TaskID(n)] {
			return nil, errors.New("invalid task ID")
		}
		seen[model.TaskID(n)] = true
		tasks = append(tasks, model.TaskID(n))
	}
	return tasks, nil
}

func DecodeShareOnce(c protocol.Call) (struct{}, error) {
	return struct{}{}, c.Arguments(0)
}
