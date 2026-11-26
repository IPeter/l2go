package testclient

import (
	"context"
	"strings"

	"github.com/IPeter/l2go/internal/packets"
)

// The community board packets (Plan 14 M11o) in the Mobius Interlude
// layouts.

// Opcodes of the community board packets.
const (
	// Client: RequestBBSwrite (a "Write" button) and RequestShowBoard (Alt+B).
	OpRequestBBSwrite  byte = 0x22
	OpRequestShowBoard byte = 0x57
	// Server: ShowBoard, and ExMailArrived (FE 2D).
	OpShowBoard     byte   = 0x6e
	ExOpMailArrived uint16 = 0x2d
)

// The decoder entries and names are added here rather than in the shared
// tables, so the board packets stay in one file.
func init() {
	serverPackets[OpShowBoard] = serverPacket{"ShowBoard", readShowBoard}
	exServerPackets[ExOpMailArrived] = serverPacket{"ExMailArrived", readExMailArrived}
	clientPacketNames[OpRequestBBSwrite] = "RequestBBSwrite"
	clientPacketNames[OpRequestShowBoard] = "RequestShowBoard"
}

// RequestShowBoard opens the community board (Alt+B): the server answers
// with its default page as ShowBoard parts.
func (c *GameConn) RequestShowBoard(ctx context.Context) error {
	return c.send(ctx, encodeInts(OpRequestShowBoard, 0))
}

// RequestBBSwrite presses a "Write" button of a board page: url names the
// board ("Mail", "Notice", ...), args are the five values the client fills
// in from the button and the page's input fields.
func (c *GameConn) RequestBBSwrite(ctx context.Context, url string, args [5]string) error {
	return c.send(ctx, encodeRequestBBSwrite(url, args))
}

// encodeRequestBBSwrite builds RequestBBSwrite: S url, S arg1 ... S arg5.
func encodeRequestBBSwrite(url string, args [5]string) []byte {
	size := 3 + 2*len(url)
	for _, a := range args {
		size += 2 + 2*len(a)
	}
	w := packets.NewWriter(size)
	w.WriteUint8(OpRequestBBSwrite)
	w.WriteString(url)
	for _, a := range args {
		w.WriteString(a)
	}
	return w.Bytes()
}

// ShowBoard opens the community board with a part of a page (0x6E).
type ShowBoard struct {
	// Show is set for a page, clear when the board closes.
	Show bool
	// Tabs are the bypasses of the eight tab buttons.
	Tabs [8]string
	// Content is the raw content: a part ID, U+0008 and the HTML.
	Content string
}

// PacketName implements Packet.
func (ShowBoard) PacketName() string { return "ShowBoard" }

// Part splits Content into its part ID ("101", "102", "103", "1001",
// "1002") and the text after the first U+0008.
func (b ShowBoard) Part() (id, html string) {
	id, html, _ = strings.Cut(b.Content, "\u0008")
	return id, html
}

func readShowBoard(f *fields) Packet {
	var p ShowBoard
	p.Show = f.flag8("show")
	for i := range p.Tabs {
		p.Tabs[i] = f.str()
	}
	p.Content = f.str()
	return p
}

// BoardPage joins the parts 101-103 of the last page in ps (Mobius
// HtmlUtil.sendCBHtml), leaving out the "null" of a missing part, or the
// "1001" page with a multiedit field. ok is false without one.
func BoardPage(ps []Packet) (html string, ok bool) {
	var parts [3]string
	for _, p := range ps {
		b, isBoard := p.(ShowBoard)
		if !isBoard {
			continue
		}
		id, text := b.Part()
		switch id {
		case "101":
			parts = [3]string{text}
			ok = true
		case "102", "103":
			if text != "null" {
				parts[id[2]-'1'] = text
			}
		case "1001":
			parts = [3]string{text}
			ok = true
		}
	}
	return parts[0] + parts[1] + parts[2], ok
}

// ExMailArrived blinks the mail icon (FE 2D); it has no fields.
type ExMailArrived struct{}

// PacketName implements Packet.
func (ExMailArrived) PacketName() string { return "ExMailArrived" }

func readExMailArrived(*fields) Packet { return ExMailArrived{} }
