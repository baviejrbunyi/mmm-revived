package database

import (
	"mmm-revived/internal/model"
)

type Guild struct {
	ID           model.GuildID
	Name         string
	SearchName   string
	Signature    string
	Announcement string
	Approval     int64
	MinimumLevel int64
	Head         int64
	Frame        int64
	Boss         model.UserID
	Level        int64
	Experience   int64
	MemberCount  int64
	Created      int64
	NameUpdated  int64
}

type GuildMember struct {
	UID          model.UserID
	GuildID      model.GuildID
	Role         model.GuildRole
	Joined       int64
	Score        int64
	HistoryScore int64
}

type GuildNews struct {
	ID      int64
	GuildID model.GuildID
	Kind    int64
	Actor   model.UserID
	Subject model.UserID
	Value   int64
	Created int64
}

type GuildPlayerState struct {
	UID        model.UserID
	LastLeft   int64
	LastSearch int64
}

type Mailbox struct {
	ID        model.MailID
	Owner     model.UserID
	Title     []byte
	Content   []byte
	Sender    []byte
	Template  int64
	Created   int64
	Expires   int64
	Status    model.MailStatus
	RewardID  model.RewardID
	Items     []byte
	HasReward int64
}
