package session

import (
	"maps"
	"slices"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

func offline(call protocol.Call) (rpc.Handler[*server.Player], error) {
	return func(p *server.Player) error {
		p.Session.Logger.Debug("offline RPC", "method", call.Name)
		return server.Acknowledge(p)
	}, nil
}

func loginUnavailable(p *server.Player, _ struct{}) error {
	if p.Account != nil {
		return server.Acknowledge(p)
	}
	return p.Send("loginReply", 1, model.Empty{})
}

func (s service) mostUsedHeroes(p *server.Player) [][2]int64 {
	heroes := slices.DeleteFunc(slices.Sorted(maps.Keys(p.Account.Heroes)), func(id model.HeroID) bool {
		_, known := s.Store.Catalog.Heroes[id]
		return !known || p.Account.Heroes[id].Trial != 0
	})
	slices.SortStableFunc(heroes, func(a, b model.HeroID) int { return p.Account.Heroes[b].Level - p.Account.Heroes[a].Level })
	result := make([][2]int64, 0, 3)
	for index, id := range heroes[:min(3, len(heroes))] {
		result = append(result, [2]int64{int64(100 - 20*index), int64(id)})
	}
	return result
}

func (s service) setGuide(p *server.Player, guide Guide) error {
	if value, exists := p.Account.Guides[guide.Key]; exists && value == guide.Value {
		return server.Acknowledge(p)
	}
	p.Account.Guides[guide.Key] = guide.Value
	if err := s.Store.Save(p.Account); err != nil {
		return err
	}
	return p.Update([]string{"lobbyComp", "guides"}, p.Account.Guides)
}
