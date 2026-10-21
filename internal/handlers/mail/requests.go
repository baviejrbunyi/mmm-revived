package mail

import (
	"errors"
	"math"
	"strconv"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

func DecodeMailID(c protocol.Call) (model.MailID, error) {
	if len(c.Args) != 1 {
		return 0, errors.New("mail operation requires an ID")
	}
	var value int64
	var err error
	if text, ok := c.Args[0].(string); ok {
		value, err = strconv.ParseInt(text, 10, 64)
	} else {
		value, err = c.Number(0, 1, math.MaxInt64)
	}
	if err != nil || value <= 0 {
		return 0, errors.New("invalid mail ID")
	}
	return model.MailID(value), nil
}
