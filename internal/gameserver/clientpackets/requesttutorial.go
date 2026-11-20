package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Opcodes of the tutorial client packets (Mobius ClientPackets 0x7B-0x7E).
const (
	OpRequestTutorialLinkHtml        = 0x7b
	OpRequestTutorialPassCmdToServer = 0x7c
	OpRequestTutorialQuestionMark    = 0x7d
	OpRequestTutorialClientEvent     = 0x7e
)

// RequestTutorialLinkHtml is a click on a "link" of the tutorial window
// (TutorialShowHtml). Mobius answers "close" (any case) with
// TutorialCloseHtml, then runs the bypass handler of Link if there is one,
// else hands Link to the Q00255_Tutorial quest as an event.
type RequestTutorialLinkHtml struct {
	Link string
}

// ParseRequestTutorialLinkHtml parses a RequestTutorialLinkHtml body
// (Mobius RequestTutorialLinkHtml.readImpl): S link. Like the Mobius
// readString it never fails: a link without its terminator runs to the end
// of the data.
func ParseRequestTutorialLinkHtml(data []byte) (RequestTutorialLinkHtml, error) {
	return RequestTutorialLinkHtml{Link: htmlActionText(data)}, nil
}

// RequestTutorialPassCmdToServer is a click on a "bypass" of the tutorial
// window. Mobius runs the bypass handler of Command if there is one, else
// hands it to the Q00255_Tutorial quest as an event.
type RequestTutorialPassCmdToServer struct {
	Command string
}

// ParseRequestTutorialPassCmdToServer parses a
// RequestTutorialPassCmdToServer body (Mobius
// RequestTutorialPassCmdToServer.readImpl): S bypass. It never fails, like
// ParseRequestTutorialLinkHtml.
func ParseRequestTutorialPassCmdToServer(data []byte) (RequestTutorialPassCmdToServer, error) {
	return RequestTutorialPassCmdToServer{Command: htmlActionText(data)}, nil
}

// RequestTutorialQuestionMark is a click on a tutorial question mark
// (TutorialShowQuestionMark). Mobius hands "QM" + Number to the
// Q00255_Tutorial quest.
type RequestTutorialQuestionMark struct {
	Number int32
}

// ParseRequestTutorialQuestionMark parses a RequestTutorialQuestionMark
// body (Mobius RequestTutorialQuestionMark.readImpl): d number.
func ParseRequestTutorialQuestionMark(data []byte) (RequestTutorialQuestionMark, error) {
	r := packets.NewReader(data)
	req := RequestTutorialQuestionMark{Number: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestTutorialQuestionMark{}, fmt.Errorf("RequestTutorialQuestionMark: %w", err)
	}
	return req, nil
}

// RequestTutorialClientEvent reports a client event that
// TutorialEnableClientEvent turned on. Mobius hands "CE" + EventID to the
// Q00255_Tutorial quest.
type RequestTutorialClientEvent struct {
	EventID int32
}

// ParseRequestTutorialClientEvent parses a RequestTutorialClientEvent body
// (Mobius RequestTutorialClientEvent.readImpl): d event.
func ParseRequestTutorialClientEvent(data []byte) (RequestTutorialClientEvent, error) {
	r := packets.NewReader(data)
	req := RequestTutorialClientEvent{EventID: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestTutorialClientEvent{}, fmt.Errorf("RequestTutorialClientEvent: %w", err)
	}
	return req, nil
}
