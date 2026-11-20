package quest

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/storage"
)

// Vars are string variables with the typed getters of Mobius StatSet, as
// used for player variables (PlayerVariables, stored in the character
// document) and global variables (GlobalVariablesManager), decision Q15.
// Values are stored as strings, as Mobius writes them: String.valueOf of the
// value. Vars track which names changed since the last TakeChanges. The zero
// value is empty and ready to use.
type Vars struct {
	m       map[string]string
	changed map[string]struct{}
}

// NewVars returns variables holding a copy of m, with no changes.
func NewVars(m map[string]string) *Vars {
	return &Vars{m: maps.Clone(m)}
}

// Get returns a variable; false if it is not set.
func (v *Vars) Get(name string) (string, bool) {
	s, ok := v.m[name]
	return s, ok
}

// Has reports whether a variable is set (Mobius hasVariable).
func (v *Vars) Has(name string) bool {
	_, ok := v.m[name]
	return ok
}

// String returns a variable, or def if it is not set (StatSet getString).
func (v *Vars) String(name, def string) string {
	if s, ok := v.m[name]; ok {
		return s
	}
	return def
}

// Int returns a variable as a Java int, or def if it is not set (StatSet
// getInt). A value that is not an int is an error, where Mobius throws.
func (v *Vars) Int(name string, def int32) (int32, error) {
	s, ok := v.m[name]
	if !ok {
		return def, nil
	}
	n, ok := parseInt(s)
	if !ok {
		return def, &VarError{Var: name, Value: s}
	}
	return n, nil
}

// Long returns a variable as a Java long, or def if it is not set (StatSet
// getLong). A value that is not a long is an error, where Mobius throws.
func (v *Vars) Long(name string, def int64) (int64, error) {
	s, ok := v.m[name]
	if !ok {
		return def, nil
	}
	n, ok := parseLong(s)
	if !ok {
		return def, &VarError{Var: name, Value: s}
	}
	return n, nil
}

// Bool returns a variable as a boolean, or def if it is not set (StatSet
// getBoolean): true only for "true" in any letter case.
func (v *Vars) Bool(name string, def bool) bool {
	s, ok := v.m[name]
	if !ok {
		return def
	}
	return strings.EqualFold(s, "true")
}

// Set sets a variable.
func (v *Vars) Set(name, value string) {
	if v.m == nil {
		v.m = make(map[string]string)
	}
	v.m[name] = value
	v.mark(name)
}

// SetInt sets a variable to a Java int.
func (v *Vars) SetInt(name string, value int32) { v.Set(name, formatInt(value)) }

// SetLong sets a variable to a Java long.
func (v *Vars) SetLong(name string, value int64) { v.Set(name, strconv.FormatInt(value, 10)) }

// SetBool sets a variable to "true" or "false".
func (v *Vars) SetBool(name string, value bool) { v.Set(name, strconv.FormatBool(value)) }

// Remove removes a variable (Mobius remove).
func (v *Vars) Remove(name string) {
	if _, ok := v.m[name]; !ok {
		return
	}
	delete(v.m, name)
	v.mark(name)
}

// Map returns a copy of the variables, nil if there are none: the
// character's Vars for a save.
func (v *Vars) Map() map[string]string {
	if len(v.m) == 0 {
		return nil
	}
	return maps.Clone(v.m)
}

// Dirty reports whether variables changed since the last TakeChanges.
func (v *Vars) Dirty() bool { return len(v.changed) > 0 }

// TakeChanges returns the changes since the last call, as they are now, and
// forgets them: the global variables' save.
func (v *Vars) TakeChanges() storage.VarChanges {
	var ch storage.VarChanges
	for _, name := range slices.Sorted(maps.Keys(v.changed)) {
		if s, ok := v.m[name]; ok {
			if ch.Set == nil {
				ch.Set = make(map[string]string)
			}
			ch.Set[name] = s
		} else {
			ch.Deleted = append(ch.Deleted, name)
		}
	}
	clear(v.changed)
	return ch
}

// SaveFailed marks the variables of changes that could not be written as
// changed again.
func (v *Vars) SaveFailed(changes storage.VarChanges) {
	for name := range changes.Set {
		v.mark(name)
	}
	for _, name := range changes.Deleted {
		v.mark(name)
	}
}

func (v *Vars) mark(name string) {
	if v.changed == nil {
		v.changed = make(map[string]struct{})
	}
	v.changed[name] = struct{}{}
}
