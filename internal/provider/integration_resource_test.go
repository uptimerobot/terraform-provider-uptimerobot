package provider

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestPagerDutyLocationFromAPI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		integration *client.Integration
		wantValue   string
		wantOK      bool
	}{
		{
			name: "custom_value2_preferred",
			integration: &client.Integration{
				CustomValue2: "EU",
				Location:     "us",
			},
			wantValue: "eu",
			wantOK:    true,
		},
		{
			name: "fallback_to_location_field",
			integration: &client.Integration{
				Location: "US",
			},
			wantValue: "us",
			wantOK:    true,
		},
		{
			name: "missing",
			integration: &client.Integration{
				CustomValue2: "",
				Location:     "",
			},
			wantValue: "",
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotValue, gotOK := pagerDutyLocationFromAPI(tt.integration)
			if gotOK != tt.wantOK {
				t.Fatalf("ok mismatch: got=%v want=%v", gotOK, tt.wantOK)
			}
			if gotValue != tt.wantValue {
				t.Fatalf("value mismatch: got=%q want=%q", gotValue, tt.wantValue)
			}
		})
	}
}

func TestIntegrationCreateLockKeyNormalizesAndHashesValue(t *testing.T) {
	t.Parallel()

	key := integrationCreateLockKey(" Mattermost ", " https://example.com/hook ")
	if key == "" {
		t.Fatal("expected non-empty lock key")
	}
	if key != integrationCreateLockKey("mattermost", "https://example.com/hook") {
		t.Fatalf("expected type and value whitespace/case normalization")
	}
	if key == integrationCreateLockKey("webhook", "https://example.com/hook") {
		t.Fatalf("expected integration type to contribute to lock key")
	}
	if strings.Contains(key, "example.com") {
		t.Fatalf("expected lock key to avoid storing the sensitive integration value")
	}
}

func TestLockIntegrationCreateSerializesSameKey(t *testing.T) {
	t.Parallel()

	key := "test-lock-" + t.Name()
	unlock, err := lockIntegrationCreate(context.Background(), key)
	if err != nil {
		t.Fatalf("first lock failed: %v", err)
	}

	acquired := make(chan struct{})
	releaseSecond := make(chan struct{})
	errCh := make(chan error, 1)
	go func() {
		unlockSecond, err := lockIntegrationCreate(context.Background(), key)
		if err != nil {
			errCh <- err
			return
		}
		close(acquired)
		<-releaseSecond
		unlockSecond()
		errCh <- nil
	}()

	select {
	case <-acquired:
		t.Fatal("second lock acquired before first lock was released")
	case err := <-errCh:
		t.Fatalf("second lock failed early: %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	unlock()

	select {
	case <-acquired:
	case err := <-errCh:
		t.Fatalf("second lock failed: %v", err)
	case <-time.After(time.Second):
		t.Fatal("second lock did not acquire after first lock was released")
	}

	close(releaseSecond)
	if err := <-errCh; err != nil {
		t.Fatalf("second lock returned error: %v", err)
	}
}

func TestPagerDutyAutoResolveFromAPI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		integration *client.Integration
		wantValue   bool
		wantOK      bool
	}{
		{
			name: "from_custom_value_true",
			integration: &client.Integration{
				CustomValue: "1",
			},
			wantValue: true,
			wantOK:    true,
		},
		{
			name: "from_custom_value_false",
			integration: &client.Integration{
				CustomValue: "false",
			},
			wantValue: false,
			wantOK:    true,
		},
		{
			name: "fallback_to_field",
			integration: &client.Integration{
				AutoResolve: true,
			},
			wantValue: true,
			wantOK:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotValue, gotOK := pagerDutyAutoResolveFromAPI(tt.integration)
			if gotOK != tt.wantOK {
				t.Fatalf("ok mismatch: got=%v want=%v", gotOK, tt.wantOK)
			}
			if gotValue != tt.wantValue {
				t.Fatalf("value mismatch: got=%v want=%v", gotValue, tt.wantValue)
			}
		})
	}
}

func TestStickyStringPreferPrevOnMismatch(t *testing.T) {
	t.Parallel()

	prev := types.StringValue("eu")
	got := stickyStringPreferPrevOnMismatch(prev, "us", nil)
	if got.IsNull() || got.IsUnknown() {
		t.Fatal("expected known value")
	}
	if got.ValueString() != "eu" {
		t.Fatalf("expected previous value to win on mismatch, got=%q", got.ValueString())
	}

	got = stickyStringPreferPrevOnMismatch(prev, "eu", nil)
	if got.ValueString() != "eu" {
		t.Fatalf("expected matching api value, got=%q", got.ValueString())
	}

	got = stickyStringPreferPrevOnMismatch(types.StringNull(), "us", nil)
	if got.ValueString() != "us" {
		t.Fatalf("expected api value when previous is null, got=%q", got.ValueString())
	}
}

func TestParseWebhookStateFields_PostValueString(t *testing.T) {
	t.Parallel()

	fields, err := parseWebhookStateFields(`{"postValue":"{\"b\":\"x\",\"a\":1}","sendJSON":"1","sendQuery":"0","sendPost":"0"}`)
	if err != nil {
		t.Fatalf("parseWebhookStateFields returned error: %v", err)
	}

	if fields.PostValue.IsNull() || fields.PostValue.ValueString() != `{"a":1,"b":"x"}` {
		t.Fatalf("expected normalized post_value, got %q", fields.PostValue.ValueString())
	}
	if !fields.PostValueKnown {
		t.Fatalf("expected postValue to be marked known")
	}
	if !fields.SendAsJSON.ValueBool() || !fields.SendAsJSONKnown {
		t.Fatalf("expected send_as_json to be true and known")
	}
	if fields.SendAsQueryString.ValueBool() || !fields.SendAsQueryKnown {
		t.Fatalf("expected send_as_query_string to be false and known")
	}
	if fields.SendAsPostParameters.ValueBool() || !fields.SendAsPostKnown {
		t.Fatalf("expected send_as_post_parameters to be false and known")
	}
}

func TestParseWebhookStateFields_PostValueObject(t *testing.T) {
	t.Parallel()

	fields, err := parseWebhookStateFields(`{"postValue":{"b":"x","a":1},"sendJSON":true,"sendQuery":false,"sendPost":0}`)
	if err != nil {
		t.Fatalf("parseWebhookStateFields returned error: %v", err)
	}

	if fields.PostValue.IsNull() || fields.PostValue.ValueString() != `{"a":1,"b":"x"}` {
		t.Fatalf("expected normalized post_value, got %q", fields.PostValue.ValueString())
	}
	if !fields.SendAsJSON.ValueBool() {
		t.Fatalf("expected send_as_json to be true")
	}
}

func TestWebhookStateKeepsPreviousValuesWhenAPIOmitsConfig(t *testing.T) {
	t.Parallel()

	prevBool := types.BoolValue(true)
	gotBool := webhookBoolState(types.BoolValue(false), false, prevBool, nil)
	if gotBool.IsNull() || !gotBool.ValueBool() {
		t.Fatalf("expected previous webhook bool value to be preserved")
	}

	topLevelTrue := true
	gotTopLevelBool := webhookBoolState(types.BoolValue(false), false, types.BoolNull(), &topLevelTrue)
	if gotTopLevelBool.IsNull() || !gotTopLevelBool.ValueBool() {
		t.Fatalf("expected top-level webhook bool value to be used")
	}

	topLevelFalse := false
	gotTopLevelFalse := webhookBoolState(types.BoolValue(true), false, prevBool, &topLevelFalse)
	if gotTopLevelFalse.IsNull() || gotTopLevelFalse.ValueBool() {
		t.Fatalf("expected explicit false from top-level webhook field to win")
	}

	gotKnownFalse := webhookBoolState(types.BoolValue(false), true, prevBool, &topLevelTrue)
	if gotKnownFalse.IsNull() || gotKnownFalse.ValueBool() {
		t.Fatalf("expected explicit false from webhook config to win")
	}

	prevPostValue := types.StringValue(`{"message":"existing"}`)
	gotPostValue, err := webhookPostValueState(types.StringNull(), false, prevPostValue, "")
	if err != nil {
		t.Fatalf("webhookPostValueState returned error: %v", err)
	}
	if gotPostValue.IsNull() || gotPostValue.ValueString() != prevPostValue.ValueString() {
		t.Fatalf("expected previous post_value to be preserved, got %q", gotPostValue.ValueString())
	}
}
