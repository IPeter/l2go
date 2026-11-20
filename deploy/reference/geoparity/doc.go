// Package geoparity compares internal/geo with Mobius's own geodata engine
// (Plan 13, test strategy 2). GeoParity.java answers height, sight, walk
// and path queries from a file with Mobius's GeoEngine and PathFinding; the
// test behind the refgeo build tag writes geotest maps as tiles, sends both
// engines the same random queries and compares the answers exactly. See
// deploy/reference/README.md, "Geodata parity", for how to run it.
package geoparity
