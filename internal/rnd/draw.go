package rnd

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Kind is the Source method of a draw.
type Kind uint8

// The kinds, one per Source method. The zero Kind is none of them.
const (
	KindGet Kind = iota + 1
	KindBetween
	KindGetInt64
	KindBetweenInt64
	KindGetFloat32
	KindBetweenFloat32
	KindGetFloat64
	KindBetweenFloat64
	KindInt32
	KindInt64
	KindFloat32
	KindFloat64
	KindBool
	KindGaussian
)

var kindNames = [...]string{
	KindGet:            "Get",
	KindBetween:        "Between",
	KindGetInt64:       "GetInt64",
	KindBetweenInt64:   "BetweenInt64",
	KindGetFloat32:     "GetFloat32",
	KindBetweenFloat32: "BetweenFloat32",
	KindGetFloat64:     "GetFloat64",
	KindBetweenFloat64: "BetweenFloat64",
	KindInt32:          "Int32",
	KindInt64:          "Int64",
	KindFloat32:        "Float32",
	KindFloat64:        "Float64",
	KindBool:           "Bool",
	KindGaussian:       "Gaussian",
}

// String returns the name of the method.
func (k Kind) String() string {
	if int(k) < len(kindNames) && kindNames[k] != "" {
		return kindNames[k]
	}
	return fmt.Sprintf("Kind(%d)", uint8(k))
}

// is32 reports whether the method works in float32.
func (k Kind) is32() bool {
	return k == KindGetFloat32 || k == KindBetweenFloat32 || k == KindFloat32
}

// Draw is one draw: the Source method called, its arguments and the value it
// returned. A script is a list of draws (NewScripted), and a Recorder lists
// the draws it saw. Build a draw with the function named after its method:
// Get(100, 37) is a call of Get(100) that returned 37.
//
// Draws compare with ==. String shows a draw as "Get(100) = 37", and
// GoString as the Go code that builds it, so %#v prints a recording as a
// script ready to paste into a test.
type Draw struct {
	Kind Kind
	// Lo and Hi are the arguments of the integer methods. Get and GetInt64
	// have Lo 0 and Hi n.
	Lo, Hi int64
	// FLo and FHi are the arguments of the float methods, the same way.
	// float32 arguments are widened, exactly.
	FLo, FHi float64
	// Int is the value of the integer methods, Float that of the float
	// methods and Gaussian (float32 values widened), Bool that of Bool.
	Int   int64
	Float float64
	Bool  bool
}

// Get is a draw of Get(n) that returns v.
func Get(n, v int) Draw { return Draw{Kind: KindGet, Hi: int64(n), Int: int64(v)} }

// Between is a draw of Between(lo, hi) that returns v.
func Between(lo, hi, v int) Draw {
	return Draw{Kind: KindBetween, Lo: int64(lo), Hi: int64(hi), Int: int64(v)}
}

// GetInt64 is a draw of GetInt64(n) that returns v.
func GetInt64(n, v int64) Draw { return Draw{Kind: KindGetInt64, Hi: n, Int: v} }

// BetweenInt64 is a draw of BetweenInt64(lo, hi) that returns v.
func BetweenInt64(lo, hi, v int64) Draw {
	return Draw{Kind: KindBetweenInt64, Lo: lo, Hi: hi, Int: v}
}

// GetFloat32 is a draw of GetFloat32(bound) that returns v.
func GetFloat32(bound, v float32) Draw {
	return Draw{Kind: KindGetFloat32, FHi: float64(bound), Float: float64(v)}
}

// BetweenFloat32 is a draw of BetweenFloat32(lo, hi) that returns v.
func BetweenFloat32(lo, hi, v float32) Draw {
	return Draw{Kind: KindBetweenFloat32, FLo: float64(lo), FHi: float64(hi), Float: float64(v)}
}

// GetFloat64 is a draw of GetFloat64(bound) that returns v.
func GetFloat64(bound, v float64) Draw {
	return Draw{Kind: KindGetFloat64, FHi: bound, Float: v}
}

// BetweenFloat64 is a draw of BetweenFloat64(lo, hi) that returns v.
func BetweenFloat64(lo, hi, v float64) Draw {
	return Draw{Kind: KindBetweenFloat64, FLo: lo, FHi: hi, Float: v}
}

// Int32 is a draw of Int32() that returns v.
func Int32(v int32) Draw { return Draw{Kind: KindInt32, Int: int64(v)} }

// Int64 is a draw of Int64() that returns v.
func Int64(v int64) Draw { return Draw{Kind: KindInt64, Int: v} }

// Float32 is a draw of Float32() that returns v.
func Float32(v float32) Draw { return Draw{Kind: KindFloat32, Float: float64(v)} }

// Float64 is a draw of Float64() that returns v.
func Float64(v float64) Draw { return Draw{Kind: KindFloat64, Float: v} }

// Bool is a draw of Bool() that returns v.
func Bool(v bool) Draw { return Draw{Kind: KindBool, Bool: v} }

// Gaussian is a draw of Gaussian() that returns v.
func Gaussian(v float64) Draw { return Draw{Kind: KindGaussian, Float: v} }

// Call returns the call the draw answers, such as "Between(-5, 5)".
func (d Draw) Call() string {
	return d.Kind.String() + "(" + strings.Join(d.args(), ", ") + ")"
}

// String returns the call and its value, such as "Between(-5, 5) = 2".
func (d Draw) String() string { return d.Call() + " = " + d.value() }

// GoString returns the Go code that builds the draw, such as
// "rnd.Between(-5, 5, 2)".
func (d Draw) GoString() string {
	return "rnd." + d.Kind.String() + "(" + strings.Join(append(d.args(), d.value()), ", ") + ")"
}

// args returns the arguments of the call as Go literals.
func (d Draw) args() []string {
	switch d.Kind {
	case KindGet, KindGetInt64:
		return []string{strconv.FormatInt(d.Hi, 10)}
	case KindBetween, KindBetweenInt64:
		return []string{strconv.FormatInt(d.Lo, 10), strconv.FormatInt(d.Hi, 10)}
	case KindGetFloat32, KindGetFloat64:
		return []string{d.fmtFloat(d.FHi)}
	case KindBetweenFloat32, KindBetweenFloat64:
		return []string{d.fmtFloat(d.FLo), d.fmtFloat(d.FHi)}
	}
	return nil
}

// value returns the value as a Go literal.
func (d Draw) value() string {
	switch d.Kind {
	case KindBool:
		return strconv.FormatBool(d.Bool)
	case KindGetFloat32, KindBetweenFloat32, KindGetFloat64, KindBetweenFloat64,
		KindFloat32, KindFloat64, KindGaussian:
		return d.fmtFloat(d.Float)
	}
	return strconv.FormatInt(d.Int, 10)
}

// fmtFloat formats x as the shortest literal that reads back as the same
// float32 or float64, whichever the method uses.
func (d Draw) fmtFloat(x float64) string {
	bits := 64
	if d.Kind.is32() {
		bits = 32
	}
	return strconv.FormatFloat(x, 'g', -1, bits)
}

// sameCall reports whether d and call are the same method with the same
// arguments.
func (d Draw) sameCall(call Draw) bool {
	return d.Kind == call.Kind && d.Lo == call.Lo && d.Hi == call.Hi &&
		d.FLo == call.FLo && d.FHi == call.FHi
}

// hiIncluded reports whether Hi is a possible value of an integer draw.
// Between and BetweenInt64 include it, as Mobius adds 1 to the Java bound,
// except at the top of the Java type, where it cannot.
func (d Draw) hiIncluded() bool {
	switch d.Kind {
	case KindBetween:
		return d.Hi != math.MaxInt32
	case KindBetweenInt64:
		return d.Hi != math.MaxInt64
	}
	return false
}

// floatRange returns the range [lo, top) of a float draw's values.
func (d Draw) floatRange() (lo, top float64) {
	switch d.Kind {
	case KindGetFloat32, KindGetFloat64:
		return 0, d.FHi
	case KindBetweenFloat32:
		return d.FLo, float64(widen32(float32(d.FHi)))
	case KindBetweenFloat64:
		return d.FLo, widen64(d.FHi)
	}
	return 0, 1
}

// badValue says why the draw's value is one its call can never return, or
// returns "" when the call can return it.
func (d Draw) badValue() string {
	switch d.Kind {
	case KindGet, KindGetInt64:
		if d.Int < 0 || d.Int >= d.Hi {
			return fmt.Sprintf("outside [0, %d)", d.Hi)
		}
	case KindBetween, KindBetweenInt64:
		switch inc := d.hiIncluded(); {
		case inc && (d.Int < d.Lo || d.Int > d.Hi):
			return fmt.Sprintf("outside [%d, %d]", d.Lo, d.Hi)
		case !inc && (d.Int < d.Lo || d.Int >= d.Hi):
			return fmt.Sprintf("outside [%d, %d)", d.Lo, d.Hi)
		}
	case KindInt32:
		if d.Int != int64(int32(d.Int)) { //nolint:gosec // the truncation is the check
			return "not an int32"
		}
	case KindGetFloat32, KindBetweenFloat32, KindGetFloat64, KindBetweenFloat64,
		KindFloat32, KindFloat64:
		if lo, top := d.floatRange(); !(d.Float >= lo && d.Float < top) {
			return "outside [" + d.fmtFloat(lo) + ", " + d.fmtFloat(top) + ")"
		}
	case KindGaussian:
		if math.IsNaN(d.Float) || math.IsInf(d.Float, 0) {
			return "not finite"
		}
	}
	return ""
}

// widen64 is Mobius's bound + MINIMUM_POSITIVE_DOUBLE in Rnd.get(double,
// double), meant to include the bound. The sum rounds back to hi whenever
// |hi| >= 2^-1020. Double.MAX_VALUE is kept as it is.
func widen64(hi float64) float64 {
	if hi == math.MaxFloat64 {
		return hi
	}
	return hi + math.SmallestNonzeroFloat64
}

// widen32 is bound + MINIMUM_POSITIVE_FLOAT in Rnd.get(float, float), which
// rounds back to hi whenever |hi| >= 2^-124. Float.MAX_VALUE is kept.
func widen32(hi float32) float32 {
	if hi == math.MaxFloat32 {
		return hi
	}
	return hi + math.SmallestNonzeroFloat32
}
