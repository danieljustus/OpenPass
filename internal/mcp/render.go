package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/danieljustus/OpenPass/internal/vault/taint"
)

func init() {
	// Bridge taint system: when taint.Render(MCP) is called, apply MCP
	// sanitization to the raw value before returning it.
	taint.SetMCPSanitizer(sanitizeMCPText)
}

// globalChokepoint is the shared MCP output sanitizer instance.
// It is used by callToolResultPayload and handleSanitizeOutput.
var globalChokepoint = &RenderChokepoint{}

// RenderChokepoint is the central output sanitization point for MCP responses.
// All tool responses should pass through this chokepoint before being sent
// to the MCP client to prevent prompt injection via vault content.
type RenderChokepoint struct{}

// NewRenderChokepoint creates a new RenderChokepoint instance.
func NewRenderChokepoint() *RenderChokepoint {
	return &RenderChokepoint{}
}

// SanitizeForMCP sanitizes text for safe inclusion in MCP responses.
// Strips ANSI escape sequences, control characters, OSC-8 hyperlinks,
// Unicode formatting/bidi/zero-width characters, and neutralizes
// XML structure injection (e.g. </data>).
func (rc *RenderChokepoint) SanitizeForMCP(text string) string {
	return sanitizeMCPText(text)
}

// sanitizeMCPText is the core MCP text sanitizer.
//
// It performs three phases:
//  1. NFKC normalization — fullwidth and compatibility characters are
//     decomposed to ASCII equivalents so downstream byte-level checks
//     can catch them.
//  2. Strip dangerous Unicode — bidirectional overrides, zero-width
//     characters, invisible formatting characters, soft hyphens, and
//     BOMs are removed.
//  3. Byte-level scanner — ANSI escape sequences, OSC-8 hyperlinks,
//     control characters, XML closing tags (</…>), and HTML comment
//     closers (-->) are stripped or neutralized.
func sanitizeMCPText(text string) string {
	// Phase 1: NFKC normalization to decompose unicode tricks.
	// Fullwidth variants (＜, ＞, ＤＡＴＡ etc.) become ASCII equivalents,
	// making them detectable by our ASCII-phase checks below.
	text = norm.NFKC.String(text)

	// Phase 2: Strip dangerous Unicode formatting/control characters.
	text = stripDangerousUnicode(text)

	// Phase 3: Byte-level scanner for ANSI escapes, control chars,
	// and XML/HTML tag injection.
	var out strings.Builder
	out.Grow(len(text))

	inEscape := false
	inOSC := false

	for i := 0; i < len(text); i++ {
		ch := text[i]

		if inEscape {
			if ch == '[' || (ch >= '0' && ch <= '9') || ch == ';' || ch == '?' {
				continue
			}
			if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
				inEscape = false
				continue
			}
			inEscape = false
			continue
		}

		if inOSC {
			if ch == 0x07 {
				inOSC = false
				continue
			}
			if ch == '\\' && i+1 < len(text) {
				inOSC = false
				i++
				continue
			}
			continue
		}

		if ch == 0x1b {
			if i+1 < len(text) && text[i+1] == ']' {
				inOSC = true
				i++
				continue
			}
			inEscape = true
			continue
		}

		if ch < 0x20 && ch != '\t' && ch != '\n' && ch != '\r' {
			continue
		}
		if ch == 0x7f {
			continue
		}

		// After NFKC normalization, any < and > are ASCII.
		// Neutralize XML closing tags: </tagname>, </tagname attr="val">
		// by injecting spaces so the sequence is no longer a valid tag.
		// The format </ tagname > is idempotent — applying the sanitizer
		// again will not change it further.
		if ch == '<' && i+2 < len(text) && text[i+1] == '/' {
			peek := text[i+2]
			if (peek >= 'a' && peek <= 'z') || (peek >= 'A' && peek <= 'Z') || peek == '_' {
				rest := text[i:]
				endIdx := strings.IndexByte(rest, '>')
				if endIdx > 2 { // at least </x>
					// Write </ with a space to break the XML closing tag
					out.WriteString("</ ")
					out.WriteString(rest[2:endIdx])
					out.WriteString(" >")
					i += endIdx
					continue
				}
			}
		}

		// Neutralize HTML comment closers that could break
		// EmbedAsData randomized markers.
		if ch == '-' && i+2 < len(text) && text[i:i+3] == "-->" {
			out.WriteString("-- >")
			i += 2
			continue
		}

		out.WriteByte(ch)
	}

	return out.String()
}

// stripDangerousUnicode removes Unicode formatting, bidirectional override,
// zero-width, and other invisible characters that can be used for prompt
// injection or tag confusion attacks.
func stripDangerousUnicode(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	for _, r := range s {
		switch {
		// Bidirectional control characters
		case r == '\u200e' || r == '\u200f': // LRM, RLM
			continue
		case r >= '\u202a' && r <= '\u202e': // LRE, RLE, PDF, LRO, RLO
			continue
		case r >= '\u2066' && r <= '\u2069': // LRI, RLI, FSI, PDI
			continue
		// Zero-width and invisible characters
		case r == '\u200b' || r == '\u200c' || r == '\u200d': // ZWSP, ZWNJ, ZWJ
			continue
		case r == '\ufeff': // BOM / ZWNBSP
			continue
		case r == '\u00ad': // Soft hyphen
			continue
		case r == '\u034f': // Combining grapheme joiner
			continue
		case r >= '\u2060' && r <= '\u2064': // Word joiner, function appl, invisible times/separator/plus
			continue
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

// generateMarker produces a cryptographically random hex string for use
// as an unpredictable marker in EmbedAsData output.
func generateMarker() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// EmbedAsData wraps untrusted content with randomized markers instead of
// XML-like tags. This prevents prompt injection via tag confusion — the
// closing marker is unpredictable and cannot be forged by untrusted content.
//
// Format:
//
//	<!-- DATA_xxxxxxxx label=NAME -->content<!-- /DATA_xxxxxxxx -->
//
// The label provides context for the LLM but is sanitized to prevent
// injection. The marker is 16 hex characters (64 bits of entropy).
func EmbedAsData(label, untrusted string) string {
	safe := sanitizeMCPText(untrusted)
	safeLabel := sanitizeMCPText(label)
	safeLabel = strings.ReplaceAll(safeLabel, "--", "-")
	safeLabel = strings.ReplaceAll(safeLabel, "\n", " ")
	safeLabel = strings.ReplaceAll(safeLabel, "\r", "")
	marker := generateMarker()
	return fmt.Sprintf("<!-- DATA_%s label=%s -->%s<!-- /DATA_%s -->", marker, safeLabel, safe, marker)
}
