package template

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"text/template"
)

// DefaultFuncMap returns the default function map for template execution.
// These functions are available in all templates.
func DefaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"b64enc": base64Encode,
		"b64url": base64URLEncode,
		"upper":  strings.ToUpper,
		"lower":  strings.ToLower,
		"tojson": toJSON,
	}
}

// base64Encode returns the standard base64 encoding of the input string.
func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// base64URLEncode returns the URL-safe base64 encoding of the input string.
func base64URLEncode(s string) string {
	return base64.URLEncoding.EncodeToString([]byte(s))
}

// toJSON returns the JSON encoding of the input value.
// Strings are quoted; other values are marshaled to JSON.
func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
