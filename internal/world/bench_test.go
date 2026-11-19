package world_test

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/world"
	"github.com/IPeter/l2go/internal/world/worldtest"
)

// BenchmarkMoveTick moves 10,000 objects (a tenth of them players) one tick
// each, at run speed, over 32×32 regions, and walks every Delta's pairs as
// the game layer would. One op is one tick; Plan 05 wants it well under
// 10 ms.
func BenchmarkMoveTick(b *testing.B) {
	const (
		n     = 10000
		area  = 32 * world.RegionSize
		speed = 25.0 // units per 100 ms tick: 250 units/s
	)
	w := world.New(world.NewLoop(world.LoopOptions{}))
	rng := rand.New(rand.NewPCG(1, 2))
	type mover struct {
		obj    world.Object
		vx, vy int32
	}
	movers := make([]mover, n)
	for i := range movers {
		pos := world.Position{X: rng.Int32N(area), Y: rng.Int32N(area), Z: -3000}
		var o world.Object
		if i%10 == 0 {
			o = worldtest.NewPlayer(world.ObjectID(i+1), pos)
		} else {
			o = worldtest.NewObject(world.ObjectID(i+1), pos)
		}
		if _, err := w.Add(o); err != nil {
			b.Fatal(err)
		}
		angle := rng.Float64() * 2 * math.Pi
		movers[i] = mover{obj: o, vx: int32(speed * math.Cos(angle)), vy: int32(speed * math.Sin(angle))}
	}

	pairs := 0
	for b.Loop() {
		for i := range movers {
			m := &movers[i]
			p := m.obj.Position()
			p.X += m.vx
			p.Y += m.vy
			if p.X < 0 || p.X >= area {
				m.vx = -m.vx
				p.X += 2 * m.vx
			}
			if p.Y < 0 || p.Y >= area {
				m.vy = -m.vy
				p.Y += 2 * m.vy
			}
			d, err := w.Move(m.obj, p)
			if err != nil {
				b.Fatal(err)
			}
			for range d.Disappear() {
				pairs++
			}
			for range d.Appear() {
				pairs++
			}
		}
	}
	b.ReportMetric(float64(pairs)/float64(b.N), "pairs/op")
}

// BenchmarkBroadcastTown sends one packet from a player to itself and the
// 499 others crowding one region of Giran.
func BenchmarkBroadcastTown(b *testing.B) {
	w := world.New(world.NewLoop(world.LoopOptions{}))
	rng := rand.New(rand.NewPCG(3, 4))
	var sent int
	sink := world.SinkFunc(func([]byte) { sent++ })
	players := make([]*benchPlayer, 500)
	for i := range players {
		pos := offset(at(104, 200, 6), rng.Int32N(2000)-1000, rng.Int32N(2000)-1000, 0)
		players[i] = &benchPlayer{Base: world.NewBase(world.ObjectID(i+1), pos), sink: sink}
		if _, err := w.Add(players[i]); err != nil {
			b.Fatal(err)
		}
	}
	body := []byte("move")
	for b.Loop() {
		w.Broadcast(players[0], body, true)
	}
	b.ReportMetric(float64(sent)/float64(b.N), "sends/op")
}

// BenchmarkTimers schedules and fires 10,000 timers.
func BenchmarkTimers(b *testing.B) {
	clk := world.NewFakeClock(time.Unix(0, 0))
	l := world.NewLoop(world.LoopOptions{Clock: clk})
	fired := 0
	fn := func() { fired++ }
	for b.Loop() {
		for i := range 10000 {
			l.After(time.Duration(i%1000)*time.Millisecond, fn)
		}
		clk.Advance(time.Second)
		if err := l.Step(); err != nil {
			b.Fatal(err)
		}
	}
	if fired != 10000*b.N {
		b.Fatalf("fired %d timers, want %d", fired, 10000*b.N)
	}
}

// benchPlayer is a viewer with a shared counting sink.
type benchPlayer struct {
	world.Base
	sink world.Sink
}

func (p *benchPlayer) Sink() world.Sink { return p.sink }

func (p *benchPlayer) SeesInvisible() bool { return false }
