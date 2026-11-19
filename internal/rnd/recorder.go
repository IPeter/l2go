package rnd

import "slices"

// Recorder is a Source that passes each call to another Source and keeps the
// draws made, in order, with their values. A recording replays as a script,
// NewScripted(t, rec.Draws()...), and %#v prints it as the Go code of one.
type Recorder struct {
	rules
	src   Source
	draws []Draw
}

// NewRecorder returns a source that draws from src and records the draws.
func NewRecorder(src Source) *Recorder {
	r := &Recorder{src: src}
	r.rules = rules{r}
	return r
}

// Draws returns the draws made so far.
func (r *Recorder) Draws() []Draw { return slices.Clone(r.draws) }

// Reset forgets the draws made so far.
func (r *Recorder) Reset() { r.draws = nil }

func (r *Recorder) draw(call Draw) Draw {
	d := call.callOn(r.src)
	r.draws = append(r.draws, d)
	return d
}

// callOn makes the call d describes on src and returns d with the value.
func (d Draw) callOn(src Source) Draw {
	switch d.Kind {
	case KindGet:
		d.Int = int64(src.Get(int(d.Hi)))
	case KindBetween:
		d.Int = int64(src.Between(int(d.Lo), int(d.Hi)))
	case KindGetInt64:
		d.Int = src.GetInt64(d.Hi)
	case KindBetweenInt64:
		d.Int = src.BetweenInt64(d.Lo, d.Hi)
	case KindGetFloat32:
		d.Float = float64(src.GetFloat32(float32(d.FHi)))
	case KindBetweenFloat32:
		d.Float = float64(src.BetweenFloat32(float32(d.FLo), float32(d.FHi)))
	case KindGetFloat64:
		d.Float = src.GetFloat64(d.FHi)
	case KindBetweenFloat64:
		d.Float = src.BetweenFloat64(d.FLo, d.FHi)
	case KindInt32:
		d.Int = int64(src.Int32())
	case KindInt64:
		d.Int = src.Int64()
	case KindFloat32:
		d.Float = float64(src.Float32())
	case KindFloat64:
		d.Float = src.Float64()
	case KindBool:
		d.Bool = src.Bool()
	case KindGaussian:
		d.Float = src.Gaussian()
	}
	return d
}
