package invite

import (
	"mmm-revived/internal/account"
	"mmm-revived/internal/handlers/journey"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("tryBindOtherInviteCode", rpc.Bind(DecodeInviteCode, s.bindInvite))
	r.Handle("shareInviteUrl", rpc.Bind(protocol.DecodeNone, func(p *server.Player, _ struct{}) error {
		return s.journeyInvite(p, func(a *account.Account, rules *account.Rules) (map[model.ItemID]int64, error) {
			return nil, a.ShareJourneyInvite(rules)
		})
	}))
	r.Handle("getTargetInviteTaskReward", rpc.Bind(journey.DecodeJourneyPair, s.claimInvite))
	r.Handle("getTargetInviteTaskRewardByList", rpc.Bind(DecodeInviteListClaim, s.claimInviteList))
}
