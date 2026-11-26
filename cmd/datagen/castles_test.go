package main

import (
	"reflect"
	"testing"
	"testing/fstest"
)

func TestJavaIntArray(t *testing.T) {
	src := `class X {
	private static final int[] MAXES =
	{
		10, 10, // Gludio
		15 /* Dion */, 20
	};
	private static final int[] OTHER = { 1 };
}`
	fsys := fstest.MapFS{"X.java": &fstest.MapFile{Data: []byte(src)}}
	got, err := javaIntArray(fsys, "X.java", "MAXES")
	if err != nil || !reflect.DeepEqual(got, []int{10, 10, 15, 20}) {
		t.Fatalf("javaIntArray = %v, %v", got, err)
	}
	if _, err := javaIntArray(fsys, "X.java", "NONE"); err == nil {
		t.Fatal("a missing array is no error")
	}
}

func TestIniTowers(t *testing.T) {
	ini := map[string]string{
		"GludioFlameTower1":   "-18154,107591,-2560,13004,70017,70019",
		"GludioFlameTower2":   "-19329,108154,-2384,13004,70018,70020",
		"GludioControlTower1": "-18325,112811,-2377,13002",
		"GludioControlTower3": "-18113,108597,-2343,13002", // no 2: Mobius stops at the gap
	}
	ft, err := iniTowers(ini, "GludioFlameTower", true)
	if err != nil || len(ft) != 2 || !reflect.DeepEqual(ft[1].Zones, []int{70018, 70020}) || ft[0].X != -18154 {
		t.Fatalf("flame towers = %+v, %v", ft, err)
	}
	ct, err := iniTowers(ini, "GludioControlTower", false)
	if err != nil || len(ct) != 1 || ct[0].NpcID != 13002 || ct[0].Zones != nil {
		t.Fatalf("control towers = %+v, %v", ct, err)
	}
}
