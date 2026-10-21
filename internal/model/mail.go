package model

type MailID int64
type MailStatus int64

const (
	MailNew MailStatus = 1 + iota
	MailOpened
	MailRewarded
	MailDeleted
)

type LocalizedText map[string]string

type MailAttachment struct {
	Item       ItemID `wire:"type"`
	Count      int64  `wire:"num"`
	Attributes string `wire:"attr"`
}

type MailInfo struct {
	Status   MailStatus       `wire:"status"`
	Expires  int64            `wire:"validity"`
	Sent     int64            `wire:"sendTime"`
	Template int64            `wire:"tableId"`
	From     UserID           `wire:"fromId"`
	Title    LocalizedText    `wire:"title"`
	Content  LocalizedText    `wire:"content"`
	Sender   LocalizedText    `wire:"sender"`
	Reward   RewardID         `wire:"rewardId"`
	Items    []MailAttachment `wire:"rewardInfo"`
}
