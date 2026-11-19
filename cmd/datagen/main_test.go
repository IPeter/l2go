package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/golden")

const (
	// sampleDir is a cut-down Mobius datapack; goldenDir holds what datagen
	// makes from it, checked by hand.
	sampleDir = "testdata/mobius"
	goldenDir = "testdata/golden"
	// The local Mobius clone, the committed data and the package of the
	// generated enums, relative to this package.
	mobiusDir = "../../L2J_Mobius/L2J_Mobius_CT_0_Interlude"
	repoData  = "../../data"
	enumsDir  = "../../internal/data"
)

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

// checkGolden runs the named domains on the sample datapack and compares
// every file they make with testdata/golden (testdata/golden/enums for the
// -enums mode). With -update it rewrites the golden files instead.
func checkGolden(t *testing.T, enums bool, names ...string) {
	t.Helper()
	files, err := generate(sampleFS(t), enums, names, io.Discard)
	if err != nil {
		t.Fatalf("generate %v: %v", names, err)
	}
	dir := goldenDir
	if enums {
		dir = filepath.Join(goldenDir, "enums")
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		got := files[name]
		golden := filepath.Join(dir, filepath.FromSlash(name))
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
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from %s (run go test ./cmd/datagen -update, then check the diff)\n--- got ---\n%s\n--- want ---\n%s", name, golden, got, want)
		}
	}
}

func TestGolden(t *testing.T) {
	t.Run("players and items", func(t *testing.T) { checkGolden(t, false, "players", "items") })
	t.Run("enums", func(t *testing.T) { checkGolden(t, true, "stats", "items", "players") })
}

// The golden folders of these domains hold no file that datagen no longer
// makes.
func TestGoldenHasNoStaleFiles(t *testing.T) {
	files, err := generate(sampleFS(t), false, []string{"players", "items"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"players", "items"} {
		entries, err := os.ReadDir(filepath.Join(goldenDir, dir))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if _, ok := files[path.Join(dir, e.Name())]; !ok {
				t.Errorf("%s/%s/%s is not generated any more", goldenDir, dir, e.Name())
			}
		}
	}
}

func TestGenerateIsStable(t *testing.T) {
	for enums, names := range map[bool][]string{false: {"players", "items"}, true: {"stats", "items", "players"}} {
		a, err := generate(sampleFS(t), enums, names, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			b, err := generate(sampleFS(t), enums, names, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if !maps.EqualFunc(a, b, bytes.Equal) {
				t.Fatalf("enums %v: two runs on the same input gave different output", enums)
			}
		}
	}
}

func TestRunWritesLoadableData(t *testing.T) {
	out := t.TempDir()
	// A gzipped form left from an earlier run is removed.
	stale := filepath.Join(out, "items", "items.json.gz")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	var log bytes.Buffer
	if err := run(sampleDir, out, false, []string{"players", "items"}, &log); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, name := range []string{data.ClassesFile, data.EquipmentFile, data.StatBonusFile, data.ExperienceFile, data.ItemsFile} {
		if !strings.Contains(log.String(), filepath.Join(out, filepath.FromSlash(name))) {
			t.Errorf("log %q does not mention %s", log.String(), name)
		}
	}
	if _, err := os.Stat(stale); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("stale %s: %v, want it removed", stale, err)
	}

	d, err := data.Load(out)
	if err != nil {
		t.Fatalf("data.Load: %v", err)
	}
	if len(d.Classes) != 3 || len(d.Templates) != 2 || len(d.Items) != 17 {
		t.Errorf("loaded %d classes, %d starting templates, %d items; want 3, 2, 17", len(d.Classes), len(d.Templates), len(d.Items))
	}
	if tpl, ok := d.Template(49); !ok || tpl.Race != 3 || tpl.Level1.CP != 47.5 {
		t.Errorf("Template(49) = %+v, %v", tpl, ok)
	}
	// Custom items are left out: 2369 comes from the main folder.
	if it, ok := d.ItemTemplate(2369); !ok || it.Name != "Squire's Sword" {
		t.Errorf("ItemTemplate(2369) = %+v, %v", it, ok)
	}

	if err := run(filepath.Join(out, "missing"), out, false, nil, io.Discard); err == nil {
		t.Error("run with a missing Mobius folder succeeded")
	}
}

// committed compares what datagen makes from the local Mobius clone with
// the files in dir. Skipped when the clone is not there (it is not in git).
func committed(t *testing.T, dir string, enums bool, names ...string) {
	t.Helper()
	if _, err := os.Stat(mobiusDir); err != nil {
		t.Skipf("no Mobius clone at %s", mobiusDir)
	}
	files, err := generate(os.DirFS(mobiusDir), enums, names, io.Discard)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		want, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !bytes.Equal(files[name], want) {
			t.Errorf("%s is out of date; run go generate ./internal/data and go run ./cmd/datagen", name)
		}
	}
}

// The committed data/ folder is what datagen makes from Mobius.
func TestCommittedDataMatchesMobius(t *testing.T) { committed(t, repoData, false, "players", "items") }

// The committed enums of internal/data are what datagen makes from Mobius.
func TestCommittedEnumsMatchMobius(t *testing.T) {
	committed(t, enumsDir, true, "stats", "items", "players")
}

func TestSelectDomains(t *testing.T) {
	all, err := selectDomains(false, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, d := range all {
		names = append(names, d.name)
		if d.enums {
			t.Errorf("data domain %s is marked as an enum domain", d.name)
		}
	}
	if !slices.IsSorted(names) || !slices.Contains(names, "players") || !slices.Contains(names, "items") {
		t.Errorf("data domains %v: want them sorted, with players and items", names)
	}

	picked, err := selectDomains(true, []string{"players", "stats", "players"})
	if err != nil {
		t.Fatal(err)
	}
	if len(picked) != 2 || picked[0].name != "players" || picked[1].name != "stats" || !picked[0].enums {
		t.Errorf("selectDomains(enums, players stats players) = %+v", picked)
	}

	if _, err := selectDomains(true, []string{"world"}); err == nil || !strings.Contains(err.Error(), `unknown domain "world"`) {
		t.Errorf("unknown enum domain: %v", err)
	}
	if _, err := generate(sampleFS(t), false, []string{"nothing"}, io.Discard); err == nil {
		t.Error("generate of an unknown domain succeeded")
	}
}

func TestRegisterTwicePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("registering players twice did not panic")
		}
	}()
	registerDomain("players", func(options) error { return nil })
}

// check refuses data that internal/data refuses, and skips domains it does
// not know.
func TestCheck(t *testing.T) {
	files, err := generate(sampleFS(t), false, []string{"players", "items"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	files[data.ClassesFile] = []byte(`[{"classId": 0, "race": "HUMAN", "parentId": -1}]`)
	err = check(files, []domain{{name: "players"}, {name: "items"}})
	if !errors.Is(err, data.ErrInvalid) || !strings.Contains(err.Error(), "the generated data does not load") {
		t.Errorf("check of broken classes: %v", err)
	}
	if err := check(files, []domain{{name: "no-such-loader"}}); err != nil {
		t.Errorf("check of a domain internal/data does not know: %v", err)
	}
}

func TestWarnf(t *testing.T) {
	var log bytes.Buffer
	options{log: &log}.warnf("class %d", 34)
	if log.String() != "warning: class 34\n" {
		t.Errorf("warnf wrote %q", log.String())
	}
}
