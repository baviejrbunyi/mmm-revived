package account

import "mmm-revived/internal/model"

type GMRules struct {
	Guides   []string                   `json:"guides"`
	Chapters map[model.ChapterID]string `json:"chapters"`
	Mail     map[int]GMMailRule         `json:"mail"`
}

func (r *Rules) MailTexts(template int) [3]string {
	if r == nil {
		return [3]string{}
	}
	mail := r.GM.Mail[template]
	return [3]string{mail.Title, mail.Content, mail.Sender}
}

func (r *Rules) seasonMail(template int, mail model.SeasonMail) model.SeasonMail {
	texts := r.MailTexts(template)
	mail.Title, mail.Content, mail.Sender = texts[0], texts[1], texts[2]
	return mail
}

type GMMailRule struct {
	Title   string         `json:"title"`
	Content string         `json:"content"`
	Sender  string         `json:"sender"`
	Reward  model.RewardID `json:"reward"`
}
