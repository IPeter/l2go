package storage

import (
	"reflect"
	"testing"
)

func TestQuestChangesMerge(t *testing.T) {
	q := func(name, cond string) Quest { return Quest{Name: name, Vars: map[string]string{"cond": cond}} }
	tests := []struct {
		name         string
		older, newer QuestChanges
		want         QuestChanges
	}{
		{"both empty", QuestChanges{}, QuestChanges{}, QuestChanges{}},
		{"only older", QuestChanges{Upserts: []Quest{q("A", "1")}}, QuestChanges{}, QuestChanges{Upserts: []Quest{q("A", "1")}}},
		{"only newer", QuestChanges{}, QuestChanges{Deleted: []string{"A"}}, QuestChanges{Deleted: []string{"A"}}},
		{
			"newer upsert replaces older upsert",
			QuestChanges{Upserts: []Quest{q("A", "1"), q("B", "1")}},
			QuestChanges{Upserts: []Quest{q("A", "2")}},
			QuestChanges{Upserts: []Quest{q("B", "1"), q("A", "2")}},
		},
		{
			"newer delete replaces older upsert",
			QuestChanges{Upserts: []Quest{q("A", "1"), q("B", "1")}},
			QuestChanges{Deleted: []string{"A"}},
			QuestChanges{Upserts: []Quest{q("B", "1")}, Deleted: []string{"A"}},
		},
		{
			"newer upsert replaces older delete",
			QuestChanges{Deleted: []string{"A", "B"}},
			QuestChanges{Upserts: []Quest{q("A", "3")}},
			QuestChanges{Upserts: []Quest{q("A", "3")}, Deleted: []string{"B"}},
		},
		{
			"newer upsert and delete of one quest: the upsert wins",
			QuestChanges{Upserts: []Quest{q("A", "1")}},
			QuestChanges{Upserts: []Quest{q("A", "2")}, Deleted: []string{"A"}},
			QuestChanges{Upserts: []Quest{q("A", "2")}},
		},
	}
	for _, tt := range tests {
		got := tt.older.Merge(tt.newer)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: Merge = %+v, want %+v", tt.name, got, tt.want)
		}
		if got.Empty() != (len(tt.want.Upserts) == 0 && len(tt.want.Deleted) == 0) {
			t.Errorf("%s: Empty() = %v", tt.name, got.Empty())
		}
	}
}

func TestVarChangesEmpty(t *testing.T) {
	if !(VarChanges{}).Empty() {
		t.Error("zero VarChanges is not empty")
	}
	if (VarChanges{Deleted: []string{"x"}}).Empty() || (VarChanges{Set: map[string]string{"x": ""}}).Empty() {
		t.Error("VarChanges with changes reports empty")
	}
}
