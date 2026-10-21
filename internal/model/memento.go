package model

type MementoChipID int32

type MementoEvent uint8

const (
	MementoNone MementoEvent = iota
	MementoEmpty
	MementoBattle
	MementoBox
	MementoKey
	MementoDoor
	MementoPortal
	MementoFreeStep
	MementoRecover
	MementoBoss
	MementoDeployment
	MementoMaintainMoon
	MementoFire
	MementoShield
	MementoOxygen
)

type MessageScope string

const (
	MementoMessages MessageScope = "memento"
	BattleMessages  MessageScope = "battle"
)
