package battle

import (
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/handlers/rogue"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

func (s service) createBattle(p *server.Player, r CreateBattle) error {
	if p.State == server.Deployment || p.State == server.Combat {
		p.State, p.Nonce, p.Deployment, p.SelectedHeroes = server.Lobby, "", nil, nil
	}
	if p.State != server.Lobby {
		return fmt.Errorf("%w: battle is already active", account.ErrProgression)
	}
	p.RogueBattle, p.BossMission, p.Limited = false, 0, nil
	p.Challenge, p.Spiral = 0, 0
	p.ActivityBattle, p.ActivityMission, p.ActivityNode, p.Brawl = 0, 0, 0, 0
	p.LimitedBoss, p.LimitedBossGroup, p.LimitedBossDifficulty = 0, 0, 0
	p.CampActivity, p.CampNode = 0, 0
	p.Mission, p.Chip, p.Deployment = 0, nil, nil
	for _, prepare := range s.modes.Prepare {
		if handled, err := prepare(s.Service, p, r); handled {
			return err
		}
	}
	campaignMission := r.Extra.Mission
	info := newBattleInfo(s.Service, p.Account)
	var extraBattle any = info
	var limited any = model.Empty{}
	mode := 0
	if rules := s.Store.Rules; rules != nil {
		var known bool
		mode, known = rules.BattleModes[r.Battle]
		if !known {
			return fmt.Errorf("%w: unknown battle", account.ErrProgression)
		}
		if mode == 14 || mode == 22 || mode == 15 {
			bundle, err := p.Account.RogueBattleData(rules, r.Battle)
			if err != nil {
				return err
			}
			if err = s.Store.Update(p.Account, func(tx *sql.Tx) error { return p.Account.BeginRogueBattle(rules, r.Battle) }); err != nil {
				return err
			}
			bundle.Extra.SeasonScience = slices.Clone(p.Account.Events.Season.Nodes)
			if mode != 14 {
				bundle.Extra.Bonds = p.Account.BondAttributes(rules)
			}
			p.RogueBattle, p.Limited = true, bundle.Limited
			extraBattle, limited = bundle.Extra, bundle.Limited
		}
		if mode == 13 {
			if err := p.Account.ValidateBossBattle(rules, r.Extra.BossMission, r.Battle, time.Now().Unix()); err != nil {
				return err
			}
			p.BossMission = r.Extra.BossMission
		}
		if r.Extra.Challenge > 0 {
			multi := r.Extra.Multi
			if multi == 0 {
				multi = 1
			}
			if err := s.Store.Update(p.Account, func(tx *sql.Tx) error {
				return p.Account.BeginJourneyChallenge(rules, r.Extra.Challenge, r.Battle, multi)
			}); err != nil {
				return err
			}
			p.Challenge = r.Extra.Challenge
			campaignMission = 0
		} else if r.Extra.GroupID > 0 && r.Extra.Mission != 0 && rules.Journey.SpiralMissions[int(r.Extra.Mission)].Battle == r.Battle {
			if !slices.Contains(rules.Journey.SpiralGroups[r.Extra.GroupID].Missions, int(r.Extra.Mission)) {
				return account.ErrProgression
			}
			if err := s.Store.Update(p.Account, func(tx *sql.Tx) error { return p.Account.BeginJourneySpiral(rules, int(r.Extra.Mission), r.Battle) }); err != nil {
				return err
			}
			p.Spiral = int(r.Extra.Mission)
			campaignMission = 0
		}
		if r.Extra.Activity > 0 {
			activity, known := rules.Journey.Activities[r.Extra.Activity]
			if known && activity.Type == 20 && r.Extra.OpenID > 0 {
				if err := p.Account.BeginLimitedBoss(rules, r.Extra.Activity, r.Extra.OpenID, r.Extra.Difficulty, r.Battle, time.Now().Unix()); err != nil {
					return err
				}
				p.LimitedBoss, p.LimitedBossGroup, p.LimitedBossDifficulty = r.Extra.Activity, r.Extra.OpenID, r.Extra.Difficulty
				campaignMission = 0
			} else if known && (activity.Type == 20 || activity.Type == 22) {
				node := int(r.Extra.Node)
				if activity.Type == 22 {
					node = r.Extra.BattleNode
				}
				if err := p.Account.ValidateJourneyActivityBattle(rules, r.Extra.Activity, int(r.Extra.Mission), node, r.Battle, time.Now().Unix()); err != nil {
					return err
				}
				p.ActivityBattle, p.ActivityMission, p.ActivityNode = r.Extra.Activity, int(r.Extra.Mission), node
				if activity.Type == 22 {
					buffs := []int{}
					for _, buff := range p.Account.Journey.Activities[r.Extra.Activity].BuffNodes {
						if !buff.Active {
							continue
						}
						if buff.Choice > 0 {
							buffs = append(buffs, buff.Choice)
						} else if rules.Journey.HyperBuffs[buff.ID].Type == 1 {
							buffs = append(buffs, buff.ID)
						}
					}
					info.HyperFlowBuffs = buffs
					extraBattle = info
				}
				campaignMission = 0
			} else if known && activity.Type == 35 {
				if err := s.prepareCamp(p, r); err != nil {
					return err
				}
				p.CampActivity, p.CampNode = r.Extra.Activity, r.Extra.BattleNode
				campaignMission = 0
			} else if known && activity.Type == 10 {
				if _, err := p.Account.JourneyActivityRule(rules, r.Extra.Activity, 10, time.Now().Unix()); err != nil {
					return err
				}
				if !slices.Contains(activity.Battles, r.Battle) {
					return account.ErrProgression
				}
				p.Brawl, info.Activity = r.Extra.Activity, r.Extra.Activity
				info.Environments = []int{mayhemEnvironment(activity.Parameter)}
				extraBattle = info
			}
		}
		if campaignMission != 0 {
			mission, ok := p.Account.Missions[r.Extra.Mission]
			if !ok || mission.Fail != 0 || mission.Node.ID != r.Extra.Node || !p.Account.MissionActive(rules, r.Extra.Mission) || rules.Missions[r.Extra.Mission].Chapter != r.Extra.Chapter {
				return fmt.Errorf("%w: battle does not match mission", account.ErrProgression)
			}
			if rules.Nodes[mission.Node.ID].Type == 5 {
				if r.Extra.Chip == nil {
					return fmt.Errorf("%w: Memento tile is missing", account.ErrProgression)
				}
				if err := p.Account.MementoBattle(rules, r.Extra.Mission, *r.Extra.Chip, r.Battle); err != nil {
					return err
				}
				mode = 4
			} else {
				if r.Extra.Chip != nil || rules.Nodes[mission.Node.ID].Battle != r.Battle {
					return fmt.Errorf("%w: battle does not match mission node", account.ErrProgression)
				}
				if rules.Missions[r.Extra.Mission].Memento {
					mode = 5
				} else if mode != 2 && mode != 3 {
					mode = 1
				}
			}
		} else if mode == 4 || mode == 5 {
			return fmt.Errorf("%w: Memento context is missing", account.ErrProgression)
		}
	}
	p.Chip, p.SelectedHeroes, p.LastBattleWon = r.Extra.Chip, nil, false
	p.Nonce, p.BattleID, p.State, p.Mission = server.RandomToken(), r.Battle, server.Deployment, campaignMission
	if p.Challenge > 0 {
		if err := s.Sync(p, "itemComp", "challengeComp"); err != nil {
			return err
		}
	}
	echo := r.Extra
	if echo.Target != nil {
		echo.RetailTarget = nil
	}
	var rule account.BattleRule
	if s.Store.Rules != nil {
		rule = s.Store.Rules.BattleSettings[r.Battle]
	}
	if rule.SkipDeploy {
		p.State, p.SelectedHeroes = server.Combat, fixedHeroes(rule)
	}
	heroes, artifacts, highlights := battleGear(p.Account, rule)
	return p.Send("createBattle", r.Battle, p.Account.Events.Season.ID, mode, heroes, artifacts, highlights, TrialInfo{}, limited, extraBattle, echo, p.Nonce)
}

func (s service) verifyBattle(p *server.Player, r VerifyBattle) error {
	err := s.settleBattle(p, r)
	if !errors.Is(err, account.ErrProgression) {
		return err
	}
	if r.Battle == p.BattleID && (p.State == server.Deployment || p.State == server.Combat) {
		p.State, p.Nonce = server.Lobby, ""
	}
	failed := r.Result
	failed.Win = false
	if sent := p.Send("onBattleEnd", r.Battle, failed); sent != nil {
		return sent
	}
	return err
}

func (s service) settleBattle(p *server.Player, r VerifyBattle) error {
	if p.State != server.Combat && (p.State != server.Deployment || r.Win) {
		return fmt.Errorf("%w: battle is not awaiting a result", account.ErrProgression)
	}
	if r.Battle != p.BattleID {
		return fmt.Errorf("%w: result battle ID does not match", account.ErrProgression)
	}
	if r.Nonce != p.Nonce {
		return fmt.Errorf("%w: battle verification token does not match", account.ErrProgression)
	}
	if r.Result.UID != 0 && r.Result.UID != p.Account.UID {
		return fmt.Errorf("%w: battle result belongs to player %d, expected %d", account.ErrProgression, r.Result.UID, p.Account.UID)
	}
	for _, finish := range s.modes.Finish {
		if handled, err := finish(s.Service, p, r); handled {
			return err
		}
	}
	nodeChanged := false
	mode := 0
	if s.Store.Rules != nil {
		mode = s.Store.Rules.BattleModes[p.BattleID]
	}
	beforeReward := *p.Account
	battleRewards := map[model.ItemID]int64{}
	previousLevel, teamBroke := p.Account.Level, false
	err := s.Store.Update(p.Account, func(tx *sql.Tx) error {
		if p.ActivityBattle > 0 {
			items, err := p.Account.FinishJourneyActivityBattle(s.Store.Rules, p.ActivityBattle, p.ActivityMission, p.ActivityNode, p.BattleID, r.Win, r.Result.StarBitmap, p.SelectedHeroes, time.Now().Unix())
			if err != nil {
				return err
			}
			if r.Win {
				p.Account.RecordJourneyHyperBattleDetails(p.ActivityBattle, p.ActivityNode, p.BattleID, r.Result.Extra.DiedCreatures, r.Result.Extra.UseTime)
			}
			for id, count := range items {
				battleRewards[id] += count
			}
		}
		if p.LimitedBoss > 0 {
			items, err := p.Account.FinishLimitedBoss(s.Store.Rules, p.LimitedBoss, p.LimitedBossGroup, p.LimitedBossDifficulty, r.Win)
			if err != nil {
				return err
			}
			for id, count := range items {
				battleRewards[id] += count
			}
		}
		if p.CampActivity > 0 {
			if err := p.Account.FinishRetailCamp(s.Store.Rules, p.CampActivity, p.CampNode, p.BattleID, r.Win, r.Result.Extra.UseTime, time.Now().Unix()); err != nil {
				return err
			}
		}
		if p.Brawl > 0 {
			if err := p.Account.RecordJourneyBrawl(s.Store.Rules, p.Brawl, p.BattleID, r.Result.BrawlStars, time.Now().Unix()); err != nil {
				return err
			}
			if err := s.Store.RecordRank(tx, p.Account.UID, model.RankInfinite, int(p.Brawl), int64(r.Result.BrawlStars), time.Now().Unix()); err != nil {
				return err
			}
		}
		if p.Challenge > 0 {
			items, err := p.Account.FinishJourneyChallenge(s.Store.Rules, p.Challenge, p.BattleID, r.Win, r.Result.StarBitmap, r.Result.Extra.UseTime, r.Result.KillCount)
			if err != nil {
				return err
			}
			for id, count := range items {
				battleRewards[id] += count
			}
		}
		if p.Spiral > 0 && r.Win {
			items, err := p.Account.FinishJourneySpiral(s.Store.Rules, p.Spiral, p.BattleID, r.Result.StarBitmap)
			if err != nil {
				return err
			}
			for id, count := range items {
				battleRewards[id] += count
			}
		}
		if p.RogueBattle {
			if len(r.Result.Extra.Inherit) > 0 {
				if err := p.Account.ApplyRogueHeroStatus(r.Result.Extra.Inherit); err != nil {
					return err
				}
			}
			if p.Account.RogueProgress.Run.Status == model.RogueBattle {
				if err := p.Account.ApplyRogueBattleProgress(r.Result.Extra.RogueLevelUp, r.Result.Extra.DiedCreatures, len(r.Result.Extra.DiedCreatures)); err != nil {
					return err
				}
			} else if mode == 22 && p.Account.RogueProgress.PVE.Status == model.RoguePVEBattle {
				p.Account.RogueProgress.PVE.Mana = r.Result.Extra.Mana
			}
			if err := p.Account.FinishRogueBattle(s.Store.Rules, p.BattleID, r.Win, r.Result.Extra.UseTime); err != nil {
				return err
			}
		}
		if p.BossMission != 0 {
			if err := p.Account.FinishBossBattle(s.Store.Rules, p.BossMission, r.Win, r.Result.Extra.UseTime); err != nil {
				return err
			}
			p.Account.RecordBossBattle(p.BossMission, r.Win, p.SelectedHeroes, time.Now().Unix())
		}
		if s.Store.Rules != nil {
			group := s.Store.Rules.BattleModes[p.BattleID]
			if p.Challenge > 0 {
				group = s.Store.Rules.Journey.Challenges[p.Challenge].Group
			}
			statistics := statisticsFrom(r.Packet)
			p.Account.RecordJourneyBattleTasks(s.Store.Rules, group, r.Win, r.Result.KillCount, statistics.Mana, r.Result.Extra.UseAssist)
			p.Account.RecordJourneyBattleStatistics(s.Store.Rules, statistics.Ultimates, statistics.Cards)
			p.Account.RecordJourneyBattleResult(s.Store.Rules, p.BattleID, r.Win, r.Packet)
			p.Account.CompleteJourneyBattle(s.Store.Rules, p.BattleID, r.Win, r.Result.StarBitmap, time.Now().Unix())
			if r.Win {
				items, broke, err := p.Account.FinishTeamBreak(s.Store.Rules, p.BattleID)
				if err != nil {
					return err
				}
				for id, count := range items {
					battleRewards[id] += count
				}
				teamBroke = broke
			}
		}
		if p.Mission != 0 {
			if r.Win {
				if r.Result.StarBitmap == 0 {
					return fmt.Errorf("%w: winning result has no stars", account.ErrProgression)
				}
				if p.Chip != nil {
					previous := p.Account.Missions[p.Mission].Node.ID
					if err := p.Account.FinishMementoBattle(s.Store.Rules, p.Mission, *p.Chip, r.Result.StarBitmap, p.SelectedHeroes); err != nil {
						return err
					}
					nodeChanged = previous != p.Account.Missions[p.Mission].Node.ID
				} else {
					if err := p.Account.FinishMissionBattle(s.Store.Rules, p.Mission, r.Result.StarBitmap); err != nil {
						return err
					}
					nodeChanged = true
				}
			} else if p.Chip != nil {
				mission := p.Account.Missions[p.Mission]
				mission.Heroes = maps.Clone(mission.Heroes)
				if mission.Heroes == nil {
					mission.Heroes = map[model.HeroID]int{}
				}
				for _, hero := range p.SelectedHeroes {
					mission.Heroes[hero] = 1
				}
				p.Account.Missions[p.Mission] = mission
			}
		}
		if r.Result.Extra.UseAssist {
			if err := s.Store.RecordSupport(p.Account, tx, time.Now().Unix()); err != nil {
				return err
			}
		}
		if r.Win {
			p.Account.AddProficiency(s.Store.Rules, p.SelectedHeroes, 50)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if teamBroke {
		p.Level = p.Account.Level
	}
	if err := s.Sync(p, "dreamlandComp", "itemComp", "heroComp", "noviceComp", "activityComp", "bossComp", "battleNewComp", "challengeComp", "spiralComp", "taskComp"); err != nil {
		return err
	}
	if p.RogueBattle {
		if err := rogue.Sync(p); err != nil {
			return err
		}
		if id, active := p.Account.ActiveRoguePVEEvent(s.Store.Rules); mode == 22 && active {
			if err := p.Send("RoguePveEvent", id); err != nil {
				return err
			}
		}
	}
	if p.ActivityBattle > 0 && s.Store.Rules.Journey.Activities[p.ActivityBattle].Type == 22 {
		if err := p.Send("battleNodeUpdate", false, p.Account.Journey.Activities[p.ActivityBattle].BattleNodes); err != nil {
			return err
		}
	}
	p.State, p.Nonce, p.LastBattleWon = server.Lobby, "", r.Win
	if err := p.Send("confirmBattleVerify"); err != nil {
		return err
	}
	if p.Chip != nil && r.Win && !nodeChanged {
		if err := p.Send("onDlVisitChip", *p.Chip, true); err != nil {
			return err
		}
	}
	if err := p.Send("onBattleEnd", p.BattleID, r.Result); err != nil {
		return err
	}
	if teamBroke {
		if err := p.Send("breakLv", previousLevel, p.Account.Level, p.Account.TeamBreak); err != nil {
			return err
		}
	}
	if len(battleRewards) > 0 {
		if err := p.Send("showReward", []model.RewardID{}, s.ItemRewards(beforeReward, p.Account, battleRewards), 1); err != nil {
			return err
		}
	}
	if nodeChanged {
		return p.Send("onInitNode", p.Mission, p.Account.Missions[p.Mission].Node.ID)
	}
	return nil
}

func (s service) prepareCamp(p *server.Player, request CreateBattle) error {
	return s.Store.Update(p.Account, func(*sql.Tx) error {
		return p.Account.BeginRetailCamp(s.Store.Rules, request.Extra.Activity, request.Extra.BattleNode, request.Battle, time.Now().Unix())
	})
}
