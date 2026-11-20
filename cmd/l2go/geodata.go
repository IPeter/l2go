package main

import (
	"fmt"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/geo"
)

// geoOptions returns the geodata engine's options of the geodata section
// (Mobius GeoEngine.ini).
func geoOptions(c config.Geodata) (geo.Options, error) {
	sizes, err := c.BufferSizes()
	if err != nil {
		return geo.Options{}, fmt.Errorf("geodata.pathFindBuffers: %w", err)
	}
	return geo.Options{
		Pathfinding:              c.Pathfinding,
		BufferSizes:              sizes,
		LowWeight:                c.LowWeight,
		MediumWeight:             c.MediumWeight,
		HighWeight:               c.HighWeight,
		DiagonalWeight:           c.DiagonalWeight,
		AdvancedDiagonalStrategy: c.AdvancedDiagonalStrategy,
		AvoidObstructedPathNodes: c.AvoidObstructedPathNodes,
		MaxPostfilterPasses:      c.MaxPostfilterPasses,
	}, nil
}
