package model

import "mmm-revived/internal/json"

type HonorTags []string

func (HonorTags) ReplacesWhole() {}

type SupportHeroes []HeroID

func (SupportHeroes) ReplacesWhole() {}

func (t *HonorTags) UnmarshalJSON(data []byte) error {
	var values []any
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	tags := make(HonorTags, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			tags = append(tags, text)
		}
	}
	*t = tags
	return nil
}

type ProfileCustomization struct {
	Signature     string           `json:"signature"`
	Head          ItemID           `json:"head"`
	Frame         ItemID           `json:"frame"`
	Background    ItemID           `json:"background"`
	Badge         ItemID           `json:"badge"`
	ShownHeroes   []HeroID         `json:"shown_heroes"`
	SupportHeroes []HeroID         `json:"support_heroes"`
	Tags          HonorTags        `json:"tags"`
	Lobby         LobbyHero        `json:"lobby"`
	HiddenIDs     []string         `json:"hidden_ids"`
	RedDots       map[string]int64 `json:"red_dots"`
}

type ShownHero struct {
	ID   HeroID `wire:"hid"`
	Skin SkinID `wire:"skin"`
}
