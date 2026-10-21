package model

type RecordCategory string

const (
	RecordDrawHistory RecordCategory = "draw_history"
	RecordDispatch    RecordCategory = "dispatch"
	RecordShop        RecordCategory = "shop"
	RecordItem        RecordCategory = "item"
	RecordHero        RecordCategory = "hero"
	RecordArtifact    RecordCategory = "artifact"
	RecordHighlight   RecordCategory = "highlight"
	RecordGuide       RecordCategory = "guide"
	RecordMission     RecordCategory = "mission"
	RecordPassed      RecordCategory = "passed"
	RecordTask        RecordCategory = "task"
	RecordChapter     RecordCategory = "chapter"
	RecordActivity    RecordCategory = "activity"
)
