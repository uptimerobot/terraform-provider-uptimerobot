package client

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUpdateMonitorRequest_AssignedAlertContacts_JSON(t *testing.T) {
	type alias UpdateMonitorRequest

	// Case 1: nil pointer -> field omitted.
	req1 := alias{}
	b1, err := json.Marshal(req1)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b1), "assignedAlertContacts") {
		t.Fatalf("expected assignedAlertContacts to be omitted, got %s", b1)
	}

	// Case 2: pointer to empty slice -> field present as [].
	empty := []AlertContactRequest{}
	req2 := alias{
		AssignedAlertContacts: &empty,
	}
	b2, err := json.Marshal(req2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b2), `"assignedAlertContacts":[]`) {
		t.Fatalf("expected assignedAlertContacts to be encoded as empty array, got %s", b2)
	}
}

func TestMonitorRequests_EmptyURL_JSON(t *testing.T) {
	updateReq := UpdateMonitorRequest{
		Name:     "heartbeat",
		Type:     MonitorTypeHeartbeat,
		Interval: 300,
	}
	updateRaw, err := json.Marshal(updateReq)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(updateRaw), `"url"`) {
		t.Fatalf("expected empty update url to be omitted, got %s", updateRaw)
	}

	createReq := CreateMonitorRequest{
		Name:     "heartbeat",
		Type:     MonitorTypeHeartbeat,
		Interval: 300,
	}
	createRaw, err := json.Marshal(createReq)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(createRaw), `"url"`) {
		t.Fatalf("expected empty create url to be omitted, got %s", createRaw)
	}

	createReq.URL = "https://example.com/health"
	createRaw, err = json.Marshal(createReq)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(createRaw), `"url":"https://example.com/health"`) {
		t.Fatalf("expected non-empty create url to be encoded, got %s", createRaw)
	}
}

func TestCreateMonitorRequest_Config_JSON(t *testing.T) {
	req := CreateMonitorRequest{
		Name:     "dns-monitor",
		URL:      "example.com",
		Type:     MonitorTypeDNS,
		Interval: 300,
	}

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["config"]; ok {
		t.Fatalf("config should be omitted when nil, got %s", raw)
	}

	days := []int64{7, 30}
	req.Config = &MonitorConfig{
		SSLExpirationPeriodDays: &days,
	}
	raw, err = json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	m = map[string]any{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	cfg, ok := m["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected config object, got %#v", m["config"])
	}
	periods, ok := cfg["sslExpirationPeriodDays"].([]any)
	if !ok {
		t.Fatalf("expected sslExpirationPeriodDays array, got %#v", cfg["sslExpirationPeriodDays"])
	}
	if len(periods) != 2 {
		t.Fatalf("expected 2 sslExpirationPeriodDays values, got %d", len(periods))
	}
}

func TestCreateMonitorRequest_Config_APIAssertions_JSON(t *testing.T) {
	req := CreateMonitorRequest{
		Name:     "api-monitor",
		URL:      "https://example.com/api/health",
		Type:     MonitorTypeAPI,
		Interval: 300,
		Config: &MonitorConfig{
			APIAssertions: &APIMonitorAssertions{
				Logic: "AND",
				Checks: []APIMonitorAssertionCheck{
					{
						Property:   "$.status",
						Comparison: "equals",
						Target:     "ok",
					},
					{
						Property:   "$.count",
						Comparison: "greater_than",
						Target:     0,
					},
				},
			},
		},
	}

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	cfg, ok := m["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected config object, got %#v", m["config"])
	}
	assertions, ok := cfg["apiAssertions"].(map[string]any)
	if !ok {
		t.Fatalf("expected apiAssertions object, got %#v", cfg["apiAssertions"])
	}
	if got := assertions["logic"]; got != "AND" {
		t.Fatalf("expected logic=AND, got %#v", got)
	}
	checks, ok := assertions["checks"].([]any)
	if !ok || len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %#v", assertions["checks"])
	}
}

func TestCreateMonitorRequest_Config_UDP_JSON(t *testing.T) {
	packetLossThreshold := int64(50)
	payload := "ping"
	req := CreateMonitorRequest{
		Name:     "udp-monitor",
		URL:      "example.com",
		Type:     MonitorTypeUDP,
		Interval: 300,
		Port:     53,
		Config: &MonitorConfig{
			UDP: &UDPMonitorConfig{
				Payload:             &payload,
				PacketLossThreshold: &packetLossThreshold,
			},
		},
	}

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	cfg, ok := m["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected config object, got %#v", m["config"])
	}
	udp, ok := cfg["udp"].(map[string]any)
	if !ok {
		t.Fatalf("expected udp object, got %#v", cfg["udp"])
	}
	if got := udp["payload"]; got != "ping" {
		t.Fatalf("expected payload=ping, got %#v", got)
	}
	if got, ok := udp["packetLossThreshold"].(float64); !ok || int64(got) != 50 {
		t.Fatalf("expected packetLossThreshold=50, got %#v", udp["packetLossThreshold"])
	}
}

func TestMonitorRequest_RegionData_JSON(t *testing.T) {
	reqThresholds := map[string]int{
		"na": 3000,
		"eu": 5000,
	}
	req := CreateMonitorRequest{
		Name:     "multi-region",
		URL:      "https://example.com",
		Type:     MonitorTypeHTTP,
		Interval: 300,
		RegionData: &RegionDataRequest{
			Regions:    []string{"na", "eu"},
			Thresholds: &reqThresholds,
		},
	}

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["regionalData"]; ok {
		t.Fatalf("new region data should use regionData, got %s", raw)
	}
	regionData, ok := m["regionData"].(map[string]any)
	if !ok {
		t.Fatalf("expected regionData object, got %#v", m["regionData"])
	}
	regions, ok := regionData["REGION"].([]any)
	if !ok || len(regions) != 2 {
		t.Fatalf("expected two REGION values, got %#v", regionData["REGION"])
	}
	expectedRegions := []string{"na", "eu"}
	for i, want := range expectedRegions {
		got, ok := regions[i].(string)
		if !ok || got != want {
			t.Fatalf("expected REGION[%d]=%q, got %#v", i, want, regions[i])
		}
	}
	thresholds, ok := regionData["THRESHOLD"].(map[string]any)
	if !ok {
		t.Fatalf("expected THRESHOLD object, got %#v", regionData["THRESHOLD"])
	}
	if len(thresholds) != len(reqThresholds) {
		t.Fatalf("expected %d threshold entries, got %#v", len(reqThresholds), thresholds)
	}
	for region, want := range reqThresholds {
		rawThreshold, ok := thresholds[region].(float64)
		if !ok {
			t.Fatalf("expected %s threshold number, got %#v", region, thresholds[region])
		}
		if got := int(rawThreshold); got != want {
			t.Fatalf("expected %s threshold=%d, got %d", region, want, got)
		}
	}
	if _, ok := regionData["MANUAL_SELECTED"]; ok {
		t.Fatalf("expected MANUAL_SELECTED to be omitted when auto_select is unmanaged, got %s", raw)
	}
}

func TestMonitorRequest_RegionData_ManualSelected_JSON(t *testing.T) {
	for name, manualSelected := range map[string]bool{
		"auto":   false,
		"manual": true,
	} {
		t.Run(name, func(t *testing.T) {
			req := CreateMonitorRequest{
				Name:     "multi-region",
				URL:      "https://example.com",
				Type:     MonitorTypeHTTP,
				Interval: 300,
				RegionData: &RegionDataRequest{
					Regions:        []string{"na"},
					ManualSelected: &manualSelected,
				},
			}

			raw, err := json.Marshal(req)
			if err != nil {
				t.Fatal(err)
			}

			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatal(err)
			}
			regionData, ok := m["regionData"].(map[string]any)
			if !ok {
				t.Fatalf("expected regionData object, got %#v", m["regionData"])
			}
			got, ok := regionData["MANUAL_SELECTED"].(bool)
			if !ok {
				t.Fatalf("expected MANUAL_SELECTED bool, got %#v in %s", regionData["MANUAL_SELECTED"], raw)
			}
			if got != manualSelected {
				t.Fatalf("expected MANUAL_SELECTED=%t, got %t", manualSelected, got)
			}
		})
	}
}

func TestMonitorRequest_RegionData_EmptyThresholds_JSON(t *testing.T) {
	reqThresholds := map[string]int{}
	req := UpdateMonitorRequest{
		Name:     "multi-region",
		Type:     MonitorTypeHTTP,
		Interval: 300,
		RegionData: &RegionDataRequest{
			Regions:    []string{"na", "eu"},
			Thresholds: &reqThresholds,
		},
	}

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	regionData, ok := m["regionData"].(map[string]any)
	if !ok {
		t.Fatalf("expected regionData object, got %#v", m["regionData"])
	}
	thresholds, ok := regionData["THRESHOLD"].(map[string]any)
	if !ok {
		t.Fatalf("expected empty THRESHOLD object, got %#v in %s", regionData["THRESHOLD"], raw)
	}
	if len(thresholds) != 0 {
		t.Fatalf("expected empty THRESHOLD object, got %#v", thresholds)
	}
}

func TestMonitorRequest_LegacyRegionalData_JSON(t *testing.T) {
	req := CreateMonitorRequest{
		Name:         "single-region",
		URL:          "https://example.com",
		Type:         MonitorTypeHTTP,
		Interval:     300,
		RegionalData: "eu",
	}

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"regionalData":"eu"`) {
		t.Fatalf("expected legacy regionalData string, got %s", raw)
	}
	if strings.Contains(string(raw), `"regionData"`) {
		t.Fatalf("did not expect new regionData object, got %s", raw)
	}
}

func TestMaintenanceWindowRequest_AutoAddMonitors_JSON(t *testing.T) {
	createReq := CreateMaintenanceWindowRequest{
		Name:     "mw-create",
		Interval: "daily",
		Time:     "01:00:00",
		Duration: 30,
	}
	raw, err := json.Marshal(createReq)
	if err != nil {
		t.Fatal(err)
	}
	var createMap map[string]any
	if err := json.Unmarshal(raw, &createMap); err != nil {
		t.Fatal(err)
	}
	if _, ok := createMap["autoAddMonitors"]; ok {
		t.Fatalf("autoAddMonitors should be omitted when nil, got %s", raw)
	}

	f := false
	createReq.AutoAddMonitors = &f
	raw, err = json.Marshal(createReq)
	if err != nil {
		t.Fatal(err)
	}
	createMap = map[string]any{}
	if err := json.Unmarshal(raw, &createMap); err != nil {
		t.Fatal(err)
	}
	if v, ok := createMap["autoAddMonitors"].(bool); !ok || v {
		t.Fatalf("expected autoAddMonitors=false in create request, got %#v", createMap["autoAddMonitors"])
	}

	updateReq := UpdateMaintenanceWindowRequest{
		Name: "mw-update",
	}
	raw, err = json.Marshal(updateReq)
	if err != nil {
		t.Fatal(err)
	}
	var updateMap map[string]any
	if err := json.Unmarshal(raw, &updateMap); err != nil {
		t.Fatal(err)
	}
	if _, ok := updateMap["autoAddMonitors"]; ok {
		t.Fatalf("autoAddMonitors should be omitted in update when nil, got %s", raw)
	}

	tVal := true
	updateReq.AutoAddMonitors = &tVal
	raw, err = json.Marshal(updateReq)
	if err != nil {
		t.Fatal(err)
	}
	updateMap = map[string]any{}
	if err := json.Unmarshal(raw, &updateMap); err != nil {
		t.Fatal(err)
	}
	if v, ok := updateMap["autoAddMonitors"].(bool); !ok || !v {
		t.Fatalf("expected autoAddMonitors=true in update request, got %#v", updateMap["autoAddMonitors"])
	}
}

func TestCreatePSPRequest_Marshal_CustomSettingsEmptyObjects(t *testing.T) {
	req := &CreatePSPRequest{
		Name:           "my-psp",
		CustomSettings: &CustomSettings{}, // page/colors/features should be {}
	}

	out, err := req.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	cs, ok := m["customSettings"].(map[string]any)
	if !ok {
		t.Fatalf("customSettings missing or wrong type")
	}
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

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	cs, ok := m["customSettings"].(map[string]any)
	if !ok {
		t.Fatalf("customSettings should be an object, got: %T (%v)", m["customSettings"], m["customSettings"])
	}
	if _, ok := cs["page"].(map[string]any); !ok {
		t.Fatalf("page must be object")
	}
	if _, ok := cs["colors"].(map[string]any); !ok {
		t.Fatalf("colors must be object")
	}
	features, ok := cs["features"].(map[string]any)
	if !ok {
		t.Fatalf("customSettings.features should be an object, got: %T (%v)", cs["features"], cs["features"])
	}
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
	cs, ok := m["customSettings"].(map[string]any)
	if !ok {
		t.Fatalf("customSettings should be an object, got: %T (%v)", m["customSettings"], m["customSettings"])
	}
	for _, k := range []string{"page", "colors", "features"} {
		if _, ok := cs[k].(map[string]any); !ok {
			t.Fatalf("customSettings.%s must be an object, got %#v", k, cs[k])
		}
	}
}

func boolPtr(b bool) *bool { return &b }
