package guildwar

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"mmm-revived/internal/account"
	"mmm-revived/internal/database"
	"mmm-revived/internal/handlers/battle"
	"mmm-revived/internal/handlers/network"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/server"
	"mmm-revived/internal/social"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("setGuildWarAreaIndex", network.Bind(s.Service, DecodeWarOrder, func(p *server.Player, order WarOrder, now int64) error {
		err := s.guildWar(p, now, func(tx *sql.Tx, w *warContext) error {
			if w.phase != social.WarPrepare {
				return fmt.Errorf("%w: guild war order is only editable during preparation", account.ErrProgression)
			}
			member, err := s.Store.Database.Queries.WithTx(tx).GetGuildMembership(context.Background(), p.Account.UID)
			if err != nil {
				return err
			}
			if member.Role != model.GuildBoss && member.Role != model.GuildManager {
				return fmt.Errorf("%w: guild leadership required", account.ErrProgression)
			}
			if len(order.Areas) != 3 || len(slices.Compact(slices.Sorted(slices.Values(order.Areas)))) != 3 || slices.Contains(order.Areas, order.Blocked) {
				return fmt.Errorf("%w: invalid guild war area order", account.ErrProgression)
			}
			areas := s.Store.Rules.Network.Wars[w.war].Areas
			for _, area := range append(slices.Clone(order.Areas), order.Blocked) {
				if !slices.Contains(areas, area) {
					return fmt.Errorf("%w: invalid guild war area", account.ErrProgression)
				}
			}
			for _, member := range w.state.Members {
				if member.Score > 0 {
					return fmt.Errorf("%w: guild war order is locked after combat", account.ErrProgression)
				}
			}
			w.state.Areas, w.state.Blocked = slices.Clone(order.Areas), order.Blocked
			return nil
		}, func(*warContext) error { return p.Send("onSetGuildWarAreaIndex", 0) })
		if errors.Is(err, account.ErrProgression) || errors.Is(err, social.ErrRejected) {
			p.Session.Logger.Info("guild war order rejected", "error", err)
			return p.Send("onSetGuildWarAreaIndex", 5)
		}
		return err
	}))
	r.Handle("reqGuildWarGeneral", network.Bind(s.Service, protocol.DecodeIgnore, func(p *server.Player, _ struct{}, now int64) error {
		return s.guildWar(p, now, nil, func(w *warContext) error {
			return p.Send("onGetGuildWarGeneralInfo", warGeneral(w.state, p.Account.UID))
		})
	}))
	r.Handle("reqGuildWarMembersInfo", network.Bind(s.Service, protocol.DecodeIgnore, func(p *server.Player, _ struct{}, now int64) error {
		return s.guildWar(p, now, nil, func(w *warContext) error {
			members, err := s.warMembers(w.guild, w.war, w.state.Members)
			if err != nil {
				return err
			}
			return p.Send("onGetGuildWarMembersInfo", members)
		})
	}))
	r.Handle("reqGuildWarLastMembersInfo", network.Bind(s.Service, protocol.DecodeIgnore, func(p *server.Player, _ struct{}, now int64) error {
		return s.guildWar(p, now, nil, func(w *warContext) error {
			last := map[model.UserID]warMember{}
			if w.war > 1 {
				data, err := s.Store.Database.Queries.GetGuildWar(context.Background(), database.GetGuildWarParams{GuildID: w.guild, War: int64(w.war - 1)})
				switch {
				case err == nil:
					var previous model.GuildWarState
					if err = json.Unmarshal(data, &previous); err != nil {
						return err
					}
					if last, err = s.warMembers(w.guild, w.war-1, previous.Members); err != nil {
						return err
					}
				case !errors.Is(err, sql.ErrNoRows):
					return err
				}
			}
			return p.Send("onGetGuildWarLastMembersInfo", last)
		})
	}))
	r.Handle("reqGuildWarAreaGroupInfo", s.warGroup(func(p *server.Player, level WarLevel, w *warContext) error {
		history, err := s.Store.GuildWarHistory(w.guild, w.war, level.Area, level.Level)
		if err != nil {
			return err
		}
		info := warGroupInfo{}
		if len(history) > 0 {
			best := historyWire(history[0])
			info.Best = &best
		}
		return p.Send("onGetGuildWarAreaGroupInfo", level.Level, info)
	}))
	r.Handle("reqGuildWarGroupHistory", s.warGroup(func(p *server.Player, level WarLevel, w *warContext) error {
		history, err := s.Store.GuildWarHistory(w.guild, w.war, level.Area, level.Level)
		if err != nil {
			return err
		}
		return p.Send("onGetGuildWarGroupHistory", level.Level, histories(history))
	}))
	r.Handle("reqGuildWarAssistHeroes", s.warGroup(s.warAssist))
	r.Handle("reqGuildWarChallengeStart", network.Bind(s.Service, DecodeWarLevel, func(p *server.Player, level WarLevel, now int64) error {
		var token string
		return s.guildWar(p, now, func(_ *sql.Tx, w *warContext) error {
			if p.State != server.Lobby && p.State != server.Deployment && p.State != server.Combat {
				return fmt.Errorf("%w: battle already active", account.ErrProgression)
			}
			if !social.WarBattling(w.phase) {
				return fmt.Errorf("%w: guild war battles are closed", account.ErrProgression)
			}
			allowed, err := s.Store.GuildWarLevelAllowed(p.Account, w.war, level.Area, level.Level, w.state)
			if err != nil {
				return err
			}
			token = server.RandomToken()
			member := w.state.Members[p.Account.UID]
			member.Tickets--
			w.state.Members[p.Account.UID] = member
			s.Store.AbandonAsyncBattle(p.Account)
			p.Account.Network.Pending = &model.NetworkBattle{Battle: allowed.Battle, Guild: w.guild, War: w.war, Area: level.Area, Level: level.Level, Token: token}
			return nil
		}, func(*warContext) error { return p.Send("onGuildWarChallengeStart", token) })
	}))
}

func FinishBattle(srv *server.Service, p *server.Player, r battle.VerifyBattle) (bool, error) {
	return service{srv}.finishGuildWarBattle(p, r)
}
