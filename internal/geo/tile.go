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
//   - flat (0): one int16, the height of all its cells; every direction is
//     open.
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
// then a flat block's height in the low 16 bits, or the offset of the
// block's values in data. A complex block has 64 values. A multilayer
// block has 65 cell starts, counted from the end of the starts (cell c's
// layers run from start c to start c+1), followed by its layers, so a cell
// is found without walking the block as Mobius does.
//
// data has fewer than 2^30 values: at most 65,536 blocks of 65 + 64×125.
const (
	entryTypeShift = 30
	entryOffset    = 1<<entryTypeShift - 1
	multiStarts    = blockCells + 1
)

// tile is one loaded tile file. Memory: 256 KiB of block entries, plus 128
// bytes per complex block and 130 bytes plus 2 per layer per multilayer
// block. A flat block costs only its entry.
type tile struct {
	blocks [tileBlocks]uint32
	data   []int16
	info   TileInfo
}

// parseTile decodes the L2J tile (x, y). Bytes after the last block are
// ignored, as in Mobius.
func parseTile(x, y int, b []byte) (*tile, error) {
	values, err := scanTile(b)
	if err != nil {
		return nil, err
	}
	t := &tile{data: make([]int16, values)}
	t.info = TileInfo{X: x, Y: y, Bytes: 4*tileBlocks + 2*values}
	off, pos := 0, 0
	for i := range tileBlocks {
		kind := b[off]
		off++
		switch kind {
		case blockFlat:
			t.blocks[i] = uint32(binary.LittleEndian.Uint16(b[off:]))
			off += 2
			t.info.Flat++
		case blockComplex:
			t.blocks[i] = blockComplex<<entryTypeShift | dataOffset(pos)
			for c := range blockCells {
				t.data[pos+c] = int16At(b, off+2*c)
			}
			off += 2 * blockCells
			pos += blockCells
			t.info.Complex++
		default: // blockMultilayer; scanTile checked the types
			t.blocks[i] = blockMultilayer<<entryTypeShift | dataOffset(pos)
			first := pos + multiStarts
			next := first
			for c := range blockCells {
				t.data[pos+c] = layerStart(next - first)
				n := int(b[off])
				off++
				for range n {
					t.data[next] = int16At(b, off)
					next++
					off += 2
				}
			}
			t.data[pos+blockCells] = layerStart(next - first)
			pos = next
			t.info.Multilayer++
		}
	}
	return t, nil
}

// scanTile checks the blocks of a tile file and returns how many values
// its data needs.
func scanTile(b []byte) (int, error) {
	off, values := 0, 0
	for i := range tileBlocks {
		if off >= len(b) {
			return 0, fmt.Errorf("%w: cut short at block %d", ErrCorrupt, i)
		}
		kind := b[off]
		off++
		switch kind {
		case blockFlat:
			off += 2
		case blockComplex:
			off += 2 * blockCells
			values += blockCells
		case blockMultilayer:
			values += multiStarts
			for c := range blockCells {
				if off >= len(b) {
					return 0, fmt.Errorf("%w: cut short at block %d", ErrCorrupt, i)
				}
				n := int(b[off])
				if n < 1 || n > maxLayers {
					return 0, fmt.Errorf("%w: block %d cell %d has %d layers", ErrCorrupt, i, c, n)
				}
				off += 1 + 2*n
				values += n
			}
		default:
			return 0, fmt.Errorf("%w: block %d has type %d", ErrCorrupt, i, kind)
		}
		if off > len(b) {
			return 0, fmt.Errorf("%w: cut short at block %d", ErrCorrupt, i)
		}
	}
	return values, nil
}

// int16At reads the little-endian int16 at b[off:].
func int16At(b []byte, off int) int16 {
	return int16(binary.LittleEndian.Uint16(b[off:])) //nolint:gosec // reinterprets the two's complement bits
}

// dataOffset packs an offset into data into a block entry.
func dataOffset(pos int) uint32 {
	return uint32(pos) //nolint:gosec // data has fewer than 2^30 values
}

// layerStart is a multilayer cell start: at most 64×125 layers.
func layerStart(n int) int16 {
	return int16(n) //nolint:gosec // at most 8,000
}

// blockIndex returns the block of geodata cell (gx, gy) in its tile.
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

// layers returns the layer values of cell (gx, gy) of the multilayer block
// with entry e.
func (t *tile) layers(e uint32, gx, gy int) []int16 {
	base := int(e & entryOffset)
	c := cellIndex(gx, gy)
	first := base + multiStarts
	return t.data[first+int(t.data[base+c]) : first+int(t.data[base+c+1])]
}

// nearestZ returns the height of cell (gx, gy) nearest to z (Mobius
// IBlock.getNearestZ).
func (t *tile) nearestZ(gx, gy, z int) int {
	e := t.blocks[blockIndex(gx, gy)]
	switch e >> entryTypeShift {
	case blockFlat:
		return flatHeight(e)
	case blockComplex:
		return cellHeight(t.data[int(e&entryOffset)+cellIndex(gx, gy)])
	}
	layers := t.layers(e, gx, gy)
	nearest := cellHeight(nearestLayer(layers, z))
	if nearest-z > multilayerMaxAbove {
		return lowerLayer(layers, z)
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
		return min(cellHeight(t.data[int(e&entryOffset)+cellIndex(gx, gy)]), z)
	}
	return lowerLayer(t.layers(e, gx, gy), z)
}

// nextHigherZ returns the height of cell (gx, gy) at or above z, or z when
// there is none (Mobius IBlock.getNextHigherZ).
func (t *tile) nextHigherZ(gx, gy, z int) int {
	e := t.blocks[blockIndex(gx, gy)]
	switch e >> entryTypeShift {
	case blockFlat:
		return max(flatHeight(e), z)
	case blockComplex:
		return max(cellHeight(t.data[int(e&entryOffset)+cellIndex(gx, gy)]), z)
	}
	return higherLayer(t.layers(e, gx, gy), z)
}

// nswe returns the movement flags of the layer of cell (gx, gy) nearest to
// z: all directions on a flat block (Mobius IBlock.checkNearestNswe).
func (t *tile) nswe(gx, gy, z int) int {
	e := t.blocks[blockIndex(gx, gy)]
	switch e >> entryTypeShift {
	case blockFlat:
		return nsweAll
	case blockComplex:
		return cellNSWE(t.data[int(e&entryOffset)+cellIndex(gx, gy)])
	}
	return cellNSWE(nearestLayer(t.layers(e, gx, gy), z))
}

// nearestLayer returns the layer closest to z: one exactly at z at once,
// otherwise the first of the closest (Mobius getNearestLayer).
func nearestLayer(layers []int16, z int) int16 {
	best, bestDZ := layers[0], -1
	for _, l := range layers {
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
func lowerLayer(layers []int16, z int) int {
	lower, found := 0, false
	for _, l := range layers {
		h := cellHeight(l)
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
func higherLayer(layers []int16, z int) int {
	higher, found := 0, false
	for _, l := range layers {
		h := cellHeight(l)
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
