package gm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"mmm-revived/internal/database"
	"mmm-revived/internal/handlers/rogue"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

func (s service) gmFailure(p *server.Player, method string, failure error) error {
	if err := s.Store.Update(p.Account, func(tx *sql.Tx) error {
		p.Account.RecordGMError(method, failure.Error(), time.Now().Unix())
		return nil
	}); err != nil {
		return err
	}
	return p.Send("gmshow", 2, failure.Error())
}

var gmCommands = [][3]string{
	{"getall", "Unlock inventory, heroes and recipes", "[count]"},
	{"item", "Grant configured item", "id [count]"},
	{"setitem", "Set stack quantity", "id count"},
	{"level", "Set team level", "level"},
	{"herolevel", "Set hero level", "hero|0 level"},
	{"herostar", "Set hero stars", "hero stars"},
	{"hero", "Unlock hero", "id"},
	{"guidefin", "Complete system guides", ""},
	{"endnovice", "Complete tutorial", ""},
	{"setnovice", "Select tutorial task", "task|0"},
	{"alldl", "Complete dreamlands", ""},
	{"enddreamland", "Change dreamland progress", "story|memento_easy|all|clear [chapter]"},
	{"nextday", "Reset daily limits", ""},
	{"gettime", "Show content calendar", ""},
	{"settime", "Set content calendar in UTC", "[YYYY.MM.DD.hh.mm.ss]"},
	{"mail", "Deliver configured mail", "template"},
	{"energy", "Set energy", "amount"},
	{"resetboss", "Reset boss attempts", ""},
	{"resetscience", "Reset science selections", ""},
	{"seasonlevel", "Set science season level", "level"},
	{"gmrogue", "Initialize rogue run", ""},
	{"gmroguepve", "Initialize rogue PvE", "[difficulty]"},
	{"doGuildGm", "Reset guild war state or deliver chest", "reset_score|reset_guild_war_ticket|reset_cd|send_box_gm"},
	{"battlecancel", "Cancel local battle", ""},
	{"checkchapterend", "Count completed chapters", ""},
	{"save", "Save profile", ""},
	{"ship", "Run a purchase", "product [count kind ref]"},
}

var errGMRules = errors.New("GM rules unavailable")

func (s service) executeGM(p *server.Player, command model.GMCommand) error {
	now := time.Now().Unix()
	switch command.Kind {
	case model.GMResetBattle, model.GMGetTime, model.GMList, model.GMSave:
	default:
		if p.State != server.Lobby {
			return s.gmFailure(p, "gmcmd", errors.New("finish the active battle first"))
		}
	}
	if command.Kind == model.GMResetBattle && p.Room != nil {
		return s.gmFailure(p, "gmcmd", errors.New("surrender the multiplayer match first"))
	}
	before := *p.Account
	var result model.GMCommandResult
	err := s.Store.Update(p.Account, func(tx *sql.Tx) error {
		var err error
		if result, err = p.Account.ExecuteGM(s.Store.Rules, s.Store.Catalog, command, now); err != nil {
			return err
		}
		if result.Mail != 0 {
			if _, err = s.Store.DeliverGMMail(tx, p.Account.UID, result.Mail, now); err != nil {
				return err
			}
		}
		if result.Clock != nil {
			if err = s.Store.Database.Queries.WithTx(tx).SaveClockOffset(context.Background(), *result.Clock); err != nil {
				return err
			}
		}
		if command.Kind == model.GMGuild {
			return s.gmGuild(p, tx, command.Text, now)
		}
		if result.ResetBattle {
			s.Store.AbandonAsyncBattle(p.Account)
			p.Account.Network.Pending = nil
			p.Account.Events.Boss.Pending = nil
		}
		return nil
	})
	if err != nil {
		return s.gmFailure(p, "gmcmd", err)
	}
	if result.Clock != nil {
		if _, err = s.Store.ConfigureCalendar(result.Clock); err != nil {
			return err
		}
		if err = server.SyncTime(p); err != nil {
			return err
		}
	}
	if result.ResetBattle {
		s.Queue().Cancel(p.Account.UID)
		p.State, p.BattleID, p.Nonce, p.Mission = server.Lobby, 0, "", 0
		p.Deployment, p.Limited, p.RogueBattle = nil, nil, false
	}
	if command.Kind == model.GMList {
		lines := make([]string, 0, len(gmCommands))
		for _, command := range gmCommands {
			lines = append(lines, strings.TrimSpace(command[0]+" "+command[2])+" - "+command[1])
		}
		return p.Send("gmshow", 1, strings.Join(lines, "\n"))
	}
	if err = s.Sync(p, "basicComp", "lobbyComp", "heroComp", "itemComp", "taskComp", "dreamlandComp", "noviceComp", "seasonComp", "bossComp", "activityComp", "dispatchComp", "shopComp"); err != nil {
		return err
	}
	if err = rogue.Sync(p); err != nil {
		return err
	}
	if result.Mail != 0 {
		mails, err := s.Store.Mailbox(p.Account.UID)
		if err != nil {
			return err
		}
		if err = p.Send("upMails", mails, 1); err != nil {
			return err
		}
	}
	if len(result.Items) > 0 {
		if err = p.ShowRewards(s.ItemRewards(before, p.Account, result.Items)); err != nil {
			return err
		}
	}
	return p.Send("gmshow", 1, result.Text)
}

func (s service) gmDiagnostic(p *server.Player, run func(*model.GMState, int64) (string, error)) error {
	now := time.Now().Unix()
	var text string
	err := s.Store.Update(p.Account, func(*sql.Tx) error {
		p.Account.InitializeGM()
		var err error
		text, err = run(&p.Account.GM, now)
		return err
	})
	if err != nil {
		return err
	}
	return p.Send("gmshow", 1, text)
}

func (s service) gmGuild(p *server.Player, tx *sql.Tx, action string, now int64) error {
	q, ctx := s.Store.Database.Queries.WithTx(tx), context.Background()
	if action == "reset_cd" {
		return q.SetGuildLeft(ctx, database.SetGuildLeftParams{UID: p.Account.UID, Time: 0})
	}
	war := s.Store.Rules.Network.DefaultWar()
	guild, state, err := s.Store.LoadGuildWar(tx, p.Account.UID, war, now)
	if err != nil {
		return err
	}
	switch action {
	case "reset_score":
		for id, value := range state.Members {
			value.Score = 0
			state.Members[id] = value
		}
	case "reset_guild_war_ticket":
		for id, value := range state.Members {
			value.Tickets = 3
			value.Day = now / 86400
			state.Members[id] = value
		}
	case "send_box_gm":
		return s.Store.DeliverGuildWarClear(tx, guild, s.Store.Rules.Network.Wars[war].BossReward, now)
	default:
		return fmt.Errorf("unknown guild GM action: %s", action)
	}
	return s.Store.SaveGuildWar(tx, guild, war, state)
}
