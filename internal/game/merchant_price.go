package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/world"
)

// Merchant price configs (decision I7). A merchant (Mobius Merchant and its
// subclasses Fisherman, PetManager, Teleporter and ClanHallManager) takes
// its price config when it spawns: the lowest-ID config of
// MerchantPriceConfig.xml whose town zone is listed in the merchant's
// zone region, else the default. Mobius's zone regions are cells of 32,768
// units (ZoneManager.SHIFT_BY = 15) that list every zone whose shape
// touches them, so a merchant near a town, not only in it, pays its tax.
// Castles have no owner and a tax of 0 yet, so only the base tax counts.

// priceZoneShift is the size of Mobius's zone regions, 2^15 units
// (ZoneManager.SHIFT_BY).
const priceZoneShift = 15

// zoneRegion is a Mobius zone region: a 32,768-unit cell of the world.
type zoneRegion struct{ x, y int }

// zoneRegionOf returns the zone region of a position (Mobius
// ZoneManager.getRegion).
func zoneRegionOf(pos world.Position) zoneRegion {
	return zoneRegion{int(pos.X) >> priceZoneShift, int(pos.Y) >> priceZoneShift}
}

// merchantSpawned gives a merchant that has just spawned its price config.
// Other NPCs are left alone.
func (s *Server) merchantSpawned(n *Npc) {
	if s.economy == nil || !s.npcIs(n, "Merchant") {
		return
	}
	n.priceConfig = s.priceConfigAt(n.Position())
}

// priceConfigAt returns the price config of a merchant standing at pos
// (Mobius MerchantPriceConfigTable.getMerchantPriceConfig), or nil without
// economy data.
func (s *Server) priceConfigAt(pos world.Position) *data.PriceConfig {
	if s.economy == nil {
		return nil
	}
	zones := s.priceZones()
	r := zoneRegionOf(pos)
	return s.economy.PriceConfigFor(func(zoneID int) bool { return zones[zoneID][r] })
}

// priceZones returns, per town zone of a price config, the zone regions its
// shape touches. It is worked out on first use: the NPC feature spawns the
// world's merchants before the shop feature is set up.
func (s *Server) priceZones() map[int]map[zoneRegion]bool {
	f := &s.shop
	if f.priceZones != nil {
		return f.priceZones
	}
	f.priceZones = make(map[int]map[zoneRegion]bool)
	if s.economy == nil || s.worldData == nil {
		return f.priceZones
	}
	for _, c := range s.economy.PriceConfigs.Configs {
		if c.ZoneID <= 0 {
			continue
		}
		// The castle configs name zones the Interlude data does not have.
		if z, ok := s.worldData.Zone(c.ZoneID); ok {
			f.priceZones[c.ZoneID] = zoneRegionsTouched(z.Shape)
		}
	}
	return f.priceZones
}

// zoneRegionsTouched returns the zone regions a shape touches (the region
// grid of Mobius ZoneManager.load, ZoneForm.intersectsRectangle).
func zoneRegionsTouched(sh data.Shape) map[zoneRegion]bool {
	const size = 1 << priceZoneShift
	minX, minY, maxX, maxY := sh.Bounds()
	regions := make(map[zoneRegion]bool)
	for rx := minX>>priceZoneShift - 1; rx <= maxX>>priceZoneShift+1; rx++ {
		for ry := minY>>priceZoneShift - 1; ry <= maxY>>priceZoneShift+1; ry++ {
			x1, y1 := rx<<priceZoneShift, ry<<priceZoneShift
			if shapeIntersectsRect(sh, x1, y1, x1+size, y1+size) {
				regions[zoneRegion{rx, ry}] = true
			}
		}
	}
	return regions
}

// shapeIntersectsRect reports whether a zone shape and the rectangle from
// (x1, y1) to (x2, y2) overlap, as Mobius's intersectsRectangle does:
// java.awt.Rectangle.intersects for a Cuboid, java.awt.Polygon.intersects
// for an NPoly (the interiors overlap; touching edges are not compared the
// way Java's crossing counter does, which only matters for a polygon that
// ends exactly on a region border), and the circle against the rectangle
// for a Cylinder.
func shapeIntersectsRect(sh data.Shape, x1, y1, x2, y2 int) bool {
	switch sh.Type {
	case data.ShapeCuboid:
		if len(sh.Points) != 2 {
			return false
		}
		minX, minY, maxX, maxY := sh.Bounds()
		return rectsOverlap(minX, minY, maxX, maxY, x1, y1, x2, y2)
	case data.ShapeNPoly:
		return polygonIntersectsRect(sh, x1, y1, x2, y2)
	case data.ShapeCylinder:
		if len(sh.Points) != 1 {
			return false
		}
		c := sh.Points[0]
		dx := int64(c.X - min(max(c.X, x1), x2))
		dy := int64(c.Y - min(max(c.Y, y1), y2))
		r := int64(sh.Radius)
		return dx*dx+dy*dy < r*r
	}
	return false
}

// rectsOverlap is java.awt.Rectangle.intersects: two non-empty rectangles
// whose interiors overlap.
func rectsOverlap(ax1, ay1, ax2, ay2, bx1, by1, bx2, by2 int) bool {
	if ax2 <= ax1 || ay2 <= ay1 || bx2 <= bx1 || by2 <= by1 {
		return false
	}
	return ax1 < bx2 && bx1 < ax2 && ay1 < by2 && by1 < ay2
}

// polygonIntersectsRect reports whether an NPoly and a rectangle overlap: a
// corner of one lies in the other, or two edges cross.
func polygonIntersectsRect(sh data.Shape, x1, y1, x2, y2 int) bool {
	pts := sh.Points
	if len(pts) < 3 {
		return false
	}
	minX, minY, maxX, maxY := sh.Bounds()
	if !rectsOverlap(minX, minY, maxX, maxY, x1, y1, x2, y2) {
		return false
	}
	for _, pt := range pts {
		if pt.X > x1 && pt.X < x2 && pt.Y > y1 && pt.Y < y2 {
			return true
		}
	}
	corners := [4]data.Point{{X: x1, Y: y1}, {X: x2, Y: y1}, {X: x2, Y: y2}, {X: x1, Y: y2}}
	for _, c := range corners {
		if sh.ContainsXY(c.X, c.Y) {
			return true
		}
	}
	for i := range pts {
		a, b := pts[i], pts[(i+1)%len(pts)]
		for j := range corners {
			if segmentsCross(a, b, corners[j], corners[(j+1)%4]) {
				return true
			}
		}
	}
	return false
}

// segmentsCross reports whether the segments ab and cd cross at a point
// inside both.
func segmentsCross(a, b, c, d data.Point) bool {
	d1 := orientation(c, d, a)
	d2 := orientation(c, d, b)
	d3 := orientation(a, b, c)
	d4 := orientation(a, b, d)
	return ((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) && ((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0))
}

// orientation is the sign of the cross product (b - a) × (c - a).
func orientation(a, b, c data.Point) int64 {
	return int64(b.X-a.X)*int64(c.Y-a.Y) - int64(b.Y-a.Y)*int64(c.X-a.X)
}

// merchantTaxRate returns the tax rate a merchant adds to its prices (Mobius
// MerchantPriceConfig.getTotalTaxRate): the base tax of its price config,
// the castle's share being 0 while castles have no owner.
func merchantTaxRate(n *Npc) float64 {
	if n == nil || n.priceConfig == nil {
		return 0
	}
	return n.priceConfig.BaseTaxRate()
}
