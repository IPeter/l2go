package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// AuthLogin asks for the character list of an account. The keys come from
// the login server; they are not checked yet.
type AuthLogin struct {
	Account   string
	PlayKey2  uint32
	PlayKey1  uint32
	LoginKey1 uint32
	LoginKey2 uint32
}

// ParseAuthLogin parses an AuthLogin body. The key order follows Mobius
// AuthLogin.java: play key 2, play key 1, login key 1, login key 2.
func ParseAuthLogin(data []byte) (AuthLogin, error) {
	r := packets.NewReader(data)
	a := AuthLogin{
		Account:   r.ReadString(),
		PlayKey2:  r.ReadUint32(),
		PlayKey1:  r.ReadUint32(),
		LoginKey1: r.ReadUint32(),
		LoginKey2: r.ReadUint32(),
	}
	if err := r.Err(); err != nil {
		return AuthLogin{}, fmt.Errorf("AuthLogin: %w", err)
	}
	return a, nil
}
