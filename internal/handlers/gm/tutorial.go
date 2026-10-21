package gm

import (
	"database/sql"
	"errors"
	"time"

	"mmm-revived/internal/handlers/rogue"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

var tutorialCommands = [...]model.GMCommandKind{model.GMNoviceFinish, model.GMGuideFinish}

func (s service) skipTutorial(p *server.Player) error {
	if s.Store.Rules == nil {
		return errGMRules
	}
	if p.Room != nil {
		return errors.New("leave the multiplayer match before skipping the tutorial")
	}
	now := time.Now().Unix()
	if err := s.Store.Update(p.Account, func(*sql.Tx) error {
		for _, kind := range tutorialCommands {
			if _, err := p.Account.ExecuteGM(s.Store.Rules, s.Store.Catalog, model.GMCommand{Kind: kind}, now); err != nil {
				return err
			}
		}
		s.Store.AbandonAsyncBattle(p.Account)
		return nil
	}); err != nil {
		return err
	}
	s.Queue().Cancel(p.Account.UID)
	p.State, p.BattleID, p.Nonce, p.Mission = server.Lobby, 0, "", 0
	p.Deployment, p.Limited, p.RogueBattle = nil, nil, false
	if err := s.Sync(p, "lobbyComp", "noviceComp", "taskComp", "dreamlandComp"); err != nil {
		return err
	}
	return rogue.Sync(p)
}
