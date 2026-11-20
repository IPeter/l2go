package recording

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// leaf is one field of a decoded packet before rendering.
type leaf struct {
	// path is the field's path in the packet struct: "X",
	// "Characters[0].Name", "Items.len" for the length of a slice.
	path string
	// val is the formatted value: a number, true or false, hex for byte
	// arrays, the text of a string (unquoted), or a placeholder.
	val string
	// kind is set for object ID fields: the kind of object the ID names,
	// used when the ID is first seen. id is the raw ID.
	kind string
	id   int32
	// text marks a string field.
	text bool
}

// flatten lists the fields of a decoded packet depth first, in
// declaration order. Byte arrays and slices become one hex value, other
// slices get a ".len" entry before their elements, and embedded structs
// add their fields to the parent's path.
func flatten(v any) []leaf {
	var out []leaf
	walk(&out, "", reflect.ValueOf(v))
	return out
}

func joinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

var stringerType = reflect.TypeFor[fmt.Stringer]()

func walk(out *[]leaf, path string, v reflect.Value) {
	if !v.IsValid() {
		*out = append(*out, leaf{path: path, val: "nil"})
		return
	}
	t := v.Type()
	if (t.Kind() == reflect.Slice || t.Kind() == reflect.Array) && t.Elem().Kind() == reflect.Uint8 {
		*out = append(*out, leaf{path: path, val: hexBytes(v)})
		return
	}
	if s, ok := stringValue(v); ok {
		*out = append(*out, leaf{path: path, val: s})
		return
	}
	switch v.Kind() {
	case reflect.Struct:
		walkStruct(out, path, v)
	case reflect.Slice:
		*out = append(*out, leaf{path: path + ".len", val: strconv.Itoa(v.Len())})
		for i := range v.Len() {
			walk(out, fmt.Sprintf("%s[%d]", path, i), v.Index(i))
		}
	case reflect.Array:
		for i := range v.Len() {
			walk(out, fmt.Sprintf("%s[%d]", path, i), v.Index(i))
		}
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			*out = append(*out, leaf{path: path, val: "nil"})
			return
		}
		walk(out, path, v.Elem())
	case reflect.String:
		*out = append(*out, leaf{path: path, val: v.String(), text: true})
	case reflect.Bool:
		*out = append(*out, leaf{path: path, val: strconv.FormatBool(v.Bool())})
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		*out = append(*out, leaf{path: path, val: strconv.FormatInt(v.Int(), 10)})
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		*out = append(*out, leaf{path: path, val: strconv.FormatUint(v.Uint(), 10)})
	case reflect.Float32, reflect.Float64:
		*out = append(*out, leaf{path: path, val: strconv.FormatFloat(v.Float(), 'g', -1, 64)})
	default:
		if v.CanInterface() {
			*out = append(*out, leaf{path: path, val: fmt.Sprint(v.Interface())})
		}
	}
}

// walkStruct lists the exported fields of a struct. A struct with fields
// but none exported (such as netip.Addr without a String method) is
// printed whole.
func walkStruct(out *[]leaf, path string, v reflect.Value) {
	t := v.Type()
	exported := 0
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		exported++
		p := joinPath(path, f.Name)
		if f.Anonymous {
			p = path
		}
		walk(out, p, v.Field(i))
	}
	if exported == 0 && t.NumField() > 0 && v.CanInterface() {
		*out = append(*out, leaf{path: path, val: fmt.Sprint(v.Interface())})
	}
}

// stringValue returns the String() of enums (named numbers with a String
// method) and of opaque structs such as netip.Addr.
func stringValue(v reflect.Value) (string, bool) {
	if !v.CanInterface() || !v.Type().Implements(stringerType) {
		return "", false
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map, reflect.Array:
		return "", false
	case reflect.Struct:
		if hasExportedFields(v.Type()) {
			return "", false
		}
	}
	s, ok := v.Interface().(fmt.Stringer)
	if !ok {
		return "", false
	}
	return s.String(), true
}

func hasExportedFields(t reflect.Type) bool {
	for i := range t.NumField() {
		if t.Field(i).IsExported() {
			return true
		}
	}
	return false
}

// hexBytes returns a byte slice or array in hex.
func hexBytes(v reflect.Value) string {
	b := make([]byte, v.Len())
	for i := range b {
		b[i] = byte(v.Index(i).Uint()) //nolint:gosec // the elements are bytes
	}
	return hex.EncodeToString(b)
}

// fieldName returns the last field name of path without its index:
// "Items[2].ObjectID" gives "ObjectID", "PaperdollObjectIDs[7]" gives
// "PaperdollObjectIDs".
func fieldName(path string) string {
	if i := strings.LastIndexByte(path, '.'); i >= 0 {
		path = path[i+1:]
	}
	if i := strings.IndexByte(path, '['); i >= 0 {
		path = path[:i]
	}
	return path
}

// genericPath replaces every index of path with []: "Servers[0].IP" gives
// "Servers[].IP". The rule tables use it.
func genericPath(path string) string {
	if !strings.Contains(path, "[") {
		return path
	}
	var b strings.Builder
	for i := 0; i < len(path); i++ {
		c := path[i]
		b.WriteByte(c)
		if c != '[' {
			continue
		}
		for i+1 < len(path) && path[i+1] != ']' {
			i++
		}
	}
	return b.String()
}
