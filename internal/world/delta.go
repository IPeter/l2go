package world

import "iter"

// Delta is the visibility change caused by adding, moving or removing one
// object, the subject. Only regions that enter or leave the subject's 3×3×3
// neighbourhood count; objects that stay in view are not listed (Mobius
// World.addVisibleObject, switchRegion and removeVisibleObject).
//
// Send its packets with Disappear first (DeleteObject), then Appear (the
// info packet for the viewer: CharInfo, NpcInfo, ...; plus MoveToLocation
// for an object that is moving, so the new viewer sees it walk).
//
// A Delta is a snapshot. Use it right away, on the loop.
type Delta struct {
	// Subject is the object that was added, moved or removed.
	Subject Object
	// Entered holds the other objects in the regions that came into the
	// subject's neighbourhood (for Add, the whole neighbourhood), in the
	// subject's instance.
	Entered []Object
	// Left holds the objects in the regions that went out of it (for
	// Remove, the whole neighbourhood), in every instance, as in Mobius.
	Left []Object

	w *World
}

// Empty reports whether nothing entered or left.
func (d Delta) Empty() bool { return len(d.Entered) == 0 && len(d.Left) == 0 }

// Appear yields (viewer, object) for every object a viewer starts to see:
// the subject, when it is a viewer, seeing each entered object, and each
// entered viewer seeing the subject. Pairs that fail World.CanSee are
// skipped (invisible GMs). For each entered object the subject's pair comes
// first, as in Mobius.
func (d Delta) Appear() iter.Seq2[Viewer, Object] {
	return func(yield func(Viewer, Object) bool) {
		if d.Subject == nil {
			return
		}
		sv, _ := d.Subject.(Viewer)
		for _, o := range d.Entered {
			if sv != nil && d.w.CanSee(sv, o) && !yield(sv, o) {
				return
			}
			if ov := o.base().viewer; ov != nil && d.w.CanSee(ov, d.Subject) && !yield(ov, d.Subject) {
				return
			}
		}
	}
}

// Disappear yields (viewer, object) for every DeleteObject to send: the
// subject, when it is a viewer, losing each left object, and each left
// viewer losing the subject. As in Mobius it ignores the visibility rule:
// deleting an object a client never saw is harmless, and this way none
// stays behind. For each left object the subject's pair comes first.
func (d Delta) Disappear() iter.Seq2[Viewer, Object] {
	return func(yield func(Viewer, Object) bool) {
		if d.Subject == nil {
			return
		}
		sv, _ := d.Subject.(Viewer)
		for _, o := range d.Left {
			if sv != nil && !yield(sv, o) {
				return
			}
			if ov := o.base().viewer; ov != nil && !yield(ov, d.Subject) {
				return
			}
		}
	}
}
