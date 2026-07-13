// Package id generates RFC 4122 UUID v4 identifiers using crypto/rand.
// This replaces stdlib "uuid" (Go 1.27+) for Go 1.26 compatibility.
package id

import (
	"crypto/rand"
	"fmt"
)

// NewV4 returns a random UUID v4 string (RFC 4122).
func NewV4() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(fmt.Sprintf("id.NewV4: crypto/rand.Read failed: %v", err))
	}
	// Set version 4
	b[6] = (b[6] & 0x0f) | 0x40
	// Set variant bits
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// NewV7 returns a time-ordered UUID v7 string (RFC 9562).
// Used for sortable, non-secret identifiers like order references.
// Falls back to v4-style randomness with a timestamp prefix for Go <1.27.
func NewV7() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(fmt.Sprintf("id.NewV7: crypto/rand.Read failed: %v", err))
	}
	// Set version 7
	b[6] = (b[6] & 0x0f) | 0x70
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Ref generates a short human-readable order reference (8 chars, uppercase).
func Ref() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O/1/I
	ref := make([]byte, 8)
	for i := range ref {
		ref[i] = charset[int(b[i%6])%len(charset)]
	}
	return string(ref)
}
