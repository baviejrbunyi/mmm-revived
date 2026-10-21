package season

import (
	"database/sql"
	"errors"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

const errorCodeRejected = 5

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("setScience", rpc.Bind(DecodeEventID, func(p *server.Player, id int) error {
		return s.event(p, func(_ *sql.Tx, rules *account.Rules, _ int64) (map[model.ItemID]int64, error) {
			return nil, p.Account.SetScience(rules, id)
		}, rewardsOrAcknowledge(p))
	}))
	r.Handle("resetScience", rpc.Bare(func(p *server.Player) error {
		return s.event(p, func(*sql.Tx, *account.Rules, int64) (map[model.ItemID]int64, error) {
			return nil, p.Account.ResetScience()
		}, rewardsOrAcknowledge(p))
	}))
	r.Handle("quickBossBattle", rpc.Bind(DecodeEventCount, func(p *server.Player, r EventCount) error {
		err := s.event(p, func(_ *sql.Tx, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return nil, p.Account.SweepBoss(rules, r.ID, r.Count, now)
		}, func([]model.ItemReward) error { return p.Send("onQuickBossBattle", 0) })
		if errors.Is(err, account.ErrProgression) {
			p.Session.Logger.Warn("boss sweep rejected", "mission", r.ID, "count", r.Count, "error", err)
			return p.Send("onQuickBossBattle", errorCodeRejected)
		}
		return err
	}))
	r.Handle("settleLastBoss", rpc.Bind(DecodeDecision, func(p *server.Player, accept bool) error {
		if !accept {
			return s.event(p, func(*sql.Tx, *account.Rules, int64) (map[model.ItemID]int64, error) {
				p.Account.DeclineBoss()
				return nil, nil
			}, func([]model.ItemReward) error { return server.Acknowledge(p) })
		}
		err := s.event(p, func(tx *sql.Tx, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			best := p.Account.BossBest()
			quantities, err := p.Account.SettleBoss(rules)
			if err != nil || p.Account.BossBest() <= best {
				return quantities, err
			}
			return quantities, s.recordBossRank(tx, p, now)
		}, func(rewards []model.ItemReward) error { return p.Send("onSettleLastBoss", rewards) })
		if errors.Is(err, account.ErrProgression) {
			p.Session.Logger.Warn("boss settlement rejected", "error", err)
			return p.Send("onSettleLastBoss", []model.ItemReward{})
		}
		return err
	}))
	r.Handle("getBossScoreRwd", rpc.Bare(func(p *server.Player) error {
		return s.event(p, func(_ *sql.Tx, rules *account.Rules, _ int64) (map[model.ItemID]int64, error) {
			return p.Account.ClaimBossScore(rules)
		}, rewardsOrAcknowledge(p))
	}))
	r.Handle("getBossRankReward", rpc.Bare(func(p *server.Player) error {
		if err := s.eventsOpen(p); err != nil {
			return err
		}
		position, total, err := s.Store.RankPosition(p.Account.UID, model.RankBoss, p.Account.Events.Boss.Open)
		if err != nil {
			return err
		}
		return s.event(p, func(_ *sql.Tx, rules *account.Rules, now int64) (map[model.ItemID]int64, error) {
			return p.Account.ClaimBossRank(rules, position, total, now)
		}, rewardsOrAcknowledge(p))
	}))
}
