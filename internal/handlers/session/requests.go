package session

import (
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type Guide struct{ Key, Value string }

type Login struct {
	Name     string
	Password string
}

func DecodeLogin(c protocol.Call) (Login, error) {
	name, _ := c.Text(0, 64)
	password, _ := c.Text(1, 256)
	return Login{Name: name, Password: password}, nil
}

func DecodeBattleID(c protocol.Call) (model.BattleID, error) { return c.BattleID() }

func DecodeGuide(c protocol.Call) (Guide, error) {
	key, err := c.Text(0, 128)
	if err != nil {
		return Guide{}, err
	}
	value, err := c.Text(1, 4096)
	return Guide{Key: key, Value: value}, err
}
