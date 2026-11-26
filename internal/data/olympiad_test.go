package data

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadOlympiadStadiums(t *testing.T) {
	fsys := fstest.MapFS{OlympiadStadiumsFile: {Data: []byte(`[{"x":-20814,"y":-21189,"z":-3030},{"x":-120324,"y":-225077,"z":-3331}]`)}}
	list, err := LoadOlympiadStadiumsFS(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0] != (OlympiadStadium{-20814, -21189, -3030}) {
		t.Errorf("stadiums = %+v", list)
	}
	for _, tc := range []struct {
		body, want string
	}{
		{`[]`, "no stadium"},
		{`[{"x":0,"y":0,"z":5}]`, "at 0,0"},
		{`[{"x":1,"y":2,"z":3},{"x":1,"y":2,"z":3}]`, "listed twice"},
		{`[{"x":1,"y":2,"z":3,"w":4}]`, "unknown field"},
	} {
		_, err := LoadOlympiadStadiumsFS(fstest.MapFS{OlympiadStadiumsFile: {Data: []byte(tc.body)}})
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want %q", tc.body, err, tc.want)
		}
	}
	if _, err := LoadOlympiadStadiums(t.TempDir()); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: %v", err)
	}
}

// The committed stadiums are Mobius's 22.
func TestCommittedOlympiadStadiums(t *testing.T) {
	list, err := LoadOlympiadStadiums(filepath.Join("..", "..", "data"))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 22 || list[0] != (OlympiadStadium{-20814, -21189, -3030}) || list[21] != (OlympiadStadium{-114413, -213241, -3331}) {
		t.Errorf("committed stadiums: %d, first %+v", len(list), list[0])
	}
}
