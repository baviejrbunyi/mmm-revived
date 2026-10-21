package dreamland

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type MissionStart struct {
	Chapter model.ChapterID
	Mission model.MissionID
}

type MissionFail struct {
	Mission model.MissionID
	Reason  int
}

type ChapterRewards struct {
	Chapter model.ChapterID
	Stars   []int
}

func DecodeMission(c protocol.Call) (model.MissionID, error) {
	id, err := c.Number(0, 1, math.MaxInt32)
	return model.MissionID(id), err
}

func DecodeMissionStart(c protocol.Call) (MissionStart, error) {
	chapter, err := c.Number(0, 1, math.MaxInt32)
	if err != nil {
		return MissionStart{}, err
	}
	mission, err := c.Number(1, 1, math.MaxInt32)
	return MissionStart{Chapter: model.ChapterID(chapter), Mission: model.MissionID(mission)}, err
}

func DecodeMissionFail(c protocol.Call) (MissionFail, error) {
	mission, err := DecodeMission(c)
	if err != nil {
		return MissionFail{}, err
	}
	reason, err := c.Number(1, 0, 100)
	return MissionFail{Mission: mission, Reason: int(reason)}, err
}

func DecodeChapterRewards(c protocol.Call) (ChapterRewards, error) {
	chapter, err := c.Number(0, 1, math.MaxInt32)
	if err != nil {
		return ChapterRewards{}, err
	}
	stars, err := c.Integers(1, 20, 1, 1000)
	return ChapterRewards{Chapter: model.ChapterID(chapter), Stars: stars}, err
}

type MementoVisit struct {
	Mission model.MissionID
	Chip    model.MementoChipID
	Heroes  []model.HeroID
}

type BoxMessage struct {
	Mission model.MissionID
	Text    string
}

func DecodeMementoVisit(c protocol.Call) (MementoVisit, error) {
	mission, err := DecodeMission(c)
	if err != nil {
		return MementoVisit{}, err
	}
	chip, err := c.Number(1, 0, 10000)
	if err != nil {
		return MementoVisit{}, err
	}
	var info struct {
		Heroes []model.HeroID `json:"heroes"`
	}
	if err = c.Record(2, &info); err != nil {
		return MementoVisit{}, err
	}
	return MementoVisit{Mission: mission, Chip: model.MementoChipID(chip), Heroes: info.Heroes}, nil
}

func message(c protocol.Call, index int) (string, error) {
	text, err := c.Text(index, 1000)
	if err != nil {
		return "", err
	}
	if !utf8.ValidString(text) || strings.ContainsRune(text, 0) || utf8.RuneCountInString(text) > 200 {
		return "", fmt.Errorf("%w: invalid community message", account.ErrProgression)
	}
	return strings.TrimSpace(text), nil
}

func DecodePassMessage(c protocol.Call) (string, error) { return message(c, 0) }

func DecodeBoxMessage(c protocol.Call) (BoxMessage, error) {
	mission, err := DecodeMission(c)
	if err != nil {
		return BoxMessage{}, err
	}
	text, err := message(c, 1)
	return BoxMessage{Mission: mission, Text: text}, err
}
