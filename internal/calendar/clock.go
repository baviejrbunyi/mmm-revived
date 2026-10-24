package calendar

import (
	"errors"
	"reflect"
	"sync/atomic"
)

const (
	maximumOffset       = 315576000
	WeekSeconds   int64 = 7 * DaySeconds
)

var (
	offset     atomic.Int64
	loopStart  atomic.Int64
	loopPeriod atomic.Int64
	loopCycle  atomic.Int64
)

func SetOffset(seconds int64) error {
	if seconds < -maximumOffset || seconds > maximumOffset {
		return errors.New("content calendar offset exceeds ten years")
	}
	offset.Store(seconds)
	return nil
}

func SetLoop(start, period int64) {
	if period < WeekSeconds {
		start, period = 0, 0
	}
	period -= period % WeekSeconds
	loopStart.Store(start)
	loopPeriod.Store(period)
}

func Loop() (start, period int64) {
	return loopStart.Load(), loopPeriod.Load()
}

func cycleAt(now int64) int64 {
	start, period := Loop()
	moment := Monotonic(now)
	if period <= 0 || moment < start+period {
		return 0
	}
	return floorDiv(moment-start, period)
}

func Advance(now int64) (int64, bool) {
	cycle := cycleAt(now)
	return cycle, loopCycle.Swap(cycle) != cycle
}

func Cycle() int64 { return loopCycle.Load() }

func BaseOffset() int64 { return offset.Load() }

func Offset() int64 {
	_, period := Loop()
	return offset.Load() - loopCycle.Load()*period
}

func Monotonic(now int64) int64 { return now + offset.Load() }

func ContentTime(now int64) int64 { return now + Offset() }

func Loops(start, end int64) bool {
	first, period := Loop()
	if period <= 0 || end <= 0 || end <= first || start >= first+period {
		return false
	}
	return start > first || end < first+period
}

func ClientTime(seconds int64) int64 {
	if seconds <= 0 {
		return seconds
	}
	return seconds + Offset()
}

func IsTimeField(name string) bool {
	switch name {
	case "create_time", "last_login_time", "server_time", "refreshTime", "expireTime", "lastDailyRewardTime", "firstShipTime", "initTime", "endTime", "startTime", "startMatchTime", "trailTime", "t", "createTime", "nameUpdateTime", "joinTime", "lastLoginTime", "lastLogoutTime", "time", "signatureLimitExpire", "childrenProtectRestTimeLastCalc", "validity", "sendTime", "lastSearchTime", "lastRecommendTime", "lastRefreshSupportTime", "guildNameUpdateTime", "lastLeaveGuildTime", "lastReq", "lastBoxReq", "timestamp", "taskStartTime", "finalBossKillTime",
		"guildInvites", "joinApply", "rankCdRecord", "bindPlayerTime", "limitHeads", "limitHeadFrames", "currPack", "valid":
		return true
	}
	return false
}

func Shift(value any) any {
	return shifted(reflect.ValueOf(value))
}

func shifted(value reflect.Value) any {
	if !value.IsValid() {
		return nil
	}
	switch value.Kind() {
	case reflect.Int64:
		return ClientTime(value.Int())
	case reflect.Slice, reflect.Array:
		result := make([]any, value.Len())
		for i := range result {
			result[i] = shifted(value.Index(i))
		}
		return result
	case reflect.Map:
		result := make(map[any]any, value.Len())
		for key, element := range value.Seq2() {
			result[key.Interface()] = shifted(element)
		}
		return result
	}
	return value.Interface()
}
