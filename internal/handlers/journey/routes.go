package journey

import (
	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("startNoviceNode", rpc.Bind(DecodeJourneyID, func(p *server.Player, id int) error {
		return s.anywhere(p, func(a *account.Account, rules *account.Rules, _ int64) (map[model.ItemID]int64, error) {
			return nil, a.StartNoviceNode(rules, id)
		}, RewardsOrAcknowledge(p))
	}))
	r.Handle("endNoviceNode", rpc.Bind(DecodeJourneyID, func(p *server.Player, id int) error {
		return s.journey(p, func(a *account.Account, rules *account.Rules, _ int64) (map[model.ItemID]int64, error) {
			return nil, a.FinishNoviceNode(rules, id)
		}, RewardsOrAcknowledge(p))
	}))
	r.Handle("getNoviceReward", rpc.Bind(DecodeJourneyID, func(p *server.Player, id int) error {
		return s.journey(p, func(a *account.Account, rules *account.Rules, _ int64) (map[model.ItemID]int64, error) {
			return a.ClaimNoviceReward(rules, id)
		}, func(rewards []model.ItemReward) error {
			if len(rewards) == 0 {
				return server.Acknowledge(p)
			}
			return p.Send("showReward", []model.RewardID{}, rewards, 2)
		})
	}))
	r.Handle("buyPassExp", rpc.Bind(DecodeJourneyID, func(p *server.Player, level int) error {
		return s.journeyPass(p, "onBuyPassExp", func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return nil, a.BuyJourneyPassLevels(rules, level, now)
		})
	}))
	r.Handle("gainPassReward", rpc.Bind(DecodePassClaim, func(p *server.Player, claim ActivityLevels) error {
		return s.journeyPass(p, "onGainPassReward", func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return a.ClaimJourneyPass(rules, claim.Levels, claim.Track, now)
		})
	}))
	r.Handle("gainLtePassReward", rpc.Bind(DecodeLimitedPassClaim, func(p *server.Player, claim ActivityLevels) error {
		return s.journeyPass(p, "onLteGainPassReward", func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return a.ClaimLimitedJourneyPass(rules, claim.Activity, claim.Levels, claim.Track, now)
		})
	}))
	r.Handle("checkMonthlyPassDailyReward", rpc.Bind(protocol.DecodeNone, func(p *server.Player, _ struct{}) error {
		return s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return a.ClaimMonthlyPass(rules, true, now)
		}, func(rewards []model.ItemReward) error {
			return s.replyThenRewards(p, rewards, "onResponseCheckMonthlyPassDailyReward")
		})
	}))
	r.Handle("checkMonthlyPassReward", rpc.Bind(protocol.DecodeNone, func(p *server.Player, _ struct{}) error {
		return s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return a.ClaimMonthlyPass(rules, false, now)
		}, RewardsOrAcknowledge(p))
	}))
	r.Handle("getGeneralSignRwd", rpc.Bind(DecodeActivityLevels, func(p *server.Player, claim ActivityLevels) error {
		return s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return a.ClaimJourneySign(rules, claim.Activity, claim.Levels, now)
		}, func(rewards []model.ItemReward) error {
			return s.replyThenRewards(p, rewards, "onGetGeneralSignRwd", rewards)
		})
	}))
	r.Handle("gainStarReward", s.activityLevels(func(a *account.Account, rules *account.Rules, claim ActivityLevels, now int64) (map[model.ItemID]int64, error) {
		return a.ClaimJourneyStars(rules, claim.Activity, claim.Levels, now)
	}))
	r.Handle("gainGameTrailerReward", s.activityLevels(func(a *account.Account, rules *account.Rules, claim ActivityLevels, now int64) (map[model.ItemID]int64, error) {
		return a.ClaimJourneyTrailers(rules, claim.Activity, claim.Levels, now)
	}))
	r.Handle("getRewardInInfiniteMayhem", rpc.Bind(DecodeActivityValue, func(p *server.Player, v ActivityNode) error {
		return s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return a.ClaimJourneyBrawl(rules, v.Activity, v.ID, now)
		}, RewardsOrAcknowledge(p))
	}))
	r.Handle("getRewardListInInfiniteMayhem", rpc.Bind(DecodeBrawlList, func(p *server.Player, claim ActivityLevels) error {
		return s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return a.ClaimJourneyBrawlList(rules, claim.Activity, claim.Levels, now)
		}, RewardsOrAcknowledge(p))
	}))
	r.Handle("getDreamguardTask", s.activity(func(a *account.Account, rules *account.Rules, activity model.ActivityID, now int64) (map[model.ItemID]int64, error) {
		return nil, a.StartDreamguard(rules, activity, now)
	}))
	r.Handle("getDreamguardTaskReward", s.activity(func(a *account.Account, rules *account.Rules, activity model.ActivityID, now int64) (map[model.ItemID]int64, error) {
		return a.ClaimDreamguard(rules, activity, now)
	}))
	r.Handle("makeDreamguardTaskForever", s.activity(func(a *account.Account, rules *account.Rules, activity model.ActivityID, now int64) (map[model.ItemID]int64, error) {
		return nil, a.MakeDreamguardPermanent(rules, activity, now)
	}))
	r.Handle("lteMissionNodeStart", rpc.Bind(DecodeActivityNode, func(p *server.Player, v ActivityNode) error {
		err := s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return nil, a.PrepareJourneyMission(rules, v.Activity, v.ID, now)
		}, func([]model.ItemReward) error { return nil })
		if err != nil {
			return err
		}
		return s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return nil, a.StartJourneyMissionNode(rules, v.Activity, v.ID, v.Node, now)
		}, RewardsOrAcknowledge(p))
	}))
	r.Handle("lteMissionNodeEnd", rpc.Bind(DecodeActivityNode, func(p *server.Player, v ActivityNode) error {
		return s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return a.FinishJourneyMissionNode(rules, v.Activity, v.ID, v.Node, now)
		}, func(rewards []model.ItemReward) error {
			if len(rewards) == 0 {
				return server.Acknowledge(p)
			}
			return p.Send("showReward", []model.RewardID{}, rewards, 15)
		})
	}))
	r.Handle("finishGridMovementLevel", rpc.Bind(DecodeActivityValue, func(p *server.Player, v ActivityNode) error {
		err := s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return a.FinishJourneyMiniGame(rules, v.Activity, v.ID, now)
		}, func(rewards []model.ItemReward) error { return p.Send("onGridMovementLevelFinish", v.ID, rewards) })
		return replyOnFailure(p, err, "onGridMovementLevelFinish", v.ID, []model.ItemReward{})
	}))
	r.Handle("sweepChallenge", rpc.Bind(DecodeJourneyPair, func(p *server.Player, v JourneyPair) error {
		err := s.journey(p, func(a *account.Account, rules *account.Rules, _ int64) (map[model.ItemID]int64, error) {
			return a.SweepJourneyChallenge(rules, v.ID, v.Node)
		}, func(rewards []model.ItemReward) error { return p.Send("onSweepChallenge", v.ID, v.Node, rewards) })
		return replyOnFailure(p, err, "onSweepChallenge", v.ID, v.Node, []model.ItemReward{})
	}))
	r.Handle("getSpiralStarReward", rpc.Bind(DecodeJourneyPair, func(p *server.Player, v JourneyPair) error {
		return s.journey(p, func(a *account.Account, rules *account.Rules, _ int64) (map[model.ItemID]int64, error) {
			return a.ClaimJourneySpiralStars(rules, v.ID, v.Node)
		}, RewardsOrAcknowledge(p))
	}))
	r.Handle("chooseBuffNode", rpc.Bind(DecodeActivityNode, func(p *server.Player, v ActivityNode) error {
		return s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return nil, a.ChooseJourneyBuff(rules, v.Activity, v.ID, v.Node, now)
		}, func(rewards []model.ItemReward) error {
			return s.replyThenRewards(p, rewards, "buffNodeUpdate", p.Account.Journey.Activities[v.Activity].BuffNodes)
		})
	}))
	r.Handle("resetHyperFlowBattleNode", rpc.Bind(DecodeActivityValue, func(p *server.Player, v ActivityNode) error {
		return s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return nil, a.ResetJourneyHyperBattle(rules, v.Activity, v.ID, now)
		}, func(rewards []model.ItemReward) error {
			return s.replyThenRewards(p, rewards, "battleNodeUpdate", true, p.Account.Journey.Activities[v.Activity].BattleNodes)
		})
	}))
	r.Handle("resetHyperFlowBuffNode", rpc.Bind(DecodeJourneyActivity, func(p *server.Player, activity model.ActivityID) error {
		return s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return nil, a.ResetJourneyBuffs(rules, activity, now)
		}, func(rewards []model.ItemReward) error {
			return s.replyThenRewards(p, rewards, "buffNodeUpdate", p.Account.Journey.Activities[activity].BuffNodes)
		})
	}))
	r.Handle("settleCampChallenge", rpc.Bind(DecodeJourneyActivity, func(p *server.Player, id model.ActivityID) error {
		var settled model.RetailCampPending
		err := s.journey(p, func(a *account.Account, rules *account.Rules, now int64) (items map[model.ItemID]int64, err error) {
			settled, items, err = a.SettleRetailCamp(rules, id, now)
			return
		}, func(rewards []model.ItemReward) error {
			return s.replyThenRewards(p, rewards, "onSettleCampChallenge", id, settled.Node, settled.Battle)
		})
		return replyOnFailure(p, err, "onSettleCampChallenge", id, 0, 0)
	}))
}

func replyOnFailure(p *server.Player, err error, method string, args ...any) error {
	if err == nil {
		return nil
	}
	if sent := p.Send(method, args...); sent != nil {
		return sent
	}
	return err
}
