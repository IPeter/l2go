package main

import (
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// Java source folders the type lists come from.
const (
	// Mobius instantiates NPC types by name from this package
	// (Spawn: Class.forName("...entity.actor.instance." + type)); their base
	// classes (Npc, Attackable, ...) are in the parent folder.
	npcTypeJavaDir  = "java/org/l2jmobius/gameserver/entity/actor/instance"
	npcBaseJavaDir  = "java/org/l2jmobius/gameserver/entity/actor"
	zoneTypeJavaDir = "java/org/l2jmobius/gameserver/entity/zone/type"
	zoneBaseJavaDir = "java/org/l2jmobius/gameserver/entity/zone"
)

// mxJavaClassRe matches a top-level class declaration, such as "public
// class Teleporter extends Merchant".
var mxJavaClassRe = regexp.MustCompile(`(?m)^public\s+(abstract\s+)?(?:final\s+)?class\s+([A-Za-z_]\w*)(?:\s+extends\s+([A-Za-z_][\w.]*))?`)

// readMobiusClasses returns the classes of typeDir, which data may name
// (Usable unless abstract), and the classes of baseDir they extend, sorted
// by name. Only the files directly in each folder are read.
func readMobiusClasses(fsys fs.FS, typeDir, baseDir string) ([]data.MobiusClass, error) {
	types, err := readJavaClassDir(fsys, typeDir)
	if err != nil {
		return nil, err
	}
	if len(types) == 0 {
		return nil, fmt.Errorf("%s: no classes found", typeDir)
	}
	bases, err := readJavaClassDir(fsys, baseDir)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]data.MobiusClass, len(types))
	for _, c := range types {
		c.Usable = !c.Abstract
		byName[c.Name] = c
	}
	// Add the base classes the type classes extend, all the way up.
	baseByName := make(map[string]data.MobiusClass, len(bases))
	for _, c := range bases {
		baseByName[c.Name] = c
	}
	for _, c := range types {
		for parent := c.Parent; parent != ""; {
			if _, ok := byName[parent]; ok {
				break
			}
			b, ok := baseByName[parent]
			if !ok {
				break
			}
			byName[b.Name] = b
			parent = b.Parent
		}
	}
	classes := make([]data.MobiusClass, 0, len(byName))
	for _, c := range byName {
		classes = append(classes, c)
	}
	slices.SortFunc(classes, func(a, b data.MobiusClass) int { return cmp.Compare(a.Name, b.Name) })
	return classes, nil
}

// readJavaClassDir reads the class declared by each .java file directly in
// dir. A file without a top-level class (an interface or enum) is skipped.
func readJavaClassDir(fsys fs.FS, dir string) ([]data.MobiusClass, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var classes []data.MobiusClass
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".java") {
			continue
		}
		name := path.Join(dir, e.Name())
		src, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		m := mxJavaClassRe.FindStringSubmatch(stripJavaComments(string(src)))
		if m == nil {
			continue
		}
		if want := strings.TrimSuffix(e.Name(), ".java"); m[2] != want {
			return nil, fmt.Errorf("%s: declares class %s, want %s", name, m[2], want)
		}
		parent := m[3]
		if i := strings.LastIndexByte(parent, '.'); i >= 0 {
			parent = parent[i+1:]
		}
		classes = append(classes, data.MobiusClass{Name: m[2], Parent: parent, Abstract: m[1] != ""})
	}
	return classes, nil
}
