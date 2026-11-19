package data

import "fmt"

// Zone spawn point types (ZoneSpawn.Type), as Mobius's ZoneRespawn reads
// them; SiegableHallZone also reads challenger points.
const (
	ZoneSpawnNormal     = ""
	ZoneSpawnOther      = "other"
	ZoneSpawnChaotic    = "chaotic"
	ZoneSpawnBanish     = "banish"
	ZoneSpawnChallenger = "challenger"
)

// FirstDynamicZoneID is the ID of the first zone without an ID in the XML.
// Mobius numbers such zones from 300001 in load order; cmd/datagen does it
// in file name order, then document order.
const FirstDynamicZoneID = 300001

// Zone is a zone of the Mobius zone data (dist/game/data/zones/*.xml; lists
// with enabled="false" are left out, as Mobius skips them). The custom_*.xml
// files are part of the data: Mobius loads them unconditionally, and the
// town zones live in custom_town.xml.
type Zone struct {
	ID   int    `json:"id"`
	Name string `json:"name,omitempty"`
	// Type is the Mobius zone class (WaterZone, PeaceZone, TownZone, ...),
	// a usable class of ZoneTypesFile. What a zone does (the ZoneId flags
	// it sets, its effects) comes from its type, as in Mobius.
	Type string `json:"type"`
	// File is the source file name, for messages.
	File  string `json:"file"`
	Shape Shape  `json:"shape"`
	// Params are the <stat name val> parameters in XML order (a name may
	// repeat, such as affectedRace), kept raw for the zone types.
	Params []ZoneParam `json:"params,omitempty"`
	// Spawns are the zone's spawn points; only the ZoneRespawn types have
	// them (Mobius ignores them on other types).
	Spawns []ZoneSpawn `json:"spawns,omitempty"`
	// RacePoints maps a player race to the map region whose respawn points
	// it uses; only RespawnZone has them. Some name a region that does not
	// exist, which makes Mobius use the default respawn region (see
	// World.RespawnRegion).
	RacePoints map[Race]string `json:"racePoints,omitempty"`
}

// ZoneParam is one <stat> parameter of a zone.
type ZoneParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ZoneSpawn is one spawn point of a zone.
type ZoneSpawn struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
	// Type is one of the ZoneSpawn* constants.
	Type string `json:"type,omitempty"`
}

// Param returns the value of the first parameter with the given name.
func (z *Zone) Param(name string) (string, bool) {
	for _, p := range z.Params {
		if p.Name == name {
			return p.Value, true
		}
	}
	return "", false
}

// Zone returns the zone with the given ID.
func (w *World) Zone(id int) (*Zone, bool) {
	i, ok := w.zones[id]
	if !ok {
		return nil, false
	}
	return &w.Zones[i], true
}

// checkZones checks the zones and indexes them:
//
//   - IDs are positive and unique;
//   - the type is a usable class of ZoneTypesFile and the shape is one
//     Mobius accepts;
//   - parameters have names;
//   - spawn points are on ZoneRespawn types and have a known type;
//   - race points are on RespawnZone types and name Mobius races (the
//     regions they name may be missing, as in Mobius).
func (w *World) checkZones(p *problems) {
	if len(w.Zones) == 0 {
		p.addf("%s: no zones", ZonesFile)
	}
	for i := range w.Zones {
		z := &w.Zones[i]
		where := fmt.Sprintf("%s: zone %d (%q, %s)", ZonesFile, z.ID, z.Name, z.File)
		if _, dup := w.zones[z.ID]; dup {
			p.addf("%s: duplicate zone ID", where)
			continue
		}
		w.zones[z.ID] = i
		if z.ID <= 0 {
			p.addf("%s: zone ID must be positive", where)
		}
		if !w.zoneTypes.usable(z.Type) {
			p.addf("%s: unknown zone type %q (not a usable class of %s)", where, z.Type, ZoneTypesFile)
		}
		z.Shape.check(p, where)
		for _, prm := range z.Params {
			if prm.Name == "" {
				p.addf("%s: parameter without a name", where)
			}
		}
		if len(z.Spawns) > 0 && !w.zoneTypes.is(z.Type, "ZoneRespawn") {
			p.addf("%s: spawn points on a %s, which is not a ZoneRespawn", where, z.Type)
		}
		for _, s := range z.Spawns {
			switch {
			case s.Type == ZoneSpawnNormal, s.Type == ZoneSpawnOther, s.Type == ZoneSpawnChaotic, s.Type == ZoneSpawnBanish:
			case s.Type == ZoneSpawnChallenger && w.zoneTypes.is(z.Type, "SiegableHallZone"):
			default:
				p.addf("%s: unknown spawn point type %q for a %s", where, s.Type, z.Type)
			}
		}
		if len(z.RacePoints) > 0 && !w.zoneTypes.is(z.Type, "RespawnZone") {
			p.addf("%s: race points on a %s, which is not a RespawnZone", where, z.Type)
		}
		for race, region := range z.RacePoints {
			if !race.Valid() || region == "" {
				p.addf("%s: race point %v: needs a known race and a map region (got %q)", where, race, region)
			}
		}
	}
}
