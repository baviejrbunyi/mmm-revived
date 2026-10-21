package network

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/database"
	"mmm-revived/internal/handlers/battle"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
	"mmm-revived/internal/wire"
)

func (s service) finishAsyncBattle(p *server.Player, r battle.VerifyBattle) error {
	pending := p.Account.Network.Pending
	if pending == nil || pending.Battle != r.Battle || pending.Token != r.Nonce || pending.Started == 0 || pending.Guild != 0 || r.Result.UID != p.Account.UID {
		return fmt.Errorf("%w: asynchronous result does not match an active battle", account.ErrProgression)
	}
	initial, err := battle.DecodeRecord(pending.InitialRecord)
	if err != nil {
		return err
	}
	completed, err := battle.DecodeRecord(r.Record)
	if err != nil {
		return err
	}
	frame, frameOK := battle.Integer(completed["frame"])
	battleID, battleOK := battle.Integer(battle.Dictionary(completed["initData"])["battleId"])
	win, winOK := completed["playerWin"].(bool)
	if !frameOK || frame < 1 || frame > 10000000 || !battleOK || battleID != int64(r.Battle) || !winOK || win != r.Win {
		return fmt.Errorf("%w: asynchronous record outcome mismatch", account.ErrProgression)
	}
	for _, key := range []string{"randomSeed", "heroConfirm"} {
		if initial[key] == nil || !reflect.DeepEqual(initial[key], completed[key]) {
			return fmt.Errorf("%w: asynchronous record changed %s", account.ErrProgression, key)
		}
	}
	initialDeployment, err := battle.RecordDeployment(initial["initActorInfos"])
	if err != nil {
		return err
	}
	completedDeployment, err := battle.RecordDeployment(completed["initActorInfos"])
	if err != nil || !reflect.DeepEqual(initialDeployment, completedDeployment) {
		return fmt.Errorf("%w: asynchronous record changed deployment", account.ErrProgression)
	}
	packet := r.Packet
	if packet == nil {
		return fmt.Errorf("%w: asynchronous result packet is missing", account.ErrProgression)
	}
	pvp := battle.Dictionary(battle.Dictionary(packet["extra"])["pvp"])
	score, okay := battle.Integer(battle.Dictionary(battle.Dictionary(pvp["detail"])["finalScore"])["score"])
	if !okay || score < -10000 || score > 10000 {
		return fmt.Errorf("%w: asynchronous score is invalid", account.ErrProgression)
	}
	packet["gid"], packet["battleId"], packet["verifyResult"] = pending.Token, int64(r.Battle), true
	encoded, err := server.EncodePayload(packet)
	if err != nil {
		return err
	}
	replay, err := s.asyncReplay(p, r)
	if err != nil {
		return err
	}
	input := AsyncBattle{Battle: r.Battle, Player: p.Account.UID, Opponent: pending.Opponent, Token: pending.Token}
	result := AsyncOutcome{Result: r.Result, ScoreDelta: int(score), Packet: encoded, Replay: replay}
	if err = s.settleAsyncBattle(p.Account, input, result, time.Now().Unix()); err != nil {
		return fmt.Errorf("%w: %w", account.ErrProgression, err)
	}
	p.State, p.Nonce, p.LastBattleWon = server.Lobby, "", r.Win
	if err = s.syncNetwork(p); err != nil {
		return err
	}
	if err = s.Sync(p, "taskComp", "activityComp", "historyComp"); err != nil {
		return err
	}
	if err = p.Send("confirmBattleVerify"); err != nil {
		return err
	}
	return p.Send("onBattleEnd", r.Battle, packet)
}

func (s service) asyncReplay(p *server.Player, r battle.VerifyBattle) (string, error) {
	pending := p.Account.Network.Pending
	extra := battle.BattleExtra{Activity: pending.Activity, Target: pending.Target, Mode: pending.Mode}
	if pending.Friend {
		extra.FriendUID = model.UserID(pending.Opponent)
	}
	info := model.AsyncBattleInfo{TeamLevel: p.Account.Level, Attacker: p.Account.NetworkDefense(s.Store.Rules), Defender: pending.Defender, Bonds: p.Account.BondAttributes(s.Store.Rules)}
	values := []any{p.Account.Heroes, p.Account.Artifacts, p.Account.Highlights, model.Empty{}, model.Empty{}, info, map[model.BattleID]battle.BattleExtra{r.Battle: extra}}
	fields := []string{"1", pending.Token, strconv.FormatInt(int64(p.Account.UID), 10), strconv.Itoa(int(r.Battle)), "0", "6", strconv.Itoa(r.Version), r.Record}
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

func (s service) startAsyncBattle(p *server.Player, r AsyncStart, now int64) error {
	pending := p.Account.Network.Pending
	if p.State != server.Combat || pending == nil || pending.Guild != 0 || pending.Battle != r.Battle || pending.Token != p.Nonce || r.Result.UID != p.Account.UID || len(p.Deployment) == 0 {
		return fmt.Errorf("%w: asynchronous battle start does not match deployment", account.ErrProgression)
	}
	if pending.Started != 0 {
		if pending.InitialRecord == r.Record {
			return nil
		}
		return fmt.Errorf("%w: asynchronous battle already started", account.ErrProgression)
	}
	record, err := battle.DecodeRecord(r.Record)
	if err != nil {
		return err
	}
	battleID, okay := battle.Integer(battle.Dictionary(record["initData"])["battleId"])
	frame, frameOkay := battle.Integer(record["frame"])
	if !okay || !frameOkay || battleID != int64(p.BattleID) || frame != 0 {
		return fmt.Errorf("%w: invalid asynchronous initial record", account.ErrProgression)
	}
	heroes, err := battle.RecordDeployment(record["initActorInfos"])
	if err != nil {
		return err
	}
	selected := make(map[model.HeroID]model.DeploymentHero, len(p.Deployment))
	for _, hero := range p.Deployment {
		selected[hero.Hero] = hero
	}
	if !reflect.DeepEqual(heroes, selected) {
		return fmt.Errorf("%w: asynchronous initial deployment differs from selected heroes", account.ErrProgression)
	}
	return s.Store.Update(p.Account, func(tx *sql.Tx) error {
		pending.Started, pending.InitialRecord = now, r.Record
		return nil
	})
}

func decodeBattlePacket(encoded string) (map[any]any, error) {
	if len(encoded) == 0 || len(encoded) > 8<<20 {
		return nil, errors.New("invalid native battle packet size")
	}
	compressed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	reader, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, 16<<20+1))
	if err != nil || len(data) > 16<<20 {
		return nil, errors.New("invalid native battle packet expansion")
	}
	value, err := wire.Unpack(data)
	if err != nil {
		return nil, err
	}
	result, ok := value.(map[any]any)
	if !ok {
		return nil, errors.New("native battle packet must be a dictionary")
	}
	return result, nil
}

func (s service) settleAsyncBattle(a *account.Account, input AsyncBattle, result AsyncOutcome, now int64) error {
	pending := a.Network.Pending
	if pending == nil || pending.Token != input.Token || pending.Started == 0 {
		return errors.New("battle settlement token mismatch")
	}
	packet, err := decodeBattlePacket(result.Packet)
	if err != nil {
		return err
	}
	uid, uidOK := battle.Integer(packet["uid"])
	battleID, battleOK := battle.Integer(packet["battleId"])
	win, winOK := packet["win"].(bool)
	if !uidOK || !battleOK || !winOK || uid != int64(input.Player) || battleID != int64(input.Battle) || packet["gid"] != input.Token || win != result.Result.Win || result.Result.UID != input.Player || len(result.Replay) == 0 || len(result.Replay) > 8<<20 {
		return errors.New("native battle result correlation mismatch")
	}
	pvp := battle.Dictionary(battle.Dictionary(packet["extra"])["pvp"])
	attacker, defender := battle.Dictionary(pvp["player"]), battle.Dictionary(pvp["opponent"])
	attackerUID, _ := battle.Integer(attacker["uid"])
	defenderUID, _ := battle.Integer(defender["uid"])
	delta, deltaOK := battle.Integer(battle.Dictionary(battle.Dictionary(pvp["detail"])["finalScore"])["score"])
	defenderDelta, defenderDeltaOK := battle.Integer(battle.Dictionary(battle.Dictionary(pvp["oppoDetail"])["finalScore"])["score"])
	if attackerUID != int64(a.UID) || defenderUID != pending.Opponent || !deltaOK || !defenderDeltaOK || delta != int64(result.ScoreDelta) || delta < -10000 || delta > 10000 || defenderDelta < -10000 || defenderDelta > 10000 {
		return errors.New("native battle scoring proof mismatch")
	}
	participants := []*account.Account{a}
	var opponent *account.Account
	if pending.Opponent > 0 && !pending.Friend {
		if peer := s.Online(model.UserID(pending.Opponent)); peer != nil {
			opponent = peer.Account
		} else {
			opponent, err = s.Store.AccountByUID(model.UserID(pending.Opponent))
			if err != nil {
				return err
			}
		}
		participants = append(participants, opponent)
	}
	oldPoints := a.Network.Points
	err = s.Store.UpdatePlayers(participants, func(tx *sql.Tx) error {
		packet["is_attack"], packet["is_friend_pvp"], packet["timestamp"], packet["battle_replay_data"] = a.UID, pending.Friend, now, result.Replay
		if pending.Target != nil {
			packet["target_data"] = *pending.Target
		}
		attacker["score"] = a.Network.Score
		defender["score"] = 1000
		if pending.Target != nil {
			defender["score"] = pending.Target.Score
		}
		if !pending.Friend {
			_, rank := s.Store.Rules.Network.Rank(a.Network.Score)
			points := rank.Lose
			if win {
				points = rank.Win
			}
			a.Network.Points += int64(points)
			a.Network.Score = max(1, a.Network.Score+int(delta))
			a.Network.MaxScore = max(a.Network.MaxScore, a.Network.Score)
			for _, id := range s.Store.Rules.Network.Activities[pending.Activity].Tasks {
				rule := s.Store.Rules.Tasks[id]
				a.UpdateTask(id, int(a.Network.Points), rule.Target)
			}
			if err := s.Store.RecordRank(tx, a.UID, model.RankAsync, int(pending.Activity), int64(a.Network.MaxScore), now); err != nil {
				return err
			}
			if opponent != nil {
				opponent.PrepareNetwork(s.Store.Rules.Network, now)
				defender["score"] = opponent.Network.Score
				opponent.Network.Score = max(1, opponent.Network.Score+int(defenderDelta))
				opponent.Network.MaxScore = max(opponent.Network.MaxScore, opponent.Network.Score)
				if err := s.Store.RecordRank(tx, opponent.UID, model.RankAsync, int(pending.Activity), int64(opponent.Network.MaxScore), now); err != nil {
					return err
				}
			}
		}
		a.RecordJourneyTaskEvent(s.Store.Rules, 106, 1, 0)
		kills, _ := battle.Integer(packet["killnum"])
		if kills >= 0 && kills <= 1000000 {
			a.RecordJourneyBattleTasks(s.Store.Rules, int(input.Battle), win, int(kills), 0, false)
		}
		record, err := server.EncodePayload(packet)
		if err != nil {
			return err
		}
		friend := int64(0)
		if pending.Friend {
			friend = 1
		}
		defenderID := sql.NullInt64{Int64: pending.Opponent, Valid: pending.Opponent > 0}
		if err = s.Store.Database.Queries.WithTx(tx).SaveAsyncRecord(context.Background(), database.SaveAsyncRecordParams{Attacker: a.UID, Defender: defenderID, Opponent: pending.Opponent, Friend: friend, Created: now, Record: record}); err != nil {
			return err
		}
		a.Network.Pending = nil
		return nil
	})
	if err != nil {
		return err
	}
	if live := s.Online(a.UID); live != nil && oldPoints != a.Network.Points {
		_ = live.Send("onAsyncPvpScoreChanged", oldPoints, a.Network.Points)
	}
	if opponent != nil {
		if live := s.Online(opponent.UID); live != nil {
			if err := s.syncNetwork(live); err != nil {
				live.Session.Logger.Error("sync opponent network state", "error", err)
			}
		}
	}
	return nil
}

func (s service) finishNetworkBattle(p *server.Player, r battle.VerifyBattle) (bool, error) {
	pending := p.Account.Network.Pending
	if pending == nil || pending.Battle != r.Battle || pending.Guild != 0 {
		return false, nil
	}
	return true, s.finishAsyncBattle(p, r)
}
