package model

type SettingKind uint8

const (
	SettingText SettingKind = iota
	SettingNumber
	SettingBoolean
)

type SettingValue struct {
	Kind    SettingKind `json:"kind"`
	Text    string      `json:"text,omitempty"`
	Number  float64     `json:"number,omitempty"`
	Boolean bool        `json:"boolean,omitempty"`
}

type LocalReceipt struct {
	ID        int64 `json:"id"`
	Product   int   `json:"product"`
	System    int   `json:"system"`
	Reference int   `json:"reference"`
	Count     int   `json:"count"`
	Created   int64 `json:"created"`
}

type SystemHeroPack struct {
	OnSale int64 `json:"on_sale" wire:"onSale"`
	Bought bool  `json:"bought" wire:"bought"`
}

type LocalLeaveReason struct {
	Stage   string `json:"stage"`
	Tags    string `json:"tags"`
	Content string `json:"content"`
	Created int64  `json:"created"`
}

type LocalBattleDiagnostic struct {
	Count        int    `json:"count"`
	Version      int    `json:"version"`
	LastTime     int64  `json:"last_time"`
	RecordDigest string `json:"record_digest"`
}

type SystemShopWindow struct {
	Count int   `json:"count"`
	Ends  int64 `json:"ends"`
}

type SystemState struct {
	LastShare       int64                              `json:"last_share"`
	ProtectionPIN   string                             `json:"protection_pin"`
	ShopWindows     map[ShopID]SystemShopWindow        `json:"shop_windows"`
	LastLeave       *LocalLeaveReason                  `json:"last_leave,omitempty"`
	Diagnostics     map[BattleID]LocalBattleDiagnostic `json:"diagnostics"`
	Settings        map[string]SettingValue            `json:"settings"`
	PushToken       string                             `json:"push_token"`
	PushTags        []string                           `json:"push_tags"`
	ProtectionHash  string                             `json:"protection_hash"`
	ProtectionSalt  string                             `json:"protection_salt"`
	ProtectionReset int64                              `json:"protection_reset"`
	QuestionData    string                             `json:"question_data"`
	DialogCount     int                                `json:"dialog_count"`
	LastDialog      int64                              `json:"last_dialog"`
	Mirage          map[ActivityID]int64               `json:"mirage"`
	Products        map[int]int                        `json:"products"`
	Gifts           map[int]int64                      `json:"gifts"`
	FinishedGifts   map[int]int                        `json:"finished_gifts"`
	HeroPacks       map[int]SystemHeroPack             `json:"hero_packs"`
	Receipts        []LocalReceipt                     `json:"receipts"`
	NextReceipt     int64                              `json:"next_receipt"`
	LastReceipt     *LocalReceipt                      `json:"last_receipt,omitempty"`
	ReturnedPayment bool                               `json:"returned_payment"`
}
