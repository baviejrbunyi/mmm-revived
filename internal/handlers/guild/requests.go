package guild

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

const textLimit = 1 << 16

type Decision[T any] struct {
	Target T
	Accept bool
}

type GuildReport struct {
	Guild  model.GuildID
	Reason string
	Text   string
}

func DecodeGuildSettings(c protocol.Call) (model.GuildSettings, error) {
	var settings model.GuildSettings
	err := c.Record(0, &settings)
	return settings, err
}

func DecodeGuildIDs(c protocol.Call) ([]model.GuildID, error) {
	if len(c.Args) == 0 {
		return nil, errors.New("missing guild list")
	}
	list, ok := c.Args[0].([]any)
	if !ok && !protocol.EmptyTable(c.Args[0]) || len(list) > 1000 {
		return nil, errors.New("invalid guild list")
	}
	guilds := make([]model.GuildID, 0, len(list))
	seen := make(map[model.GuildID]bool, len(list))
	for _, value := range list {
		id, err := protocol.Integer(value)
		if err != nil || id < 1 || seen[model.GuildID(id)] {
			continue
		}
		seen[model.GuildID(id)] = true
		guilds = append(guilds, model.GuildID(id))
	}
	return guilds, nil
}

func DecodeGuildText(c protocol.Call) (string, error) {
	return c.Text(0, textLimit)
}

func DecodeGuildSearch(c protocol.Call) (string, error) {
	if len(c.Args) == 0 {
		return "", nil
	}
	text, _ := c.Args[0].(string)
	if text = strings.TrimSpace(text); len(text) > 256 || !protocol.ValidText(text) {
		return "", nil
	}
	return text, nil
}

func DecodeGuildID(c protocol.Call) (model.GuildID, error) {
	id, err := c.Number(0, 1, math.MaxInt64)
	return model.GuildID(id), err
}

func DecodeGuildSearchID(c protocol.Call) (model.GuildID, error) {
	if len(c.Args) == 0 {
		return 0, nil
	}
	if text, ok := c.Args[0].(string); ok {
		id, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil || id < 1 {
			return 0, nil
		}
		return model.GuildID(id), nil
	}
	id, err := c.Number(0, 1, math.MaxInt64)
	if err != nil {
		return 0, nil
	}
	return model.GuildID(id), nil
}

func decision(c protocol.Call) (bool, error) {
	if len(c.Args) != 2 {
		return false, errors.New("missing guild decision")
	}
	flag, ok := c.Args[1].(bool)
	if !ok {
		return false, errors.New("invalid guild decision")
	}
	return flag, nil
}

func decodeDecision[T ~int64 | ~uint64](c protocol.Call, minimum int64) (Decision[T], error) {
	id, err := c.Number(0, minimum, math.MaxInt64)
	if err != nil {
		return Decision[T]{}, err
	}
	accept, err := decision(c)
	return Decision[T]{Target: T(id), Accept: accept}, err
}

func DecodeAdminDecision(c protocol.Call) (Decision[model.UserID], error) {
	return decodeDecision[model.UserID](c, 1)
}

func DecodeApplicationDecision(c protocol.Call) (Decision[model.UserID], error) {
	return decodeDecision[model.UserID](c, 0)
}

func DecodeInviteDecision(c protocol.Call) (Decision[model.GuildID], error) {
	return decodeDecision[model.GuildID](c, 1)
}

func DecodeGuildReport(c protocol.Call) (GuildReport, error) {
	guild, err := DecodeGuildSearchID(c)
	if err != nil || guild == 0 {
		return GuildReport{}, errors.New("invalid guild report target")
	}
	reason, err := c.Text(1, textLimit)
	if err != nil {
		return GuildReport{}, err
	}
	var text string
	if len(c.Args) > 2 {
		if text, err = c.Text(2, textLimit); err != nil {
			return GuildReport{}, err
		}
	}
	if !utf8.ValidString(reason) || !utf8.ValidString(text) {
		return GuildReport{}, errors.New("invalid guild report")
	}
	return GuildReport{Guild: guild, Reason: reason, Text: text}, nil
}
