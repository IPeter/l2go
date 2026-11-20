// Package recording is the comparison side of cmd/refdiff. It normalizes
// what a scenario recorded (scenario.Capture) into a Recording that no
// longer depends on the server or the run, saves and loads recordings as
// golden JSON files, compares two recordings and reports the differences
// side by side or as JSON.
//
// Normalization rules (Normalize):
//   - Object IDs become canonical names by first appearance, counted per
//     kind of object: #player1, #npc3, #item2, #clan1, #crest1, #vehicle1,
//     and #obj1 for IDs first seen elsewhere. 0 stays 0. Visibility bursts
//     (consecutive NpcInfo, CharInfo, DeleteObject, ... packets) are first
//     sorted by their content, because their order comes from the
//     server's iteration order, not from the protocol. For the same
//     reason the items of ItemList, InventoryUpdate, SellList and the
//     warehouse windows are sorted by item ID. Items on the ground
//     (DropItem, SpawnItem, GetItem) are #item too.
//   - Random and time values become placeholders: the session ID
//     (<session-id>), the login and play keys (<login-ok1> ... <play-ok2>),
//     the Blowfish key, the RSA modulus, the XOR key and the game time.
//     So do the game server's ID (<server-id>, only where it equals the
//     target's), address and player count in the server list.
//   - Account and character names become the scenario's roles:
//     <account:alice>, <char:alice>, also inside strings.
//   - A new character starts at a random creation point, so the
//     coordinates of the scenario's own characters are relative to where
//     each first stood (start, start+500). The start points themselves are
//     kept as anchors and compared with a tolerance. So are the target
//     coordinates of MoveToPawn and Attack when the target is one of them,
//     and the coordinates of an item one of them dropped (relative to the
//     dropper).
//   - Random positions become placeholders: a monster's coordinates and
//     heading (<monster>; it spawns at a random point and wanders), and
//     where its loot falls (<drop>).
//   - Random values become placeholders (rules.go): a monster's loot
//     (item, count; <drop>), current HP, MP and CP below the maximum
//     (<cur>), the damage of hit messages (<damage>). The player's adena
//     becomes relative to its first ItemList (adena0+900), since the
//     starting amount is configuration.
//   - The swings of a fight (fight.go: Attack, StatusUpdate, hit and miss
//     messages, the chase) are random in number; each run of them becomes
//     one Fight entry naming the attackers and targets.
//   - Object IDs inside strings (dialog links) and inside packets the
//     client does not parse are replaced by their canonical names too.
//
// Recordings compare field by field on the decoded packets, and as bytes
// for packets the client does not parse.
package recording

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Version is the format version of saved recordings. Version 2 added the
// Milestone 5–7 rules (monsters, loot, adena, HP, fights).
const Version = 2

// Recording is the normalized packet streams of one scenario run against
// one server.
type Recording struct {
	Version  int    `json:"version"`
	Scenario string `json:"scenario"`
	// Server labels the side, such as "l2go" or "reference".
	Server string `json:"server"`
	// Source says where it was recorded.
	Source string `json:"source,omitempty"`
	// Errors holds the error that ended the scenario early, if any.
	Errors []string `json:"errors,omitempty"`
	// Notes are remarks of the scenario, such as which NPC it talked to.
	Notes []string `json:"notes,omitempty"`
	// Anchors are the start points of the scenario's characters.
	Anchors []Anchor `json:"anchors,omitempty"`
	Streams []Stream `json:"streams"`
}

// Packets returns the number of server packets in all streams.
func (r Recording) Packets() int {
	n := 0
	for _, st := range r.Streams {
		for _, e := range st.Entries {
			if e.Dir == DirServer {
				n++
			}
		}
	}
	return n
}

// Anchor is where a scenario character first stood (its first
// CharSelected). Its coordinates in the streams are relative to it.
type Anchor struct {
	Object string `json:"object"`
	X      int32  `json:"x"`
	Y      int32  `json:"y"`
	Z      int32  `json:"z"`
}

// Stream is the packets of one connection, in order.
type Stream struct {
	Name    string  `json:"name"`
	Entries []Entry `json:"entries"`
}

// Directions of an Entry.
const (
	DirServer = "server"
	DirClient = "client"
)

// Entry is one normalized packet. Client packets keep only their name:
// the scenario writes them, so they are context, not compared.
type Entry struct {
	Dir      string `json:"dir"`
	Name     string `json:"name"`
	Opcode   byte   `json:"opcode"`
	ExOpcode uint16 `json:"ex,omitempty"`
	// Fields are the decoded fields of a server packet, in packet order.
	Fields Fields `json:"fields,omitempty"`
	// Raw is the body of a server packet the client does not parse (or
	// could not parse) in hex. Known object IDs and masked values appear
	// in braces: {#npc3}, {game-time}.
	Raw string `json:"raw,omitempty"`
	// Malformed marks a server packet whose layout did not match Mobius;
	// Error says why.
	Malformed bool   `json:"malformed,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Field is one normalized field: its path in the packet struct, such as
// "Items[2].Count", and its value. Strings are quoted, object IDs are
// canonical names and placeholders are in angle brackets.
type Field struct {
	Path  string
	Value string
}

// Fields are the fields of a packet. They are saved as one JSON object
// whose keys keep the packet order.
type Fields []Field

// MarshalJSON writes the fields as an object in field order.
func (fs Fields) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, f := range fs {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := writeJSONString(&buf, f.Path); err != nil {
			return nil, err
		}
		buf.WriteByte(':')
		if err := writeJSONString(&buf, f.Value); err != nil {
			return nil, err
		}
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// UnmarshalJSON reads an object written by MarshalJSON, keeping the key
// order.
func (fs *Fields) UnmarshalJSON(b []byte) error {
	if string(bytes.TrimSpace(b)) == "null" {
		*fs = nil
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("recording: fields: %w", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return fmt.Errorf("recording: fields: want an object, got %v", tok)
	}
	out := Fields{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return fmt.Errorf("recording: fields: %w", err)
		}
		path, ok := tok.(string)
		if !ok {
			return fmt.Errorf("recording: fields: key %v is not a string", tok)
		}
		var value string
		if err := dec.Decode(&value); err != nil {
			return fmt.Errorf("recording: field %q: %w", path, err)
		}
		out = append(out, Field{Path: path, Value: value})
	}
	if _, err := dec.Token(); err != nil {
		return fmt.Errorf("recording: fields: %w", err)
	}
	*fs = out
	return nil
}

// writeJSONString writes s as a JSON string without HTML escaping, so that
// placeholders such as <play-ok1> stay readable.
func writeJSONString(buf *bytes.Buffer, s string) error {
	var tmp bytes.Buffer
	enc := json.NewEncoder(&tmp)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return fmt.Errorf("recording: %w", err)
	}
	buf.Write(bytes.TrimSuffix(tmp.Bytes(), []byte("\n")))
	return nil
}

// writeJSON writes v as indented JSON without HTML escaping.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("recording: %w", err)
	}
	return nil
}

// Write writes rec as indented JSON, the format of golden files.
func Write(w io.Writer, rec Recording) error {
	rec.Version = Version
	return writeJSON(w, rec)
}

// Read reads a recording written by Write.
func Read(r io.Reader) (Recording, error) {
	var rec Recording
	if err := json.NewDecoder(r).Decode(&rec); err != nil {
		return Recording{}, fmt.Errorf("recording: %w", err)
	}
	if rec.Version != Version {
		return Recording{}, fmt.Errorf("recording: format version %d, want %d", rec.Version, Version)
	}
	return rec, nil
}

// Save writes rec to the golden file path.
func Save(path string, rec Recording) error {
	var buf bytes.Buffer
	if err := Write(&buf, rec); err != nil {
		return err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil { //nolint:gosec // golden files are shared test data
		return fmt.Errorf("recording: save: %w", err)
	}
	return nil
}

// Load reads the golden file path.
func Load(path string) (Recording, error) {
	f, err := os.Open(path)
	if err != nil {
		return Recording{}, fmt.Errorf("recording: load: %w", err)
	}
	defer func() { _ = f.Close() }()
	rec, err := Read(f)
	if err != nil {
		return Recording{}, fmt.Errorf("%s: %w", path, err)
	}
	return rec, nil
}
