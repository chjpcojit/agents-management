// Package system adapts the operating system (wall clock, random source) to the use case ports.
// Package system nối các tài nguyên hệ điều hành (đồng hồ, nguồn random) vào port của use case.
package system

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/chjpcojit/agents-management/internal/usecase/port"
)

var (
	_ port.Clock       = Clock{}
	_ port.IDGenerator = UUIDGenerator{}
)

// Clock reads the wall clock in UTC.
// Clock đọc thời gian hệ thống theo UTC.
type Clock struct{}

// Now returns the current UTC time.
// Now trả về thời gian UTC hiện tại.
func (Clock) Now() time.Time { return time.Now().UTC() }

// UUIDGenerator returns random RFC 9562 version 4 UUIDs from crypto/rand.
// UUIDGenerator sinh UUID version 4 (RFC 9562) ngẫu nhiên từ crypto/rand.
type UUIDGenerator struct{}

// NewID returns a new UUIDv4 string.
// crypto/rand.Read never returns an error on supported platforms (Go 1.24+); it panics instead.
//
// NewID trả về một UUIDv4 dạng string.
// crypto/rand.Read không trả lỗi trên các platform được hỗ trợ (Go 1.24+); nếu hỏng thì panic.
func (UUIDGenerator) NewID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4 / phiên bản 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 9562 variant / variant RFC 9562
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
