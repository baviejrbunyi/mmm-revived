package friend

import (
	"errors"
	"math"
	"strings"
	"unicode/utf8"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type FriendApplication struct {
	Target model.UserID
	Source model.FriendSource
}

type FriendDecision struct {
	Target model.UserID
	Accept bool
}

type FriendName struct {
	Target model.UserID
	Name   string
}

type SendChat struct {
	Channel model.ChatChannel
	Kind    int
	Text    string
	Target  model.UserID
}

func DecodeSearchID(c protocol.Call) (model.UserID, error) {
	if len(c.Args) == 0 || c.Args[0] == nil {
		return 0, nil
	}
	id, err := protocol.Integer(c.Args[0])
	if err != nil || id < 0 {
		return 0, nil
	}
	return model.UserID(id), nil
}

func DecodeChatPeer(c protocol.Call) (model.UserID, error) { return c.Player(0) }

func DecodePlayers(c protocol.Call) ([]model.UserID, error) {
	ids, err := c.Integers(0, 150, 1, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	players := make([]model.UserID, len(ids))
	for i, id := range ids {
		players[i] = model.UserID(id)
	}
	return players, nil
}

func DecodeSearchName(c protocol.Call) (string, error) {
	if len(c.Args) == 0 {
		return "", nil
	}
	name, _ := c.Args[0].(string)
	if name = strings.TrimSpace(name); len(name) > 256 || !protocol.ValidText(name) {
		return "", nil
	}
	return name, nil
}

func DecodeFriendApplication(c protocol.Call) (FriendApplication, error) {
	target, err := protocol.DecodePlayer(c)
	if err != nil {
		return FriendApplication{}, err
	}
	var source model.FriendSource
	if len(c.Args) > 1 {
		value, _ := c.Args[1].(string)
		source, _ = model.ParseFriendSource(value)
	}
	return FriendApplication{Target: target, Source: source}, nil
}

func DecodeFriendDecision(c protocol.Call) (FriendDecision, error) {
	target, err := protocol.DecodePlayer(c)
	if err != nil {
		return FriendDecision{}, err
	}
	if len(c.Args) != 2 {
		return FriendDecision{}, errors.New("missing friend decision")
	}
	accept, ok := c.Args[1].(bool)
	if !ok {
		return FriendDecision{}, errors.New("invalid friend decision")
	}
	return FriendDecision{Target: target, Accept: accept}, nil
}

func DecodeFriendName(c protocol.Call) (FriendName, error) {
	target, err := protocol.DecodePlayer(c)
	if err != nil {
		return FriendName{}, err
	}
	name, err := c.Text(1, 128)
	if err != nil {
		return FriendName{}, err
	}
	if !utf8.ValidString(name) {
		return FriendName{}, errors.New("invalid friend nickname")
	}
	return FriendName{Target: target, Name: name}, nil
}

func DecodeChat(c protocol.Call) (SendChat, error) {
	channel, err := c.Number(0, 1, 3)
	if err != nil {
		return SendChat{}, err
	}
	kind, err := c.Number(1, 0, 7)
	if err != nil {
		return SendChat{}, err
	}
	text, err := c.Text(2, 1<<16)
	if err != nil {
		return SendChat{}, err
	}
	var destination struct {
		Target model.UserID `json:"targetUid"`
	}
	if len(c.Args) > 3 {
		if err = c.Record(3, &destination); err != nil {
			return SendChat{}, err
		}
	}
	return SendChat{Channel: model.ChatChannel(channel), Kind: int(kind), Text: text, Target: destination.Target}, nil
}
