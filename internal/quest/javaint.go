package quest

import "strconv"

// Java int arithmetic. Quest variables are strings that Mobius parses with
// Integer.parseInt and Long.parseLong, and the cond flags are Java ints with
// Java's wrap-around and shift rules; these helpers keep both.

// parseInt is Integer.parseInt: an optional sign and decimal digits that fit
// in 32 bits, nothing else.
func parseInt(s string) (int32, bool) {
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, false
	}
	return int32(n), true
}

// parseLong is Long.parseLong.
func parseLong(s string) (int64, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// formatInt is Integer.toString and String.valueOf(int).
func formatInt(n int32) string {
	return strconv.FormatInt(int64(n), 10)
}

// bit is the Java int expression 1 << n: Java uses only the low five bits of
// the shift distance, so 1 << 32 is 1 and 1 << -1 is 1 << 31.
func bit(n int32) int32 {
	return int32(uint32(1) << (uint32(n) & 31)) //nolint:gosec // Java int semantics on purpose
}
