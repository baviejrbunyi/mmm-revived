package battle

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"

	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type BattleExtra struct {
	EnemyIndex       int                  `json:"enemyIdx" wire:"enemyIdx,omitempty"`
	IsRevenge        int                  `json:"isRevenge" wire:"isRevenge,omitempty"`
	RevengeIndex     int                  `json:"revengeIdx" wire:"revengeIdx,omitempty"`
	RevengeMineIndex string               `json:"revengeMineIdx" wire:"revengeMineIdx,omitempty"`
	RetailTarget     *model.RetailPVPCard `json:"-" wire:"target,omitempty"`

	BattleNode   int                  `json:"battleNodeId" wire:"battleNodeId,omitempty"`
	Challenge    int                  `json:"challengeId" wire:"challengeId,omitempty"`
	Multi        int                  `json:"multi" wire:"multi,omitempty"`
	MaterialType int                  `json:"materialType" wire:"materialType,omitempty"`
	BossMission  int                  `json:"bossMissionId" wire:"bossMissionId,omitempty"`
	Difficulty   int                  `json:"difficulty" wire:"difficulty,omitempty"`
	IsRecord     bool                 `json:"isRecord" wire:"isRecord,omitempty"`
	EventID      int                  `json:"eventId" wire:"eventId,omitempty"`
	Monster      model.HeroID         `json:"monster" wire:"monster,omitempty"`
	Mode         int                  `json:"mode" wire:"mode,omitempty"`
	Activity     model.ActivityID     `json:"activityId" wire:"activityId,omitempty"`
	OpenID       int                  `json:"openId" wire:"openId,omitempty"`
	Target       *model.AsyncTarget   `json:"target" wire:"target,omitempty"`
	FriendUID    model.UserID         `json:"friend_uid_pvp" wire:"friend_uid_pvp,omitempty"`
	AreaID       int                  `json:"areaId" wire:"areaId,omitempty"`
	GroupID      int                  `json:"groupId" wire:"groupId,omitempty"`
	LevelID      int                  `json:"levelId" wire:"levelId,omitempty"`
	BattleToken  string               `json:"battleToken" wire:"battleToken,omitempty"`
	Chip         *model.MementoChipID `json:"chip" wire:"chip,omitempty"`
	Mission      model.MissionID      `json:"missionId" wire:"missionId,omitempty"`
	Chapter      model.ChapterID      `json:"chapterId" wire:"chapterId,omitempty"`
	Node         model.NodeID         `json:"nodeId" wire:"nodeId,omitempty"`
}

type BattleResultExtra struct {
	DiedCreatures []model.HeroID                         `json:"dieCreatures" wire:"dieCreatures"`
	UseTime       int64                                  `json:"useTime" wire:"useTime"`
	Inherit       map[model.HeroID]model.RogueHeroStatus `json:"inherit" wire:"inherit"`
	RogueLevelUp  map[model.HeroID]int                   `json:"rogueLvUp" wire:"rogueLvUp"`
	Mana          int64                                  `json:"mana" wire:"mana"`
	UseAssist     bool                                   `json:"useAssist" wire:"useAssist"`
}

type BattleResult struct {
	Extra      BattleResultExtra `json:"extra" wire:"extra"`
	KillCount  int               `json:"killnum" wire:"killnum"`
	BrawlStars int               `json:"skillBrawlStar" wire:"skillBrawlStar"`
	UID        model.UserID      `json:"uid" wire:"uid"`
	Stars      int               `json:"star" wire:"star"`
	StarBitmap int               `json:"starBitMap" wire:"starBitMap"`
	Win        bool              `json:"win" wire:"win"`
	Duration   int64             `json:"battleTime" wire:"battleTime"`
}

type CreateBattle struct {
	Battle model.BattleID
	Extra  BattleExtra
	raw    any
}

func (r CreateBattle) RetailCard() (*model.RetailPVPCard, error) { return decodeMultiPVPCard(r.raw) }

type VerifyBattle struct {
	Record  string
	Packet  map[any]any
	Battle  model.BattleID
	Version int
	Win     bool
	Result  BattleResult
	Nonce   string
}

func DecodeCreateBattle(c protocol.Call) (CreateBattle, error) {
	id, err := c.BattleID()
	r := CreateBattle{Battle: id}
	if err == nil && len(c.Args) >= 3 {
		err = c.Record(2, &r.Extra)
		r.raw = c.Args[2]
	}
	return r, err
}

func DecodeVerifyBattle(c protocol.Call) (VerifyBattle, error) {
	id, err := c.BattleID()
	if err != nil {
		return VerifyBattle{}, err
	}
	version, err := c.Number(1, 0, math.MaxInt32)
	if err != nil {
		return VerifyBattle{}, err
	}
	if len(c.Args) < 6 || len(c.Args) > 8 {
		return VerifyBattle{}, errors.New("battleVerify requires a battle result")
	}
	win, ok := c.Args[2].(bool)
	if !ok {
		return VerifyBattle{}, errors.New("battleVerify requires a boolean outcome")
	}
	r := VerifyBattle{Battle: id, Version: int(version), Win: win}
	if r.Record, err = c.Text(3, 4<<20); err != nil {
		return VerifyBattle{}, err
	}
	if r.Packet, ok = c.Args[5].(map[any]any); !ok {
		return VerifyBattle{}, errors.New("battle result must be a dictionary")
	}
	if err = c.Record(5, &r.Result); err != nil {
		return VerifyBattle{}, err
	}
	if len(c.Args) == 8 && c.Args[7] != nil {
		r.Nonce, err = c.Text(7, 128)
	}
	if r.Result.Extra.UseTime < 0 || r.Result.Extra.UseTime > 86400 || r.Result.KillCount < 0 || r.Result.KillCount > 1000000 {
		return VerifyBattle{}, errors.New("invalid battle metrics")
	}
	if r.Result.Stars < 0 || r.Result.Stars > 3 || r.Result.StarBitmap < 0 || r.Result.StarBitmap > 7 || r.Result.Win != win {
		return VerifyBattle{}, errors.New("invalid battle result")
	}
	return r, err
}

func decodeMultiPVPCard(extra any) (*model.RetailPVPCard, error) {
	values, okay := extra.(map[any]any)
	if !okay {
		return nil, nil
	}
	target, okay := values["target"].(map[any]any)
	if !okay {
		return nil, nil
	}
	name, okay := target["name"].(string)
	if !okay {
		return nil, nil
	}
	if len(name) > 256 {
		return nil, errors.New("invalid Halloween player name")
	}
	uid, err := protocol.Integer(target["uid"])
	if err != nil {
		return nil, err
	}
	card := &model.RetailPVPCard{UID: uid, Name: name}
	number := func(value any) (int64, error) {
		if value == nil {
			return 0, nil
		}
		if text, okay := value.(string); okay {
			return strconv.ParseInt(text, 10, 32)
		}
		return protocol.Integer(value)
	}
	head, err := number(target["h"])
	if err != nil || head < 0 || head > math.MaxInt32 {
		return nil, errors.New("invalid Halloween player head")
	}
	frame, err := number(target["hf"])
	if err != nil || frame < 0 || frame > math.MaxInt32 {
		return nil, errors.New("invalid Halloween player frame")
	}
	card.Head, card.Frame = model.ItemID(head), model.ItemID(frame)
	return card, nil
}

type Deployment struct {
	Heroes []model.DeploymentHero
	Cards  []int
}

func DecodeDeployment(c protocol.Call) (Deployment, error) {
	if len(c.Args) != 2 {
		return Deployment{}, errors.New("lockHero requires deployment and card data")
	}
	r := Deployment{Heroes: []model.DeploymentHero{}, Cards: []int{}}
	values, ok := c.Args[0].([]any)
	if empty, dictionary := c.Args[0].(map[any]any); dictionary && len(empty) == 0 {
		values, ok = []any{}, true
	}
	if !ok || len(values) > 10 {
		return Deployment{}, errors.New("invalid deployment list")
	}
	slots, heroes, orders := map[int64]bool{}, map[int64]bool{}, map[int64]bool{}
	for _, value := range values {
		row, ok := value.([]any)
		if !ok || len(row) != 4 {
			return Deployment{}, errors.New("invalid deployment hero")
		}
		parsed := [4]int64{}
		for index, value := range row {
			if index == 2 {
				if trial, ok := value.(bool); ok {
					if trial {
						parsed[index] = 1
					}
					continue
				}
			}
			n, err := protocol.Integer(value)
			if err != nil || n > math.MaxInt32 || index == 0 && n < math.MinInt32 || index != 0 && n < 0 {
				return Deployment{}, errors.New("invalid deployment integer")
			}
			parsed[index] = n
		}
		if parsed[1] == 0 || parsed[2] > 1 || parsed[3] == 0 || slots[parsed[0]] || heroes[parsed[1]] || orders[parsed[3]] {
			return Deployment{}, fmt.Errorf("invalid or duplicate deployment hero: slot=%d hero=%d trial=%d order=%d", parsed[0], parsed[1], parsed[2], parsed[3])
		}
		slots[parsed[0]], heroes[parsed[1]], orders[parsed[3]] = true, true, true
		r.Heroes = append(r.Heroes, model.DeploymentHero{Slot: int(parsed[0]), Hero: model.HeroID(parsed[1]), Trial: int(parsed[2]), Order: int(parsed[3])})
	}
	info, ok := c.Args[1].(map[any]any)
	if !ok {
		return Deployment{}, errors.New("invalid deployment card data")
	}
	if info["selectedCommonCard"] == nil {
		return r, nil
	}
	cards, ok := info["selectedCommonCard"].([]any)
	if protocol.EmptyTable(info["selectedCommonCard"]) {
		cards, ok = []any{}, true
	}
	if !ok {
		return Deployment{}, errors.New("invalid common card list")
	}
	if len(cards) > 20 {
		return Deployment{}, errors.New("too many common cards")
	}
	seen := map[int]bool{}
	for _, value := range cards {
		id, err := protocol.Integer(value)
		if err != nil || id <= 0 || id > math.MaxInt32 || seen[int(id)] {
			return Deployment{}, errors.New("invalid common card")
		}
		seen[int(id)] = true
		r.Cards = append(r.Cards, int(id))
	}
	return r, nil
}

func decodeBattleList[T any](data json.RawMessage) ([]T, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return []T{}, nil
	}
	if data[0] == '{' {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			return nil, err
		}
		if len(fields) == 0 || len(fields) == 1 && bytes.Equal(bytes.TrimSpace(fields["_n"]), []byte("0")) {
			return []T{}, nil
		}
		return nil, errors.New("battle list must be an array or empty Lua table")
	}
	var values []T
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func (r *BattleResultExtra) UnmarshalJSON(data []byte) error {
	type plain BattleResultExtra
	var result plain
	fields := struct {
		*plain
		DiedCreatures json.RawMessage `json:"dieCreatures"`
		Inherit       json.RawMessage `json:"inherit"`
		RogueLevelUp  json.RawMessage `json:"rogueLvUp"`
		Mana          *float64        `json:"mana"`
	}{plain: &result}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	values, err := decodeBattleList[model.HeroID](fields.DiedCreatures)
	if err != nil {
		return fmt.Errorf("dieCreatures: %w", err)
	}
	result.DiedCreatures = values
	inherit, err := decodeBattleDictionary[model.HeroID, inheritedHero](fields.Inherit)
	if err != nil {
		return fmt.Errorf("inherit: %w", err)
	}
	if inherit != nil {
		result.Inherit = make(map[model.HeroID]model.RogueHeroStatus, len(inherit))
		for id, hero := range inherit {
			result.Inherit[id] = model.RogueHeroStatus{Energy: int64(math.Floor(hero.Energy)), HP: int64(math.Floor(hero.HP)), MaximumHP: int64(math.Floor(hero.MaximumHP))}
		}
	}
	if fields.Mana != nil {
		result.Mana = int64(math.Floor(*fields.Mana))
	}
	if result.RogueLevelUp, err = decodeBattleDictionary[model.HeroID, int](fields.RogueLevelUp); err != nil {
		return fmt.Errorf("rogueLvUp: %w", err)
	}
	*r = BattleResultExtra(result)
	return nil
}

type inheritedHero struct {
	Energy    float64 `json:"energy"`
	HP        float64 `json:"hp"`
	MaximumHP float64 `json:"maxhp"`
}

func decodeBattleDictionary[K comparable, V any](data json.RawMessage) (map[K]V, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil, nil
	}
	if data[0] == '[' {
		var values []json.RawMessage
		if err := json.Unmarshal(data, &values); err != nil || len(values) > 0 {
			return nil, errors.New("battle dictionary must not be a list")
		}
		return map[K]V{}, nil
	}
	var values map[K]V
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func (r *BattleResult) UnmarshalJSON(data []byte) error {
	type plain BattleResult
	var result plain
	fields := struct {
		*plain
		StarBitmap json.RawMessage `json:"starBitMap"`
	}{plain: &result}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	bitmap := bytes.TrimSpace(fields.StarBitmap)
	if len(bitmap) > 0 && !bytes.Equal(bitmap, []byte("null")) {
		switch bitmap[0] {
		case '[':
			var stars []*bool
			if err := json.Unmarshal(bitmap, &stars); err != nil || len(stars) > 3 {
				return errors.New("invalid battle star bitmap")
			}
			for index, star := range stars {
				if star != nil && *star {
					result.StarBitmap |= 1 << index
				}
			}
		case '{':
			var stars map[int]bool
			if err := json.Unmarshal(bitmap, &stars); err != nil {
				return errors.New("invalid battle star bitmap")
			}
			for index, star := range stars {
				if index < 1 || index > 3 {
					return errors.New("invalid battle star index")
				}
				if star {
					result.StarBitmap |= 1 << (index - 1)
				}
			}
		default:
			if err := json.Unmarshal(bitmap, &result.StarBitmap); err != nil {
				return fmt.Errorf("starBitMap: %w", err)
			}
		}
	}
	if result.StarBitmap < 0 || result.StarBitmap > 7 {
		return errors.New("invalid battle star bitmap")
	}
	*r = BattleResult(result)
	return nil
}
