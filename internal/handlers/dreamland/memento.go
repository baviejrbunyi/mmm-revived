package dreamland

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/database"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

type community struct {
	scope   model.MessageScope
	mission model.MissionID
	upload  bool
	text    string
}

func (s service) visitMemento(p *server.Player, r MementoVisit) error {
	err := fmt.Errorf("%w: battle is active", account.ErrProgression)
	if s.Store.Rules == nil {
		err = server.ErrProgressionRules
	} else if p.State == server.Lobby {
		err = p.Account.VisitMemento(s.Store.Rules, r.Mission, r.Chip, r.Heroes)
	}
	if err != nil {
		if errors.Is(err, account.ErrProgression) || errors.Is(err, server.ErrProgressionRules) {
			return p.Send("onDlVisitChip", r.Chip, false)
		}
		return err
	}
	if err := s.SaveAndSync(p, "dreamlandComp", "itemComp"); err != nil {
		return err
	}
	return p.Send("onDlVisitChip", r.Chip, true)
}

func (s service) failMementoTeam(p *server.Player, id model.MissionID) error {
	rules := s.Store.Rules
	if rules == nil {
		return server.ErrProgressionRules
	}
	if !p.Account.MissionActive(rules, id) || !rules.Missions[id].Memento || p.State != server.Lobby {
		return fmt.Errorf("%w: Memento is not active", account.ErrProgression)
	}
	mission := p.Account.Missions[id]
	if rules.Nodes[mission.Node.ID].Type != 5 {
		return fmt.Errorf("%w: no Memento team to fail", account.ErrProgression)
	}
	heroes := maps.Clone(mission.Heroes)
	if heroes == nil {
		heroes = map[model.HeroID]int{}
	}
	for _, hero := range p.SelectedHeroes {
		heroes[hero] = 1
	}
	if err := s.Store.Update(p.Account, func(*sql.Tx) error {
		mission.Heroes = heroes
		p.Account.Missions[id] = mission
		return nil
	}); err != nil {
		return err
	}
	return s.Sync(p, "dreamlandComp")
}

func (s service) communityMessage(p *server.Player, r community) error {
	ctx := context.Background()
	subject := int64(p.BattleID)
	if r.scope == model.MementoMessages {
		if s.Store.Rules == nil {
			return server.ErrProgressionRules
		}
		if _, ok := p.Account.Missions[r.mission]; !ok || !s.Store.Rules.Missions[r.mission].Memento {
			return fmt.Errorf("%w: unknown Memento", account.ErrProgression)
		}
		subject = int64(r.mission)
	} else if p.BattleID == 0 || p.State != server.Lobby || r.upload && !p.LastBattleWon {
		return fmt.Errorf("%w: no completed battle", account.ErrProgression)
	}
	if r.upload {
		err := s.Store.Update(p.Account, func(tx *sql.Tx) error {
			q := s.Store.Database.Queries.WithTx(tx)
			if err := q.SaveCommunityMessage(ctx, database.SaveCommunityMessageParams{Scope: r.scope, Subject: subject, UID: p.Account.UID, Content: r.text, Updated: time.Now().Unix()}); err != nil {
				return err
			}
			if r.scope == model.MementoMessages {
				mission := p.Account.Missions[r.mission]
				mission.BoxMessageUID, mission.BoxMessage = p.Account.UID, r.text
				if r.text == "" {
					mission.BoxMessageUID = 0
				}
				p.Account.Missions[r.mission] = mission
			}
			return nil
		})
		if err != nil {
			return err
		}
		if r.scope == model.MementoMessages {
			if err = s.Sync(p, "dreamlandComp"); err != nil {
				return err
			}
			return p.Send("onUploadBoxMessage", r.mission)
		}
		return p.Send("onUploadPassMessage")
	}
	rows, err := s.Store.Database.Queries.ListCommunityMessages(ctx, database.ListCommunityMessagesParams{Scope: r.scope, Subject: subject, MaxRows: 20})
	if err != nil {
		return err
	}
	rows = slices.DeleteFunc(rows, func(row database.ListCommunityMessagesRow) bool { return row.Content == "" })
	if r.scope == model.MementoMessages {
		mission := p.Account.Missions[r.mission]
		mission.BoxMessageUID, mission.BoxMessage = 0, ""
		if len(rows) > 0 {
			mission.BoxMessageUID, mission.BoxMessage = rows[0].UID, rows[0].Content
		}
		p.Account.Missions[r.mission] = mission
		if err = s.SaveAndSync(p, "dreamlandComp"); err != nil {
			return err
		}
		return p.Send("onDownloadBoxMessage", r.mission)
	}
	messages := map[model.UserID]string{}
	for _, row := range rows {
		messages[row.UID] = row.Content
	}
	return p.Update([]string{"battleNewComp", "passMsg"}, messages)
}
