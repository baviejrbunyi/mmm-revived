package network

import (
	"database/sql"

	"mmm-revived/internal/account"
	"mmm-revived/internal/handlers/battle"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("reqRecommendSupportHeroes", network(s, protocol.DecodeIgnore, func(p *server.Player, _ struct{}, now int64) error {
		return s.recommendSupport(p, now)
	}))
	r.Handle("chooseSupportHero", network(s, DecodeSupport, func(p *server.Player, support Support, _ int64) error {
		return s.chooseSupport(p, support, false)
	}))
	r.Handle("guildWarChooseSupportHero", network(s, DecodeSupport, func(p *server.Player, support Support, _ int64) error {
		return s.chooseSupport(p, support, true)
	}))
	r.Handle("getSupportHeroReward", network(s, protocol.DecodeIgnore, func(p *server.Player, _ struct{}, _ int64) error {
		return s.claimSupport(p)
	}))
	r.Handle("updatePvPTeamInfo", network(s, DecodeDefense, func(p *server.Player, defense Defense, _ int64) error {
		err := s.Store.Update(p.Account, func(*sql.Tx) error {
			return p.Account.SetNetworkDefense(s.Store.Rules.Network, defense.Heroes, defense.Order, defense.Interactions, defense.Environment, defense.Camp)
		})
		if err != nil {
			return err
		}
		if err = s.syncNetwork(p); err != nil {
			return err
		}
		return p.Send("onAttachPlayGround")
	}))
	r.Handle("attachPlayGround", network(s, protocol.DecodeIgnore, func(p *server.Player, _ struct{}, now int64) error {
		if err := s.refreshAsyncTargets(p, false, now); err != nil {
			return err
		}
		if err := notifyTargets(p); err != nil {
			return err
		}
		return p.Send("onAttachPlayGround")
	}))
	r.Handle("clientRefreshTargets", network(s, protocol.DecodeIgnore, func(p *server.Player, _ struct{}, now int64) error {
		if err := s.refreshAsyncTargets(p, true, now); err != nil {
			return err
		}
		return notifyTargets(p)
	}))
	r.Handle("getFriendsPvpInfo", network(s, protocol.DecodeIgnore, func(p *server.Player, _ struct{}, _ int64) error {
		return s.asyncFriends(p, 0)
	}))
	r.Handle("getSpecifyFriendPvpInfo", network(s, DecodeFriendTarget, func(p *server.Player, uid model.UserID, _ int64) error {
		return s.asyncFriends(p, uid)
	}))
	r.Handle("getPvpRankReward", network(s, DecodeRankRewards, func(p *server.Player, ids []int, _ int64) error {
		return s.claimNetwork(p, func() (map[model.ItemID]int64, error) { return p.Account.ClaimNetworkRanks(s.Store.Rules, ids) })
	}))
	r.Handle("gainAsyncPvpReward", network(s, DecodeActivityRewards, func(p *server.Player, r ActivityRewards, _ int64) error {
		return s.claimNetwork(p, func() (map[model.ItemID]int64, error) {
			return p.Account.ClaimNetworkTasks(s.Store.Rules, r.Activity, r.IDs)
		})
	}))
	r.Handle("startAsyncPvPBattle", network(s, DecodeAsyncStart, s.startAsyncBattle))
	r.Handle("getFriendRank", network(s, DecodeRankQuery, func(p *server.Player, q RankQuery, now int64) error {
		return s.personalRanks(p, q, false, now)
	}))
	r.Handle("getGuildRank", network(s, DecodeRankQuery, func(p *server.Player, q RankQuery, now int64) error {
		return s.personalRanks(p, q, true, now)
	}))
	r.Handle("genRankPercent", network(s, DecodeRankPercent, func(p *server.Player, r RankPercent, _ int64) error {
		percent := int64(100)
		if r.Maximum > 0 {
			percent = min(percent, int64(r.Rank)*100/int64(r.Maximum))
		}
		return p.Send("onGenRankPercent", r.Rank, r.Maximum, percent)
	}))
	r.Handle("joinWorldChannel", network(s, DecodeChannel, func(p *server.Player, name string, _ int64) error {
		channel, err := s.Store.Social.JoinLocalWorld(p.Account.UID, name)
		if err != nil {
			return err
		}
		return sendWorld(p, "onJoinWorldChannel", channel)
	}))
}

func notifyTargets(p *server.Player) error {
	for slot := 1; slot <= 3; slot++ {
		if err := p.Send("onClientRefreshTargets", slot); err != nil {
			return err
		}
	}
	return nil
}

func Bind[T any](srv *server.Service, decode func(protocol.Call) (T, error), handle func(p *server.Player, request T, now int64) error) rpc.Endpoint[*server.Player] {
	return network(service{srv}, decode, handle)
}

func Profiles(srv *server.Service, ids []model.UserID) (map[model.UserID]account.PublicProfile, error) {
	return service{srv}.networkProfiles(ids)
}

func PrepareBattle(srv *server.Service, p *server.Player, r battle.CreateBattle) (bool, error) {
	return service{srv}.prepareNetworkBattle(p, r)
}

func FinishBattle(srv *server.Service, p *server.Player, r battle.VerifyBattle) (bool, error) {
	return service{srv}.finishNetworkBattle(p, r)
}
