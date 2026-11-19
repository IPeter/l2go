package world

import (
	"bytes"
	"runtime"
	"strconv"
)

// goid returns the ID of the calling goroutine, read from the header of its
// stack trace ("goroutine 42 [running]:"). It takes about a microsecond, so
// it is used only for owner checks.
func goid() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	b := bytes.TrimPrefix(buf[:n], []byte("goroutine "))
	i := bytes.IndexByte(b, ' ')
	if i <= 0 {
		return 0
	}
	id, err := strconv.ParseUint(string(b[:i]), 10, 64)
	if err != nil {
		return 0
	}
	return id
}
