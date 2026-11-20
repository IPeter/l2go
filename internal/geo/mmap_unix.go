//go:build unix

package geo

import (
	"fmt"
	"os"
	"syscall"
)

// mapFile maps the file at path read-only, as Mobius maps its tiles. The
// mapping lives as long as the process; the file must not change while the
// server runs.
func mapFile(path string) ([]byte, error) {
	f, err := os.Open(path) //nolint:gosec // a tile in the configured geodata folder
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	if size == 0 {
		return nil, nil
	}
	if size >= maxTileBytes {
		return nil, fmt.Errorf("%w: %d bytes, more than any tile", ErrCorrupt, size)
	}
	return syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED) //nolint:gosec // a file descriptor and a size below 1 GiB
}

// unmapFile releases a mapping of mapFile.
func unmapFile(b []byte) {
	if len(b) > 0 {
		_ = syscall.Munmap(b)
	}
}
