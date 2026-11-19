// Package world is the shared game world: the loop that owns it, the objects
// in it, who sees whom, and the fan-out of encoded packets.
//
// # Threading
//
// One goroutine, the loop (Loop.Run), owns all world state (decision W1).
// Other goroutines hand it work with Loop.Post, Loop.PostGuarded or
// Loop.Call. Everything else is loop-only: call it from a command, a timer or
// a tick hook. That covers World, Base and Delta methods, Loop.After, Every,
// Cancel, OnTick, OnTickEnd, RemoveHook and Now, and
// GameTime.OnDayNightChange. Before Run starts (or between Steps in tests)
// the one goroutine setting things up may call them too.
//
// Safe from any goroutine: Loop.Post, PostGuarded, Call, Run, Step, Done,
// OnLoop, Clock and TickInterval; World.Loop, GameTime and IDs, which return
// fixed values; GameTime.Ticks, Minutes, Hour, Minute and IsNight; the
// clocks; IDAllocator.
//
// LoopOptions.CheckOwner makes loop-only methods verify the goroutine they
// run on. A violation is reported and the call does nothing (methods with an
// error result return ErrNotOnLoop), so a test sees the bug instead of a
// corrupted world.
//
// # Visibility
//
// The world is cut into regions of 2048×2048 units and 16 layers of 2000
// units in Z (Mobius World). An object sees what is in the 3×3×3 regions
// around its own, in the same instance (decision W5). There are no known
// lists: Add, Move and Remove return a Delta built from region differences,
// and the game layer turns it into packets, Disappear first, then Appear:
//
//	d, err := w.Move(p, pos)
//	for v, o := range d.Disappear() {
//		world.Send(v, deleteObject(o)) // DeleteObject
//	}
//	for v, o := range d.Appear() {
//		world.Send(v, infoFor(v, o)) // CharInfo, NpcInfo, ...
//		if moving(o) {
//			world.Send(v, moveToLocation(o)) // a new viewer sees it walk
//		}
//	}
//
// Appear applies the visibility rule (World.CanSee: invisible GMs); like
// Mobius, Disappear does not, so a client never keeps a stale object. Mobius
// also clears targets and makes creatures forget each other for every pair
// that loses sight; Delta.Left lists all of those objects, players or not.
//
// # Broadcasting
//
// Packets are encoded once (decision W6). World.Broadcast and friends hand the
// same body to every receiver's Sink, whose writer encrypts a copy and sends
// it. Bodies are therefore read-only once handed over. Refreshes such as
// UserInfo and CharInfo are coalesced per tick with World.MarkUpdate.
package world
