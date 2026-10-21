package calendar

import (
	"errors"
	"reflect"
	"sync/atomic"
)

const maximumOffset = 315576000

var offset atomic.Int64

func SetOffset(seconds int64) error {
	if seconds < -maximumOffset || seconds > maximumOffset {
		return errors.New("content calendar offset exceeds ten years")
	}
	offset.Store(seconds)
	return nil
}

func ClientTime(seconds int64) int64 {
	if seconds <= 0 {
		return seconds
	}
	return seconds + offset.Load()
}

func IsTimeField(name string) bool {
	switch name {
	case "create_time", "last_login_time", "server_time", "refreshTime", "expireTime", "lastDailyRewardTime", "firstShipTime", "initTime", "endTime", "startTime", "startMatchTime", "trailTime", "t", "createTime", "nameUpdateTime", "joinTime", "lastLoginTime", "lastLogoutTime", "time", "signatureLimitExpire", "childrenProtectRestTimeLastCalc", "validity", "sendTime", "lastSearchTime", "lastRecommendTime", "lastRefreshSupportTime", "guildNameUpdateTime", "lastLeaveGuildTime", "lastReq", "timestamp", "taskStartTime", "finalBossKillTime",
		"guildInvites", "joinApply", "rankCdRecord", "bindPlayerTime", "limitHeads", "limitHeadFrames", "currPack":
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
