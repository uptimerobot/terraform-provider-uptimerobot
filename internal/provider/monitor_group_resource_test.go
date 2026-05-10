package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestMonitorGroupResource_Metadata(t *testing.T) {
	t.Parallel()

	r := NewMonitorGroupResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "uptimerobot"}, resp)

	if resp.TypeName != "uptimerobot_monitor_group" {
		t.Fatalf("unexpected type name %q", resp.TypeName)
	}
}

func TestMonitorGroupResource_Schema(t *testing.T) {
	t.Parallel()

	r := NewMonitorGroupResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	for _, name := range []string{"id", "name", "monitors_new_group_id", "created_at", "updated_at"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Fatalf("expected schema attribute %q", name)
		}
	}
}

func TestMonitorGroupResourceModelApplyAPI(t *testing.T) {
	t.Parallel()

	model := monitorGroupResourceModel{
		MonitorsNewGroupID: types.Int64Value(123),
	}
	model.applyAPI(&client.MonitorGroup{
		ID:        456,
		Name:      "Production",
		CreatedAt: "2026-05-10T10:00:00.000Z",
		UpdatedAt: "2026-05-10T10:01:00.000Z",
	})

	if model.ID.ValueString() != "456" {
		t.Fatalf("unexpected ID %q", model.ID.ValueString())
	}
	if model.Name.ValueString() != "Production" {
		t.Fatalf("unexpected name %q", model.Name.ValueString())
	}
	if model.MonitorsNewGroupID.ValueInt64() != 123 {
		t.Fatalf("delete target was not preserved")
	}
}

func TestMonitorGroupResourceModelIntID(t *testing.T) {
	t.Parallel()

	model := monitorGroupResourceModel{ID: types.StringValue("789")}
	id, err := model.intID()
	if err != nil {
		t.Fatalf("intID returned error: %v", err)
	}
	if id != 789 {
		t.Fatalf("expected 789, got %d", id)
	}

	model.ID = types.StringValue("not-an-int")
	if _, err := model.intID(); err == nil {
		t.Fatal("expected invalid ID error")
	}
}
