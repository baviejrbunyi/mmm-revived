package invite

import (
	"math"

	"mmm-revived/internal/handlers/journey"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

func DecodeInviteCode(c protocol.Call) (string, error) {
	if err := c.Arguments(1); err != nil {
		return "", err
	}
	return c.Text(0, 256)
}

type InviteListClaim struct {
	Target  model.UserID
	Indices []int
}

func DecodeInviteListClaim(c protocol.Call) (InviteListClaim, error) {
	if err := c.Arguments(2); err != nil {
		return InviteListClaim{}, err
	}
	uid, err := c.Number(0, 1, math.MaxInt32)
	if err != nil {
		return InviteListClaim{}, err
	}
	indices, err := journey.ClaimList(c, 1, 1)
	return InviteListClaim{Target: model.UserID(uid), Indices: indices}, err
}
