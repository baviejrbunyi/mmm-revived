package helper

import (
	"reflect"
	"testing"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
)

func TestAvailableArtifactIDs(t *testing.T) {
	rules := &account.Rules{
		Artifacts: map[model.ArtifactID]account.ArtifactRule{
			5002: {Name: "B", Star: 5, Templates: []int{2}},
			5001: {Name: "A", Star: 3, Templates: []int{1}},
			5003: {Name: "Hidden", Star: 4, Templates: []int{3}, Unreleased: true},
			5004: {Name: "Low rarity", Star: 2, Templates: []int{1}},
			5005: {Name: "No template", Star: 4, Templates: []int{4}},
		},
		ArtifactTemplates: map[int]account.ArtifactTemplate{
			1: {},
			2: {},
		},
	}
	if got, want := availableArtifactIDs(rules), "5001, 5002"; got != want {
		t.Fatalf("availableArtifactIDs() = %q, want %q", got, want)
	}
	if got, want := artifactIDNames(rules), []string{"5001 - A", "5002 - B"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("artifactIDNames() = %q, want %q", got, want)
	}
}

func TestParseFeaturedBanners(t *testing.T) {
	pools := map[model.DrawPoolID]account.DrawPoolRule{
		2:  {},
		5:  {},
		12: {},
	}
	got, err := parseFeaturedBanners("5, 12", pools)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int32{5, 12}; !reflect.DeepEqual(got, want) {
		t.Fatalf("parseFeaturedBanners() = %v, want %v", got, want)
	}
	if got, err := parseFeaturedBanners("none", pools); err != nil || len(got) != 0 {
		t.Fatalf("parseFeaturedBanners(none) = %v, %v; want an empty list", got, err)
	}
	for _, input := range []string{"1", "5,5", "5,,12", "29"} {
		if _, err := parseFeaturedBanners(input, pools); err == nil {
			t.Errorf("parseFeaturedBanners(%q) unexpectedly succeeded", input)
		}
	}
}

func TestParseHelperBoolean(t *testing.T) {
	for _, input := range []string{"true", "on", "yes", "1"} {
		if got, err := parseHelperBoolean(input); err != nil || !got {
			t.Errorf("parseHelperBoolean(%q) = %t, %v; want true, nil", input, got, err)
		}
	}
	for _, input := range []string{"false", "off", "no", "0"} {
		if got, err := parseHelperBoolean(input); err != nil || got {
			t.Errorf("parseHelperBoolean(%q) = %t, %v; want false, nil", input, got, err)
		}
	}
	if _, err := parseHelperBoolean("maybe"); err == nil {
		t.Error("parseHelperBoolean(maybe) succeeded; want an error")
	}
}

func TestResolveHelperItem(t *testing.T) {
	rules := &account.Rules{
		ItemRules: map[model.ItemID]account.ItemRule{
			101095: {Name: "Dreamcore"},
			555401: {Name: "Spinning\u00a0Success", Type: 1015},
			555101: {Name: "Blessing of Vishanti"},
			913:    {Name: "Random Rare Artifact"},
			921:    {Name: "Random Rare Artifact"},
		},
	}
	for input, want := range map[string]model.ItemID{
		"dreamcore":                101095,
		"spinning_success":         555401,
		"555101":                   555101,
		"random_rare_artifact_913": 913,
	} {
		got, alternatives := resolveHelperItem(rules, input)
		if got != want || len(alternatives) != 0 {
			t.Errorf("resolveHelperItem(%q) = %d, %v; want %d", input, got, alternatives, want)
		}
	}
	if got, alternatives := resolveHelperItem(rules, "random_rare_artifact"); got != 0 || !reflect.DeepEqual(alternatives, []string{"random_rare_artifact_913", "random_rare_artifact_921"}) {
		t.Errorf("ambiguous resolveHelperItem() = %d, %v", got, alternatives)
	}
	if got, alternatives := resolveHelperItem(rules, "missing_item"); got != 0 || len(alternatives) != 0 {
		t.Errorf("unknown resolveHelperItem() = %d, %v", got, alternatives)
	}
	if got, alternatives := resolveHelperItem(rules, "999999"); got != 0 || len(alternatives) != 0 {
		t.Errorf("unknown numeric resolveHelperItem() = %d, %v", got, alternatives)
	}
}

func TestHelperItemSlug(t *testing.T) {
	for input, want := range map[string]string{
		"Dreamcore":             "dreamcore",
		"Spinning\u00a0Success": "spinning_success",
		"SealingSpell":          "sealing_spell",
		"!Hero EXP":             "hero_exp",
	} {
		if got := helperItemSlug(input); got != want {
			t.Errorf("helperItemSlug(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestGrantHelperItemOnlyChangesTargetAccount(t *testing.T) {
	rules := &account.Rules{ItemRules: map[model.ItemID]account.ItemRule{101095: {Type: 1000, Name: "Dreamcore"}}}
	sender := &account.Account{Items: map[model.ItemID]int64{}}
	other := &account.Account{Items: map[model.ItemID]int64{}}
	if err := sender.GrantItems(rules, map[model.ItemID]int64{101095: 999}); err != nil {
		t.Fatalf("GrantItems() error = %v", err)
	}
	if got := sender.Items[101095]; got != 999 {
		t.Errorf("sender Dreamcore = %d, want 999", got)
	}
	if got := other.Items[101095]; got != 0 {
		t.Errorf("other player's Dreamcore = %d, want 0", got)
	}
}
