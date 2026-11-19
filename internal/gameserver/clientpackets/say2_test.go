package clientpackets

import "testing"

// Mobius Say2.readImpl: S text, d type, and S target only for a whisper
// (type 2).

func TestParseSay2(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want Say2
	}{
		{
			"general",
			new(m3Body).s("hello world").d(0).b,
			Say2{Text: "hello world", Type: 0},
		},
		{
			"shout",
			new(m3Body).s("WTS stuff").d(1).b,
			Say2{Text: "WTS stuff", Type: 1},
		},
		{
			"whisper",
			new(m3Body).s("psst").d(2).s("Elmore").b,
			Say2{Text: "psst", Type: 2, Target: "Elmore"},
		},
		{
			// Only a whisper has a target: the bytes after a trade line
			// are ignored, as in Mobius.
			"trade with trailing data",
			new(m3Body).s("+buy").d(8).s("Elmore").b,
			Say2{Text: "+buy", Type: 8},
		},
		{
			// Unknown types parse; the chat code disconnects, as Mobius.
			"unknown type",
			new(m3Body).s("x").d(99).b,
			Say2{Text: "x", Type: 99},
		},
		{
			"empty text",
			new(m3Body).s("").d(0).b,
			Say2{Text: "", Type: 0},
		},
		{
			"not ASCII",
			new(m3Body).s("Hérőke 😀 говорит").d(3).b,
			Say2{Text: "Hérőke 😀 говорит", Type: 3},
		},
	}
	for _, tt := range tests {
		got, err := ParseSay2(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}

	m3Truncations(t, "Say2 general", new(m3Body).s("hi").d(0).b, ParseSay2)
	m3Truncations(t, "Say2 whisper", new(m3Body).s("hi").d(2).s("Bob").b, ParseSay2)
}

func TestParseSay2Garbage(t *testing.T) {
	inputs := [][]byte{
		nil,
		{0x41},                         // half a code unit
		{0x41, 0x00, 0x42},             // no terminator, odd length
		{0xff, 0xff, 0xff, 0xff, 0xff}, // never terminated
		new(m3Body).s("hi").b,          // no type
		new(m3Body).s("hi").c(2).b,     // a short type
		new(m3Body).s("hi").d(2).b,     // a whisper without a target
		append(new(m3Body).s("hi").d(2).b, 'B', 0, 'o'), // an unterminated target
	}
	for _, in := range inputs {
		got, err := ParseSay2(in)
		m3Failed(t, "Say2", got, err)
	}
}

func FuzzParseSay2(f *testing.F) {
	m3Fuzz(f, ParseSay2, -1,
		new(m3Body).s("hello").d(0).b,
		new(m3Body).s("psst").d(2).s("Elmore").b,
		new(m3Body).s("").d(2).b,
	)
}
