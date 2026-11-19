package testclient

import (
	"encoding/binary"
	"fmt"
	"slices"
	"sync"
	"time"
)

// Direction tells which side sent a recorded packet.
type Direction uint8

// Directions.
const (
	// FromServer marks a packet the server sent.
	FromServer Direction = iota + 1
	// FromClient marks a packet the client sent.
	FromClient
)

func (d Direction) String() string {
	switch d {
	case FromServer:
		return "server"
	case FromClient:
		return "client"
	}
	return fmt.Sprintf("Direction(%d)", uint8(d))
}

// Record is one recorded packet.
type Record struct {
	// Seq numbers the records of a Recorder from 0, in the order the client
	// sent and read the packets.
	Seq  int
	Time time.Time
	Dir  Direction
	// Name is the Mobius class name of the packet. A server packet the
	// client does not parse has the name Unknown.PacketName gives it.
	Name string
	// Opcode is the first byte. ExOpcode is the sub-opcode of an Ex packet
	// (0xfe from the server, 0xd0 from the client), else 0.
	Opcode   byte
	ExOpcode uint16
	// Raw is the decrypted packet, opcode included.
	Raw []byte
	// Packet is the decoded server packet: an Unknown when the client does
	// not parse it, nil for client packets and malformed server packets.
	Packet Packet
	// Err is why a malformed server packet could not be decoded.
	Err error
}

// Recorder keeps every packet of the game connections it is given to with
// WithRecorder, in both directions and in order: the decoded server
// packets with their raw bytes, and the raw client packets. cmd/refdiff
// records the same scenario against two servers and compares the records.
// It is safe for concurrent use. A Recorder grows without limit.
type Recorder struct {
	mu      sync.Mutex
	records []Record
}

// NewRecorder returns an empty recorder.
func NewRecorder() *Recorder { return &Recorder{} }

// Records returns every record so far.
func (r *Recorder) Records() []Record {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.records)
}

// ServerPackets returns the decoded server packets so far, in order,
// without the malformed ones.
func (r *Recorder) ServerPackets() []Packet {
	r.mu.Lock()
	defer r.mu.Unlock()
	var ps []Packet
	for _, rec := range r.records {
		if rec.Dir == FromServer && rec.Packet != nil {
			ps = append(ps, rec.Packet)
		}
	}
	return ps
}

// Len returns the number of records.
func (r *Recorder) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.records)
}

// Reset forgets every record.
func (r *Recorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = nil
}

// add records one packet. A nil recorder records nothing.
func (r *Recorder) add(dir Direction, raw []byte, p Packet, err error) {
	if r == nil || len(raw) == 0 {
		return
	}
	rec := Record{Time: time.Now(), Dir: dir, Opcode: raw[0], Raw: raw, Packet: p, Err: err}
	ex := OpExServer
	if dir == FromClient {
		ex = OpExClient
	}
	if raw[0] == ex && len(raw) >= 3 {
		rec.ExOpcode = binary.LittleEndian.Uint16(raw[1:])
	}
	switch {
	case dir == FromClient:
		rec.Name = clientPacketName(raw)
	case p != nil:
		rec.Name = p.PacketName()
	default:
		rec.Name = Unknown{Opcode: raw[0], Body: raw[1:]}.PacketName()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	rec.Seq = len(r.records)
	r.records = append(r.records, rec)
}
