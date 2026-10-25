package account

import (
	"errors"
	"fmt"
	"slices"

	"mmm-revived/internal/model"
)

const freeDrawKeys int64 = 10

type EconomyOptions struct {
	InfiniteEnergy  bool
	FreeDraws       bool
	DreamstoneShop  bool
	AllBanners      bool
	FeaturedBanners []model.DrawPoolID
}

type Economy struct {
	InfiniteEnergy bool
	FreeDraws      bool
	DreamstoneShop bool
	Pools          map[model.DrawPoolID]bool
}

func (s *Store) ConfigureEconomy(o EconomyOptions) error {
	if s.Rules == nil {
		if o.InfiniteEnergy || o.FreeDraws || o.DreamstoneShop || o.AllBanners || len(o.FeaturedBanners) > 0 {
			return errors.New("economy options need the rules data tables")
		}
		return nil
	}
	e := Economy{InfiniteEnergy: o.InfiniteEnergy, FreeDraws: o.FreeDraws, DreamstoneShop: o.DreamstoneShop}
	for _, id := range o.FeaturedBanners {
		if _, known := s.Rules.Draw.Pools[id]; !known {
			return fmt.Errorf("featured_banners: unknown draw pool %d", id)
		}
	}
	if o.AllBanners || len(o.FeaturedBanners) > 0 {
		e.Pools = map[model.DrawPoolID]bool{}
		for id, pool := range s.Rules.Draw.Pools {
			if pool.End > 0 {
				e.Pools[id] = o.AllBanners || slices.Contains(o.FeaturedBanners, id)
			}
		}
	}
	s.Rules.Economy = e
	return nil
}

func (r *Rules) drawPoolOpen(id model.DrawPoolID, pool DrawPoolRule, now int64) bool {
	if open, forced := r.Economy.Pools[id]; forced {
		return open
	}
	calendar := now + r.Draw.CalendarOffset
	return calendar >= pool.Start && (pool.End <= 0 || calendar < pool.End)
}

func (r *Rules) EconomySnapshot(draw *model.DrawComponent) {
	if r == nil {
		return
	}
	if len(r.Economy.Pools) > 0 {
		draw.Open = make(map[model.DrawPoolID]int, len(r.Economy.Pools))
		for id, open := range r.Economy.Pools {
			draw.Open[id] = 0
			if open {
				draw.Open[id] = 1
			}
		}
	}
	if r.Economy.DreamstoneShop {
		draw.DreamstoneShop = 1
	}
}

func (a *Account) applyEconomy(r *Rules) {
	if r == nil || a.Items == nil {
		return
	}
	if r.Economy.InfiniteEnergy {
		a.Items[r.Constants.Items.Energy] = max(a.Items[r.Constants.Items.Energy], r.TeamLevels[a.Level].EnergyLimit)
	}
	if r.Economy.FreeDraws {
		for _, pool := range r.Draw.Pools {
			if pool.Ticket != 0 {
				a.Items[pool.Ticket] = max(a.Items[pool.Ticket], freeDrawKeys)
			}
		}
	}
}
