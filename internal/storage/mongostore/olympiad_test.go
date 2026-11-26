package mongostore

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/IPeter/l2go/internal/storage"
)

// The olympiad document round-trips through BSON with Mobius's names; no
// database is needed.
func TestOlympiadDocRoundTrip(t *testing.T) {
	in := storage.Olympiad{
		Cycle: 3, Period: 1, OlympiadEnd: 11, ValidationEnd: 12, NextWeeklyChange: 13,
		Nobles:    []storage.OlympiadNoble{{CharID: 1, ClassID: 88, Name: "A", Points: 21, Done: 9, Won: 5, Lost: 3, Drawn: 1}},
		NoblesEOM: []storage.OlympiadNoble{{CharID: 2, ClassID: 93, Name: "B", Points: 3}},
		Heroes:    []storage.Hero{{CharID: 1, ClassID: 88, Name: "A", Count: 2, Played: true, Claimed: true, Message: "hi"}},
		Unclaimed: []storage.UnclaimedPasses{{CharID: 1, Passes: 100000}},
	}
	raw, err := bson.Marshal(toOlympiadDoc(in))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"current_cycle", "period", "olympiad_end", "validation_end", "next_weekly_change", "nobles", "nobles_eom", "heroes", "unclaimed"} {
		if _, err := bson.Raw(raw).LookupErr(key); err != nil {
			t.Errorf("no %s in %v", key, bson.Raw(raw))
		}
	}
	if v, err := bson.Raw(raw).LookupErr("nobles", "0", "olympiad_points"); err != nil || v.Int32() != 21 {
		t.Errorf("nobles.0.olympiad_points = %v, %v", v, err)
	}
	var back olympiadDoc
	if err := bson.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if got := back.state(); !reflect.DeepEqual(got, in) {
		t.Errorf("read back %+v, want %+v", got, in)
	}
	f := storage.OlympiadFight{CharOneID: 1, CharTwoID: 2, CharOneName: "A", CharTwoName: "B", CharOneClass: 88, CharTwoClass: 93, Winner: 2, Start: 5, Time: 6, Classed: true}
	raw, err = bson.Marshal(fightDoc(f))
	if err != nil {
		t.Fatal(err)
	}
	var fd fightDoc
	if err := bson.Unmarshal(raw, &fd); err != nil || storage.OlympiadFight(fd) != f {
		t.Errorf("fight read back %+v, %v", fd, err)
	}
	if _, err := bson.Raw(raw).LookupErr("charOneId"); err != nil {
		t.Errorf("fight document %v", bson.Raw(raw))
	}
}
