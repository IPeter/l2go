package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDefaultEnchant(t *testing.T) {
	e := Default().GameServers[0].Enchant
	if !e.DisableOverEnchanting || !e.OverEnchantProtection || e.Punishment() != PunishJail ||
		time.Duration(e.FloodEnchantItem) != 1500*time.Millisecond || len(e.EnchantBlackList) != 19 {
		t.Errorf("DefaultEnchant = %+v", e)
	}
	// The default list is a copy.
	e.EnchantBlackList[0] = 1
	if DefaultEnchantBlackList[0] != 7816 {
		t.Error("DefaultEnchant shares its black list")
	}
}

// A section that names some keys keeps the defaults of the others.
func TestParseEnchant(t *testing.T) {
	cfg, err := Parse([]byte(`{"gameservers": [{"id": 1, "enchant": {"overEnchantPunishment": "kick", "enchantBlackList": [57]}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	e := cfg.GameServers[0].Enchant
	want := DefaultEnchant()
	want.OverEnchantPunishment = "kick"
	want.EnchantBlackList = []int{57}
	if !reflect.DeepEqual(e, want) {
		t.Errorf("enchant = %+v, want %+v", e, want)
	}
	if e.Punishment() != PunishKick {
		t.Errorf("Punishment = %q", e.Punishment())
	}
	cfg, err = Parse([]byte(`{"gameservers": [{"id": 1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.GameServers[0].Enchant; !reflect.DeepEqual(got, DefaultEnchant()) {
		t.Errorf("enchant without the section = %+v", got)
	}
}

func TestValidateEnchant(t *testing.T) {
	cfg := Default()
	gs := &cfg.GameServers[0]
	gs.Enchant.EnchantBlackList = []int{0}
	gs.Enchant.OverEnchantPunishment = "PRISON"
	gs.Enchant.FloodEnchantItem = Duration(-time.Second)
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate passed a bad enchant section")
	}
	for _, want := range []string{"enchant.enchantBlackList", "enchant.overEnchantPunishment", "enchant.floodEnchantItem"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
	for _, p := range []string{"none", "Broadcast", "KICK", "kickban", "jail"} {
		cfg := Default()
		cfg.GameServers[0].Enchant.OverEnchantPunishment = p
		if err := cfg.Validate(); err != nil {
			t.Errorf("punishment %q: %v", p, err)
		}
	}
}
