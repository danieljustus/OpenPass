package template

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBase64Encode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", base64.StdEncoding.EncodeToString([]byte("hello"))},
		{"", base64.StdEncoding.EncodeToString([]byte(""))},
		{"special!@#$%", base64.StdEncoding.EncodeToString([]byte("special!@#$%"))},
		{"unicode: \u00e4\u00f6\u00fc", base64.StdEncoding.EncodeToString([]byte("unicode: \u00e4\u00f6\u00fc"))},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := base64Encode(tt.input)
			if got != tt.want {
				t.Errorf("base64Encode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBase64URLEncode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", base64.URLEncoding.EncodeToString([]byte("hello"))},
		{"", base64.URLEncoding.EncodeToString([]byte(""))},
		{"with+slash/", base64.URLEncoding.EncodeToString([]byte("with+slash/"))},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := base64URLEncode(tt.input)
			if got != tt.want {
				t.Errorf("base64URLEncode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUpper(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "HELLO"},
		{"Hello World", "HELLO WORLD"},
		{"", ""},
		{"ALREADY", "ALREADY"},
		{"mixed123", "MIXED123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := strings.ToUpper(tt.input)
			if got != tt.want {
				t.Errorf("upper(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLower(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"HELLO", "hello"},
		{"Hello World", "hello world"},
		{"", ""},
		{"already", "already"},
		{"MIXED123", "mixed123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := strings.ToLower(tt.input)
			if got != tt.want {
				t.Errorf("lower(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToJSON(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"string", "hello", `"hello"`},
		{"empty string", "", `""`},
		{"string with quotes", `say "hello"`, `"say \"hello\""`},
		{"number int", 42, "42"},
		{"number float", 3.14, "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, "null"},
		{"slice", []string{"a", "b"}, `["a","b"]`},
		{"map", map[string]int{"x": 1}, `{"x":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toJSON(tt.input)
			if got != tt.want {
				t.Errorf("toJSON(%#v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDefaultFuncMap(t *testing.T) {
	funcs := DefaultFuncMap()

	required := []string{"b64enc", "b64url", "upper", "lower", "tojson"}
	for _, name := range required {
		if _, ok := funcs[name]; !ok {
			t.Errorf("DefaultFuncMap missing required function %q", name)
		}
	}

	if len(funcs) != len(required) {
		t.Errorf("DefaultFuncMap has %d functions, want %d", len(funcs), len(required))
	}
}

func TestDefaultFuncMapIntegration(t *testing.T) {
	funcs := DefaultFuncMap()

	tests := []struct {
		name     string
		fnName   string
		input    any
		wantFunc func(string) bool
	}{
		{
			name:   "b64enc",
			fnName: "b64enc",
			input:  "test",
			wantFunc: func(s string) bool {
				return s == base64.StdEncoding.EncodeToString([]byte("test"))
			},
		},
		{
			name:   "b64url",
			fnName: "b64url",
			input:  "test",
			wantFunc: func(s string) bool {
				return s == base64.URLEncoding.EncodeToString([]byte("test"))
			},
		},
		{
			name:   "upper",
			fnName: "upper",
			input:  "test",
			wantFunc: func(s string) bool {
				return s == "TEST"
			},
		},
		{
			name:   "lower",
			fnName: "lower",
			input:  "TEST",
			wantFunc: func(s string) bool {
				return s == "test"
			},
		},
		{
			name:   "tojson",
			fnName: "tojson",
			input:  "test",
			wantFunc: func(s string) bool {
				return s == `"test"`
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := funcs[tt.fnName]
			if !ok {
				t.Fatalf("function %q not found in func map", tt.fnName)
			}

			var got string
			switch f := fn.(type) {
			case func(string) string:
				got = f(tt.input.(string))
			case func(any) string:
				got = f(tt.input)
			default:
				t.Fatalf("unexpected function type for %q: %T", tt.fnName, fn)
			}

			if !tt.wantFunc(got) {
				t.Errorf("%s(%v) = %q, unexpected result", tt.fnName, tt.input, got)
			}
		})
	}
}
