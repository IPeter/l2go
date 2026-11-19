package world

import "fmt"

// Region geometry, from Mobius World: map tiles 16–26 × 10–25 of 32768
// units, cut into regions of 2048×2048 units, and 16 layers of 2000 units
// from -16000 to 16000 in Z.
const (
	RegionShift  = 11
	RegionSize   = 1 << RegionShift
	RegionZSize  = 2000
	RegionLayers = 16

	WorldXMin = -131072 // (16 - 20) * 32768
	WorldXMax = 229376  // (26 - 20 + 1) * 32768, exclusive
	WorldYMin = -262144 // (10 - 18) * 32768
	WorldYMax = 262144  // (25 - 18 + 1) * 32768, exclusive
	WorldZMin = -16000
	WorldZMax = 16000

	regionOffsetX = -WorldXMin >> RegionShift              // 64
	regionOffsetY = -WorldYMin >> RegionShift              // 128
	regionsX      = WorldXMax>>RegionShift + regionOffsetX // 176
	regionsY      = WorldYMax>>RegionShift + regionOffsetY // 256
)

// RegionKey identifies a region by Mobius's indices: X 0–175, Y 0–255 and
// Z 0–15.
type RegionKey struct{ X, Y, Z int32 }

func (k RegionKey) String() string { return fmt.Sprintf("(%d, %d, %d)", k.X, k.Y, k.Z) }

// RegionOf returns the region holding (x, y, z), or false when (x, y) lies
// outside the world. Every z has a layer: below WorldZMin is layer 0, from
// WorldZMax up is layer 15.
func RegionOf(x, y, z int32) (RegionKey, bool) {
	if x < WorldXMin || x >= WorldXMax || y < WorldYMin || y >= WorldYMax {
		return RegionKey{}, false
	}
	return RegionKey{X: x>>RegionShift + regionOffsetX, Y: y>>RegionShift + regionOffsetY, Z: layerOf(z)}, true
}

func layerOf(z int32) int32 {
	switch {
	case z < WorldZMin:
		return 0
	case z >= WorldZMax:
		return RegionLayers - 1
	}
	return (z - WorldZMin) / RegionZSize
}

// Adjacent reports whether k and o lie in each other's 3×3×3 neighbourhood.
// A region is adjacent to itself.
func (k RegionKey) Adjacent(o RegionKey) bool {
	return absDiff(k.X, o.X) <= 1 && absDiff(k.Y, o.Y) <= 1 && absDiff(k.Z, o.Z) <= 1
}

func absDiff(a, b int32) int32 {
	if a > b {
		return a - b
	}
	return b - a
}

func (k RegionKey) valid() bool {
	return k.X >= 0 && k.X < regionsX && k.Y >= 0 && k.Y < regionsY && k.Z >= 0 && k.Z < RegionLayers
}

// pack turns a valid key into a map key: X takes 8 bits, Y 9, Z 4.
func (k RegionKey) pack() uint32 { return uint32(k.X)<<13 | uint32(k.Y)<<4 | uint32(k.Z) }

// region holds the objects whose position lies in it. Regions are created
// when first entered and never removed.
type region struct {
	key       RegionKey
	objects   []*Base
	viewers   []*Base   // the objects that are viewers, for fast broadcasts
	neighbors []*region // existing regions of the 3×3×3 neighbourhood, itself first
}

func (r *region) add(b *Base) {
	b.region = r
	b.slot = int32(len(r.objects))
	r.objects = append(r.objects, b)
	b.vslot = -1
	if b.viewer != nil {
		b.vslot = int32(len(r.viewers))
		r.viewers = append(r.viewers, b)
	}
}

// remove takes b out by moving the last object into its slot.
func (r *region) remove(b *Base) {
	last := len(r.objects) - 1
	moved := r.objects[last]
	r.objects[b.slot] = moved
	moved.slot = b.slot
	r.objects[last] = nil
	r.objects = r.objects[:last]
	if b.vslot >= 0 {
		last = len(r.viewers) - 1
		moved = r.viewers[last]
		r.viewers[b.vslot] = moved
		moved.vslot = b.vslot
		r.viewers[last] = nil
		r.viewers = r.viewers[:last]
	}
	b.region = nil
	b.slot, b.vslot = -1, -1
}

// regionAt returns the region with key k, creating it (and linking it to its
// neighbours) when create is set.
func (w *World) regionAt(k RegionKey, create bool) *region {
	if r := w.regions[k.pack()]; r != nil || !create {
		return r
	}
	r := &region{key: k}
	r.neighbors = append(r.neighbors, r)
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			for dz := int32(-1); dz <= 1; dz++ {
				nk := RegionKey{k.X + dx, k.Y + dy, k.Z + dz}
				if nk == k || !nk.valid() {
					continue
				}
				if n := w.regions[nk.pack()]; n != nil {
					r.neighbors = append(r.neighbors, n)
					n.neighbors = append(n.neighbors, r)
				}
			}
		}
	}
	w.regions[k.pack()] = r
	return r
}

// neighbourhood returns the existing regions around b: its region's
// neighbours when it is in the world, otherwise those around its position
// (Mobius looks regions up by position too).
func (w *World) neighbourhood(b *Base) []*region {
	if b.region != nil {
		return b.region.neighbors
	}
	k, ok := RegionOf(b.pos.X, b.pos.Y, b.pos.Z)
	if !ok {
		return nil
	}
	return w.around(k)
}

// around returns the existing regions in the neighbourhood of k.
func (w *World) around(k RegionKey) []*region {
	if r := w.regions[k.pack()]; r != nil {
		return r.neighbors
	}
	var list []*region
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			for dz := int32(-1); dz <= 1; dz++ {
				nk := RegionKey{k.X + dx, k.Y + dy, k.Z + dz}
				if !nk.valid() {
					continue
				}
				if n := w.regions[nk.pack()]; n != nil {
					list = append(list, n)
				}
			}
		}
	}
	return list
}
