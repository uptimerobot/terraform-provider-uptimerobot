package client

import (
	"encoding/json"
	"testing"
)

func TestBoolOrString_Unmarshal_Bool(t *testing.T) {
	var b BoolOrString
	in := []byte(`true`)
	if err := json.Unmarshal(in, &b); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Val == nil || *b.Val != true {
		t.Fatalf("expected true, got %#v", b.Val)
	}
}

func TestBoolOrString_Unmarshal_StringTrue(t *testing.T) {
	var b BoolOrString
	in := []byte(`"true"`)
	if err := json.Unmarshal(in, &b); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Val == nil || *b.Val != true {
		t.Fatalf("expected true, got %#v", b.Val)
	}
}

func TestBoolOrString_Unmarshal_StringFalse(t *testing.T) {
	var b BoolOrString
	in := []byte(`"FALSE"`) // case-insensitive
	if err := json.Unmarshal(in, &b); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Val == nil || *b.Val != false {
		t.Fatalf("expected false, got %#v", b.Val)
	}
}

func TestBoolOrString_Unmarshal_Null(t *testing.T) {
	var b BoolOrString
	in := []byte(`null`)
	if err := json.Unmarshal(in, &b); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Val != nil {
		t.Fatalf("expected nil, got %#v", b.Val)
	}
}

func TestBoolOrString_Unmarshal_Invalid(t *testing.T) {
	var b BoolOrString
	in := []byte(`"yes"`)
	if err := json.Unmarshal(in, &b); err == nil {
		t.Fatalf("expected error, got none (val=%#v)", b.Val)
	}
}
