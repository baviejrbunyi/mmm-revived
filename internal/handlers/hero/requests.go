package hero

import (
	"math"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type HeroValue struct {
	Hero  model.HeroID
	Value int
}

type Equip struct {
	Hero     model.HeroID
	Instance model.InstanceID
	Slot     int
}

type Lock struct {
	Instance model.InstanceID
	State    model.LockState
}

func DecodeHero(c protocol.Call) (model.HeroID, error) {
	id, err := c.Number(0, 1, math.MaxInt32)
	return model.HeroID(id), err
}

func DecodeHeroValue(c protocol.Call) (HeroValue, error) {
	hero, err := DecodeHero(c)
	if err != nil {
		return HeroValue{}, err
	}
	value, err := c.Number(1, 0, math.MaxInt32)
	return HeroValue{Hero: hero, Value: int(value)}, err
}

func DecodeLock(c protocol.Call) (Lock, error) {
	instance, err := c.InstanceID(0)
	if err != nil {
		return Lock{}, err
	}
	command, err := c.Number(1, 0, 1)
	return Lock{Instance: instance, State: model.LockState(command)}, err
}

func DecodeEquip(c protocol.Call) (Equip, error) {
	hero, err := DecodeHero(c)
	if err != nil {
		return Equip{}, err
	}
	instance, err := c.InstanceID(1)
	return Equip{Hero: hero, Instance: instance}, err
}

func DecodeEquipSlot(c protocol.Call) (Equip, error) {
	r, err := DecodeEquip(c)
	if err != nil {
		return Equip{}, err
	}
	slot, err := c.Number(2, 1, 2)
	r.Slot = int(slot)
	return r, err
}

func DecodeUnequipHighlight(c protocol.Call) (Equip, error) {
	hero, err := DecodeHero(c)
	if err != nil || len(c.Args) <= 1 {
		return Equip{Hero: hero}, err
	}
	slot, err := c.Number(1, 1, 2)
	return Equip{Hero: hero, Slot: int(slot)}, err
}

type HeroLevel struct {
	Hero   model.HeroID
	Levels int
}

type InstanceLevel struct {
	Instance model.InstanceID
	Levels   int
}

func DecodeHeroLevel(c protocol.Call) (HeroLevel, error) {
	hero, err := c.Number(0, 1, math.MaxInt32)
	if err != nil {
		return HeroLevel{}, err
	}
	levels, err := c.Number(1, 0, 100)
	return HeroLevel{Hero: model.HeroID(hero), Levels: int(levels)}, err
}

func DecodeInstanceLevel(c protocol.Call) (InstanceLevel, error) {
	id, err := c.InstanceID(0)
	if err != nil {
		return InstanceLevel{}, err
	}
	levels, err := c.Number(1, 0, 100)
	return InstanceLevel{Instance: id, Levels: int(levels)}, err
}
