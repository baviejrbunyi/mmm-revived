package model

type MatchMode int

const (
	MatchQueue MatchMode = iota
	MatchOwnHeroes
	MatchRandomHeroes
	MatchDraft
)

func (m MatchMode) UsesDraft() bool { return m == MatchQueue || m == MatchDraft }

type DuelResult int

const (
	DuelSent DuelResult = iota
	DuelDeclined
	DuelBusy
	DuelOffline
	DuelExpired
	DuelNotFriend
	DuelNoDraft
	DuelCancelled
	DuelNoHeroes
)
