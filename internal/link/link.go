// Package link is the protocol between the login server and its game
// servers.
//
// A game server connects to the login server and registers with a shared
// secret: the login server sends a random challenge, and the game server
// answers with its ID and HMAC-SHA256(secret, challenge), so the secret
// never goes over the wire. A registered game server then reports its
// player count, asks the login server to confirm the session keys of each
// player that joins it, and pings to keep the link alive.
//
// Every message is one packet in the shared framing (packets.ReadPacket
// and packets.WritePacket): an opcode byte followed by little-endian
// fields. The link runs on the internal network and is not encrypted.
//
// Server is the login server side, Client the game server side.
package link

import (
	"errors"
	"fmt"

	"github.com/IPeter/l2go/internal/session"
)

// ErrNotConnected is returned by Client.VerifySession while the client is
// not registered with the login server.
var ErrNotConnected = errors.New("link: not connected to the login server")

// Reason says why the login server refused a registration.
type Reason uint8

// Registration results. Zero means the registration was accepted.
const (
	reasonNone Reason = iota
	// ReasonWrongSecret: the HMAC did not match the server's secret.
	ReasonWrongSecret
	// ReasonUnknownServer: the login server has no game server with that ID.
	ReasonUnknownServer
	// ReasonAlreadyRegistered: another connection holds that ID.
	ReasonAlreadyRegistered
	// ReasonUnsupportedVersion: the two sides speak different protocol
	// versions.
	ReasonUnsupportedVersion
)

func (r Reason) String() string {
	switch r {
	case reasonNone:
		return "accepted"
	case ReasonWrongSecret:
		return "wrong secret"
	case ReasonUnknownServer:
		return "unknown server ID"
	case ReasonAlreadyRegistered:
		return "server ID already registered"
	case ReasonUnsupportedVersion:
		return "unsupported protocol version"
	default:
		return fmt.Sprintf("reason %d", uint8(r))
	}
}

// RejectedError reports a registration the login server refused.
//
// Client.Run returns it for every reason except ReasonAlreadyRegistered,
// which it retries: retrying cannot fix a wrong secret or ID.
type RejectedError struct {
	Reason Reason
}

func (e *RejectedError) Error() string {
	return "link: registration rejected: " + e.Reason.String()
}

// SessionConsumer confirms the session keys a player shows to a game
// server. *session.Store implements it.
type SessionConsumer interface {
	// Consume reports whether k are the current keys of account, and
	// forgets them if so.
	Consume(account string, k session.Keys) bool
}

// AccessLevelSetter stores an account's access level when a game server
// asks for it (a GM's account ban). It is called on the link's goroutine
// of that game server.
type AccessLevelSetter interface {
	SetAccessLevel(account string, level int) error
}

// ServerStatus is what the login server knows about one game server.
type ServerStatus struct {
	// Online is true while the game server is registered.
	Online bool
	// Players is the last reported player count; 0 while offline.
	Players int
	// MaxPlayers is the last reported limit. It is kept while offline,
	// and 0 if the server never registered.
	MaxPlayers int
}
