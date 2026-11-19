package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpSay2 is the opcode of Say2.
const OpSay2 = 0x38

// say2Whisper is the chat type of a whisper (serverpackets.ChatWhisper).
// Only a whisper carries a target name.
const say2Whisper = 2

// Say2 is a chat line the player sends.
type Say2 struct {
	Text string
	// Type is the chat type, one of the serverpackets.ChatType values.
	// It is not checked here: Mobius disconnects the player for an unknown
	// type.
	Type   int32
	Target string // the whisper target name; empty for other chat types
}

// ParseSay2 parses a Say2 body (Mobius Say2.readImpl): the text, the type,
// and the target name for a whisper.
func ParseSay2(data []byte) (Say2, error) {
	r := packets.NewReader(data)
	s := Say2{
		Text: r.ReadString(),
		Type: r.ReadInt32(),
	}
	if s.Type == say2Whisper {
		s.Target = r.ReadString()
	}
	if err := r.Err(); err != nil {
		return Say2{}, fmt.Errorf("Say2: %w", err)
	}
	return s, nil
}
