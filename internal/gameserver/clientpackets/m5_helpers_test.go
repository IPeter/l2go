package clientpackets

import "testing"

// The helpers of the M5 parser tests (items and economy). Bodies are built
// with m3Body; results are checked with m3Parsed, m3Truncations and m3Fuzz.

// m5List is the body of a shop or warehouse packet as the client sends it:
// the header ints, the entry count, then the ints of every entry.
func m5List(header []int32, n int32, entries ...[]int32) []byte {
	b := new(m3Body)
	for _, v := range header {
		b.d(v)
	}
	b.d(n)
	for _, e := range entries {
		for _, v := range e {
			b.d(v)
		}
	}
	return b.b
}

// m5Repeat returns n copies of entry.
func m5Repeat(n int, entry []int32) [][]int32 {
	out := make([][]int32, n)
	for i := range out {
		out[i] = entry
	}
	return out
}

// m5FuzzList runs a list packet parser on fuzzed input. It must not panic.
// It fails exactly on a body shorter than the header, with ErrShortPacket
// and the zero value. A list it accepts has 1 to MaxItemsInPacket entries of
// entrySize bytes that fill the rest of the body, each passing ok.
func m5FuzzList[T, E any](f *testing.F, parse func([]byte) (T, error), items func(T) []E,
	header, entrySize int, ok func(E) bool, seeds ...[]byte,
) {
	f.Add([]byte{})
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := parse(data)
		if err != nil {
			m3Failed(t, "fuzzed input", got, err)
		}
		if (err == nil) != (len(data) >= header) {
			t.Fatalf("%d bytes: error %v, header of %d bytes", len(data), err, header)
		}
		list := items(got)
		if list == nil {
			return
		}
		if len(list) < 1 || len(list) > MaxItemsInPacket || header+len(list)*entrySize != len(data) {
			t.Errorf("%d bytes: %d entries accepted", len(data), len(list))
		}
		for _, e := range list {
			if !ok(e) {
				t.Errorf("entry %+v accepted", e)
			}
		}
	})
}
