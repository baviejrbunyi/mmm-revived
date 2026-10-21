package duel

import (
	"errors"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type Challenge struct {
	Target model.UserID
	Mode   model.MatchMode
	Manual bool
}

type Reply struct {
	Target model.UserID
	Accept bool
}

func DecodeChallenge(c protocol.Call) (Challenge, error) {
	if len(c.Args) != 3 {
		if err := c.Arguments(2); err != nil {
			return Challenge{}, err
		}
	}
	target, err := protocol.DecodePlayer(c)
	if err != nil {
		return Challenge{}, err
	}
	mode, err := c.Number(1, int64(model.MatchOwnHeroes), int64(model.MatchDraft))
	if err != nil {
		return Challenge{}, err
	}
	challenge := Challenge{Target: target, Mode: model.MatchMode(mode)}
	if len(c.Args) == 3 {
		manual, ok := c.Args[2].(bool)
		if !ok {
			return Challenge{}, errors.New("invalid duel control")
		}
		challenge.Manual = manual
	}
	return challenge, nil
}

func DecodeReply(c protocol.Call) (Reply, error) {
	if err := c.Arguments(2); err != nil {
		return Reply{}, err
	}
	target, err := protocol.DecodePlayer(c)
	if err != nil {
		return Reply{}, err
	}
	accept, ok := c.Args[1].(bool)
	if !ok {
		return Reply{}, errors.New("invalid duel reply")
	}
	return Reply{Target: target, Accept: accept}, nil
}
