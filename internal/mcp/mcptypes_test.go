package mcp

import (
	"testing"
)

func TestMCPTypes_RequireString(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		key     string
		want    string
		wantErr bool
	}{
		{
			name:    "missing key returns error",
			args:    map[string]any{},
			key:     "name",
			want:    "",
			wantErr: true,
		},
		{
			name:    "non-string value returns error",
			args:    map[string]any{"name": 123},
			key:     "name",
			want:    "",
			wantErr: true,
		},
		{
			name:    "valid string returns value",
			args:    map[string]any{"name": "alice"},
			key:     "name",
			want:    "alice",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := CallToolRequest{Arguments: tt.args}
			got, err := r.RequireString(tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RequireString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("RequireString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMCPTypes_RequireFloat(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		key     string
		want    float64
		wantErr bool
	}{
		{
			name:    "missing key returns error",
			args:    map[string]any{},
			key:     "count",
			want:    0,
			wantErr: true,
		},
		{
			name:    "float64 value",
			args:    map[string]any{"count": float64(3.14)},
			key:     "count",
			want:    3.14,
			wantErr: false,
		},
		{
			name:    "float32 value",
			args:    map[string]any{"count": float32(2.5)},
			key:     "count",
			want:    2.5,
			wantErr: false,
		},
		{
			name:    "int value",
			args:    map[string]any{"count": int(42)},
			key:     "count",
			want:    42,
			wantErr: false,
		},
		{
			name:    "int64 value",
			args:    map[string]any{"count": int64(99)},
			key:     "count",
			want:    99,
			wantErr: false,
		},
		{
			name:    "int32 value",
			args:    map[string]any{"count": int32(7)},
			key:     "count",
			want:    7,
			wantErr: false,
		},
		{
			name:    "valid string parses correctly",
			args:    map[string]any{"count": "12.34"},
			key:     "count",
			want:    12.34,
			wantErr: false,
		},
		{
			name:    "invalid string returns error",
			args:    map[string]any{"count": "not-a-number"},
			key:     "count",
			want:    0,
			wantErr: true,
		},
		{
			name:    "unsupported type returns error",
			args:    map[string]any{"count": true},
			key:     "count",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := CallToolRequest{Arguments: tt.args}
			got, err := r.RequireFloat(tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RequireFloat() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("RequireFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMCPTypes_GetString(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		key  string
		def  string
		want string
	}{
		{
			name: "missing returns default",
			args: map[string]any{},
			key:  "name",
			def:  "default",
			want: "default",
		},
		{
			name: "non-string returns default",
			args: map[string]any{"name": 123},
			key:  "name",
			def:  "default",
			want: "default",
		},
		{
			name: "valid string returns value",
			args: map[string]any{"name": "alice"},
			key:  "name",
			def:  "default",
			want: "alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := CallToolRequest{Arguments: tt.args}
			got := r.GetString(tt.key, tt.def)
			if got != tt.want {
				t.Errorf("GetString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMCPTypes_GetFloat(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		key  string
		def  float64
		want float64
	}{
		{
			name: "missing returns default",
			args: map[string]any{},
			key:  "count",
			def:  1.23,
			want: 1.23,
		},
		{
			name: "float64 value",
			args: map[string]any{"count": float64(3.14)},
			key:  "count",
			def:  0,
			want: 3.14,
		},
		{
			name: "float32 value",
			args: map[string]any{"count": float32(2.5)},
			key:  "count",
			def:  0,
			want: 2.5,
		},
		{
			name: "int value",
			args: map[string]any{"count": int(42)},
			key:  "count",
			def:  0,
			want: 42,
		},
		{
			name: "int64 value",
			args: map[string]any{"count": int64(99)},
			key:  "count",
			def:  0,
			want: 99,
		},
		{
			name: "int32 value",
			args: map[string]any{"count": int32(7)},
			key:  "count",
			def:  0,
			want: 7,
		},
		{
			name: "valid string parses correctly",
			args: map[string]any{"count": "12.34"},
			key:  "count",
			def:  0,
			want: 12.34,
		},
		{
			name: "invalid string returns default",
			args: map[string]any{"count": "not-a-number"},
			key:  "count",
			def:  1.23,
			want: 1.23,
		},
		{
			name: "unsupported type returns default",
			args: map[string]any{"count": true},
			key:  "count",
			def:  1.23,
			want: 1.23,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := CallToolRequest{Arguments: tt.args}
			got := r.GetFloat(tt.key, tt.def)
			if got != tt.want {
				t.Errorf("GetFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMCPTypes_GetBool(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		key  string
		def  bool
		want bool
	}{
		{
			name: "missing returns default",
			args: map[string]any{},
			key:  "active",
			def:  true,
			want: true,
		},
		{
			name: "true bool",
			args: map[string]any{"active": true},
			key:  "active",
			def:  false,
			want: true,
		},
		{
			name: "false bool",
			args: map[string]any{"active": false},
			key:  "active",
			def:  true,
			want: false,
		},
		{
			name: "valid true string",
			args: map[string]any{"active": "true"},
			key:  "active",
			def:  false,
			want: true,
		},
		{
			name: "valid false string",
			args: map[string]any{"active": "false"},
			key:  "active",
			def:  true,
			want: false,
		},
		{
			name: "valid 1 string",
			args: map[string]any{"active": "1"},
			key:  "active",
			def:  false,
			want: true,
		},
		{
			name: "valid 0 string",
			args: map[string]any{"active": "0"},
			key:  "active",
			def:  true,
			want: false,
		},
		{
			name: "invalid string returns default",
			args: map[string]any{"active": "maybe"},
			key:  "active",
			def:  true,
			want: true,
		},
		{
			name: "unsupported type returns default",
			args: map[string]any{"active": 123},
			key:  "active",
			def:  true,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := CallToolRequest{Arguments: tt.args}
			got := r.GetBool(tt.key, tt.def)
			if got != tt.want {
				t.Errorf("GetBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewToolResultError(t *testing.T) {
	msg := "something went wrong"
	result := NewToolResultError(msg)
	if result == nil {
		t.Fatal("NewToolResultError() returned nil")
	}
	if !result.IsError {
		t.Error("NewToolResultError().IsError = false, want true")
	}
	if result.Text != msg {
		t.Errorf("NewToolResultError().Text = %q, want %q", result.Text, msg)
	}
}

func TestNewToolResultText(t *testing.T) {
	text := "hello world"
	result := NewToolResultText(text)
	if result == nil {
		t.Fatal("NewToolResultText() returned nil")
	}
	if result.IsError {
		t.Error("NewToolResultText().IsError = true, want false")
	}
	if result.Text != text {
		t.Errorf("NewToolResultText().Text = %q, want %q", result.Text, text)
	}
}

func TestNewTool_WithNilOptions(t *testing.T) {
	var nilOpt ToolOption
	tool := NewTool("test_tool", nilOpt, WithDescription("desc"), nilOpt)
	if tool.Name != "test_tool" {
		t.Errorf("Name = %q, want test_tool", tool.Name)
	}
	if tool.Description != "desc" {
		t.Errorf("Description = %q, want desc", tool.Description)
	}
}

func TestToolOptions_WithNilInnerOptions(t *testing.T) {
	var nilOpt ToolOption

	// All of these should not panic when nil inner options are passed.
	tests := []struct {
		opt  ToolOption
		name string
	}{
		{
			name: "WithString with nil inner options",
			opt:  WithString("param", nilOpt, WithDescription("inner"), nilOpt),
		},
		{
			name: "WithNumber with nil inner options",
			opt:  WithNumber("param", nilOpt, WithDescription("inner"), nilOpt),
		},
		{
			name: "WithBoolean with nil inner options",
			opt:  WithBoolean("param", nilOpt, WithDescription("inner"), nilOpt),
		},
		{
			name: "Required",
			opt:  Required(),
		},
		{
			name: "Description",
			opt:  Description("desc"),
		},
		{
			name: "DefaultNumber",
			opt:  DefaultNumber(42),
		},
		{
			name: "DefaultBool",
			opt:  DefaultBool(true),
		},
		{
			name: "Default",
			opt:  Default("value"),
		},
		{
			name: "Enum",
			opt:  Enum("a", "b", "c"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply to a tool to ensure no panic
			tool := NewTool("test", tt.opt)
			if tool.Name != "test" {
				t.Errorf("Name = %q, want test", tool.Name)
			}
		})
	}
}
