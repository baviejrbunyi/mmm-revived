package profile

import (
	"errors"
	"math"
	"strings"
	"unicode/utf8"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

const (
	codeCommon                = 5
	codeSignatureLong         = 7
	codeNameLength            = 35
	codeNameNumeric           = 64
	codeSignature             = 71
	codeItemsMissing          = 132
	codeNameCharacter         = 139
	codeNameEmpty             = 141
	nameMinimum               = 2
	nameMaximum               = 14
	signatureMaximum          = 50
	lineupHeroes              = 3
	nameLatinLetters          = "ÀÁÂÃÄÇÈÉÊËÌÍÎÏÑÒÓÔÕÖÙÚÛÜàáâãäçèéêëìíîïñòóôõöùúûüßŒœŸẞ"
	nameCJKFirst, nameCJKLast = 0x4E00, 0x9FA5
)

var honorTags = map[string]bool{"": true, "story": true, "heroes": true, "skins": true, "createTime": true, "season": true, "achievePoint": true, "1004102": true, "1004111": true, "1004121": true, "1004131": true, "2000000": true}

type RedDot struct {
	Key   string
	Value int64
}

type Lineup struct {
	Slot   int
	Kind   int
	Name   string
	Heroes []model.HeroID
}

func DecodeText(c protocol.Call) (string, error) { return c.Text(0, 4096) }

func nameLength(value string, latin int) (int, bool) {
	length := 0
	for _, r := range value {
		switch {
		case '0' <= r && r <= '9', 'A' <= r && r <= 'Z', 'a' <= r && r <= 'z':
			length++
		case strings.ContainsRune(nameLatinLetters, r):
			length += latin
		case nameCJKFirst <= r && r <= nameCJKLast:
			length += 2
		default:
			return 0, false
		}
	}
	return length, true
}

func nameCode(value string, latin int) int {
	length, ok := nameLength(value, latin)
	switch {
	case value == "":
		return codeNameEmpty
	case !ok:
		return codeNameCharacter
	case length < nameMinimum || length > nameMaximum:
		return codeNameLength
	}
	return 0
}

func teamNameCode(value string) int {
	if code := nameCode(value, 1); code != 0 {
		return code
	}
	if strings.Trim(value, "0123456789") == "" {
		return codeNameNumeric
	}
	return 0
}

func lineupNameCode(value string) int { return nameCode(value, 2) }

func signatureCode(value string) int {
	switch {
	case !protocol.ValidText(value):
		return codeSignature
	case utf8.RuneCountInString(value) > signatureMaximum:
		return codeSignatureLong
	}
	return 0
}

func DecodeRedDot(c protocol.Call) (RedDot, error) {
	key, err := c.Text(0, 640)
	if err != nil || !protocol.ValidText(key) {
		return RedDot{}, errors.New("invalid red dot key")
	}
	value, err := c.Number(1, 0, math.MaxInt64)
	return RedDot{Key: key, Value: value}, err
}

func DecodeShownHeroes(c protocol.Call) ([]model.HeroID, error) { return c.HeroList(0, 3) }

func DecodeTags(c protocol.Call) (model.HonorTags, error) {
	values, err := c.Strings(0, 6, 64)
	if err != nil {
		return nil, err
	}
	tags := make(model.HonorTags, len(values))
	for index, value := range values {
		if honorTags[value] {
			tags[index] = value
		}
	}
	return tags, nil
}

func DecodeCosmetic(c protocol.Call) (model.ItemID, error) {
	value, err := c.Number(0, 0, math.MaxInt32)
	return model.ItemID(value), err
}

func DecodeLineupSlot(c protocol.Call) (int, error) {
	slot, err := c.Number(0, 0, math.MaxInt32)
	return int(slot), err
}

func DecodeLineup(c protocol.Call) (Lineup, error) {
	slot, err := DecodeLineupSlot(c)
	if err != nil {
		return Lineup{}, err
	}
	kind, err := c.Number(1, 0, math.MaxInt32)
	if err != nil {
		return Lineup{}, err
	}
	heroes, err := c.HeroList(2, 5)
	return Lineup{Slot: slot, Kind: int(kind), Heroes: heroes}, err
}

func DecodeLineupName(c protocol.Call) (Lineup, error) {
	slot, err := DecodeLineupSlot(c)
	if err != nil {
		return Lineup{}, err
	}
	name, err := c.Text(1, 4096)
	return Lineup{Slot: slot, Name: name}, err
}

func DecodeLineupType(c protocol.Call) (Lineup, error) {
	slot, err := DecodeLineupSlot(c)
	if err != nil {
		return Lineup{}, err
	}
	kind, err := c.Number(1, 0, math.MaxInt32)
	return Lineup{Slot: slot, Kind: int(kind)}, err
}

type NameCards struct {
	Players []model.UserID
	Keys    []string
}

func DecodeNameCards(c protocol.Call) (NameCards, error) {
	if len(c.Args) != 2 {
		return NameCards{}, errors.New("name cards require keys and player IDs")
	}
	ids, ok := c.Args[1].([]any)
	if !ok || len(ids) == 0 || len(ids) > 100 {
		return NameCards{}, errors.New("invalid name card player list")
	}
	keys, ok := c.Args[0].([]any)
	if !ok || len(keys) > 40 {
		return NameCards{}, errors.New("invalid name card fields")
	}
	request := NameCards{Players: make([]model.UserID, 0, len(ids)), Keys: make([]string, 0, len(keys))}
	for _, value := range keys {
		key, ok := value.(string)
		if !ok || len(key) > 64 {
			return NameCards{}, errors.New("invalid name card field")
		}
		request.Keys = append(request.Keys, key)
	}
	for _, value := range ids {
		id, err := protocol.Integer(value)
		if err != nil || id <= 0 {
			return NameCards{}, errors.New("invalid player ID")
		}
		request.Players = append(request.Players, model.UserID(id))
	}
	return request, nil
}
