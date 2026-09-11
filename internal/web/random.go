package web

import (
	"crypto/rand"
	"encoding/binary"
)

// randomHex32 returns 32 random bits, used for the reactive stack's request id.
func randomHex32() uint32 {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0
	}
	return binary.BigEndian.Uint32(buf[:])
}
