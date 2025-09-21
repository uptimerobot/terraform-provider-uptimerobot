package client

import (
	"encoding/json"
	"testing"
)

func TestCreatePSPRequest_Marshal_CustomSettingsEmptyObjects(t *testing.T) {
	req := &CreatePSPRequest{
		Name:           "my-psp",
		CustomSettings: &CustomSettings{}, // page/colors/features should be {}
	}

	out, err := req.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}

	// Decode back to map to introspect
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	cs, ok := m["customSettings"].(map[string]any)
	if !ok {
		t.Fatalf("customSettings missing or wrong type")
	}
	// All must exist and be objects
	for _, k := range []string{"page", "colors", "features"} {
		if _, ok := cs[k].(map[string]any); !ok {
			t.Fatalf("customSettings.%s must be an object, got %#v", k, cs[k])
		}
	}
}

func TestCreatePSPRequest_Marshal_CustomSettingsWithFeatures(t *testing.T) {
	req := &CreatePSPRequest{
		Name: "my-psp",
		CustomSettings: &CustomSettings{
			Features: &FeatureSettings{ShowBars: boolPtr(true)},
		},
	}

	raw, err := json.Marshal(req) // hits the custom MarshalJSON
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	cs := m["customSettings"].(map[string]any)

	// page/colors present as objects
	if _, ok := cs["page"].(map[string]any); !ok {
		t.Fatalf("page must be object")
	}
	if _, ok := cs["colors"].(map[string]any); !ok {
		t.Fatalf("colors must be object")
	}

	features := cs["features"].(map[string]any)
	if v, ok := features["showBars"].(bool); !ok || !v {
		t.Fatalf("features.showBars must be true, got %#v", features["showBars"])
	}
}

func TestUpdatePSPRequest_Marshal_CustomSettingsEmptyObjects(t *testing.T) {
	req := &UpdatePSPRequest{
		Name:           "my-psp",
		CustomSettings: &CustomSettings{},
	}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	cs := m["customSettings"].(map[string]any)
	for _, k := range []string{"page", "colors", "features"} {
		if _, ok := cs[k].(map[string]any); !ok {
			t.Fatalf("customSettings.%s must be an object, got %#v", k, cs[k])
		}
	}
}

func boolPtr(b bool) *bool { return &b }
