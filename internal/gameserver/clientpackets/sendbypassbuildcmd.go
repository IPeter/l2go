package clientpackets

import (
	"fmt"
	"strings"

	"github.com/IPeter/l2go/internal/packets"
)

// OpSendBypassBuildCmd is the opcode of SendBypassBuildCmd.
const OpSendBypassBuildCmd = 0x5b

// SendBypassBuildCmd is a GM command typed in the chat as "//command",
// without the slashes.
type SendBypassBuildCmd struct {
	// Command is the command line, trimmed like Java's String.trim: without
	// leading and trailing spaces and control characters.
	Command string
}

// ParseSendBypassBuildCmd parses a SendBypassBuildCmd body (Mobius
// SendBypassBuildCmd.readImpl).
func ParseSendBypassBuildCmd(data []byte) (SendBypassBuildCmd, error) {
	r := packets.NewReader(data)
	cmd := r.ReadString()
	if err := r.Err(); err != nil {
		return SendBypassBuildCmd{}, fmt.Errorf("SendBypassBuildCmd: %w", err)
	}
	return SendBypassBuildCmd{Command: strings.TrimFunc(cmd, func(c rune) bool { return c <= ' ' })}, nil
}
