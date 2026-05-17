package taint

type Provenance struct {
	Source string
}

type TerminalStyle int

const Terminal TerminalStyle = 0

type Untrusted string

func Wrap(value string, provenance ...Provenance) Untrusted {
	return Untrusted(value)
}

func (u Untrusted) Render(style TerminalStyle) string {
	return string(u)
}

func (u Untrusted) UnsafeRawForStorage() string {
	return string(u)
}

func (u Untrusted) Bytes() []byte {
	return []byte(u)
}

func (u Untrusted) Provenance() []Provenance {
	return nil
}

func (u Untrusted) Tags() map[string]string {
	return nil
}
