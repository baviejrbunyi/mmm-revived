package network

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"mmm-revived/internal/handlers/battle"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type Support struct {
	Owner model.UserID
	Hero  model.HeroID
}

type Defense struct {
	Heroes       map[model.HeroID]int
	Order        []model.HeroID
	Interactions map[int][]int
	Environment  int
	Camp         int
}

type ActivityRewards struct {
	Activity model.ActivityID
	IDs      []int
}

type RankQuery struct {
	Mode string
	Rank int
}

type RankPercent struct {
	Rank    int
	Maximum int
}

type AsyncStart struct {
	Battle  model.BattleID
	Version int
	Record  string
	Result  battle.BattleResult
}

const rankModes = "|boss|challenge|achieve|infinite_mayhem|async_pvp|rogue|rogue_pve|guild_war|multi_pvp|"

func DecodeChannel(c protocol.Call) (string, error) {
	if len(c.Args) == 0 || c.Args[0] == nil {
		return "", nil
	}
	if id, err := protocol.Integer(c.Args[0]); err == nil {
		return strconv.FormatInt(id, 10), nil
	}
	return c.Text(0, 64)
}

func DecodeSupport(c protocol.Call) (Support, error) {
	key, err := c.Text(0, 64)
	if err != nil || key == "" {
		return Support{}, err
	}
	ownerText, heroText, ok := strings.Cut(key, ":")
	if !ok || strings.Contains(heroText, ":") {
		return Support{}, errors.New("invalid support key")
	}
	owner, err := strconv.ParseUint(ownerText, 10, 63)
	if err != nil {
		return Support{}, err
	}
	hero, err := strconv.ParseInt(heroText, 10, 32)
	if err == nil && (owner == 0 || hero <= 0) {
		err = errors.New("invalid support identity")
	}
	return Support{Owner: model.UserID(owner), Hero: model.HeroID(hero)}, err
}

func DecodeFriendTarget(c protocol.Call) (model.UserID, error) { return c.Player(1) }

func DecodeDefense(c protocol.Call) (Defense, error) {
	if len(c.Args) == 0 {
		return Defense{}, errors.New("missing defensive hero list")
	}
	var values []any
	if !protocol.EmptyTable(c.Args[0]) {
		var ok bool
		if values, ok = c.Args[0].([]any); !ok || len(values) > 3 {
			return Defense{}, errors.New("invalid defensive hero list")
		}
	}
	r := Defense{Heroes: make(map[model.HeroID]int, len(values)), Order: make([]model.HeroID, 0, len(values))}
	for _, value := range values {
		row, ok := value.([]any)
		if !ok || len(row) != 2 {
			return Defense{}, errors.New("invalid defensive hero")
		}
		hero, err := protocol.Integer(row[0])
		if err != nil || hero <= 0 || hero > math.MaxInt32 {
			return Defense{}, errors.New("invalid defensive hero")
		}
		born, err := protocol.Integer(row[1])
		if err != nil || born < 0 || born > 65535 {
			return Defense{}, errors.New("invalid defensive born position")
		}
		if _, exists := r.Heroes[model.HeroID(hero)]; exists {
			return Defense{}, errors.New("duplicate defensive hero")
		}
		r.Heroes[model.HeroID(hero)] = int(born)
		r.Order = append(r.Order, model.HeroID(hero))
	}
	if len(c.Args) > 1 && protocol.EmptyTable(c.Args[1]) {
		r.Interactions = map[int][]int{}
	} else if err := c.Record(1, &r.Interactions); err != nil {
		return Defense{}, err
	}
	environment, err := c.Number(2, 0, math.MaxInt32)
	if err != nil {
		return Defense{}, err
	}
	camp, err := c.Number(3, 0, math.MaxInt32)
	r.Environment, r.Camp = int(environment), int(camp)
	return r, err
}

func DecodeRankRewards(c protocol.Call) ([]int, error) { return c.Integers(0, 100, 1, 100) }

func DecodeActivityRewards(c protocol.Call) (ActivityRewards, error) {
	activity, err := protocol.DecodeActivity(c)
	if err != nil {
		return ActivityRewards{}, err
	}
	ids, err := c.Integers(1, 100, 1, math.MaxInt32)
	return ActivityRewards{Activity: activity, IDs: ids}, err
}

func DecodeRankQuery(c protocol.Call) (RankQuery, error) {
	mode, err := c.Text(0, 32)
	if err != nil {
		return RankQuery{}, err
	}
	if !strings.Contains(rankModes, "|"+mode+"|") {
		return RankQuery{}, errors.New("invalid rank mode")
	}
	rank, err := c.Number(1, 0, math.MaxInt32)
	return RankQuery{Mode: mode, Rank: int(rank)}, err
}

func DecodeRankPercent(c protocol.Call) (RankPercent, error) {
	rank, err := c.Number(0, 0, math.MaxInt32)
	if err != nil {
		return RankPercent{}, err
	}
	maximum, err := c.Number(1, 0, math.MaxInt32)
	return RankPercent{Rank: int(rank), Maximum: int(maximum)}, err
}

func DecodeAsyncStart(c protocol.Call) (AsyncStart, error) {
	battle, err := c.Number(0, 1, math.MaxInt32)
	if err != nil {
		return AsyncStart{}, err
	}
	version, err := c.Number(1, 0, math.MaxInt32)
	if err != nil {
		return AsyncStart{}, err
	}
	r := AsyncStart{Battle: model.BattleID(battle), Version: int(version)}
	if r.Record, err = c.Text(2, 1<<20); err != nil {
		return AsyncStart{}, err
	}
	err = c.Record(3, &r.Result)
	return r, err
}
