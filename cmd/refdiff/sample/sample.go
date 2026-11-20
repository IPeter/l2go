// Package sample is the sample mode of cmd/refdiff (Plan 09, "Reference
// diff, sample mode"). Single fights are random, so instead of comparing
// packets it compares the statistics of many swings recorded on both
// servers: the mean damage, the miss and critical rates (each within 3
// standard errors) and the time between swings (the medians within 50
// ms).
//
// The plan's third check, that every damage lies inside the [min, max]
// envelope the combat formulas allow, needs L2Go's internal/combat package
// (Milestone 6); until then the report shows each side's damage range.
package sample

import (
	"fmt"
	"io"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/IPeter/l2go/cmd/refdiff/scenario"
	"github.com/IPeter/l2go/internal/testclient"
)

// Limits of the checks.
const (
	// MaxSigma is how many standard errors two means or rates may differ.
	MaxSigma = 3.0
	// MaxSpacingDiff is how far the median swing spacings may differ.
	MaxSpacingDiff = 50 * time.Millisecond
	// maxGap is the longest time between two swings at the same target
	// that counts as the swing rhythm; longer gaps are walks or pauses.
	maxGap = 5 * time.Second
)

// Stats are the swings of one character in one capture.
type Stats struct {
	// Server labels the side.
	Server string
	// Swings counts the character's Attack packets, Hits their hits (two
	// per swing for dual weapons and fists), Misses and Crits the missed
	// and critical hits.
	Swings, Hits, Misses, Crits int
	// Damage is the damage of every hit that landed.
	Damage []int64
	// Gaps are the times between consecutive swings at the same target.
	Gaps []time.Duration
}

// Collect gathers the swings of role's character from the game streams of
// c. The character is the one its first CharSelected names.
func Collect(c *scenario.Capture, role string) (Stats, error) {
	st := Stats{Server: c.Target.Label}
	var (
		me         int32
		last       time.Time
		lastTarget int32
	)
	for _, s := range c.Streams {
		if !isGameStream(s.Name, role) {
			continue
		}
		for _, p := range s.Packets {
			switch v := p.Value.(type) {
			case testclient.CharSelected:
				if me == 0 {
					me = v.ObjectID
				}
			case testclient.Attack:
				if me == 0 || v.AttackerID != me || len(v.Hits) == 0 {
					continue
				}
				st.add(v)
				target := v.Hits[0].TargetID
				if !last.IsZero() && target == lastTarget {
					if gap := p.Time.Sub(last); gap > 0 && gap <= maxGap {
						st.Gaps = append(st.Gaps, gap)
					}
				}
				last, lastTarget = p.Time, target
			}
		}
	}
	if me == 0 {
		return st, fmt.Errorf("sample: %s has no character of role %s", c.Target.Label, role)
	}
	return st, nil
}

// isGameStream reports whether name is a game stream of role: "game alice"
// or "game alice 2".
func isGameStream(name, role string) bool {
	f := strings.Fields(name)
	return len(f) >= 2 && f[0] == "game" && f[1] == role
}

// add counts one swing.
func (s *Stats) add(a testclient.Attack) {
	s.Swings++
	for _, h := range a.Hits {
		s.Hits++
		switch {
		case h.Miss():
			s.Misses++
		default:
			if h.Critical() {
				s.Crits++
			}
			s.Damage = append(s.Damage, int64(h.Damage))
		}
	}
}

// meanSE returns the mean of xs and its standard error.
func meanSE(xs []int64) (mean, se float64) {
	n := float64(len(xs))
	if n == 0 {
		return 0, 0
	}
	for _, x := range xs {
		mean += float64(x)
	}
	mean /= n
	if n < 2 {
		return mean, 0
	}
	var ss float64
	for _, x := range xs {
		d := float64(x) - mean
		ss += d * d
	}
	return mean, math.Sqrt(ss / (n - 1) / n)
}

// rateSE returns k/n and its standard error.
func rateSE(k, n int) (rate, se float64) {
	if n == 0 {
		return 0, 0
	}
	p := float64(k) / float64(n)
	return p, math.Sqrt(p * (1 - p) / float64(n))
}

// median returns the median of ds, 0 for none.
func median(ds []time.Duration) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	s := slices.Clone(ds)
	slices.Sort(s)
	if len(s)%2 == 1 {
		return s[len(s)/2]
	}
	return (s[len(s)/2-1] + s[len(s)/2]) / 2
}

// Check is one comparison of the two sides.
type Check struct {
	Name        string
	Left, Right string
	OK          bool
	// Info marks a line that only informs, without a verdict.
	Info   bool
	Detail string
}

// Report is the outcome of a sample comparison.
type Report struct {
	Left, Right Stats
	Checks      []Check
}

// OK reports whether every check passed.
func (r Report) OK() bool {
	return !slices.ContainsFunc(r.Checks, func(c Check) bool { return !c.Info && !c.OK })
}

// Compare compares the swings of the two sides. Each side needs at least
// minSwings swings.
func Compare(l, r Stats, minSwings int) Report {
	rep := Report{Left: l, Right: r}
	add := func(c Check) { rep.Checks = append(rep.Checks, c) }

	add(Check{
		Name: "swings", Left: fmt.Sprint(l.Swings), Right: fmt.Sprint(r.Swings),
		OK: l.Swings >= minSwings && r.Swings >= minSwings, Detail: fmt.Sprintf("at least %d each", minSwings),
	})

	ml, sl := meanSE(l.Damage)
	mr, sr := meanSE(r.Damage)
	add(within("mean damage", ml, sl, mr, sr, "%.2f ± %.2f"))

	pl, el := rateSE(l.Misses, l.Hits)
	pr, er := rateSE(r.Misses, r.Hits)
	add(within("miss rate", pl, el, pr, er, "%.3f ± %.3f"))

	cl, ecl := rateSE(l.Crits, l.Hits-l.Misses)
	cr, ecr := rateSE(r.Crits, r.Hits-r.Misses)
	add(within("critical rate", cl, ecl, cr, ecr, "%.3f ± %.3f"))

	gl, gr := median(l.Gaps), median(r.Gaps)
	diff := gl - gr
	if diff < 0 {
		diff = -diff
	}
	add(Check{
		Name: "swing spacing (median)", Left: gl.Round(time.Millisecond).String(), Right: gr.Round(time.Millisecond).String(),
		OK: len(l.Gaps) > 0 && len(r.Gaps) > 0 && diff <= MaxSpacingDiff, Detail: fmt.Sprintf("within %v", MaxSpacingDiff),
	})

	add(Check{
		Name: "damage range", Left: damageRange(l.Damage), Right: damageRange(r.Damage), Info: true,
		Detail: "the envelope check needs internal/combat",
	})
	return rep
}

// within checks that two estimates differ by at most MaxSigma combined
// standard errors.
func within(name string, a, sa, b, sb float64, format string) Check {
	se := math.Sqrt(sa*sa + sb*sb)
	d := math.Abs(a - b)
	ok := d <= MaxSigma*se || d < 1e-9
	sigma := "0"
	if se > 0 {
		sigma = fmt.Sprintf("%.1f", d/se)
	}
	return Check{
		Name: name, Left: fmt.Sprintf(format, a, sa), Right: fmt.Sprintf(format, b, sb),
		OK: ok, Detail: fmt.Sprintf("%s standard errors apart, at most %.0f", sigma, MaxSigma),
	}
}

// damageRange formats the lowest and highest damage.
func damageRange(ds []int64) string {
	if len(ds) == 0 {
		return "-"
	}
	return fmt.Sprintf("%d-%d", slices.Min(ds), slices.Max(ds))
}

// WriteText writes the report as a table.
func WriteText(w io.Writer, scenarioName string, rep Report) error {
	var b strings.Builder
	fmt.Fprintf(&b, "sample %s: %s (left) vs %s (right)\n", scenarioName, rep.Left.Server, rep.Right.Server)
	fmt.Fprintf(&b, "  %-24s %-20s %-20s %s\n", "check", rep.Left.Server, rep.Right.Server, "verdict")
	for _, c := range rep.Checks {
		verdict := "ok"
		switch {
		case c.Info:
			verdict = "info"
		case !c.OK:
			verdict = "FAIL"
		}
		fmt.Fprintf(&b, "  %-24s %-20s %-20s %s (%s)\n", c.Name, c.Left, c.Right, verdict, c.Detail)
	}
	if rep.OK() {
		b.WriteString("the samples agree\n")
	} else {
		b.WriteString("the samples differ\n")
	}
	_, err := io.WriteString(w, b.String())
	return err
}
