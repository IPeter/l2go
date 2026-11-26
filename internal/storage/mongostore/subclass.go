package mongostore

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/IPeter/l2go/internal/storage"
)

// subclassFields are the subclass fields of a character document (M11e;
// Mobius characters.nobless and character_subclasses), inlined into
// characterDoc. Each is absent while it has its zero value, so documents
// written before subclasses existed read as a character on its base class
// (class index 0), without noblesse or subclasses.
type subclassFields struct {
	ClassIndex int32         `bson:"class_index,omitempty"`
	Noble      bool          `bson:"nobless,omitempty"`
	Subclasses []subclassDoc `bson:"subclasses,omitempty"`
	BaseLevel  int32         `bson:"base_level,omitempty"`
	BaseExp    int64         `bson:"base_exp,omitempty"`
	BaseSP     int32         `bson:"base_sp,omitempty"`
}

// subclassDoc is the stored form of storage.Subclass, with the column names
// of Mobius character_subclasses.
type subclassDoc struct {
	ClassIndex int32 `bson:"class_index"`
	ClassID    int32 `bson:"class_id"`
	Level      int32 `bson:"level"`
	Exp        int64 `bson:"exp"`
	SP         int32 `bson:"sp"`
}

// toSubclassFields returns the subclass fields of c.
func toSubclassFields(c *storage.Character) subclassFields {
	return subclassFields{
		ClassIndex: c.ClassIndex,
		Noble:      c.Noble,
		Subclasses: convertList(c.Subclasses, func(s storage.Subclass) subclassDoc { return subclassDoc(s) }),
		BaseLevel:  c.BaseLevel,
		BaseExp:    c.BaseExp,
		BaseSP:     c.BaseSP,
	}
}

// fromSubclassDocs returns the subclasses of docs, nil for none.
func fromSubclassDocs(docs []subclassDoc) []storage.Subclass {
	return convertList(docs, func(d subclassDoc) storage.Subclass { return storage.Subclass(d) })
}

// subclassUpdate adds the subclass fields to the $set and $unset of a
// characterUpdate: a field with its zero value is removed, as the document
// leaves it out.
func subclassUpdate(f *subclassFields, set, unset bson.D) (bson.D, bson.D) {
	for _, v := range []struct {
		key   string
		value any
		zero  bool
	}{
		{"class_index", f.ClassIndex, f.ClassIndex == 0},
		{"nobless", f.Noble, !f.Noble},
		{"subclasses", f.Subclasses, f.Subclasses == nil},
		{"base_level", f.BaseLevel, f.BaseLevel == 0},
		{"base_exp", f.BaseExp, f.BaseExp == 0},
		{"base_sp", f.BaseSP, f.BaseSP == 0},
	} {
		if v.zero {
			unset = append(unset, bson.E{Key: v.key, Value: ""})
		} else {
			set = append(set, bson.E{Key: v.key, Value: v.value})
		}
	}
	return set, unset
}
