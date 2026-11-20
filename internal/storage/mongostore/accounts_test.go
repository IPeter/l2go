package mongostore

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/IPeter/l2go/internal/storage"
)

// An account document written before L2Go stored the last server reads as
// the Mobius column default; a stored one is kept.
func TestAccountDocLastServer(t *testing.T) {
	tests := []struct {
		name string
		doc  bson.D
		want int
	}{
		{"missing", bson.D{{Key: "username", Value: "old"}}, storage.DefaultLastServer},
		{"stored", bson.D{{Key: "username", Value: "new"}, {Key: "last_server", Value: int32(7)}}, 7},
	}
	for _, tt := range tests {
		raw, err := bson.Marshal(tt.doc)
		if err != nil {
			t.Fatal(err)
		}
		var d accountDoc
		if err := bson.Unmarshal(raw, &d); err != nil {
			t.Fatalf("%s: Unmarshal: %v", tt.name, err)
		}
		if got := d.account().LastServer; got != tt.want {
			t.Errorf("%s: LastServer = %d, want %d", tt.name, got, tt.want)
		}
	}
}
