package system

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type LobbyHero struct {
	Hero    model.HeroID
	Skin    model.SkinID
	Dialogs []string
	Kind    int
}

type Setting struct {
	Key     string
	Value   model.SettingValue
	Missing bool
}

type Purchase struct {
	Product   int
	Count     int
	Kind      int
	Reference int
}

type LeaveReason struct {
	Stage   string
	Tags    string
	Content string
}

type BugReport struct {
	Battle  model.BattleID
	Version int
	Digest  string
}

func DecodeLobbyHero(c protocol.Call) (LobbyHero, error) {
	if err := c.Arguments(5); err != nil {
		return LobbyHero{}, err
	}
	if slot, err := c.Number(0, 1, 1); err != nil || slot != 1 {
		return LobbyHero{}, errors.New("invalid lobby slot")
	}
	hero, err := c.Number(1, 0, math.MaxInt32)
	if err != nil {
		return LobbyHero{}, err
	}
	r := LobbyHero{Hero: model.HeroID(hero)}
	if c.Args[2] != nil {
		skin, err := c.Number(2, 0, math.MaxInt32)
		if err != nil {
			return LobbyHero{}, err
		}
		r.Skin = model.SkinID(skin)
	}
	if r.Dialogs, err = c.Strings(3, 32, 256); err != nil {
		return LobbyHero{}, err
	}
	kind, err := c.Number(4, 0, 1)
	r.Kind = int(kind)
	return r, err
}

func DecodeMirage(c protocol.Call) (model.ActivityID, error) {
	if err := c.Arguments(1); err != nil {
		return 0, err
	}
	return protocol.DecodeActivity(c)
}

func DecodePIN(c protocol.Call) (string, error) {
	if err := c.Arguments(1); err != nil {
		return "", err
	}
	return c.Text(0, 4)
}

func DecodePushToken(c protocol.Call) (string, error) {
	if err := c.Arguments(1); err != nil {
		return "", err
	}
	return c.Text(0, 4096)
}

func DecodePushTags(c protocol.Call) ([]string, error) {
	if err := c.Arguments(1); err != nil {
		return nil, err
	}
	return c.Strings(0, 32, 128)
}

func DecodeSetting(c protocol.Call) (Setting, error) {
	if len(c.Args) != 1 && len(c.Args) != 2 {
		return Setting{}, errors.New("invalid setting argument count")
	}
	key, err := c.Text(0, 64)
	if err != nil {
		return Setting{}, err
	}
	r := Setting{Key: key}
	if len(c.Args) == 1 {
		r.Missing = true
		return r, nil
	}
	switch value := c.Args[1].(type) {
	case nil:
		r.Missing = true
	case string:
		r.Value = model.SettingValue{Kind: model.SettingText, Text: value}
	case bool:
		r.Value = model.SettingValue{Kind: model.SettingBoolean, Boolean: value}
	case int64:
		r.Value = model.SettingValue{Kind: model.SettingNumber, Number: float64(value)}
	case uint64:
		r.Value = model.SettingValue{Kind: model.SettingNumber, Number: float64(value)}
	case float64:
		r.Value = model.SettingValue{Kind: model.SettingNumber, Number: value}
	default:
		return Setting{}, errors.New("invalid setting value")
	}
	return r, nil
}

func DecodePurchase(c protocol.Call) (Purchase, error) {
	if err := c.Arguments(4); err != nil {
		return Purchase{}, err
	}
	var values [4]int
	for index := range values {
		value, err := c.Number(index, 0, math.MaxInt32)
		if err != nil {
			return Purchase{}, err
		}
		values[index] = int(value)
	}
	return Purchase{Product: values[0], Count: values[1], Kind: values[2], Reference: values[3]}, nil
}

func DecodeGiftOffer(c protocol.Call) (int, error) {
	if err := c.Arguments(1); err != nil {
		return 0, err
	}
	value, err := c.Number(0, 1, math.MaxInt32)
	return int(value), err
}

func DecodeFlag(c protocol.Call) (bool, error) {
	if err := c.Arguments(1); err != nil {
		return false, err
	}
	value, ok := c.Args[0].(bool)
	if !ok {
		return false, errors.New("invalid boolean setting")
	}
	return value, nil
}

func DecodeLeaveReason(c protocol.Call) (LeaveReason, error) {
	if err := c.Arguments(3); err != nil {
		return LeaveReason{}, err
	}
	stage, err := c.Text(0, 64)
	if err != nil {
		return LeaveReason{}, err
	}
	tags, err := c.Text(1, 1<<16)
	if err != nil {
		return LeaveReason{}, err
	}
	content, err := c.Text(2, 1<<16)
	if err != nil {
		return LeaveReason{}, err
	}
	return LeaveReason{Stage: stage, Tags: tags, Content: content}, nil
}

func DecodeBugReport(c protocol.Call) (BugReport, error) {
	if len(c.Args) < 4 || len(c.Args) > 8 {
		return BugReport{}, errors.New("invalid bug report argument count")
	}
	battle, err := c.Number(0, 1, math.MaxInt32)
	if err != nil {
		return BugReport{}, err
	}
	version, err := c.Number(1, 0, math.MaxInt32)
	if err != nil {
		return BugReport{}, err
	}
	record, err := c.Text(3, 8<<20)
	if err != nil {
		return BugReport{}, err
	}
	digest := sha256.Sum256([]byte(record))
	return BugReport{Battle: model.BattleID(battle), Version: int(version), Digest: hex.EncodeToString(digest[:])}, nil
}
