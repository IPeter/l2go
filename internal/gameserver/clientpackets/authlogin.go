package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/session"
)

// AuthLogin asks for the character list of an account. The keys come from
// the login server, which confirms them over the link.
type AuthLogin struct {
	// Account is the account name as the client sent it. Mobius lowercases
	// it; the server does too before using it.
	Account   string
	PlayKey2  int32
	PlayKey1  int32
	LoginKey1 int32
	LoginKey2 int32
}

// Keys returns the session keys in the form the link checks.
func (a AuthLogin) Keys() session.Keys {
	return session.Keys{
		LoginOK1: a.LoginKey1,
		LoginOK2: a.LoginKey2,
		PlayOK1:  a.PlayKey1,
		PlayOK2:  a.PlayKey2,
	}
}

// ParseAuthLogin parses an AuthLogin body. The key order follows Mobius
// AuthLogin.java: play key 2, play key 1, login key 1, login key 2.
func ParseAuthLogin(data []byte) (AuthLogin, error) {
	r := packets.NewReader(data)
	a := AuthLogin{
		Account:   r.ReadString(),
		PlayKey2:  r.ReadInt32(),
		PlayKey1:  r.ReadInt32(),
		LoginKey1: r.ReadInt32(),
		LoginKey2: r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return AuthLogin{}, fmt.Errorf("AuthLogin: %w", err)
	}
	return a, nil
}
