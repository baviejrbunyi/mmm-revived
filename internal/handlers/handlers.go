package handlers

import (
	"mmm-revived/internal/handlers/battle"
	"mmm-revived/internal/handlers/dispatch"
	"mmm-revived/internal/handlers/draw"
	"mmm-revived/internal/handlers/dreamland"
	"mmm-revived/internal/handlers/duel"
	"mmm-revived/internal/handlers/friend"
	"mmm-revived/internal/handlers/gm"
	"mmm-revived/internal/handlers/guild"
	"mmm-revived/internal/handlers/guildwar"
	"mmm-revived/internal/handlers/hero"
	"mmm-revived/internal/handlers/inventory"
	"mmm-revived/internal/handlers/invite"
	"mmm-revived/internal/handlers/journey"
	"mmm-revived/internal/handlers/mail"
	"mmm-revived/internal/handlers/multipvp"
	"mmm-revived/internal/handlers/network"
	"mmm-revived/internal/handlers/profile"
	"mmm-revived/internal/handlers/realtimepvp"
	"mmm-revived/internal/handlers/rogue"
	"mmm-revived/internal/handlers/season"
	"mmm-revived/internal/handlers/session"
	"mmm-revived/internal/handlers/system"
	"mmm-revived/internal/handlers/task"
	"mmm-revived/internal/server"
)

func Register(s *server.Service, r *server.Router) {
	session.Register(s, r)
	battle.Register(s, r, battle.Modes{
		Prepare: []battle.Prepare{multipvp.PrepareBattle, network.PrepareBattle},
		Lock:    []battle.Lock{multipvp.LockDeployment},
		Finish:  []battle.Finish{multipvp.FinishBattle, network.FinishBattle, guildwar.FinishBattle},
	})
	dreamland.Register(s, r)
	hero.Register(s, r)
	task.Register(s, r)
	inventory.Register(s, r)
	draw.Register(s, r)
	dispatch.Register(s, r)
	mail.Register(s, r)
	season.Register(s, r)
	gm.Register(s, r)
	profile.Register(s, r)
	system.Register(s, r)
	journey.Register(s, r)
	invite.Register(s, r)
	rogue.Register(s, r)
	friend.Register(s, r)
	guild.Register(s, r)
	guildwar.Register(s, r)
	network.Register(s, r)
	realtimepvp.Register(s, r)
	duel.Register(s, r)
	multipvp.Register(s, r)
}
