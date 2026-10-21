package model

import (
	"strconv"

	"mmm-revived/internal/json"
)

func (id *InstanceID) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		*id = InstanceID(value)
		return nil
	}
	n, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return err
	}
	*id = InstanceID(strconv.FormatInt(n, 10))
	return nil
}
