// Package storage defines the persistent data models and the store
// interfaces the servers use. Implementations live in sub-packages.
package storage

import (
	"context"
	"errors"
)

var (
	// ErrNotFound is returned when the requested record does not exist.
	ErrNotFound = errors.New("storage: not found")
	// ErrExists is returned when a record with the same unique key exists.
	ErrExists = errors.New("storage: already exists")
)

// AccessLevel is an account's permission level.
type AccessLevel int8

// Access levels. Anything below AccessLevelPlayer may not log in.
const (
	AccessLevelBanned AccessLevel = -1
	AccessLevelPlayer AccessLevel = 0
	AccessLevelAdmin  AccessLevel = 1
)

// DefaultLastServer is the last server of an account that never joined one,
// the default of the Mobius accounts.lastServer column.
const DefaultLastServer = 1

// Account is a login account.
type Account struct {
	ID          string // assigned by the store on Create
	Username    string
	Password    string // bcrypt hash
	AccessLevel AccessLevel
	// LastServer is the ID of the game server the account joined last
	// (Mobius accounts.lastServer). The server list preselects it.
	LastServer int
}

// AccountStore stores login accounts. Usernames are unique and compared
// exactly (case-sensitive).
type AccountStore interface {
	// FindByUsername returns the account or ErrNotFound.
	FindByUsername(ctx context.Context, username string) (Account, error)
	// Create stores a new account. It returns ErrExists if the username is
	// taken. Any ID set on acc is ignored. A LastServer of 0 is stored as
	// DefaultLastServer.
	Create(ctx context.Context, acc Account) error
	// SetLastServer stores the account's last server. It returns
	// ErrNotFound if the account does not exist.
	SetLastServer(ctx context.Context, username string, serverID int) error
	// SetAccessLevel stores the account's access level; AccessLevelBanned
	// bans it. It returns ErrNotFound if the account does not exist.
	SetAccessLevel(ctx context.Context, username string, level AccessLevel) error
}
