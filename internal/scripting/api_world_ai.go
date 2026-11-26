package scripting

import (
	"math"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// The world, state and cinematics modules of the AI API (Plan 13, Phase
// 10.6): world scans and geodata, zones, doors, game time, grand bosses
// and cinematics.

// worldAIFuncs are the world module's AI functions.
func (v *vm) worldAIFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"npcs_by_id": func(L *lua.LState) int {
			const fn = "world.npcs_by_id"
			id := int(checkInt(L, 1, fn))
			L.Push(v.npcList(L, worldAs[WorldScan](v, L, fn).NpcsByID(id)))
			return 1
		},
		"npcs_around": func(L *lua.LState) int {
			const fn = "world.npcs_around"
			obj := checkObject(L, 1, fn)
			r := int(optInt(L, 2, fn, 0))
			L.Push(v.npcList(L, worldAs[WorldScan](v, L, fn).NpcsAround(obj, r)))
			return 1
		},
		"players_around": func(L *lua.LState) int {
			const fn = "world.players_around"
			obj := checkObject(L, 1, fn)
			r := int(optInt(L, 2, fn, 0))
			L.Push(v.playerList(L, worldAs[WorldScan](v, L, fn).PlayersAround(obj, r)))
			return 1
		},
		"visible_objects": v.visibleObjects,
		"valid_location": func(L *lua.LState) int {
			const fn = "world.valid_location"
			from := locationArg(L, 1, fn)
			to := locationArg(L, 2, fn)
			if g, ok := v.needWorld(L, fn).(WorldGeo); ok {
				to = g.ValidLocation(from, to)
			}
			L.Push(locationTable(L, to))
			return 1
		},
	}
}

// visibleObjects is world.visible_objects(obj, class[, radius]): the
// objects of a Mobius class around obj (World.getVisibleObjects(obj,
// class)). "Player" and "Playable" are players, "Npc" every NPC, "Creature"
// and "WorldObject" both, and any other class the NPCs of that class
// (npc:is_type).
func (v *vm) visibleObjects(L *lua.LState) int {
	const fn = "world.visible_objects"
	obj := checkObject(L, 1, fn)
	class := L.OptString(2, "WorldObject")
	r := int(optInt(L, 3, fn, 0))
	scan := worldAs[WorldScan](v, L, fn)
	t := L.NewTable()
	players := class == "Player" || class == "Playable" || class == "Creature" || class == "WorldObject"
	if players {
		for _, p := range scan.PlayersAround(obj, r) {
			if p != nil {
				t.Append(v.newPlayer(L, p))
			}
		}
	}
	if class != "Player" && class != "Playable" {
		for _, n := range scan.NpcsAround(obj, r) {
			if n == nil {
				continue
			}
			if !players && class != "Npc" {
				d, ok := n.(NpcDetails)
				if !ok || !d.IsType(class) {
					continue
				}
			}
			t.Append(v.newNpc(L, n))
		}
	}
	L.Push(t)
	return 1
}

// playerList returns players as a Lua list.
func (v *vm) playerList(L *lua.LState, players []PlayerRef) *lua.LTable {
	t := L.CreateTable(len(players), 0)
	for _, p := range players {
		if p != nil {
			t.Append(v.newPlayer(L, p))
		}
	}
	return t
}

// timeAIFuncs are the time module's game time functions (Mobius
// GameTimeTaskManager).
func (v *vm) timeAIFuncs() map[string]lua.LGFunction {
	clock := func(L *lua.LState, fn string) GameClock { return worldAs[GameClock](v, L, fn) }
	return map[string]lua.LGFunction{
		"game_ticks": func(L *lua.LState) int {
			L.Push(lua.LNumber(clock(L, "time.game_ticks").GameTicks()))
			return 1
		},
		"game_time": func(L *lua.LState) int {
			L.Push(lua.LNumber(clock(L, "time.game_time").GameTime()))
			return 1
		},
		"game_hour": func(L *lua.LState) int {
			L.Push(lua.LNumber(clock(L, "time.game_hour").GameTime() / 60))
			return 1
		},
		"is_night": func(L *lua.LState) int {
			L.Push(lua.LBool(clock(L, "time.is_night").IsNight()))
			return 1
		},
	}
}

// calculateAngleFrom is util.calculate_angle_from(a, b): Mobius
// LocationUtil.calculateAngleFrom, the angle in degrees [0, 360) from a to
// b.
func calculateAngleFrom(L *lua.LState) int {
	const fn = "util.calculate_angle_from"
	a := locationArg(L, 1, fn)
	b := locationArg(L, 2, fn)
	angle := math.Atan2(float64(b.Y-a.Y), float64(b.X-a.X)) * 180 / math.Pi
	if angle < 0 {
		angle += 360
	}
	L.Push(lua.LNumber(angle))
	return 1
}

// zonesFuncs is the zones module (Mobius ZoneType and BossZone).
func (v *vm) zonesFuncs() map[string]lua.LGFunction {
	zones := func(L *lua.LState, fn string) (ZoneControl, int) {
		id := int(checkInt(L, 1, fn))
		return worldAs[ZoneControl](v, L, fn), id
	}
	m := map[string]lua.LGFunction{
		"players_inside": func(L *lua.LState) int {
			z, id := zones(L, "zones.players_inside")
			L.Push(v.playerList(L, z.ZonePlayers(id)))
			return 1
		},
		"is_inside": func(L *lua.LState) int {
			const fn = "zones.is_inside"
			z, id := zones(L, fn)
			var obj any
			if _, isPos := L.Get(2).(*lua.LTable); isPos {
				obj = locationArg(L, 2, fn) // a position (Mobius isInsideZone(x, y, z))
			} else {
				obj = checkObject(L, 2, fn)
			}
			L.Push(lua.LBool(z.ZoneIsInside(id, obj)))
			return 1
		},
		"move_players": func(L *lua.LState) int {
			const fn = "zones.move_players"
			z, id := zones(L, fn)
			z.ZoneMovePlayers(id, locationArg(L, 2, fn))
			return 0
		},
		"oust_all": func(L *lua.LState) int {
			z, id := zones(L, "zones.oust_all")
			z.ZoneOustAll(id)
			return 0
		},
		"allow_entry": func(L *lua.LState) int {
			const fn = "zones.allow_entry"
			z, id := zones(L, fn)
			p := checkPlayerArg(L, 2, fn)
			secs := checkInt(L, 3, fn)
			z.ZoneAllowEntry(id, p, time.Duration(secs)*time.Second)
			return 0
		},
		"remove_player": func(L *lua.LState) int {
			const fn = "zones.remove_player"
			z, id := zones(L, fn)
			z.ZoneRemovePlayer(id, checkPlayerArg(L, 2, fn))
			return 0
		},
	}
	for name, fn := range v.areaZonesFuncs() { // ai_areas.go
		m[name] = fn
	}
	return m
}

// doorsFuncs is the doors module (Mobius DoorData, Door).
func (v *vm) doorsFuncs() map[string]lua.LGFunction {
	doors := func(L *lua.LState, fn string) (DoorControl, int) {
		id := int(checkInt(L, 1, fn))
		return worldAs[DoorControl](v, L, fn), id
	}
	return map[string]lua.LGFunction{
		"open": func(L *lua.LState) int {
			d, id := doors(L, "doors.open")
			L.Push(lua.LBool(d.OpenDoor(id)))
			return 1
		},
		"close": func(L *lua.LState) int {
			d, id := doors(L, "doors.close")
			L.Push(lua.LBool(d.CloseDoor(id)))
			return 1
		},
		"is_open": func(L *lua.LState) int {
			d, id := doors(L, "doors.is_open")
			open, ok := d.DoorIsOpen(id)
			if !ok {
				L.Push(lua.LNil)
			} else {
				L.Push(lua.LBool(open))
			}
			return 1
		},
	}
}

// grandbossFuncs is the grandboss module (Mobius GrandBossManager).
func (v *vm) grandbossFuncs() map[string]lua.LGFunction {
	store := func(L *lua.LState, fn string) (GrandBossStore, int) {
		id := int(checkInt(L, 1, fn))
		return worldAs[GrandBossStore](v, L, fn), id
	}
	return map[string]lua.LGFunction{
		"status": func(L *lua.LState) int {
			s, id := store(L, "grandboss.status")
			if st, ok := s.BossStatus(id); ok {
				L.Push(lua.LNumber(st))
			} else {
				L.Push(lua.LNil)
			}
			return 1
		},
		"set_status": func(L *lua.LState) int {
			const fn = "grandboss.set_status"
			s, id := store(L, fn)
			s.SetBossStatus(id, int(checkInt(L, 2, fn)))
			return 0
		},
		"info": func(L *lua.LState) int {
			s, id := store(L, "grandboss.info")
			info, ok := s.BossInfo(id)
			if !ok {
				L.Push(lua.LNil)
				return 1
			}
			t := L.CreateTable(0, 7)
			t.RawSetString("respawn_time", lua.LNumber(info.RespawnTime))
			t.RawSetString("x", lua.LNumber(info.X))
			t.RawSetString("y", lua.LNumber(info.Y))
			t.RawSetString("z", lua.LNumber(info.Z))
			t.RawSetString("heading", lua.LNumber(info.Heading))
			t.RawSetString("hp", lua.LNumber(info.HP))
			t.RawSetString("mp", lua.LNumber(info.MP))
			L.Push(t)
			return 1
		},
		"set_info": func(L *lua.LState) int {
			const fn = "grandboss.set_info"
			s, id := store(L, fn)
			t := L.CheckTable(2)
			num := func(k string) float64 {
				n, _ := t.RawGetString(k).(lua.LNumber)
				return float64(n)
			}
			s.SetBossInfo(id, BossInfo{
				RespawnTime: int64(num("respawn_time")),
				X:           int(num("x")), Y: int(num("y")), Z: int(num("z")), Heading: int(num("heading")),
				HP: num("hp"), MP: num("mp"),
			})
			return 0
		},
		"boss": func(L *lua.LState) int {
			s, id := store(L, "grandboss.boss")
			L.Push(v.newNpc(L, s.LiveBoss(id)))
			return 1
		},
		"add_boss": func(L *lua.LState) int {
			const fn = "grandboss.add_boss"
			n := optNpcArg(L, 1, fn)
			s := worldAs[GrandBossStore](v, L, fn)
			if n == nil {
				L.ArgError(1, fn+": an NPC expected")
			}
			s.AddBoss(n)
			return 0
		},
	}
}

// audienceArg reads argument n, who sees a cinematic: a Player (that
// player), an Npc (the players around it) or a number (the players in
// that zone).
func audienceArg(L *lua.LState, n int, fn string) Audience {
	if num, ok := L.Get(n).(lua.LNumber); ok {
		id, ok := toID(num)
		if !ok {
			L.ArgError(n, fn+": a zone ID must be an integer")
		}
		return Audience{Zone: id}
	}
	switch x := checkObject(L, n, fn).(type) {
	case PlayerRef:
		return Audience{Player: x}
	case NpcRef:
		return Audience{Around: x}
	}
	L.ArgError(n, fn+": a player, an NPC or a zone ID expected")
	return Audience{}
}

// cinematicsFuncs is the cinematics module: SpecialCamera, Earthquake,
// PlaySound and ExShowScreenMessage for one player, the players around an
// object or a zone. Skill animations are npc:magic_skill_use and
// player:magic_skill_use.
func (v *vm) cinematicsFuncs() map[string]lua.LGFunction {
	cin := func(L *lua.LState, fn string) Cinematics { return worldAs[Cinematics](v, L, fn) }
	return map[string]lua.LGFunction{
		"special_camera": func(L *lua.LState) int {
			const fn = "cinematics.special_camera"
			to := audienceArg(L, 1, fn)
			target := checkObject(L, 2, fn)
			arg := func(i int) int { return int(optInt(L, i, fn, 0)) }
			cam := Camera{
				Force: arg(3), Angle1: arg(4), Angle2: arg(5), Time: arg(6), Range: arg(7), Duration: arg(8),
				RelYaw: arg(9), RelPitch: arg(10), Wide: arg(11), RelAngle: arg(12), Unk: arg(13),
			}
			cin(L, fn).SpecialCamera(to, target, cam)
			return 0
		},
		"earthquake": func(L *lua.LState) int {
			const fn = "cinematics.earthquake"
			to := audienceArg(L, 1, fn)
			loc := locationArg(L, 2, fn)
			intensity := int(checkInt(L, 3, fn))
			duration := int(checkInt(L, 4, fn))
			cin(L, fn).Earthquake(to, loc, intensity, duration)
			return 0
		},
		"sound": func(L *lua.LState) int {
			const fn = "cinematics.sound"
			to := audienceArg(L, 1, fn)
			name := L.CheckString(2)
			var source any
			if L.Get(3) != lua.LNil {
				source = checkObject(L, 3, fn)
			}
			cin(L, fn).PlaySoundFrom(to, name, source)
			return 0
		},
		"screen_message": func(L *lua.LState) int {
			const fn = "cinematics.screen_message"
			to := audienceArg(L, 1, fn)
			text := L.CheckString(2)
			ms := int(checkInt(L, 3, fn))
			pos := int(optInt(L, 4, fn, screenTopCenter))
			cin(L, fn).ScreenMessage(to, text, pos, ms)
			return 0
		},
	}
}

// screenTopCenter is ExShowScreenMessage.TOP_CENTER, the default position.
const screenTopCenter = 2
