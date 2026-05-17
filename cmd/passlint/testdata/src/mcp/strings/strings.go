package strings

import "taint"

func safeUsage() {
	u := taint.Wrap("secret", taint.Provenance{Source: "test"})
	_ = u.Render(taint.Terminal)
	_ = u.UnsafeRawForStorage()
	_ = u.Bytes()
}

func unsafeUsage() {
	u := taint.Wrap("secret", taint.Provenance{Source: "test"})
	_ = string(u) // want "direct string\\(\\) cast on taint.Untrusted"
}
