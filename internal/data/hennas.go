package data

// Henna data: the dyes symbol makers draw, from Mobius stats/hennaList.xml,
// loaded with LoadHennas. The rules that use them are in internal/henna.

import (
	"fmt"
	"io/fs"
	"os"
	"slices"
)

// HennasFile is the path of the henna file, relative to the data folder.
const HennasFile = "hennas/hennas.json"

// Henna is one <henna> of hennaList.xml (Mobius Henna): a dye, the stats it
// changes, what drawing and removing it cost, and the classes that may wear
// it.
type Henna struct {
	// DyeID is the symbol's ID (Mobius dyeId), the ID the henna packets
	// carry.
	DyeID int `json:"dyeId"`
	// DyeItemID is the item drawing it uses up (Mobius dyeItemId).
	DyeItemID int `json:"dyeItemId"`
	// The base stat changes, 0 when the XML leaves one out.
	STR int `json:"str,omitempty"`
	CON int `json:"con,omitempty"`
	DEX int `json:"dex,omitempty"`
	INT int `json:"int,omitempty"`
	MEN int `json:"men,omitempty"`
	WIT int `json:"wit,omitempty"`
	// WearCount dye items and WearFee adena draw it; removing it gives back
	// CancelCount dye items for CancelFee adena.
	WearCount   int `json:"wearCount"`
	WearFee     int `json:"wearFee"`
	CancelCount int `json:"cancelCount"`
	CancelFee   int `json:"cancelFee"`
	// Classes are the classes that may wear it, sorted.
	Classes []int `json:"classes"`
}

// AllowedClass reports whether the class may wear the dye (Mobius
// Henna.isAllowedClass).
func (h *Henna) AllowedClass(classID int) bool {
	_, found := slices.BinarySearch(h.Classes, classID)
	return found
}

// Hennas is the checked henna data. It must not be modified after NewHennas
// or LoadHennas returns, and is then safe for concurrent use.
type Hennas struct {
	// List holds the dyes sorted by dye ID.
	List []Henna
	byID map[int]int // dye ID -> index in List
}

// LoadHennas reads and checks the henna file in dir.
func LoadHennas(dir string) (*Hennas, error) {
	h, err := LoadHennasFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return h, nil
}

// LoadHennasFS reads and checks the henna file in fsys.
func LoadHennasFS(fsys fs.FS) (*Hennas, error) {
	var list []Henna
	if err := readJSON(fsys, HennasFile, &list); err != nil {
		return nil, err
	}
	return NewHennas(list)
}

// NewHennas checks the dyes: positive dye and item IDs, sorted by dye ID
// without repeats, counts and fees not negative, and sorted classes without
// repeats, each a known player class. The slice is used as it is.
func NewHennas(list []Henna) (*Hennas, error) {
	var p problems
	byID := make(map[int]int, len(list))
	for i, h := range list {
		where := fmt.Sprintf("%s: henna %d (dye %d)", HennasFile, i, h.DyeID)
		switch {
		case h.DyeID <= 0:
			p.addf("%s: dye ID not positive", where)
		case i > 0 && h.DyeID <= list[i-1].DyeID:
			p.addf("%s: not sorted by dye ID, or listed twice", where)
		case h.DyeItemID <= 0:
			p.addf("%s: dye item ID not positive", where)
		case h.WearCount < 0 || h.WearFee < 0 || h.CancelCount < 0 || h.CancelFee < 0:
			p.addf("%s: a negative count or fee", where)
		}
		for j, c := range h.Classes {
			if j > 0 && c <= h.Classes[j-1] {
				p.addf("%s: classes not sorted, or listed twice", where)
				break
			}
			if !PlayerClass(c).Valid() {
				p.addf("%s: unknown class %d", where, c)
			}
		}
		byID[h.DyeID] = i
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return &Hennas{List: list, byID: byID}, nil
}

// Henna returns the dye with the dye ID, or nil (Mobius HennaData.getHenna).
func (h *Hennas) Henna(dyeID int) *Henna {
	if h == nil {
		return nil
	}
	i, ok := h.byID[dyeID]
	if !ok {
		return nil
	}
	return &h.List[i]
}

// ForClass returns the dyes the class may wear, by dye ID (Mobius
// HennaData.getHennaList, whose HashMap of small integer keys iterates in
// key order).
func (h *Hennas) ForClass(classID int) []*Henna {
	if h == nil {
		return nil
	}
	var out []*Henna
	for i := range h.List {
		if h.List[i].AllowedClass(classID) {
			out = append(out, &h.List[i])
		}
	}
	return out
}
