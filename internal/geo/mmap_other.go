//go:build !unix

package geo

import "os"

// mapFile reads the file at path: this system has no mapping here, so the
// tile takes memory.
func mapFile(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec // a tile in the configured geodata folder
}

// unmapFile does nothing.
func unmapFile([]byte) {}
