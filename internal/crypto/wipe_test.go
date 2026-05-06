package crypto

import "testing"

func TestWipeZeroesBytes(t *testing.T) {
	buf := []byte("sensitive data")
	Wipe(buf)
	for i, b := range buf {
		if b != 0 {
			t.Errorf("byte %d not zeroed: got %d", i, b)
		}
	}
}
