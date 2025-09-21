// provider/upgrader_helpers_test.go
package provider

import "testing"

func TestUpgradeFeaturesMap(t *testing.T) {
	old := map[string]any{
		"show_bars":           "true",
		"show_monitor_url":    "0",
		"enable_details_page": "YES",
		"garbage":             "maybe",
		"empty":               "",
	}
	got := upgradeFeaturesMap(old)
	if got["show_bars"] != true {
		t.Fatalf("show_bars expected true, got %#v", got["show_bars"])
	}
	if got["show_monitor_url"] != false {
		t.Fatalf("show_monitor_url expected false, got %#v", got["show_monitor_url"])
	}
	if _, ok := got["enable_details_page"]; ok {
		t.Fatalf("enable_details_page should be dropped for non-ParseBool strings")
	}
	if _, ok := got["garbage"]; ok {
		t.Fatalf("garbage should be dropped")
	}
	if _, ok := got["empty"]; ok {
		t.Fatalf("empty should be dropped")
	}
}
