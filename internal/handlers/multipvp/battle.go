package multipvp

import (
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/handlers/battle"
	"mmm-revived/internal/handlers/network"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

func (s service) prepareBattle(p *server.Player, request battle.CreateBattle) (bool, error) {
	r := s.Store.Rules
	if r == nil {
		return false, nil
	}
	recognized := false
	for _, mine := range r.RetailPVP.Mines {
		recognized = recognized || slices.Contains(mine.Battles, request.Battle)
	}
	_, activityKnown := r.RetailPVP.Activities[request.Extra.Activity]
	if !recognized && !activityKnown {
		return false, nil
	}
	now := time.Now().Unix()
	if request.Extra.IsRevenge < 0 || request.Extra.IsRevenge > 1 {
		return true, fmt.Errorf("%w: invalid Halloween revenge flag", account.ErrProgression)
	}
	revenge := 0
	if request.Extra.IsRevenge == 1 {
		if request.Extra.RevengeIndex < 0 {
			return true, fmt.Errorf("%w: Halloween revenge record is missing", account.ErrProgression)
		}
		revenge = request.Extra.RevengeIndex + 1
	}
	target, err := p.Account.RetailPVPBattleTarget(r, request.Extra.Activity, request.Extra.EnemyIndex, revenge, request.Extra.RevengeMineIndex, now)
	if err != nil {
		return true, err
	}
	if request.Extra.Target == nil || request.Extra.Target.UID != target.UID {
		return true, fmt.Errorf("%w: Halloween target identity differs from authorized snapshot", account.ErrProgression)
	}
	var defender model.AsyncDefense
	card := model.RetailPVPCard{UID: target.UID, Name: "Local Challenger", Level: target.Level, Head: 5001, Frame: 6001, Host: 1, Server: 1}
	if request.Extra.RetailTarget != nil && request.Extra.RetailTarget.UID == target.UID {
		card = *request.Extra.RetailTarget
		card.Level = target.Level
		card.Host, card.Server = 1, 1
	}
	if target.UID < 0 {
		row, exists := r.RetailPVP.AI[int(-target.UID)]
		if !exists || row.Mine != target.Mine {
			return true, fmt.Errorf("%w: Halloween AI snapshot is unavailable", account.ErrProgression)
		}
		_, defender, err = s.botTarget(int(-target.UID), target.Level)
	} else {
		profiles, loadErr := network.Profiles(s.Service, []model.UserID{model.UserID(target.UID)})
		if loadErr != nil {
			return true, loadErr
		}
		peer, exists := profiles[model.UserID(target.UID)]
		if !exists {
			return true, fmt.Errorf("%w: Halloween opponent is unavailable", account.ErrProgression)
		}
		a := peer.Account
		var okay bool
		target, okay, err = s.mineTarget(&a, request.Extra.Activity, target.MineIndex)
		if !okay {
			return true, fmt.Errorf("%w: Halloween opponent mine has no defense", account.ErrProgression)
		}
		defender, okay = a.RetailPVPMineDefense(request.Extra.Activity, target.MineIndex, r)
		if !okay {
			return true, fmt.Errorf("%w: Halloween opponent defense is unavailable", account.ErrProgression)
		}
		card = model.RetailPVPCard{UID: int64(a.UID), Name: a.TeamName, Level: a.Level, Head: a.Profile.Head, Frame: a.Profile.Frame, Host: 1, Server: 1}
	}
	if err != nil {
		return true, err
	}
	defender.HeroInfo.TeamLevel = target.Level
	attacker := p.Account.NetworkDefense(r)
	attacker.HeroInfo.TeamLevel = p.Account.Level
	attacker.UID = int64(p.Account.UID)
	attacker.WinFix, attacker.LoseFix = 1, 1
	token := server.RandomToken()
	if err = s.Store.Update(p.Account, func(*sql.Tx) error {
		if err := p.Account.BeginRetailPVPBattle(r, request.Extra.Activity, request.Battle, target, request.Extra.EnemyIndex, revenge, token, now); err != nil {
			return err
		}
		pending := p.Account.Journey.Activities[request.Extra.Activity].RetailPVP.Pending
		pending.Defender, pending.Card = defender, card
		return nil
	}); err != nil {
		return true, err
	}
	p.RogueBattle, p.BossMission, p.Limited = false, 0, nil
	p.Challenge, p.Spiral, p.CampNode = 0, 0, 0
	p.ActivityBattle, p.Brawl, p.CampActivity = 0, 0, 0
	p.ActivityMission, p.ActivityNode = 0, 0
	p.Deployment, p.SelectedHeroes, p.Chip = nil, nil, nil
	p.Nonce, p.BattleID, p.State, p.Mission = token, request.Battle, server.Deployment, 0
	p.LastBattleWon = false
	extra := request.Extra
	extra.Target = nil
	extra.RetailTarget = &card
	artifacts, highlights := battle.GearInfo(p.Account)
	return true, p.Send("createBattle", request.Battle, p.Account.Events.Season.ID, 28, p.Account.Heroes, artifacts, highlights, model.Empty{}, model.Empty{}, model.AsyncBattleInfo{TeamLevel: p.Account.Level, SkinCollection: attacker.HeroInfo.SkinCollection, Attacker: attacker, Defender: defender, Bonds: attacker.HeroInfo.Bonds}, extra, token)
}

func (s service) lockDeployment(p *server.Player, heroes []model.HeroID) error {
	id, pending := p.Account.PendingRetailPVP(p.Nonce)
	if pending == nil {
		return nil
	}
	return s.Store.Update(p.Account, func(*sql.Tx) error {
		return p.Account.LockRetailPVPBattle(s.Store.Rules, id, p.Nonce, heroes, time.Now().Unix())
	})
}

func (s service) finishBattle(p *server.Player, request battle.VerifyBattle) (bool, error) {
	id, pending := p.Account.PendingRetailPVP(p.Nonce)
	if pending == nil {
		return false, nil
	}
	if pending.Battle != request.Battle || !pending.Locked || request.Result.UID != p.Account.UID || request.Result.Duration < 0 || request.Result.Duration > 86400 {
		return true, fmt.Errorf("%w: Halloween result does not match battle", account.ErrProgression)
	}
	replay, err := battle.DecodeRecord(request.Record)
	if err != nil {
		return true, err
	}
	battleID, okay := battle.Integer(battle.Dictionary(replay["initData"])["battleId"])
	if !okay || battleID != int64(request.Battle) {
		return true, fmt.Errorf("%w: Halloween replay belongs to another battle", account.ErrProgression)
	}
	frame, frameOkay := battle.Integer(replay["frame"])
	win, winOkay := replay["playerWin"].(bool)
	if !frameOkay || frame < 1 || frame > 10000000 || !winOkay || win != request.Win {
		return true, fmt.Errorf("%w: Halloween replay outcome is invalid", account.ErrProgression)
	}
	recorded, err := battle.RecordDeployment(replay["initActorInfos"])
	if err != nil {
		return true, err
	}
	if !battle.MatchesDeployment(recorded, p.Deployment) {
		return true, fmt.Errorf("%w: Halloween replay deployment differs from authorized heroes", account.ErrProgression)
	}
	packet := request.Packet
	packetID, packetOkay := battle.Integer(packet["battleId"])
	if !packetOkay || packetID != int64(request.Battle) {
		return true, fmt.Errorf("%w: Halloween result packet belongs to another battle", account.ErrProgression)
	}
	pvp := battle.Dictionary(battle.Dictionary(packet["extra"])["pvp"])
	player, opponent := battle.Dictionary(pvp["player"]), battle.Dictionary(pvp["opponent"])
	playerUID, playerOkay := battle.Integer(player["uid"])
	opponentUID, opponentOkay := battle.Integer(opponent["uid"])
	if !playerOkay || !opponentOkay || playerUID != int64(p.Account.UID) || opponentUID != pending.Target.UID {
		return true, fmt.Errorf("%w: Halloween result participants differ from battle", account.ErrProgression)
	}
	now := time.Now().Unix()
	nativeReplay, err := s.replay(p, id, pending, request)
	if err != nil {
		return true, err
	}
	packet["is_attack"], packet["timestamp"], packet["battle_replay_data"], packet["version"], packet["give_up_uid"] = p.Account.UID, now, nativeReplay, request.Version, int64(0)
	packet["verifyResult"] = true
	history, err := server.EncodePayload(packet)
	if err != nil {
		return true, err
	}
	participants := []*account.Account{p.Account}
	var peer *account.Account
	if pending.Target.UID > 0 {
		uid := model.UserID(pending.Target.UID)
		if live := s.Online(uid); live != nil {
			peer = live.Account
		} else {
			peer, err = s.Store.AccountByUID(uid)
		}
		if err != nil {
			return true, err
		}
		participants = append(participants, peer)
	}
	err = s.Store.UpdatePlayers(participants, func(tx *sql.Tx) error {
		currentID, current := p.Account.PendingRetailPVP(request.Nonce)
		if current == nil || currentID != id {
			return fmt.Errorf("%w: Halloween result has already settled", account.ErrProgression)
		}
		if peer != nil {
			peer.InitializeRetailPVP(s.Store.Rules, now)
			if err := peer.RefreshRetailPVPBox(s.Store.Rules, id, now); err != nil {
				return err
			}
		}
		record, err := p.Account.FinishRetailPVPBattle(s.Store.Rules, id, request.Nonce, request.Win, history, now)
		if err != nil {
			return err
		}
		if peer != nil {
			if err = peer.RecordRetailPVPDefense(s.Store.Rules, id, pending.Target.MineIndex, p.Account.UID, request.Win, record, now); err != nil {
				return err
			}
		}
		p.Account.RecordJourneyBattleTasks(s.Store.Rules, 28, request.Win, request.Result.KillCount, 0, false)
		p.Account.RecordJourneyBattleResult(s.Store.Rules, request.Battle, request.Win, packet)
		return s.Store.RecordRank(tx, p.Account.UID, model.RankRetailPVP, int(id), p.Account.Journey.Activities[id].RetailPVP.Point, now)
	})
	if err != nil {
		return true, err
	}
	p.State, p.Nonce, p.LastBattleWon = server.Lobby, "", request.Win
	if err = s.Sync(p, "activityComp", "itemComp", "heroComp", "taskComp", "historyComp"); err != nil {
		return true, err
	}
	if peer != nil {
		if live := s.Online(peer.UID); live != nil {
			if err := s.Sync(live, "activityComp"); err != nil {
				live.Session.Logger.Error("sync opponent activity", "error", err)
			}
		}
	}
	if err = p.Send("confirmBattleVerify"); err != nil {
		return true, err
	}
	return true, p.Send("onBattleEnd", request.Battle, packet)
}

func (s service) replay(p *server.Player, id model.ActivityID, pending *model.RetailPVPBattle, r battle.VerifyBattle) (string, error) {
	extra := battle.BattleExtra{Activity: id, EnemyIndex: pending.EnemyIndex, RetailTarget: &pending.Card}
	if pending.RevengeIndex > 0 {
		extra.IsRevenge = 1
		extra.RevengeIndex = pending.RevengeIndex - 1
		extra.RevengeMineIndex = pending.Target.MineIndex
	}
	info := model.AsyncBattleInfo{TeamLevel: p.Account.Level, SkinCollection: p.Account.SkinCollectionLevel(s.Store.Rules), Attacker: p.Account.NetworkDefense(s.Store.Rules), Defender: pending.Defender, Bonds: p.Account.BondAttributes(s.Store.Rules)}
	values := []any{p.Account.Heroes, p.Account.Artifacts, p.Account.Highlights, model.Empty{}, model.Empty{}, info, map[model.BattleID]battle.BattleExtra{r.Battle: extra}}
	fields := []string{"1", pending.Token, strconv.FormatUint(uint64(p.Account.UID), 10), strconv.Itoa(int(r.Battle)), "0", "28", strconv.Itoa(r.Version), r.Record}
	for _, value := range values {
		encoded, err := server.EncodePayload(value)
		if err != nil {
			return "", err
		}
		fields = append(fields, encoded)
	}
	empty, err := server.EncodePayload(model.Empty{})
	if err != nil {
		return "", err
	}
	return strings.Join(append(fields, "0", empty), "||"), nil
}
