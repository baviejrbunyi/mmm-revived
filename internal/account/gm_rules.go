package account

import "mmm-revived/internal/model"

type GMRules struct {
	Guides   []string                   `json:"guides"`
	Chapters map[model.ChapterID]string `json:"chapters"`
	Mail     map[int]GMMailRule         `json:"mail"`
}

type GMMailRule struct {
	Title   string         `json:"title"`
	Content string         `json:"content"`
	Sender  string         `json:"sender"`
	Reward  model.RewardID `json:"reward"`
}
