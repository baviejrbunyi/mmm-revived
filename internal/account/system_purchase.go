package account

import (
	"errors"
	"math"

	"mmm-revived/internal/model"
)

const (
	ShipCommon          = 1
	ShipProduct         = 2
	ShipLevel           = 3
	ShipPredecessor     = 4
	ShipLimit           = 5
	ShipTime            = 6
	ShipCount           = 7
	ShipConfiguration   = 8
	ShipSystem          = 9
	ShipGiftUnavailable = 11
)

type ShipError struct {
	Code   int
	Reason string
}

func (e ShipError) Error() string { return ErrProgression.Error() + ": " + e.Reason }

func (e ShipError) Unwrap() error { return ErrProgression }

func ShipErrorCode(err error) int {
	if shipError, ok := errors.AsType[ShipError](err); ok {
		return shipError.Code
	}
	return ShipCommon
}

func shipRejected(code int, reason string) error { return ShipError{Code: code, Reason: reason} }

func (a *Account) PurchaseLocalProduct(r *Rules, product, count, kind, reference int, now int64) (map[model.ItemID]int64, model.LocalReceipt, error) {
	a.InitializeSystem()
	sku, known := r.System.Products[product]
	empty := model.LocalReceipt{}
	if a.System.NextReceipt == math.MaxInt64 {
		return nil, empty, shipRejected(ShipCommon, "local receipt limit reached")
	}
	if !known {
		return nil, empty, shipRejected(ShipProduct, "unknown local product")
	}
	if count != 1 || a.System.Products[product] > math.MaxInt32-count {
		return nil, empty, shipRejected(ShipCount, "invalid local product count")
	}
	values := map[model.ItemID]int64{}
	var err error
	calendar := now + r.Journey.CalendarOffset
	switch kind {
	case 1:
		if sku.Yuanbao > 0 && reference == 0 {
			values[3] = sku.Yuanbao + sku.Bonus
			break
		}
		id := model.ShopID(reference)
		rule, exists := r.System.Shops[id]
		if !exists {
			return nil, empty, shipRejected(ShipConfiguration, "unknown local shop product")
		}
		if a.Level < rule.Level {
			return nil, empty, shipRejected(ShipLevel, "local shop product requires a higher level")
		}
		expected := rule.Product
		if rule.DiscountProduct != 0 && calendar >= rule.DiscountStart && rule.DiscountEnd > calendar {
			expected = rule.DiscountProduct
		}
		if product != expected {
			return nil, empty, shipRejected(ShipProduct, "local shop product does not match sale")
		}
		if calendar < rule.Start || rule.End > 0 && calendar > rule.End {
			return nil, empty, shipRejected(ShipTime, "local shop sale is closed")
		}
		purchase := a.ShopPurchases[id]
		period := shopPeriod(rule.Period, now)
		bought := purchase.Count
		if rule.Period == 5 && rule.PeriodHours > 0 {
			window := a.System.ShopWindows[id]
			bought = window.Count
			if window.Ends <= now {
				bought = 0
			}
			period = 0
		}
		if rule.Period != 5 && purchase.Period != period {
			bought = 0
		}
		if rule.Limit > 0 && bought+count > rule.Limit {
			return nil, empty, shipRejected(ShipLimit, "local shop purchase limit reached")
		}
		if rule.Predecessor != 0 && a.ShopPurchases[rule.Predecessor].Total == 0 {
			return nil, empty, shipRejected(ShipPredecessor, "required shop purchase is missing")
		}
		values, err = rewardItems(r, rule.Reward, int64(count))
		if err != nil {
			return nil, empty, err
		}
		a.ShopPurchases[id] = model.ShopPurchase{Total: purchase.Total + count, Count: bought + count, Period: period, PeriodType: rule.Period}
		if rule.Period == 5 && rule.PeriodHours > 0 {
			duration := rule.PeriodHours * 3600
			remaining := duration - (calendar-rule.Start)%duration
			a.System.ShopWindows[id] = model.SystemShopWindow{Count: bought + count, Ends: now + remaining}
			purchase = a.ShopPurchases[id]
			purchase.PeriodType = 0
			a.ShopPurchases[id] = purchase
		}
	case 2:
		offers, exists := r.System.Passes[reference]
		offer, offered := offers[product]
		if !exists || !offered {
			return nil, empty, shipRejected(ShipProduct, "unknown local pass product")
		}
		if a.Journey.Pass.ID != reference {
			return nil, empty, shipRejected(ShipTime, "local pass is not active")
		}
		activation := offer.Kind
		switch activation {
		case 2:
			activation = 3
		case 3:
			activation = 2
		}
		if err = a.ActivateJourneyPass(r, activation, now); err != nil {
			return nil, empty, err
		}
		values, err = rewardItems(r, offer.Reward, 1)
		if err != nil {
			return nil, empty, err
		}
	case 3:
		if !sku.Monthly || reference != 0 {
			return nil, empty, shipRejected(ShipProduct, "invalid local monthly pass")
		}
		if err = a.ActivateMonthlyPass(r, now); err != nil {
			return nil, empty, err
		}
	case 4:
		rule, exists := r.System.Gifts[reference]
		expires, offered := a.System.Gifts[reference]
		if !exists || !offered || expires != 0 && expires <= now || a.System.FinishedGifts[reference] != 0 {
			return nil, empty, shipRejected(ShipGiftUnavailable, "local gift offer is unavailable")
		}
		if rule.Product != product {
			return nil, empty, shipRejected(ShipProduct, "local gift product does not match")
		}
		values, err = rewardItems(r, rule.Reward, 1)
		if err != nil {
			return nil, empty, err
		}
		delete(a.System.Gifts, reference)
		a.System.FinishedGifts[reference] = 1
	case 6:
		rule, exists := r.System.HeroPacks[reference]
		state := a.System.HeroPacks[reference]
		if !exists {
			return nil, empty, shipRejected(ShipConfiguration, "unknown local hero pack")
		}
		if product != rule.Product && product != rule.Discount {
			return nil, empty, shipRejected(ShipProduct, "local hero pack product does not match")
		}
		if state.Bought {
			return nil, empty, shipRejected(ShipLimit, "local hero pack is already bought")
		}
		values[rule.Item] = 1
		state.OnSale = now
		state.Bought = true
		a.System.HeroPacks[reference] = state
	default:
		return nil, empty, shipRejected(ShipSystem, "unsupported local purchase system")
	}
	if err = a.GrantItems(r, values); err != nil {
		return nil, empty, err
	}
	if kind == 2 && values[40] > 0 {
		rule := r.Journey.Passes[a.Journey.Pass.ID]
		a.Journey.Pass.Level = min(rule.MaximumLevel, a.Journey.Pass.Level+int(values[40]))
		if a.Journey.Pass.Level == rule.MaximumLevel {
			a.Journey.Pass.Experience = 0
		}
	}
	a.System.Products[product] += count
	a.System.NextReceipt++
	receipt := model.LocalReceipt{ID: a.System.NextReceipt, Product: product, System: kind, Reference: reference, Count: count, Created: now}
	a.System.Receipts = appendRecent(a.System.Receipts, receipt, 128)
	a.System.LastReceipt = &receipt
	return values, receipt, nil
}
