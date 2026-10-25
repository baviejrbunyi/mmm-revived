package realtimepvp

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"reflect"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/matchmaking"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

func (s service) cancelMatch(p *server.Player) error {
	s.Queue().Cancel(p.Account.UID)
	if p.State == server.Queued {
		p.State = server.Lobby
	}
	if err := s.MatchStatus(p, 0); err != nil {
		return err
	}
	return p.Send("onAttachPlayGround")
}

const (
	roundGrace      = 5 * time.Second
	resultConfirm   = 15 * time.Second
	botWait         = 10 * time.Second
	botThinkMinimum = 3 * time.Second
	botThinkSpread  = 5 * time.Second
)

type matchedBot struct {
	UID    int64                       `wire:"uid"`
	Heroes map[model.HeroID]model.Hero `wire:"limitedHeros"`
	Seed   int64                       `wire:"aiRandSeed"`
}

type battleBot struct {
	UID      int64                       `wire:"uid"`
	Heroes   map[model.HeroID]model.Hero `wire:"limitedHeroInfo"`
	Selected []model.Selection           `wire:"selectedHeroInfo"`
	Callback model.Empty                 `wire:"callbackData"`
}

func (s service) joinMatch(p *server.Player, r MatchStart) error {
	if p.State != server.Lobby {
		return fmt.Errorf("%w: player is not in lobby", account.ErrProgression)
	}
	if err := s.season(p); err != nil {
		return err
	}
	if r.Activity != p.Account.RealtimeActivity(s.Store.Rules) || p.Account.Activities[r.Activity].Status != model.PVPMatching {
		return fmt.Errorf("%w: matchmaking preparation is incomplete", account.ErrProgression)
	}
	arena, err := s.arena(r.Activity)
	if err != nil {
		return err
	}
	var first *server.Player
	for first == nil {
		joined := time.Now()
		pair, err := s.Queue().Join(matchmaking.Entry{Player: p.Account.UID, Activity: r.Activity, Version: r.Version, Joined: joined})
		if err != nil {
			return fmt.Errorf("%w: %w", account.ErrProgression, err)
		}
		if pair == nil {
			p.State = server.Queued
			s.After(botWait, func() { s.botFallback(p, joined) })
			return s.MatchStatus(p, 1)
		}
		if candidate := s.Online(pair.First.Player); candidate != nil && candidate.State == server.Queued && candidate.Room == nil {
			first = candidate
		}
	}
	heroes := map[model.UserID]map[model.HeroID]model.Hero{
		first.Account.UID: DraftHeroes(first.Account, r.Activity),
		p.Account.UID:     DraftHeroes(p.Account, r.Activity),
	}
	return s.openRoom([2]*server.Player{first, p}, r.Activity, model.MatchQueue, false, heroes, arena, nil)
}

func (s service) botFallback(p *server.Player, joined time.Time) {
	entry, queued := s.Queue().Entry(p.Account.UID)
	if !queued || !entry.Joined.Equal(joined) || s.Online(p.Account.UID) != p || p.State != server.Queued || p.Room != nil {
		return
	}
	err := s.openBotRoom(p, entry.Activity)
	if err == nil {
		return
	}
	p.Session.Logger.Error("open realtime bot match", "error", err)
	s.Queue().Cancel(p.Account.UID)
	if p.Room != nil {
		return
	}
	p.State = server.Lobby
	if err = s.MatchStatus(p, 0); err == nil {
		err = p.Send("onRealtimePvpMatchFailed")
	}
	if err != nil {
		p.Session.Logger.Error("report realtime bot match failure", "error", err)
	}
}

func (s service) openBotRoom(p *server.Player, activity model.ActivityID) error {
	arena, err := s.arena(activity)
	if err != nil {
		return err
	}
	picks, err := s.Store.Rules.RealtimeBotPlan(activity, p.Account.Activities[activity], s.Store.Rules.Settings.Realtime.Rounds*s.Store.Rules.Settings.Realtime.HeroesPerRound)
	if err != nil {
		return err
	}
	bot := server.NewBot()
	lineup := make(map[model.HeroID]model.Hero, len(picks))
	plan := make([]model.Selection, 0, len(picks))
	for _, pick := range picks {
		lineup[pick.Hero.ID] = BattleHero(model.Hero{ID: pick.Hero.ID, Level: pick.Hero.Level, Star: pick.Hero.Star})
		plan = append(plan, model.Selection{Hero: pick.Hero.ID, Born: pick.Born})
	}
	heroes := map[model.UserID]map[model.HeroID]model.Hero{p.Account.UID: DraftHeroes(p.Account, activity), bot.Account.UID: lineup}
	return s.openRoom([2]*server.Player{p, bot}, activity, model.MatchQueue, false, heroes, arena, plan)
}

func (s service) botSelect(match *server.Room) error {
	bot := match.Bot()
	if bot == nil || match.Confirmed[bot.Account.UID] {
		return nil
	}
	uid := bot.Account.UID
	for index := len(match.Choices[uid]); index < min(len(match.Plan), match.Round*s.Store.Rules.Settings.Realtime.HeroesPerRound); index++ {
		if selected := match.Plan[index]; !selectionTaken(match, uid, selected) {
			addSelection(match, uid, selected, false)
		}
	}
	match.Confirmed[uid] = true
	for _, member := range match.Players {
		if member.Bot() {
			continue
		}
		if err := member.Send("syncSelectHeroNum", server.WireUID(uid), len(match.Choices[uid])); err != nil {
			return err
		}
	}
	return nil
}

func (s service) botThink(match *server.Room) {
	if match.Bot() == nil {
		return
	}
	round := match.Round
	s.After(botThinkMinimum+rand.N(botThinkSpread), func() {
		if match.Finished || match.Started || match.Round != round || len(match.Ready) != 2 {
			return
		}
		err := s.botSelect(match)
		if err == nil && len(match.Confirmed) == 2 {
			err = s.completeRound(match)
		}
		if err != nil {
			match.Players[0].Session.Logger.Error("realtime bot deployment", "room", match.ID, "error", err)
		}
	})
}

func (s service) season(p *server.Player) error {
	if s.Store.Rules == nil {
		return nil
	}
	before := maps.Clone(p.Account.Activities)
	p.Account.PrepareRealtimePVP(s.Store.Rules, time.Now().Unix())
	if !reflect.DeepEqual(before, p.Account.Activities) {
		if err := s.SaveAndSync(p, "activityComp", "taskComp"); err != nil {
			return err
		}
	}
	return s.DeliverSeasonMail(p)
}

type arena struct {
	battles      []int
	environments []int
}

func (s service) arena(activity model.ActivityID) (arena, error) {
	rule, ok := s.Store.Rules.PVP[activity]
	if !ok {
		return arena{}, fmt.Errorf("%w: activity is not open", account.ErrProgression)
	}
	battles, err := s.Store.Rules.RealtimeBattlePool(activity)
	if err != nil {
		return arena{}, err
	}
	if len(rule.Environments) == 0 {
		return arena{}, fmt.Errorf("%w: empty matchmaking rules", account.ErrProgression)
	}
	return arena{battles: battles, environments: rule.Environments}, nil
}

func StartMatch(srv *server.Service, players [2]*server.Player, activity model.ActivityID, mode model.MatchMode, manual bool, heroes map[model.UserID]map[model.HeroID]model.Hero) error {
	s := service{srv}
	arena, err := s.arena(activity)
	if err != nil {
		return err
	}
	return s.openRoom(players, activity, mode, manual, heroes, arena, nil)
}

func (s service) openRoom(players [2]*server.Player, activity model.ActivityID, mode model.MatchMode, manual bool, heroes map[model.UserID]map[model.HeroID]model.Hero, arena arena, plan []model.Selection) error {
	seed := time.Now().Unix() % 2147483647
	match := &server.Room{ID: server.RandomToken(), Players: players, Round: 1, Ready: map[model.UserID]bool{}, Choices: map[model.UserID][]model.Selection{}, Confirmed: map[model.UserID]bool{}, Rows: map[model.UserID][]any{}, Reports: map[model.UserID]model.MatchReport{}, Heroes: heroes, Seed: seed, Deadline: time.Now().Add(s.Config().LoadTimeout), Mode: mode, Manual: manual, Activity: activity, Expire: s.expireRound, Plan: plan}
	if bot := match.Bot(); bot != nil {
		match.Ready[bot.Account.UID] = true
	}
	match.Battle = arena.battles[int(seed)%len(arena.battles)]
	match.Environment = arena.environments[int(seed)%len(arena.environments)]
	first, second := players[0].Account.UID, players[1].Account.UID
	s.CloseInvites(s.Queue().Invites().Remove(func(invite matchmaking.Invite) bool {
		return invite.Involves(first) || invite.Involves(second)
	}), model.DuelBusy, model.DuelCancelled)
	s.OpenRoom(match)
	for _, member := range match.Players {
		s.Queue().Cancel(member.Account.UID)
		member.Room, member.State = match, server.PVPDeployment
		if member.Bot() {
			continue
		}
		if err := s.MatchStatus(member, 0); err != nil {
			return err
		}
	}
	primary := max(first, second)
	if match.Bot() != nil {
		primary = min(first, second)
	}
	var extra any = model.Empty{}
	if manual {
		extra = map[string]any{"mmmManual": true}
	}
	for i, member := range match.Players {
		if member.Bot() {
			continue
		}
		opponent := match.Players[1-i]
		self := model.MatchedPlayer{UID: member.Account.UID, Heroes: s.matchHeroes(member), Seed: match.Seed}
		var other any = model.MatchedPlayer{UID: opponent.Account.UID, Heroes: s.matchHeroes(opponent), Seed: match.Seed}
		if opponent.Bot() {
			other = matchedBot{UID: server.WireUID(opponent.Account.UID), Heroes: s.matchHeroes(opponent), Seed: match.Seed}
		}
		if err := member.Send("pvpBattleMatched", match.Battle, match.ID, primary, self, other, extra, match.Environment); err != nil {
			return err
		}
	}
	return nil
}

func (s service) loadMatch(p *server.Player) error {
	match := p.Room
	if match == nil {
		return nil
	}
	if match.Finished || p.State != server.PVPDeployment {
		return fmt.Errorf("%w: match is not loading", account.ErrProgression)
	}
	if match.Ready[p.Account.UID] {
		return nil
	}
	match.Ready[p.Account.UID] = true
	if len(match.Ready) != 2 {
		return nil
	}
	for _, member := range match.Players {
		if member.Bot() {
			continue
		}
		if err := member.Send("pvpBattleAllLoadReady"); err != nil {
			return err
		}
	}
	return s.newRound(match)
}

func (s service) newRound(match *server.Room) error {
	seconds := s.Store.Rules.Constants.Constants.RealtimeDeployRoundTime
	if match.Round == 1 {
		seconds = s.Store.Rules.Constants.Constants.RealtimeFirstDeployRoundTime
	}
	ends := time.Now().Add(time.Duration(seconds) * time.Second)
	match.Deadline = ends.Add(roundGrace)
	for _, member := range match.Players {
		if member.Bot() {
			continue
		}
		if err := member.Send("pvpBattleNewRound", s.Store.Rules.Settings.Realtime.Rounds, match.Round, ends.Unix(), s.Store.Rules.Settings.Realtime.HeroesPerRound); err != nil {
			return err
		}
	}
	s.botThink(match)
	return nil
}

func (s service) confirmSelection(p *server.Player, r ConfirmSelection) error {
	match := p.Room
	if match == nil || match.Finished || match.Started || r.Round != match.Round || match.Confirmed[p.Account.UID] {
		return nil
	}
	if len(match.Ready) != 2 || len(match.Choices[p.Account.UID])+len(r.Heroes) > match.Round*s.Store.Rules.Settings.Realtime.HeroesPerRound {
		return fmt.Errorf("%w: invalid deployment state", account.ErrProgression)
	}
	if err := s.validatePVPPlacement(p, r); err != nil {
		return err
	}
	for _, selected := range r.Heroes {
		if _, ok := s.matchHeroes(p)[selected.Hero]; !ok {
			return fmt.Errorf("%w: hero is outside the draft", account.ErrProgression)
		}
		if selectionTaken(match, p.Account.UID, selected) {
			return fmt.Errorf("%w: hero or slot is already selected", account.ErrProgression)
		}
	}
	for _, selected := range r.Heroes {
		addSelection(match, p.Account.UID, selected, false)
	}
	match.Confirmed[p.Account.UID] = true
	if opponent := p.Opponent(); !opponent.Bot() {
		if err := opponent.Send("syncSelectHeroNum", p.Account.UID, len(match.Choices[p.Account.UID])); err != nil {
			return err
		}
	}
	if len(match.Confirmed) != 2 {
		return nil
	}
	return s.completeRound(match)
}

func selectionTaken(match *server.Room, uid model.UserID, selected model.Selection) bool {
	for _, previous := range match.Choices[uid] {
		if previous.Hero == selected.Hero || previous.Born == selected.Born {
			return true
		}
	}
	return false
}

func addSelection(match *server.Room, uid model.UserID, selected model.Selection, timeout bool) {
	match.Choices[uid] = append(match.Choices[uid], selected)
	match.Rows[uid] = append(match.Rows[uid], []any{selected.Hero, selected.Born, timeout})
}

func (s service) expireRound(match *server.Room) error {
	if match.Finished || match.Started || len(match.Ready) != 2 {
		return fmt.Errorf("%w: match round cannot expire", account.ErrProgression)
	}
	if err := s.botSelect(match); err != nil {
		return err
	}
	for _, member := range match.Players {
		uid := member.Account.UID
		if match.Confirmed[uid] {
			continue
		}
		match.Confirmed[uid] = true
		placement := match.Positions[uid]
		if placement.Round != match.Round {
			continue
		}
		heroes := s.matchHeroes(member)
		for hero, born := range placement.Heroes {
			selected := model.Selection{Hero: hero, Born: born}
			if _, ok := heroes[hero]; ok && !selectionTaken(match, uid, selected) {
				addSelection(match, uid, selected, true)
			}
		}
	}
	return s.completeRound(match)
}

func (s service) completeRound(match *server.Room) error {
	var round any
	if match.Bot() == nil {
		rows := make(map[model.UserID][]any, len(match.Players))
		for _, member := range match.Players {
			rows[member.Account.UID] = append([]any{}, match.Rows[member.Account.UID]...)
		}
		round = rows
	} else {
		rows := make(map[int64][]any, len(match.Players))
		for _, member := range match.Players {
			rows[server.WireUID(member.Account.UID)] = append([]any{}, match.Rows[member.Account.UID]...)
		}
		round = rows
	}
	match.Rows = map[model.UserID][]any{}
	for _, member := range match.Players {
		if member.Bot() {
			continue
		}
		if err := member.Send("pvpBattleSync", round); err != nil {
			return err
		}
	}
	if match.Round < s.Store.Rules.Settings.Realtime.Rounds {
		match.Round++
		match.Confirmed = map[model.UserID]bool{}
		return s.newRound(match)
	}
	match.Started, match.Deadline = true, time.Now().Add(s.Config().CombatTimeout)
	match.BeginDuel(time.Now())
	for i, member := range match.Players {
		member.State = server.PVPCombat
		if member.Bot() {
			continue
		}
		opponent := match.Players[1-i]
		self, err := encodeBattlePlayer(model.BattlePlayer{UID: member.Account.UID, Heroes: s.matchHeroes(member), Selected: match.Choices[member.Account.UID]})
		if err != nil {
			return err
		}
		var other string
		if opponent.Bot() {
			other, err = server.EncodePayload(battleBot{UID: server.WireUID(opponent.Account.UID), Heroes: s.matchHeroes(opponent), Selected: match.Choices[opponent.Account.UID]})
		} else {
			other, err = encodeBattlePlayer(model.BattlePlayer{UID: opponent.Account.UID, Heroes: s.matchHeroes(opponent), Selected: match.Choices[opponent.Account.UID]})
		}
		if err != nil {
			return err
		}
		if err = member.Send("pvpBattleStart", match.Seed, self, other, match.Environment); err != nil {
			return err
		}
	}
	return nil
}

func encodeBattlePlayer(info model.BattlePlayer) (string, error) {
	return server.EncodePayload(info)
}

func (s service) reportMatch(p *server.Player, report model.MatchReport) error {
	match := p.Room
	if match == nil || match.Finished || !match.Started || report.GID == "" {
		return nil
	}
	if p.State != server.PVPCombat || report.Player != p.Account.UID || report.GID != match.ID {
		return fmt.Errorf("%w: invalid match report", account.ErrProgression)
	}
	a, b := match.Players[0].Account.UID, match.Players[1].Account.UID
	if !(report.Winner == a && report.Opponent == b || report.Winner == b && report.Opponent == a) {
		return fmt.Errorf("%w: report participants do not match room", account.ErrProgression)
	}
	if previous, ok := match.Reports[p.Account.UID]; ok {
		if previous == report {
			return nil
		}
		return fmt.Errorf("%w: conflicting repeated report", account.ErrProgression)
	}
	match.Reports[p.Account.UID] = report
	if match.Bot() != nil {
		return s.FinishMatch(match, report.Winner)
	}
	if len(match.Reports) == 1 {
		return s.AnnounceMatch(match, report.Winner, resultConfirm)
	}
	if match.Claim != report.Winner {
		match.Claim = 0
		return s.FinishMatch(match, 0)
	}
	return s.FinishMatch(match, report.Winner)
}

func (s service) matchHeroes(p *server.Player) map[model.HeroID]model.Hero {
	return p.Room.Heroes[p.Account.UID]
}

func DraftHeroes(a *account.Account, activity model.ActivityID) map[model.HeroID]model.Hero {
	lineup := a.Activities[activity].Lineup
	heroes := make(map[model.HeroID]model.Hero, len(lineup))
	for _, entry := range lineup {
		heroes[entry.ID] = BattleHero(model.Hero{ID: entry.ID, Level: entry.Level, Star: entry.Star})
	}
	return heroes
}

func BattleHero(hero model.Hero) model.Hero {
	if hero.Talents == nil {
		hero.Talents = map[int]int{}
	}
	if hero.Artifacts == nil {
		hero.Artifacts = map[int]model.InstanceID{}
	}
	if hero.Highlights == nil {
		hero.Highlights = map[int]model.InstanceID{}
	}
	if hero.AwakeIDs == nil {
		hero.AwakeIDs = []int{}
	}
	return hero
}
