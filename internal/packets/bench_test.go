package packets

import (
	"bytes"
	"testing"
)

func BenchmarkWriter(b *testing.B) {
	for b.Loop() {
		w := NewWriter(64)
		w.WriteUint8(0x0b)
		w.WriteString("Character")
		for range 12 {
			w.WriteUint32(42)
		}
		_ = w.Bytes()
	}
}

func BenchmarkReader(b *testing.B) {
	w := NewWriter(64)
	w.WriteString("Character")
	for range 12 {
		w.WriteUint32(42)
	}
	data := w.Bytes()
	for b.Loop() {
		r := NewReader(data)
		r.ReadString()
		for range 12 {
			r.ReadUint32()
		}
	}
}

func BenchmarkReadWritePacket(b *testing.B) {
	body := bytes.Repeat([]byte{1}, 256)
	var buf bytes.Buffer
	for b.Loop() {
		buf.Reset()
		if err := WritePacket(&buf, body); err != nil {
			b.Fatal(err)
		}
		if _, err := ReadPacket(&buf, MaxBodySize); err != nil {
			b.Fatal(err)
		}
	}
}
