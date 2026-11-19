package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/golden")

const (
	sampleDir = "testdata/mobius"
	goldenDir = "testdata/golden"
	// The local Mobius clone and the committed data, relative to this
	// package.
	mobiusDir = "../../L2J_Mobius/L2J_Mobius_CT_0_Interlude"
	repoData  = "../../data"
)

var outputs = []string{data.TemplatesFile, data.EquipmentFile, data.ItemsFile}

// The sample in testdata/mobius is a cut-down Mobius datapack. The golden
// files were written by hand from it: classes and items sorted by ID, level
// 1 picked by its val, unused and custom items left out.
func TestGenerateGolden(t *testing.T) {
	files, err := generate(os.DirFS(sampleDir))
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(files) != len(outputs) {
		t.Errorf("generate returned %d files, want %d", len(files), len(outputs))
	}
	for _, name := range outputs {
		got, ok := files[name]
		if !ok {
			t.Errorf("no output for %s", name)
			continue
		}
		golden := filepath.Join(goldenDir, filepath.FromSlash(name))
		if *update {
			if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(golden, got, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from %s\n--- got ---\n%s\n--- want ---\n%s", name, golden, got, want)
		}
	}
}

func TestGenerateIsStable(t *testing.T) {
	a, err := generate(os.DirFS(sampleDir))
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		b, err := generate(os.DirFS(sampleDir))
		if err != nil {
			t.Fatal(err)
		}
		if !maps.EqualFunc(a, b, bytes.Equal) {
			t.Fatal("two runs on the same input gave different output")
		}
	}
}

func TestRunWritesLoadableData(t *testing.T) {
	out := t.TempDir()
	var log bytes.Buffer
	if err := run(sampleDir, out, &log); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, name := range outputs {
		if !strings.Contains(log.String(), filepath.Join(out, filepath.FromSlash(name))) {
			t.Errorf("log %q does not mention %s", log.String(), name)
		}
	}
	d, err := data.Load(out)
	if err != nil {
		t.Fatalf("data.Load: %v", err)
	}
	if len(d.Templates) != 2 {
		t.Errorf("loaded %d templates, want 2", len(d.Templates))
	}
	if tpl, ok := d.Template(49); !ok || tpl.Race != 3 || tpl.Level1.CP != 47.5 {
		t.Errorf("Template(49) = %+v, %v", tpl, ok)
	}

	if err := run(filepath.Join(out, "missing"), out, io.Discard); err == nil {
		t.Error("run with a missing Mobius folder succeeded")
	}
}

// The committed data/ folder must be what datagen makes from the local
// Mobius clone. Skipped when the clone is not there (it is not in git).
func TestCommittedDataMatchesMobius(t *testing.T) {
	if _, err := os.Stat(mobiusDir); err != nil {
		t.Skipf("no Mobius clone at %s", mobiusDir)
	}
	files, err := generate(os.DirFS(mobiusDir))
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, name := range outputs {
		want, err := os.ReadFile(filepath.Join(repoData, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(files[name], want) {
			t.Errorf("data/%s is out of date; run: go run ./cmd/datagen", name)
		}
	}
}

// sampleFS returns the sample datapack as an in-memory file system that a
// test can change.
func sampleFS(t *testing.T) fstest.MapFS {
	t.Helper()
	root := os.DirFS(sampleDir)
	m := fstest.MapFS{}
	err := fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(root, p)
		if err != nil {
			return err
		}
		m[p] = &fstest.MapFile{Data: b}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func file(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }

const (
	humanFighter = startingClassDir + "/HumanFighter.xml"
	orcMystic    = startingClassDir + "/OrcMystic.xml"
)

func TestGenerateErrors(t *testing.T) {
	tests := []struct {
		name   string
		change func(fstest.MapFS)
		want   string
	}{
		{"missing class list", func(m fstest.MapFS) { delete(m, classListFile) }, "read " + classListFile},
		{"missing equipment", func(m fstest.MapFS) { delete(m, equipmentFile) }, "read " + equipmentFile},
		{"missing Race.java", func(m fstest.MapFS) { delete(m, raceFile) }, "read " + raceFile},
		{"no templates", func(m fstest.MapFS) {
			delete(m, humanFighter)
			delete(m, orcMystic)
		}, "no templates found"},
		{"bad XML", func(m fstest.MapFS) { m[classListFile] = file("<list><class") }, "parse " + classListFile},
		{"class without ID", func(m fstest.MapFS) {
			m[classListFile] = file(`<list><class name="Nobody" /></list>`)
		}, `class "Nobody" has no classId`},
		{"template not in class list", func(m fstest.MapFS) {
			m[startingClassDir+"/Unknown.xml"] = file(`<list><classId>77</classId></list>`)
		}, "class 77 is not in " + classListFile},
		{"template of a child class", func(m fstest.MapFS) {
			m[startingClassDir+"/Warrior.xml"] = file(`<list><classId>1</classId></list>`)
		}, "class 1 (Warrior) has a parent class"},
		{"template without class ID", func(m fstest.MapFS) {
			m[humanFighter] = file(`<list><staticData /></list>`)
		}, humanFighter + ": no classId"},
		{"no level 1", func(m fstest.MapFS) {
			m[orcMystic] = file(`<list><classId>49</classId><lvlUpgainData><level val="2"><hp>1</hp></level></lvlUpgainData></list>`)
		}, "class 49 has no level 1 data"},
		{"class without race", func(m fstest.MapFS) {
			m[playerClassFile] = file("FIGHTER(0, false, Race.HUMAN, null),\n")
		}, "class 49 has no race"},
		{"unknown race", func(m fstest.MapFS) {
			m[playerClassFile] = file("FIGHTER(0, false, Race.HUMAN, null),\nORC_MAGE(49, true, Race.KAMAEL, null);\n")
		}, "Race.KAMAEL"},
		{"no classes in PlayerClass.java", func(m fstest.MapFS) { m[playerClassFile] = file("public enum PlayerClass {}") }, "no classes found"},
		{"no Race enum", func(m fstest.MapFS) { m[raceFile] = file("public class Race {}") }, "enum Race not found"},
		{"item without count", func(m fstest.MapFS) {
			m[equipmentFile] = file(`<list><equipment classId="0"><item id="10" /></equipment></list>`)
		}, "item without id or count"},
		{"unknown item", func(m fstest.MapFS) {
			m[equipmentFile] = file(`<list><equipment classId="0"><item id="9999" count="1" /></equipment></list>`)
		}, "item 9999 (initial equipment of class 0) not found"},
		{"item defined twice", func(m fstest.MapFS) {
			m[itemsDir+"/09900-09999.xml"] = file(`<list><item id="10" type="Weapon" name="Dagger" /></list>`)
		}, "item 10 is defined in both"},
		{"no item files", func(m fstest.MapFS) {
			for p := range m {
				if strings.HasPrefix(p, itemsDir+"/") {
					delete(m, p)
				}
			}
		}, "no item files found"},
		{"fails data validation", func(m fstest.MapFS) {
			m[orcMystic] = file(`<list><classId>49</classId><staticData><baseSTR>1</baseSTR></staticData>` +
				`<lvlUpgainData><level val="1"><hp>1</hp><mp>1</mp><cp>1</cp></level></lvlUpgainData></list>`)
		}, "no creation points"},
	}
	for _, tt := range tests {
		m := sampleFS(t)
		tt.change(m)
		_, err := generate(m)
		if err == nil {
			t.Errorf("%s: generate succeeded", tt.name)
			continue
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error %q does not mention %q", tt.name, err, tt.want)
		}
	}

	// Validation errors from internal/data keep their sentinel.
	m := sampleFS(t)
	m[equipmentFile] = file(`<list><equipment classId="0"><item id="10" count="0" /></equipment></list>`)
	if _, err := generate(m); !errors.Is(err, data.ErrInvalid) {
		t.Errorf("zero count: generate = %v, want data.ErrInvalid", err)
	}
}

func TestReadRaceEnum(t *testing.T) {
	got, err := readRaceEnum(os.DirFS(sampleDir))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"HUMAN": 0, "ELF": 1, "DARK_ELF": 2, "ORC": 3, "DWARF": 4, "ANIMAL": 5}
	if !maps.Equal(got, want) {
		t.Errorf("readRaceEnum = %v, want %v", got, want)
	}
}

func TestRaceName(t *testing.T) {
	tests := map[string]string{
		"HUMAN":    "Human",
		"DARK_ELF": "Dark Elf",
		"ORC":      "Orc",
		"A__B":     "A  B",
	}
	for in, want := range tests {
		if got := raceName(in); got != want {
			t.Errorf("raceName(%q) = %q, want %q", in, got, want)
		}
	}
}
