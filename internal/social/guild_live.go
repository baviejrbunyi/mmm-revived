package social

import (
	"maps"
	"slices"
	"sync"

	"mmm-revived/internal/model"
)

type guildProgress struct {
	mu      sync.Mutex
	changed map[model.GuildID][]model.GuildNews
}

type GuildProgress struct {
	Guild model.GuildID
	News  []model.GuildNews
}

func (g *guildProgress) note(id model.GuildID, news []model.GuildNews) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.changed == nil {
		g.changed = map[model.GuildID][]model.GuildNews{}
	}
	g.changed[id] = append(g.changed[id], news...)
}

func (r *Repository) TakeGuildProgress() []GuildProgress {
	g := &r.progress
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.changed) == 0 {
		return nil
	}
	result := make([]GuildProgress, 0, len(g.changed))
	for _, id := range slices.Sorted(maps.Keys(g.changed)) {
		result = append(result, GuildProgress{Guild: id, News: g.changed[id]})
	}
	clear(g.changed)
	return result
}
