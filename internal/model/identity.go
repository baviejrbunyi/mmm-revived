package model

type UserID uint64
type GuildID int64
type HeroID int32
type ItemID int32
type SkinID int32
type BattleID int32
type ArtifactID int32
type HighlightID int32
type InstanceID string
type TaskID int32
type MissionID int32
type ChapterID int32
type NodeID int32
type RewardID int32
type ActivityID int32

type Empty struct{}

type LoginInfo struct {
	HostNumber      int      `wire:"hostnum"`
	TestServer      int      `wire:"istest"`
	GMEnabled       int      `wire:"gmcmd"`
	Server          string   `wire:"server_version"`
	PlatformSession string   `wire:"unisdk_login_json"`
	ServerTime      [2]int64 `wire:"server_time"`
}
