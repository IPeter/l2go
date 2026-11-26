package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// The community board client packets (Plan 14 M11o, Mobius ClientPackets).
const (
	OpRequestBBSwrite  = 0x22
	OpRequestShowBoard = 0x57
)

// RequestShowBoard opens the community board (Alt+B); the server sends its
// default page.
type RequestShowBoard struct {
	// Unused is a value the client sends and Mobius ignores.
	Unused int32
}

// ParseRequestShowBoard parses a RequestShowBoard body (Mobius
// RequestShowBoard.readImpl): d unused.
func ParseRequestShowBoard(data []byte) (RequestShowBoard, error) {
	r := packets.NewReader(data)
	req := RequestShowBoard{Unused: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestShowBoard{}, fmt.Errorf("RequestShowBoard: %w", err)
	}
	return req, nil
}

// RequestBBSwrite is a "Write" button of a board page: the board (URL) and
// five values, the page's arguments and the input fields it names.
type RequestBBSwrite struct {
	URL                          string
	Arg1, Arg2, Arg3, Arg4, Arg5 string
}

// ParseRequestBBSwrite parses a RequestBBSwrite body (Mobius
// RequestBBSwrite.readImpl): S url, then S arg1 to arg5.
func ParseRequestBBSwrite(data []byte) (RequestBBSwrite, error) {
	r := packets.NewReader(data)
	req := RequestBBSwrite{
		URL:  r.ReadString(),
		Arg1: r.ReadString(),
		Arg2: r.ReadString(),
		Arg3: r.ReadString(),
		Arg4: r.ReadString(),
		Arg5: r.ReadString(),
	}
	if err := r.Err(); err != nil {
		return RequestBBSwrite{}, fmt.Errorf("RequestBBSwrite: %w", err)
	}
	return req, nil
}
