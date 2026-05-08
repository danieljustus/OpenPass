package autotype

import "testing"

func TestEscapeAppleScriptString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{`back\slash`, `back\\slash`},
		{`say "hello"`, `say \"hello\"`},
		{"line\nbreak", "line\\nbreak"},
		{"car\rriage", "car\\rriage"},
		{"tab\there", "tab\\there"},
		{`a\b"c\nd\re\tf`, `a\\b\"c\\nd\\re\\tf`},
		{"test{123}+^%~", "test{123}+^%~"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeAppleScriptString(tt.input)
			if got != tt.expected {
				t.Errorf("escapeAppleScriptString(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEscapeSendKeysString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"plus+sign", "plus{+}sign"},
		{"caret^", "caret{^}"},
		{"percent%", "percent{%}"},
		{"tilde~", "tilde{~}"},
		{"brace{", "brace{{}"},
		{"brace}", "brace{}}"},
		{"bracket[", "bracket{[}"},
		{"bracket]", "bracket{]}"},
		{"test{123}+^%~", "test{{}123{}}{+}{^}{%}{~}"},
		{"normal.text", "normal.text"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeSendKeysString(tt.input)
			if got != tt.expected {
				t.Errorf("escapeSendKeysString(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
