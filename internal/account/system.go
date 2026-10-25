package account

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"mmm-revived/internal/model"
)

func (a *Account) InitializeSystem() {
	s := &a.System
	if s.ShopWindows == nil {
		s.ShopWindows = map[model.ShopID]model.SystemShopWindow{}
	}
	if s.Diagnostics == nil {
		s.Diagnostics = map[model.BattleID]model.LocalBattleDiagnostic{}
	}
	if s.Settings == nil {
		s.Settings = map[string]model.SettingValue{}
	}
	if s.Mirage == nil {
		s.Mirage = map[model.ActivityID]int64{}
	}
	if s.Products == nil {
		s.Products = map[int]int{}
	}
	if s.Gifts == nil {
		s.Gifts = map[int]int64{}
	}
	if s.FinishedGifts == nil {
		s.FinishedGifts = map[int]int{}
	}
	if s.HeroPacks == nil {
		s.HeroPacks = map[int]model.SystemHeroPack{}
	}
	if s.PushTags == nil {
		s.PushTags = []string{}
	}
	if s.Receipts == nil {
		s.Receipts = []model.LocalReceipt{}
	}
}

func (a *Account) ApplySystemSnapshot(r *Rules, profile *model.Avatar, now int64) {
	s := &a.System
	if s.ProtectionHash != "" && !a.protectionExpired(now) && r != nil {
		profile.Basic.ProtectionCode = s.ProtectionPIN
		if s.ProtectionReset != 0 {
			profile.Basic.ProtectionTime = r.Settings.ProtectionForget
			profile.Basic.ProtectionUpdated = s.ProtectionReset - r.Settings.ProtectionForget
		}
	}
	profile.Questions.Questions = s.QuestionData
	profile.Purchases.ReturnedPayment = s.ReturnedPayment
	profile.Purchases.FirstPayment = map[int]map[int]int{1: {}}
	maps.Copy(profile.Purchases.FirstPayment[1], s.Products)
	profile.Purchases.TotalPrice = map[model.Currency]int64{}
	profile.Purchases.TotalPriceCeil = map[model.Currency]int64{}
	profile.Purchases.FreePrice = map[model.Currency]int64{}
	profile.Purchases.FreePriceCeil = map[model.Currency]int64{}
	if len(s.Receipts) > 0 {
		profile.Purchases.FirstPurchase = s.Receipts[0].Created
	}
	profile.Purchases.LastPurchase = []int64{}
	if receipt := s.LastReceipt; receipt != nil {
		profile.Purchases.LastPurchase = []int64{int64(receipt.Product), int64(receipt.System), int64(receipt.Reference)}
	}
	profile.PopupPacks.Current = map[int]int64{}
	profile.PopupPacks.Finished = []int{}
	for _, id := range slices.Sorted(maps.Keys(s.FinishedGifts)) {
		if s.FinishedGifts[id] != 0 {
			profile.PopupPacks.Finished = append(profile.PopupPacks.Finished, id)
		}
	}
	for id, expires := range s.Gifts {
		if expires == 0 || expires > now {
			profile.PopupPacks.Current[id] = expires
			profile.PopupPacks.ClientID = max(profile.PopupPacks.ClientID, id)
		}
	}
	profile.HeroPacks.Packs = s.HeroPacks
	for id, window := range s.ShopWindows {
		count := 0
		if window.Ends > now {
			count = window.Count
		}
		profile.Shop.Counts[id] = count
	}
}

func (a *Account) ChangeSystemSetting(key string, value model.SettingValue) error {
	if len(key) < 1 || len(key) > 64 || !utf8.ValidString(key) || strings.ContainsAny(key, "\x00\r\n") || value.Kind > model.SettingBoolean || value.Kind == model.SettingText && (len(value.Text) > 1024 || !utf8.ValidString(value.Text)) || value.Kind == model.SettingNumber && (math.IsNaN(value.Number) || math.IsInf(value.Number, 0) || math.Abs(value.Number) > 1e12) {
		return rejected("invalid game setting")
	}
	if _, exists := a.System.Settings[key]; !exists && len(a.System.Settings) >= 128 {
		return rejected("too many game settings")
	}
	a.System.Settings[key] = value
	return nil
}

func (a *Account) SetLobbyHero(r *Rules, hero model.HeroID, skin model.SkinID, hidden []string) error {
	if hero == 0 {
		hero = a.Profile.Lobby.Hero
	}
	value, owned := a.Heroes[hero]
	if !owned || value.Trial != 0 || len(hidden) > 32 {
		return rejected("lobby hero is not owned")
	}
	hiddenSkins := make(map[model.SkinID]bool, len(hidden))
	hiddenPools := make(map[int64]bool, len(hidden))
	for _, encoded := range hidden {
		if encoded == "" {
			continue
		}
		parts := strings.Split(encoded, "_")
		if len(parts) != 5 {
			return rejected("invalid lobby hero preference")
		}
		var numbers [5]int64
		for index, part := range parts {
			if index == 2 && part == "nil" {
				continue
			}
			number, err := strconv.ParseInt(part, 10, 64)
			if err != nil || number < 0 {
				return rejected("invalid lobby hero preference")
			}
			numbers[index] = number
		}
		if numbers[0] > math.MaxInt32 || numbers[1] > math.MaxInt32 || numbers[2] > math.MaxInt32 || hiddenPools[numbers[0]] {
			return rejected("invalid lobby preference identity")
		}
		hiddenPools[numbers[0]] = true
		if _, known := r.Draw.Pools[model.DrawPoolID(numbers[0])]; !known {
			return rejected("unknown lobby draw preference")
		}
		preferred, exists := a.Heroes[model.HeroID(numbers[1])]
		if !exists || preferred.Trial != 0 || numbers[4] > 1 {
			return rejected("preferred lobby hero is unavailable")
		}
		selectedSkin := model.SkinID(numbers[2])
		if selectedSkin != 0 && r.Skins[selectedSkin] != model.HeroID(numbers[1]) {
			return rejected("preferred lobby skin does not match hero")
		}
		hiddenSkins[selectedSkin] = true
	}
	if skin != 0 && r.Skins[skin] != hero {
		if !hiddenSkins[skin] {
			return rejected("lobby skin does not match hero")
		}
		skin = a.Profile.Lobby.Skin
	}
	a.Profile.Lobby = model.LobbyHero{Hero: hero, Skin: skin}
	a.Profile.HiddenIDs = slices.Clone(hidden)
	return nil
}

func (a *Account) TriggerHeroDialog(r *Rules, now int64) bool {
	if a.System.LastDialog == now {
		return false
	}
	a.System.LastDialog = now
	a.System.DialogCount = min(math.MaxInt32, a.System.DialogCount+1)
	changed := false
	for id, rule := range r.Tasks {
		if rule.Type != 77 {
			continue
		}
		task, exists := a.Tasks[id]
		if !exists || task.Status < 2 || task.Status >= 4 {
			continue
		}
		a.UpdateTask(id, min(rule.Target, task.Value+1), rule.Target)
		changed = changed || a.Tasks[id] != task
	}
	return changed
}

func (a *Account) MirageInteract(r *Rules, id model.ActivityID, now int64) (bool, error) {
	if !slices.Contains(r.System.MirageActivities, id) {
		return false, rejected("unknown mirage activity")
	}
	if a.System.Mirage[id] == 0 {
		a.System.Mirage[id] = now
	}
	value, exists := a.Journey.Activities[id]
	if !exists || value.Interacted == 1 {
		return false, nil
	}
	value.Interacted = 1
	a.Journey.Activities[id] = value
	return true, nil
}

func validProtectionPIN(pin string) bool {
	if len(pin) != 4 {
		return false
	}
	for _, value := range pin {
		if value < '0' || value > '9' {
			return false
		}
	}
	return true
}

func protectionDigest(salt, pin string) string {
	value := sha256.Sum256([]byte(salt + ":" + pin))
	return hex.EncodeToString(value[:])
}

func (a *Account) protectionExpired(now int64) bool {
	return a.System.ProtectionHash != "" && a.System.ProtectionReset != 0 && now >= a.System.ProtectionReset
}

func (a *Account) expireProtection(now int64) {
	if a.protectionExpired(now) {
		a.clearProtection()
	}
}

func (a *Account) SetProtectionPIN(_ *Rules, pin string, now int64) error {
	a.expireProtection(now)
	if !validProtectionPIN(pin) || a.System.ProtectionHash != "" {
		return rejected("protection PIN cannot be set")
	}
	var salt [16]byte
	rand.Read(salt[:])
	a.System.ProtectionSalt = hex.EncodeToString(salt[:])
	a.System.ProtectionHash = protectionDigest(a.System.ProtectionSalt, pin)
	a.System.ProtectionPIN = pin
	a.System.ProtectionReset = 0
	return nil
}

func (a *Account) clearProtection() {
	a.System.ProtectionHash, a.System.ProtectionSalt, a.System.ProtectionReset, a.System.ProtectionPIN = "", "", 0, ""
}

func (a *Account) UnsetProtectionPIN(pin string, now int64) error {
	a.expireProtection(now)
	if !validProtectionPIN(pin) || a.System.ProtectionHash == "" || subtle.ConstantTimeCompare([]byte(a.System.ProtectionHash), []byte(protectionDigest(a.System.ProtectionSalt, pin))) != 1 {
		return rejected("incorrect protection PIN")
	}
	a.clearProtection()
	return nil
}

func (a *Account) ForgetProtectionPIN(r *Rules, now int64) bool {
	a.expireProtection(now)
	if a.System.ProtectionHash == "" || a.System.ProtectionReset != 0 {
		return false
	}
	a.System.ProtectionReset = now + r.Settings.ProtectionForget
	return true
}

func (a *Account) OfferLocalGift(r *Rules, id int, now int64) error {
	rule, known := r.System.Gifts[id]
	if !known || rule.Product == 0 || rule.Reward == 0 || a.System.FinishedGifts[id] != 0 {
		return rejected("gift offer is unavailable")
	}
	if _, exists := a.System.Gifts[id]; exists {
		return nil
	}
	expires := int64(0)
	if rule.Duration > 0 {
		expires = now + rule.Duration
	}
	a.System.Gifts[id] = expires
	return nil
}

func (a *Account) giftConditionMet(conditions []SystemGiftCondition) bool {
	for _, condition := range conditions {
		switch {
		case condition.Level > 0 && a.Level >= condition.Level:
			return true
		case condition.Gift != 0 && a.System.FinishedGifts[condition.Gift] != 0:
			return true
		case condition.Activity != 0 && a.Journey.Activities[condition.Activity].Open:
			return true
		}
	}
	return false
}

func (a *Account) OfferTriggeredGifts(r *Rules, now int64) bool {
	if r == nil {
		return false
	}
	changed := false
	for _, id := range slices.Sorted(maps.Keys(r.System.Gifts)) {
		rule := r.System.Gifts[id]
		if rule.Product == 0 || rule.Reward == 0 || a.System.FinishedGifts[id] != 0 {
			continue
		}
		if expires, offered := a.System.Gifts[id]; offered && (expires == 0 || expires > now) {
			continue
		}
		offers := a.System.GiftOffers[id]
		triggered := a.giftConditionMet(rule.Triggers)
		if offers > 0 {
			triggered = r.System.ReofferGifts && triggered || offers == 1 && a.giftConditionMet(rule.Retriggers)
		}
		if !triggered {
			continue
		}
		a.InitializeSystem()
		if a.System.GiftOffers == nil {
			a.System.GiftOffers = map[int]int{}
		}
		expires := int64(0)
		if rule.Duration > 0 {
			expires = now + rule.Duration
		}
		a.System.Gifts[id] = expires
		a.System.GiftOffers[id] = min(math.MaxInt32, offers+1)
		changed = true
	}
	if changed {
		a.System.GiftsPending = true
	}
	return changed
}
