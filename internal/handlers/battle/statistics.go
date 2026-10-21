package battle

import "mmm-revived/internal/protocol"

type battleStatistics struct {
	Mana      int64
	Cards     int64
	Ultimates int64
}

func statisticsFrom(packet map[any]any) battleStatistics {
	var result battleStatistics
	extra, _ := packet["extra"].(map[any]any)
	var actors []any
	switch values := extra["statistics"].(type) {
	case map[any]any:
		for _, value := range values {
			actors = append(actors, value)
		}
	case []any:
		actors = values
	}
	number := func(values map[any]any, key string) int64 {
		if real, ok := values[key].(float64); ok && real >= 0 && real <= 1000000 {
			return int64(real)
		}
		value, err := protocol.Integer(values[key])
		if err != nil || value < 0 || value > 1000000 {
			return 0
		}
		return value
	}
	for _, actor := range actors {
		values, ok := actor.(map[any]any)
		if !ok {
			continue
		}
		result.Mana += number(values, "mana")
		result.Cards += number(values, "use_card_time")
		result.Ultimates += number(values, "use_active_skill_card_time")
	}
	return result
}
