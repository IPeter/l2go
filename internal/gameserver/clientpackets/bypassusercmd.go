package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpBypassUserCmd is the opcode of BypassUserCmd.
const OpBypassUserCmd = 0xaa

// User command IDs (the COMMAND_IDS of the Mobius user command handlers).
const (
	UserCmdLoc = 0 // /loc
)

// BypassUserCmd is a user command typed in the chat, like /loc. The client
// sends its number.
type BypassUserCmd struct {
	Command int32
}

// ParseBypassUserCmd parses a BypassUserCmd body (Mobius
// BypassUserCmd.readImpl).
func ParseBypassUserCmd(data []byte) (BypassUserCmd, error) {
	r := packets.NewReader(data)
	b := BypassUserCmd{Command: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return BypassUserCmd{}, fmt.Errorf("BypassUserCmd: %w", err)
	}
	return b, nil
}
