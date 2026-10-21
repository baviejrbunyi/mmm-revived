package model

import (
	"errors"

	"mmm-revived/internal/json"
)

type LockState uint8

func (s *LockState) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "false":
		*s = 0
		return nil
	case "true":
		*s = 1
		return nil
	}
	var value uint8
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value > 1 {
		return errors.New("invalid equipment lock state")
	}
	*s = LockState(value)
	return nil
}

type Hero struct {
	Liked             int                `json:"liked" wire:"like"`
	ID                HeroID             `json:"id" wire:"id"`
	Level             int                `json:"level" wire:"lv"`
	Star              int                `json:"star" wire:"star"`
	Break             int                `json:"break" wire:"breakLv"`
	Experience        int64              `json:"experience" wire:"exp"`
	Talents           map[int]int        `json:"talents" wire:"talents"`
	AwakeIDs          []int              `json:"awake_ids" wire:"awakeIds"`
	Skin              SkinID             `json:"skin" wire:"skin"`
	Artifacts         map[int]InstanceID `json:"artifacts" wire:"installArti"`
	Highlights        map[int]InstanceID `json:"highlights" wire:"-"`
	Highlight         InstanceID         `json:"-" wire:"installHighlight"`
	Trial             int                `json:"trial" wire:"trial"`
	ProficiencyLevel  int                `json:"proficiency_level" wire:"profLv"`
	ProficiencyXP     int64              `json:"proficiency_xp" wire:"profExp"`
	ProficiencyReward int                `json:"proficiency_reward" wire:"profRwd"`
}

func (h *Hero) UnmarshalJSON(data []byte) error {
	type plain Hero
	if err := json.Unmarshal(data, (*plain)(h)); err != nil {
		return err
	}
	h.Highlight = h.Highlights[1]
	return nil
}

type HeroHistory struct {
	MaxLevel int `wire:"maxLv"`
}

type Artifact struct {
	Growth        map[string][]int64 `wire:"subGrow"`
	PendingGrowth map[string][]int64 `wire:"subGrowBak"`
	Install       HeroID             `wire:"install"`
	Template      int                `wire:"temp"`
	ID            ArtifactID         `wire:"id"`
	Level         int                `wire:"lv"`
	Experience    int64              `wire:"exp"`
	Locked        LockState          `wire:"lock"`
	Sub           map[string]int64   `wire:"subAttr"`
}

type Highlight struct {
	Install    HeroID      `wire:"install"`
	ID         HighlightID `wire:"id"`
	Level      int         `wire:"lv"`
	Break      int         `wire:"breakLv"`
	Awake      int         `wire:"awake"`
	Experience int64       `wire:"exp"`
	Locked     LockState   `wire:"lock"`
}

type HeroComponent struct {
	Heroes          map[HeroID]Hero          `wire:"heros"`
	History         map[HeroID]HeroHistory   `wire:"heroHistory"`
	Artifacts       map[InstanceID]Artifact  `wire:"artifacts"`
	ArtifactCounts  map[ArtifactID]int       `wire:"arti2num"`
	Highlights      map[InstanceID]Highlight `wire:"highlights"`
	HighlightCounts map[HighlightID]int      `wire:"highlight2num"`
	Skins           map[SkinID]int           `wire:"skins"`
	ShowHero        []HeroID                 `wire:"showHero"`
}
