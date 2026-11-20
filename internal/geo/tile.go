package geo

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// L2J geodata, as Mobius reads it (IRegion, IBlock and the block classes).
//
// A tile file holds 256×256 blocks and no header, x-major: block (bx, by)
// is the (bx*256+by)-th. A block covers 8×8 cells, also x-major, and starts
// with a type byte:
//
//   - flat (0): one int16, the height of all its cells (not doubled); every
//     direction is open.
//   - complex (1): 64 int16 cell values.
//   - multilayer (2): per cell a layer count byte (1–125), then that many
//     int16 layer values, one per floor.
//
// A cell or layer value keeps the height, doubled, in its upper 12 bits and
// the NSWE movement flags in its low 4 bits. Everything is little endian.
const (
	tileBlocksX = 256                       // blocks per tile side
	tileBlocks  = tileBlocksX * tileBlocksX // blocks per tile
	blockCellsX = 8                         // cells per block side
	blockCells  = blockCellsX * blockCellsX // cells per block
	tileCellsX  = tileBlocksX * blockCellsX // cells per tile side: 2,048

	blockFlat       = 0
	blockComplex    = 1
	blockMultilayer = 2

	// maxLayers is the most layers Mobius accepts in a cell.
	maxLayers = 125
	// multilayerMaxAbove: a nearest floor more than this above the asked
	// height is another storey, and the height falls to the floor below
	// (Mobius MultilayerBlock.getNearestZ).
	multilayerMaxAbove = 1000
)

// NSWE movement flags of a cell (Mobius Cell): a flag is set when a
// creature may step to the neighbouring cell that way. North is towards
// smaller Y.
const (
	nsweEast  = 1 << 0
	nsweWest  = 1 << 1
	nsweSouth = 1 << 2
	nsweNorth = 1 << 3

	nsweNorthEast = nsweNorth | nsweEast
	nsweNorthWest = nsweNorth | nsweWest
	nsweSouthEast = nsweSouth | nsweEast
	nsweSouthWest = nsweSouth | nsweWest
	nsweAll       = nsweEast | nsweWest | nsweSouth | nsweNorth
)

// ErrCorrupt reports a tile file that is not valid L2J geodata.
var ErrCorrupt = errors.New("corrupt geodata tile")

// A tile keeps one entry per block: the block type in the top two bits,
// then a flat block's height in the low 16 bits, or the byte offset of a
// complex or multilayer block's cells in the file, after the type byte.
// Cell values are read from the file's bytes, which are memory-mapped, so
// an idle tile costs address space rather than memory; like Mobius, a
// multilayer cell is found by walking the layer counts of the cells before
// it.
const (
	entryTypeShift = 30
	entryValue     = 1<<entryTypeShift - 1
	// maxTileBytes: offsets must fit in an entry. Real tiles are a few MB.
	maxTileBytes = 1 << entryTypeShift
)

// tile is one loaded tile file.
type tile struct {
	data   []byte // the file
	blocks [tileBlocks]uint32
	info   TileInfo
}

// parseTile indexes the L2J tile (x, y) in b, which it keeps. Bytes after
// the last block are ignored, as in Mobius.
func parseTile(x, y int, b []byte) (*tile, error) {
	if len(b) >= maxTileBytes {
		return nil, fmt.Errorf("%w: %d bytes, more than any tile", ErrCorrupt, len(b))
	}
	t := &tile{data: b, info: TileInfo{X: x, Y: y, Bytes: len(b)}}
	cut := func(i int) error { return fmt.Errorf("%w: cut short at block %d", ErrCorrupt, i) }
	off := 0
	for i := range tileBlocks {
		if off >= len(b) {
			return nil, cut(i)
		}
		kind := b[off]
		off++
		switch kind {
		case blockFlat:
			if off+2 > len(b) {
				return nil, cut(i)
			}
			t.blocks[i] = uint32(binary.LittleEndian.Uint16(b[off:]))
			off += 2
			t.info.Flat++
		case blockComplex:
			t.blocks[i] = blockComplex<<entryTypeShift | entryOffset(off)
			off += 2 * blockCells
			if off > len(b) {
				return nil, cut(i)
			}
			t.info.Complex++
		case blockMultilayer:
			t.blocks[i] = blockMultilayer<<entryTypeShift | entryOffset(off)
			for c := range blockCells {
				if off >= len(b) {
					return nil, cut(i)
				}
				n := int(b[off])
				if n < 1 || n > maxLayers {
					return nil, fmt.Errorf("%w: block %d cell %d has %d layers", ErrCorrupt, i, c, n)
				}
				off += 1 + 2*n
				if off > len(b) {
					return nil, cut(i)
				}
			}
			t.info.Multilayer++
		default:
			return nil, fmt.Errorf("%w: block %d has type %d", ErrCorrupt, i, kind)
		}
	}
	return t, nil
}

// entryOffset packs a byte offset into a block entry.
func entryOffset(off int) uint32 {
	return uint32(off) //nolint:gosec // below maxTileBytes
}

// blockIndex returns the block of geodata cell (gx, gy) in its tile. A cell
// of a neighbouring tile wraps around to this tile's other edge, as in
// Mobius's Region.getBlock.
func blockIndex(gx, gy int) int {
	return (gx>>3&(tileBlocksX-1))<<8 | gy>>3&(tileBlocksX-1)
}

// cellIndex returns the cell (gx, gy) in its block.
func cellIndex(gx, gy int) int {
	return (gx&(blockCellsX-1))<<3 | gy&(blockCellsX-1)
}

// flatHeight returns the height a flat block's entry keeps in its low 16
// bits.
func flatHeight(e uint32) int {
	return int(int16(e & 0xFFFF)) //nolint:gosec // the low 16 bits are the int16 height
}

// cellHeight returns the height of a cell or layer value: its upper 12
// bits as a signed short, halved (Mobius getCellHeight).
func cellHeight(v int16) int { return int(v&^0xF) >> 1 }

// cellNSWE returns the movement flags of a cell or layer value.
func cellNSWE(v int16) int { return int(v) & nsweAll }

// value returns the int16 at byte offset off.
func (t *tile) value(off int) int16 {
	return int16(binary.LittleEndian.Uint16(t.data[off:])) //nolint:gosec // reinterprets the two's complement bits
}

// complexValue returns the value of cell (gx, gy) of the complex block with
// entry e.
func (t *tile) complexValue(e uint32, gx, gy int) int16 {
	return t.value(int(e&entryValue) + 2*cellIndex(gx, gy))
}

// layers returns the byte offset of the first layer of cell (gx, gy) of the
// multilayer block with entry e and its layer count (Mobius
// getCellDataOffset).
func (t *tile) layers(e uint32, gx, gy int) (off, n int) {
	off = int(e & entryValue)
	for range cellIndex(gx, gy) {
		off += 1 + 2*int(t.data[off])
	}
	return off + 1, int(t.data[off])
}

// nearestZ returns the height of cell (gx, gy) nearest to z (Mobius
// IBlock.getNearestZ).
func (t *tile) nearestZ(gx, gy, z int) int {
	e := t.blocks[blockIndex(gx, gy)]
	switch e >> entryTypeShift {
	case blockFlat:
		return flatHeight(e)
	case blockComplex:
		return cellHeight(t.complexValue(e, gx, gy))
	}
	off, n := t.layers(e, gx, gy)
	nearest := cellHeight(t.nearestLayer(off, n, z))
	if nearest-z > multilayerMaxAbove {
		return t.lowerLayer(off, n, z)
	}
	return nearest
}

// nextLowerZ returns the height of cell (gx, gy) at or below z, or z when
// there is none (Mobius IBlock.getNextLowerZ).
func (t *tile) nextLowerZ(gx, gy, z int) int {
	e := t.blocks[blockIndex(gx, gy)]
	switch e >> entryTypeShift {
	case blockFlat:
		return min(flatHeight(e), z)
	case blockComplex:
		return min(cellHeight(t.complexValue(e, gx, gy)), z)
	}
	off, n := t.layers(e, gx, gy)
	return t.lowerLayer(off, n, z)
}

// nextHigherZ returns the height of cell (gx, gy) at or above z, or z when
// there is none (Mobius IBlock.getNextHigherZ).
func (t *tile) nextHigherZ(gx, gy, z int) int {
	e := t.blocks[blockIndex(gx, gy)]
	switch e >> entryTypeShift {
	case blockFlat:
		return max(flatHeight(e), z)
	case blockComplex:
		return max(cellHeight(t.complexValue(e, gx, gy)), z)
	}
	off, n := t.layers(e, gx, gy)
	return t.higherLayer(off, n, z)
}

// nswe returns the movement flags of the layer of cell (gx, gy) nearest to
// z: all directions on a flat block (Mobius IBlock.checkNearestNswe).
func (t *tile) nswe(gx, gy, z int) int {
	e := t.blocks[blockIndex(gx, gy)]
	switch e >> entryTypeShift {
	case blockFlat:
		return nsweAll
	case blockComplex:
		return cellNSWE(t.complexValue(e, gx, gy))
	}
	off, n := t.layers(e, gx, gy)
	return cellNSWE(t.nearestLayer(off, n, z))
}

// nearestLayer returns the layer closest to z of the n layers at off: one
// exactly at z at once, otherwise the first of the closest (Mobius
// getNearestLayer).
func (t *tile) nearestLayer(off, n, z int) int16 {
	var best int16
	bestDZ := -1
	for i := range n {
		l := t.value(off + 2*i)
		h := cellHeight(l)
		if h == z {
			return l
		}
		if dz := abs(h - z); bestDZ < 0 || dz < bestDZ {
			best, bestDZ = l, dz
		}
	}
	return best
}

// lowerLayer returns z when a layer is exactly at z, otherwise the highest
// layer below z, or z when none is (Mobius getNextLowerZ).
func (t *tile) lowerLayer(off, n, z int) int {
	lower, found := 0, false
	for i := range n {
		h := cellHeight(t.value(off + 2*i))
		if h == z {
			return h
		}
		if h < z && (!found || h > lower) {
			lower, found = h, true
		}
	}
	if !found {
		return z
	}
	return lower
}

// higherLayer returns z when a layer is exactly at z, otherwise the lowest
// layer above z, or z when none is (Mobius getNextHigherZ).
func (t *tile) higherLayer(off, n, z int) int {
	higher, found := 0, false
	for i := range n {
		h := cellHeight(t.value(off + 2*i))
		if h == z {
			return h
		}
		if h > z && (!found || h < higher) {
			higher, found = h, true
		}
	}
	if !found {
		return z
	}
	return higher
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
