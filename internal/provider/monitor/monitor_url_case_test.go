package monitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestNormalizeMonitorURLCase(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ raw, want string }{
		{"HTTPS://example.com", "https://example.com"},
		{"https://EXAMPLE.com", "https://example.com"},
		{"Https://Example.COM:8443/MyPath?Token=AbC#Frag", "https://example.com:8443/MyPath?Token=AbC#Frag"},
		{"HTTPS://User:Pass@Host.COM/p", "https://User:Pass@host.com/p"},
		{"HTTPS://User@Pass@Host.COM/p", "https://User@Pass@host.com/p"},
		{"HTTPS://[2001:DB8::A]:8443/x", "https://[2001:db8::a]:8443/x"},
		{"HTTPS://ΟΣ-A.COM/Ä", "https://ΟΣ-a.com/Ä"},
		{`https://EVIL.com\@GOOD.com/Path`, `https://evil.com\@GOOD.com/Path`},
		{"HTTP://Example.com/path?user=A@B#Frag", "http://example.com/path?user=A@B#Frag"},
		{"https://EXAMPLE.com/%2F?Q=%3A\nRest", "https://example.com/%2F?Q=%3A\nRest"},
		{"WS+X.Y-Z://EXAMPLE.COM:ABC/Path", "ws+x.y-z://example.com:ABC/Path"},
		{"Example.COM", "Example.COM"},
		{"//Example.COM/Path", "//Example.COM/Path"},
		{"192.168.1.1", "192.168.1.1"},
		{"[2001:DB8::1]", "[2001:DB8::1]"},
		{"https://example.com/Path?Q=1", "https://example.com/Path?Q=1"},
		{"", ""},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			t.Parallel()
			if got := normalizeMonitorURLCase(tc.raw); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMonitorURLsEquivalentForState_Case(t *testing.T) {
	t.Parallel()
	for _, monitorType := range []string{MonitorTypeHTTP, MonitorTypeKEYWORD, MonitorTypeAPI, MonitorTypePING, MonitorTypePORT, MonitorTypeUDP, MonitorTypeDNS, MonitorTypeHEARTBEAT} {
		t.Run(monitorType, func(t *testing.T) {
			t.Parallel()
			configured := "HTTPS://EXAMPLE.com/Path?Token=AbC"
			apiURL := "https://example.com/Path?Token=AbC"
			if !monitorURLsEquivalentForState(monitorType, configured, apiURL) {
				t.Fatal("scheme and host casing should be equivalent")
			}
			if !monitorURLForState(monitorType, types.StringValue(configured), apiURL).Equal(types.StringValue(configured)) {
				t.Fatal("state should preserve configured spelling")
			}
			want := monComparable{Type: &monitorType, URL: &configured}
			got := monComparable{Type: &monitorType, URL: &apiURL}
			if !equalComparable(want, got) || len(fieldsStillDifferent(want, got)) != 0 {
				t.Fatal("equivalent URL should settle without differences")
			}
		})
	}
	for _, tc := range []struct {
		name, monitorType, configured, api string
		equal                              bool
	}{
		{"userinfo", MonitorTypeHTTP, "https://User:Pass@EXAMPLE.com/", "https://user:pass@example.com/", false},
		{"path", MonitorTypeHTTP, "https://EXAMPLE.com/Path", "https://example.com/path", false},
		{"query", MonitorTypeHTTP, "https://EXAMPLE.com/?Key=Value", "https://example.com/?key=value", false},
		{"fragment", MonitorTypeHTTP, "https://EXAMPLE.com/#Frag", "https://example.com/#frag", false},
		{"unicode host", MonitorTypeHTTP, "https://ΟΣ.COM/", "https://οσ.com/", false},
		{"port", MonitorTypeHTTP, "https://EXAMPLE.com:443/", "https://example.com/", false},
		{"escaping", MonitorTypeHTTP, "https://EXAMPLE.com/%2F", "https://example.com/%2f", false},
		{"scheme change", MonitorTypeHTTP, "https://EXAMPLE.com/", "http://example.com/", false},
		{"bare host unchanged", MonitorTypePING, "EXAMPLE.com", "example.com", false},
		{"ping scheme", MonitorTypePING, "HTTPS://EXAMPLE.com/Path", "example.com/Path", true},
		{"port scheme", MonitorTypePORT, "HTTPS://[2001:DB8::1]:443/Path", "[2001:db8::1]:443/Path", true},
		{"ping legacy host", MonitorTypePING, "HTTPS://EXAMPLE.com/Health", "EXAMPLE.com/Health", true},
		{"ping legacy mixed host", MonitorTypePING, "https://example.com/Health", "ExAmPlE.com/Health", true},
		{"port legacy host", MonitorTypePORT, "HTTPS://EXAMPLE.com/Health", "EXAMPLE.com/Health", true},
		{"port bare host unchanged", MonitorTypePORT, "EXAMPLE.com/Health", "example.com/Health", false},
		{"ping unicode host unchanged", MonitorTypePING, "HTTPS://ΟΣ.COM/Health", "οσ.com/Health", false},
		{"port legacy path", MonitorTypePORT, "HTTPS://EXAMPLE.com/Health", "EXAMPLE.com/health", false},
		{"ping path", MonitorTypePING, "HTTPS://EXAMPLE.com/Path", "example.com/path", false},
		{"http requires scheme", MonitorTypeHTTP, "HTTPS://EXAMPLE.com/Path", "example.com/Path", false},
		{"udp scheme unchanged", MonitorTypeUDP, "HTTPS://EXAMPLE.com/Path", "example.com/Path", false},
		{"dns scheme unchanged", MonitorTypeDNS, "HTTPS://EXAMPLE.com/Path", "example.com/Path", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := monitorURLsEquivalentForState(tc.monitorType, tc.configured, tc.api); got != tc.equal {
				t.Fatalf("equivalence = %v, want %v", got, tc.equal)
			}
			if got := monitorURLsEquivalentForState(tc.monitorType, tc.api, tc.configured); got != tc.equal {
				t.Fatal("equivalence must be symmetric")
			}
			if tc.equal && monitorURLForState(tc.monitorType, types.StringValue(tc.configured), tc.api).ValueString() != tc.configured {
				t.Fatal("equivalent URL should preserve configured spelling in state")
			}
		})
	}
	for _, configured := range []types.String{types.StringNull(), types.StringUnknown()} {
		if got := monitorURLForState(MonitorTypeHTTP, configured, "https://example.com/Path"); got.ValueString() != "https://example.com/Path" {
			t.Fatal("unknown or imported URL should use API spelling")
		}
	}
	if got := monitorURLForState(MonitorTypeHTTP, types.StringValue("HTTPS://EXAMPLE.com/Old"), "https://example.com/New"); got.ValueString() != "https://example.com/New" {
		t.Fatal("meaningful drift must be reflected in state")
	}
}

func TestWarnMonitorURLCase(t *testing.T) {
	t.Parallel()
	for _, value := range []types.String{types.StringNull(), types.StringUnknown(), types.StringValue("https://example.com/Path?Q=ABC"), types.StringValue("EXAMPLE.com"), types.StringValue("https://ΟΣ.com")} {
		var diagnostics diag.Diagnostics
		warnMonitorURLCase(value, &diagnostics)
		if len(diagnostics) != 0 {
			t.Fatalf("unexpected diagnostic for %s: %v", value, diagnostics)
		}
	}
	var diagnostics diag.Diagnostics
	warnMonitorURLCase(types.StringValue("HTTPS://User:Secret@EXAMPLE.com/Private?Token=Secret#Fragment"), &diagnostics)
	if diagnostics.HasError() || diagnostics.WarningsCount() != 1 {
		t.Fatalf("expected one warning, got %v", diagnostics)
	}
	detail := diagnostics[0].Detail()
	if !strings.Contains(detail, "https://example.com") {
		t.Fatalf("missing normalized scheme and host: %s", detail)
	}
	for _, secret := range []string{"User", "Secret", "Private", "Token", "Fragment"} {
		if strings.Contains(detail, secret) {
			t.Fatalf("warning disclosed URL component %q", secret)
		}
	}
}

func TestWarnMonitorURLCase_MalformedBracketedHost(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"HTTPS://[::1/?token=Secret]",
		"HTTPS://[::1#Secret]",
		`HTTPS://[::1\Secret]`,
		"HTTPS://[::1@Secret]",
	} {
		var diagnostics diag.Diagnostics
		warnMonitorURLCase(types.StringValue(raw), &diagnostics)
		if diagnostics.HasError() || diagnostics.WarningsCount() != 1 {
			t.Fatalf("expected one warning for %q, got %v", raw, diagnostics)
		}
		if strings.Contains(strings.ToLower(diagnostics[0].Detail()), "secret") {
			t.Fatalf("warning disclosed malformed URL contents: %s", diagnostics[0].Detail())
		}
	}
}

func TestWarnMonitorURLCase_IPv6(t *testing.T) {
	t.Parallel()
	var diagnostics diag.Diagnostics
	warnMonitorURLCase(types.StringValue("HTTPS://[2001:DB8::1]:8443/Private?Token=Secret"), &diagnostics)
	if diagnostics.HasError() || diagnostics.WarningsCount() != 1 {
		t.Fatalf("expected one IPv6 warning, got %v", diagnostics)
	}
	detail := diagnostics[0].Detail()
	if !strings.Contains(detail, "https://[2001:db8::1]") || strings.Contains(detail, "Secret") || strings.Contains(detail, "Private") {
		t.Fatalf("unexpected IPv6 warning: %s", detail)
	}
}

func TestMonitorModifyPlan_WarnsWithoutChangingURL(t *testing.T) {
	t.Parallel()
	r := &monitorResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(t.Context(), resource.SchemaRequest{}, &schemaResp)
	const configured = "HTTPS://EXAMPLE.com/Health"
	state := urlCaseMonitorState(t, schemaResp.Schema, configured)
	plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: state.Raw}
	resp := resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(t.Context(), resource.ModifyPlanRequest{
		Plan: plan, State: state,
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: state.Raw},
	}, &resp)
	if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("expected one plan warning, got %v", resp.Diagnostics)
	}
	var got types.String
	if diags := resp.Plan.GetAttribute(t.Context(), path.Root("url"), &got); diags.HasError() {
		t.Fatal(diags)
	}
	if got.ValueString() != configured {
		t.Fatalf("planned URL = %s, want %q", got, configured)
	}
}

func TestMonitorUpdate_URLCase(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	const configured = "HTTPS://EXAMPLE.com/Health?Token=AbC"
	const stored = "https://example.com/Health?Token=AbC"
	var mu sync.Mutex
	var sent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch req.Method {
		case http.MethodPatch:
			var update client.UpdateMonitorRequest
			if err := json.NewDecoder(req.Body).Decode(&update); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			sent = update.URL
		case http.MethodGet:
		default:
			t.Errorf("unexpected method %s", req.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := json.NewEncoder(w).Encode(client.Monitor{ID: 42, Type: MonitorTypeHTTP, Name: "URL case", URL: stored, Interval: 300, Timeout: 30}); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()
	api := client.NewClient("test-key")
	api.SetBaseURL(srv.URL)
	r := &monitorResource{client: api}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	plan := urlCaseMonitorState(t, schemaResp.Schema, configured)
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Update(ctx, resource.UpdateRequest{
		Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: plan.Raw},
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: plan.Raw},
		State:  urlCaseMonitorState(t, schemaResp.Schema, stored),
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", resp.Diagnostics)
	}
	var got types.String
	if diags := resp.State.GetAttribute(ctx, path.Root("url"), &got); diags.HasError() {
		t.Fatal(diags)
	}
	mu.Lock()
	defer mu.Unlock()
	if sent != configured || got.ValueString() != configured {
		t.Fatalf("sent %q and stored %s, want configured %q for both", sent, got, configured)
	}
}

func urlCaseMonitorState(t *testing.T, s schema.Schema, url string) tfsdk.State {
	t.Helper()
	objType, ok := s.Type().TerraformType(t.Context()).(tftypes.Object)
	if !ok {
		t.Fatal("monitor schema must be an object")
	}
	values := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	for name, value := range map[string]any{
		"id": "42", "type": MonitorTypeHTTP, "name": "URL case", "url": url,
		"interval": int64(300), "timeout": int64(30),
	} {
		values[name] = tftypes.NewValue(objType.AttributeTypes[name], value)
	}
	return tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, values)}
}
