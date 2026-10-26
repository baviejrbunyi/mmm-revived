package account

import (
	"fmt"
	"math"
)

func (s *Store) ConfigureEnergyRegen(rate float64) error {
	if !(rate > 0) || math.IsInf(rate, 0) {
		return fmt.Errorf("energy_regen must be greater than 0, got %v", rate)
	}
	if s.Rules != nil {
		s.Rules.EnergyRegen = rate
	}
	return nil
}

func (r *Rules) energyInterval(seconds int64) int64 {
	if r.EnergyRegen <= 0 || r.EnergyRegen == 1 || seconds <= 0 {
		return seconds
	}
	return max(1, int64(math.Round(float64(seconds)/r.EnergyRegen)))
}

func (r *Rules) energyIntervals() (int64, int64) {
	if r.EnergyRegen <= 0 || r.EnergyRegen == 1 {
		return 0, 0
	}
	start := r.Defaults.Retail
	return r.energyInterval(start.EnergySeconds), r.energyInterval(start.BankSeconds)
}
