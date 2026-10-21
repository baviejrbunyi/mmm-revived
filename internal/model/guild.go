package model

type GuildRole string

const (
	GuildMember  GuildRole = "normal"
	GuildManager GuildRole = "manager"
	GuildBoss    GuildRole = "boss"
)

type GuildSettings struct {
	Name         *string `json:"name"`
	Signature    *string `json:"sig"`
	Announcement *string `json:"announce"`
	Approval     *int    `json:"approveMode"`
	MinimumLevel *int    `json:"requireLv"`
	Head         *int    `json:"head"`
	Frame        *int    `json:"frame"`
}

type GuildInfo struct {
	ID           GuildID `wire:"id"`
	Name         string  `wire:"name"`
	Signature    string  `wire:"sig"`
	Announcement string  `wire:"announce"`
	Approval     int     `wire:"approveMode"`
	MinimumLevel int     `wire:"requireLv"`
	Head         int     `wire:"head"`
	Frame        int     `wire:"frame"`
	Boss         UserID  `wire:"bossUid"`
	Level        int     `wire:"lv"`
	Experience   int64   `wire:"exp"`
	Members      int     `wire:"memberCount"`
	Created      int64   `wire:"createTime"`
	NameUpdated  int64   `wire:"nameUpdateTime"`
	Server       int     `wire:"initServer"`
}

type GuildMemberInfo struct {
	Joined     int64 `wire:"joinTime"`
	Online     int64 `wire:"lastLoginTime"`
	Offline    int64 `wire:"lastLogoutTime"`
	Score      int64 `wire:"score"`
	TotalScore int64 `wire:"historyScore"`
}

type GuildMembersInfo struct {
	Members map[UserID]GuildMemberInfo `wire:"members"`
	Admins  []UserID                   `wire:"adminList"`
	Boss    UserID                     `wire:"bossUid"`
}

type GuildChatMessage struct {
	ID        int64  `wire:"gid"`
	Sender    UserID `wire:"src"`
	Kind      int    `wire:"type"`
	Text      string `wire:"content"`
	Time      int64  `wire:"time"`
	Timestamp int64  `wire:"ts"`
	Poster    string `wire:"poster"`
}

type GuildNews struct {
	ID         int64    `wire:"id"`
	Kind       int      `wire:"type"`
	Time       int64    `wire:"time"`
	Parameters []UserID `wire:"params"`
}
