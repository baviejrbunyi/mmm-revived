package gm

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"mmm-revived/internal/handlers/system"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

func parseGMNumber(value string, minimum, maximum int64) (int64, error) {
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil || number < minimum || number > maximum {
		return 0, errors.New("invalid numeric argument")
	}
	return number, nil
}

func parseGMCommand(text string) (model.GMCommand, error) {
	if len(text) > 4096 || !utf8.ValidString(text) || strings.ContainsAny(text, "\x00\r\n") {
		return model.GMCommand{}, errors.New("invalid command text")
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return model.GMCommand{}, errors.New("empty command")
	}
	name := strings.ToLower(fields[0])
	result := model.GMCommand{Name: name, Kind: model.GMUnknown}
	count := func(minimum, maximum int) bool { return len(fields)-1 >= minimum && len(fields)-1 <= maximum }
	number := func(index int, minimum, maximum int64) (int64, error) {
		if index >= len(fields) {
			return 0, errors.New("missing numeric argument")
		}
		return parseGMNumber(fields[index], minimum, maximum)
	}
	var err error
	switch name {
	case "gmlist", "gmdoc", "help":
		result.Kind = model.GMList
	case "getall", "allitems":
		result.Kind = model.GMGetAll
		result.Value = 9999999
		if !count(0, 1) {
			return result, errors.New("getall [count]")
		}
		if len(fields) == 2 {
			result.Value, err = number(1, 1, 1000000000)
		}
	case "item", "additem", "getitem", "giveitem":
		result.Kind = model.GMAddItem
		result.Value = 1
		if !count(1, 2) {
			return result, errors.New("item <id> [count]")
		}
		result.ID, err = number(1, 1, math.MaxInt32)
		if err == nil && len(fields) == 3 {
			result.Value, err = number(2, 1, 1000000000)
		}
	case "setitem":
		result.Kind = model.GMSetItem
		if !count(2, 2) {
			return result, errors.New("setitem <id> <count>")
		}
		result.ID, err = number(1, 1, math.MaxInt32)
		if err == nil {
			result.Value, err = number(2, 0, 1000000000)
		}
	case "level", "teamlv", "setlevel", "setlv":
		result.Kind = model.GMTeamLevel
		if !count(1, 1) {
			return result, errors.New("level <level>")
		}
		result.Value, err = number(1, 1, 1000)
	case "herolevel", "herolv":
		result.Kind = model.GMHeroLevel
		if !count(2, 2) {
			return result, errors.New("herolevel <hero|0> <level>")
		}
		result.ID, err = number(1, 0, math.MaxInt32)
		if err == nil {
			result.Value, err = number(2, 1, 600)
		}
	case "herostar":
		result.Kind = model.GMHeroStar
		if !count(2, 2) {
			return result, errors.New("herostar <hero> <star>")
		}
		result.ID, err = number(1, 1, math.MaxInt32)
		if err == nil {
			result.Value, err = number(2, 1, 5)
		}
	case "hero", "addhero":
		result.Kind = model.GMAddHero
		if !count(1, 1) {
			return result, errors.New("hero <id>")
		}
		result.ID, err = number(1, 1, math.MaxInt32)
	case "guidefin":
		result.Kind = model.GMGuideFinish
	case "endnovice":
		result.Kind = model.GMNoviceFinish
	case "setnovice":
		result.Kind = model.GMNoviceSet
		if !count(1, 1) {
			return result, errors.New("setnovice <task|0>")
		}
		result.ID, err = number(1, 0, math.MaxInt32)
	case "alldl":
		result.Kind = model.GMDreamlandFinish
		result.Text = "all"
	case "enddreamland":
		result.Kind = model.GMDreamlandFinish
		if !count(1, 2) {
			return result, errors.New("enddreamland <story|memento_easy|all|clear> [chapter]")
		}
		result.Text = strings.ToLower(fields[1])
		if len(fields) == 3 {
			result.ID, err = number(2, 0, 100)
		}
	case "nextday":
		result.Kind = model.GMNextDay
	case "gettime":
		result.Kind = model.GMGetTime
	case "settime":
		result.Kind = model.GMSetTime
		if !count(0, 1) {
			return result, errors.New("settime [YYYY.MM.DD.hh.mm.ss]")
		}
		if len(fields) == 2 {
			date, parseErr := time.ParseInLocation("2006.01.02.15.04.05", fields[1], time.UTC)
			if parseErr != nil {
				result.Value, err = number(1, 1, math.MaxInt32)
			} else {
				result.Value = date.Unix()
			}
		}
	case "mail":
		result.Kind = model.GMMail
		if !count(1, 1) {
			return result, errors.New("mail <template>")
		}
		result.ID, err = number(1, 1, math.MaxInt32)
	case "energy":
		result.Kind = model.GMSetEnergy
		if !count(1, 1) {
			return result, errors.New("energy <amount>")
		}
		result.Value, err = number(1, 0, 1000000000)
	case "resetboss":
		result.Kind = model.GMBossReset
	case "resetscience":
		result.Kind = model.GMScienceReset
	case "seasonlevel":
		result.Kind = model.GMScienceLevel
		if !count(1, 1) {
			return result, errors.New("seasonlevel <level>")
		}
		result.Value, err = number(1, 1, 10000)
	case "gmrogue":
		result.Kind = model.GMRogueInitialize
	case "gmroguepve":
		result.Kind = model.GMRoguePVEInitialize
		result.ID = 1
		if !count(0, 1) {
			return result, errors.New("gmroguepve [difficulty]")
		}
		if len(fields) == 2 {
			result.ID, err = number(1, 1, math.MaxInt32)
		}
	case "save":
		result.Kind = model.GMSave
	case "checkchapterend":
		result.Kind = model.GMCheckChapters
	case "battlecancel":
		result.Kind = model.GMResetBattle
	case "doguildgm":
		result.Kind = model.GMGuild
		if !count(1, 1) {
			return result, errors.New("doGuildGm <action>")
		}
		result.Text = strings.ToLower(fields[1])
	}
	if err == nil {
		switch result.Kind {
		case model.GMList, model.GMGuideFinish, model.GMNoviceFinish, model.GMNextDay, model.GMGetTime, model.GMBossReset, model.GMScienceReset, model.GMRogueInitialize, model.GMSave, model.GMCheckChapters, model.GMResetBattle:
			if !count(0, 0) {
				err = fmt.Errorf("%s takes no arguments", name)
			}
		case model.GMDreamlandFinish:
			if name == "alldl" && !count(0, 0) {
				err = errors.New("alldl takes no arguments")
			}
		}
	}
	return result, err
}

type GMCommand struct {
	Text       string
	Command    model.GMCommand
	Validation string
	Ship       *system.Purchase
	Recharge   string
}

func parseShip(text string) (*system.Purchase, string, bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 || strings.ToLower(fields[0]) != "ship" {
		return nil, "", false
	}
	if len(fields) != 2 && len(fields) != 5 {
		return nil, "", true
	}
	var values [4]int64
	values[1], values[2] = 1, 1
	recharge := ""
	for index, field := range fields[1:] {
		number, err := parseGMNumber(field, 0, math.MaxInt32)
		if err != nil {
			if index != 0 || len(field) > 128 {
				return nil, "", true
			}
			recharge = field
		}
		values[index] = number
	}
	return &system.Purchase{Product: int(values[0]), Count: int(values[1]), Kind: int(values[2]), Reference: int(values[3])}, recharge, true
}

func DecodeGMCommand(c protocol.Call) (GMCommand, error) {
	if err := c.Arguments(1); err != nil {
		return GMCommand{}, err
	}
	text, err := c.Text(0, 4096)
	if err != nil {
		return GMCommand{}, err
	}
	r := GMCommand{Text: text}
	if ship, recharge, matched := parseShip(text); matched {
		if ship == nil {
			r.Validation = "ship <rechargeId> [count kind reference]"
		}
		r.Ship, r.Recharge = ship, recharge
		return r, nil
	}
	if r.Command, err = parseGMCommand(text); err != nil {
		r.Validation = err.Error()
	}
	return r, nil
}

func DecodeGMTokenRequest(c protocol.Call) (struct{}, error) {
	switch len(c.Args) {
	case 0:
		return struct{}{}, nil
	case 1:
		if _, ok := c.Args[0].(map[any]any); ok {
			return struct{}{}, nil
		}
	}
	return struct{}{}, errors.New("invalid GM token request")
}

func DecodeGMUploadURL(c protocol.Call) (string, error) {
	if err := c.Arguments(1); err != nil {
		return "", err
	}
	return c.Text(0, 2048)
}

func DecodeCoverageUser(c protocol.Call) (string, error) {
	switch len(c.Args) {
	case 0:
		return "", nil
	case 1:
		return c.Text(0, 128)
	}
	return "", errors.New("invalid coverage user")
}

func DecodeGMHistory(c protocol.Call) (string, error) {
	if err := c.Arguments(1); err != nil {
		return "", err
	}
	return c.Text(0, 1<<20)
}
