package config

import (
	"fmt"
	"strings"
)

// CommunityBoard configures the community board (Alt+B, Plan 14 M11o):
// Mobius General.ini (EnableCommunityBoard, BBSDefault) and the two keys of
// Custom/CommunityBoard.ini that also apply to the standard board's home
// page. The custom board itself (CustomCommunityBoard: buffer, teleports,
// shops) is not built (decision X8).
type CommunityBoard struct {
	Comment string `json:"_comment,omitempty"`
	// Enabled turns the board on; when off, every board request answers
	// "The community server is currently offline." (Mobius
	// EnableCommunityBoard).
	Enabled bool `json:"enableCommunityBoard"`
	// Default is the bypass Alt+B opens (Mobius BBSDefault).
	Default string `json:"bbsDefault"`
	// PeaceOnly lets the home page open only in a peace zone (Mobius
	// CommunityBoardPeaceOnly).
	PeaceOnly bool `json:"communityBoardPeaceOnly"`
	// KarmaDisabled keeps players with karma off the home page (Mobius
	// CommunityKarmaDisabled).
	KarmaDisabled bool `json:"communityKarmaDisabled"`
}

// DefaultCommunityBoard returns the Mobius defaults.
func DefaultCommunityBoard() CommunityBoard {
	return CommunityBoard{Enabled: true, Default: "_bbshome", KarmaDisabled: true}
}

// validateCommunityBoard checks the communityBoard section.
func (g GameServer) validateCommunityBoard() []error {
	if strings.TrimSpace(g.CommunityBoard.Default) == "" {
		return []error{fmt.Errorf("communityBoard.bbsDefault: empty")}
	}
	return nil
}
