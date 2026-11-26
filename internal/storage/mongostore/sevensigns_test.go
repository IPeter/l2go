package mongostore

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/IPeter/l2go/internal/storage"
)

// The Seven Signs documents round-trip through BSON with Mobius's names; no
// database is needed.
func TestSevenSignsDocRoundTrip(t *testing.T) {
	st := storage.SevenSignsStatus{
		Cycle: 3, Period: 1, PreviousWinner: 2, DawnStoneScore: 1234, DuskStoneScore: 99,
		DawnFestivalScore: 60, DuskFestivalScore: 70, AvariceOwner: 2, GnosisOwner: 1, StrifeOwner: 2,
		AvariceDawnScore: 1, GnosisDawnScore: 2, StrifeDawnScore: 3, AvariceDuskScore: 4, GnosisDuskScore: 5, StrifeDuskScore: 6,
		FestivalCycle: 7, AccumulatedBonus: []int32{10, 20, 30, 40, 50}, Date: 1796040000123, NextPeriodChange: 1796126400000,
	}
	raw, err := bson.Marshal(toSevenSignsStatusDoc(st))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"_id", "current_cycle", "active_period", "previous_winner", "dawn_stone_score", "dawn_festival_score",
		"dusk_stone_score", "dusk_festival_score", "avarice_owner", "gnosis_owner", "strife_owner", "avarice_dawn_score",
		"strife_dusk_score", "festival_cycle", "accumulated_bonus0", "accumulated_bonus4", "date", "next_period_change"} {
		if _, err := bson.Raw(raw).LookupErr(key); err != nil {
			t.Errorf("no %s in %v", key, bson.Raw(raw))
		}
	}
	var back sevenSignsStatusDoc
	if err := bson.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if got := back.status(); !reflect.DeepEqual(got, st) {
		t.Errorf("status read back %+v, want %+v", got, st)
	}

	p := storage.SevenSignsPlayer{CharID: 0x10000001, Cabal: "dawn", Seal: 3, RedStones: 1, GreenStones: 2, BlueStones: 3, AncientAdena: 25, Contribution: 25}
	raw, err = bson.Marshal(toSevenSignsPlayerDoc(p))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"_id", "charId", "cabal", "seal", "red_stones", "green_stones", "blue_stones", "ancient_adena_amount", "contribution_score"} {
		if _, err := bson.Raw(raw).LookupErr(key); err != nil {
			t.Errorf("no %s in %v", key, bson.Raw(raw))
		}
	}
	var pd sevenSignsPlayerDoc
	if err := bson.Unmarshal(raw, &pd); err != nil || pd.player() != p {
		t.Errorf("player read back %+v, %v", pd, err)
	}

	f := storage.SevenSignsFestivalScore{Festival: 4, Cabal: "dusk", Cycle: 2, Date: 5, Score: 120, Members: "A,B"}
	raw, err = bson.Marshal(sevenSignsFestivalDoc(f))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bson.Raw(raw).LookupErr("festivalId"); err != nil {
		t.Errorf("festival document %v", bson.Raw(raw))
	}
	var fd sevenSignsFestivalDoc
	if err := bson.Unmarshal(raw, &fd); err != nil || storage.SevenSignsFestivalScore(fd) != f {
		t.Errorf("festival read back %+v, %v", fd, err)
	}
}
