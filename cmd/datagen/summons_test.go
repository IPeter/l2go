package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

const samplePetXML = `<?xml version="1.0" encoding="UTF-8"?>
<pets xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="../../xsd/PetData.xsd">
	<!-- pet_wolf_a -->
	<pet id="12077" itemId="2375" index="1">
		<set name="food" val="2515;2516;2515" />
		<set name="hungry_limit" val="55" />
		<set name="load" val="54510" />
		<set name="evolve" val="12" />
		<stats>
			<stat level="2">
				<set name="max_meal" val="244" />
				<set name="exp" val="636" />
				<set name="get_exp_type" val="73" />
				<set name="consume_meal_in_battle" val="2" />
				<set name="consume_meal_in_normal" val="2" />
				<set name="org_pattack" val="2.33050847457627" />
				<set name="org_pdefend" val="11.5384615384615" />
				<set name="org_mattack" val="1.59143518518519" />
				<set name="org_mdefend" val="8.44334539296436" />
				<set name="org_hp" val="24.0458413461538" />
				<set name="org_mp" val="23.64" />
				<set name="org_hp_regen" val="2" />
				<set name="org_mp_regen" val="0.9" />
				<set name="soulshot_count" val="1" />
				<set name="spiritshot_count" val="1" />
			</stat>
			<stat level="1">
				<set name="max_meal" val="248" />
				<set name="exp" val="0" />
				<set name="get_exp_type" val="73" />
				<set name="consume_meal_in_battle" val="2" />
				<set name="consume_meal_in_normal" val="2" />
				<set name="org_pattack" val="2.11864406779661" />
				<set name="org_pdefend" val="11.1111111111111" />
				<set name="org_mattack" val="1.44675925925926" />
				<set name="org_mdefend" val="8.13062889692864" />
				<set name="org_hp" val="19.8725961538461" />
				<set name="org_mp" val="20" />
				<set name="org_hp_regen" val="2" />
				<set name="org_mp_regen" val="0.9" />
				<set name="soulshot_count" val="1" />
				<set name="spiritshot_count" val="1" />
			</stat>
		</stats>
	</pet>
</pets>
`

const sampleWyvernXML = `<?xml version="1.0" encoding="UTF-8"?>
<pets>
	<pet id="12621" itemId="-1" index="99">
		<set name="food" val="6316" />
		<set name="sync_level" val="1" />
		<skills>
			<skill minLevel="1" skillId="4289" skillLevel="0" />
		</skills>
		<stats>
			<stat level="1">
				<set name="exp" val="1.5E3" />
				<set name="get_exp_type" val="-1" />
				<set name="consume_meal_in_battle" val="20" />
				<set name="consume_meal_in_normal" val="10" />
				<set name="max_meal" val="508" />
				<set name="soulshot_count" val="1" />
				<set name="spiritshot_count" val="1" />
				<set name="org_hp" val="28" />
				<set name="org_mp" val="25" />
				<set name="org_pdefend" val="20" />
				<set name="org_mdefend" val="64" />
				<set name="org_pattack" val="9" />
				<set name="org_mattack" val="6" />
				<set name="org_hp_regen" val="2" />
				<set name="org_mp_regen" val="1" />
				<set name="speed_on_ride" val="0" walk="250" run="250" slowSwim="140" fastSwim="140" slowFly="250" fastFly="250" />
			</stat>
		</stats>
	</pet>
</pets>
`

const samplePetSkillXML = `<?xml version="1.0" encoding="UTF-8"?>
<list xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="xsd/PetSkillData.xsd">
	<!-- SUMMON CORRUPTED MAN ACTIVE SKILLS -->
	<skill npcId="14038" skillId="4260" skillLevel="0" />
	<skill npcId="12077" skillId="4025" skillLevel="1" />
	<skill npcId="14038" skillId="4138" skillLevel="2" />
</list>
`

const sampleCubicXML = `<?xml version="1.0" encoding="UTF-8"?>
<list xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="../../xsd/cubics.xsd">
	<!-- Storm Cubic -->
	<cubic id="1" level="2" slot="1" duration="900" delay="10" maxCount="30" useUp="1" power="434.0" selectionMode="RANDOM" targetType="TARGET">
		<skills>
			<skill id="4049" level="2" /> <!-- Cubic DD -->
		</skills>
	</cubic>
	<cubic id="1" level="1" slot="1" duration="900" delay="10" maxCount="30" useUp="1" power="282.0" selectionMode="RANDOM" targetType="TARGET">
		<skills>
			<skill id="4049" level="1" />
		</skills>
	</cubic>
</list>
`

const sampleCubicSubXML = `<?xml version="1.0" encoding="UTF-8"?>
<list>
	<cubic id="3" level="1" slot="3" duration="900" delay="13" maxCount="20" useUp="1" selectionMode="ORDERED" targetType="HEAL">
		<conditions>
			<range value="900" />
		</conditions>
		<skills>
			<skill id="4051" level="1" triggerRate="50" target="MASTER" targetDebuff="true" priority="true">
				<conditions>
					<hp type="LESSER" percent="60" />
					<healthPercent min="10" max="90" />
				</conditions>
			</skill>
		</skills>
	</cubic>
	<cubic id="2" level="1" slot="2" duration="900" delay="15" maxCount="20" useUp="0">
		<skills />
	</cubic>
</list>
`

const (
	sampleWolfFile   = petXMLDir + "/12077_Wolf.xml"
	sampleWyvernFile = petXMLDir + "/12621_Wyvern.xml"
	sampleCubicFile  = cubicXMLDir + "/cubic-01.xml"
	sampleCubicSub   = cubicXMLDir + "/more/cubic-03.xml"
)

func summonSample(t *testing.T) fstest.MapFS {
	t.Helper()
	m := sampleFS(t)
	m[sampleWolfFile] = file(samplePetXML)
	m[sampleWyvernFile] = file(sampleWyvernXML)
	m[petXMLDir+"/readme.txt"] = file("not a pet")
	m[petXMLDir+"/sub/12000_Other.xml"] = file("not read: Mobius reads the pet folder only")
	m[petSkillXML] = file(samplePetSkillXML)
	m[sampleCubicFile] = file(sampleCubicXML)
	m[sampleCubicSub] = file(sampleCubicSubXML)
	return m
}

func TestSummons(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(summonSample(t), false, []string{"summons"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		data.PetsFile: "[\n" +
			`{"npcId":12077,"itemId":2375,"index":1,"food":[2515,2516],"hungryLimit":55,"load":54510,"levels":[` + "\n" +
			`{"level":1,"maxMeal":248,"exp":0,"ownerExpTaken":73,"mealInBattle":2,"mealInNormal":2,"pAtk":2.1186439990997314,"pDef":11.11111068725586,"mAtk":1.4467592239379883,"mDef":8.13062858581543,"hp":19.872596740722656,"mp":20,"hpRegen":2,"mpRegen":0.8999999761581421,"soulshotCount":1,"spiritshotCount":1},` + "\n" +
			`{"level":2,"maxMeal":244,"exp":636,"ownerExpTaken":73,"mealInBattle":2,"mealInNormal":2,"pAtk":2.3305084705352783,"pDef":11.538461685180664,"mAtk":1.591435194015503,"mDef":8.443345069885254,"hp":24.045841217041016,"mp":23.639999389648438,"hpRegen":2,"mpRegen":0.8999999761581421,"soulshotCount":1,"spiritshotCount":1}` + "\n" +
			"]},\n" +
			// Mobius defaults: hungry limit 1, load 20000.
			`{"npcId":12621,"itemId":-1,"index":99,"food":[6316],"hungryLimit":1,"load":20000,"syncLevel":true,"skills":[{"skillId":4289,"skillLevel":0,"minLevel":1}],"levels":[` + "\n" +
			`{"level":1,"maxMeal":508,"exp":1500,"ownerExpTaken":-1,"mealInBattle":20,"mealInNormal":10,"pAtk":9,"pDef":20,"mAtk":6,"mDef":64,"hp":28,"mp":25,"hpRegen":2,"mpRegen":1,"soulshotCount":1,"spiritshotCount":1,"walkOnRide":250,"runOnRide":250,"slowSwimOnRide":140,"fastSwimOnRide":140,"slowFlyOnRide":250,"fastFlyOnRide":250}` + "\n" +
			"]}\n]\n",
		data.PetSkillsFile: "[\n" +
			`{"npcId":14038,"skillId":4260,"skillLevel":0},` + "\n" +
			`{"npcId":12077,"skillId":4025,"skillLevel":1},` + "\n" +
			`{"npcId":14038,"skillId":4138,"skillLevel":2}` + "\n]\n",
		data.CubicsFile: "[\n" +
			`{"id":1,"level":1,"slot":1,"duration":900,"delay":10,"maxCount":30,"useUp":1,"power":282,"selectionMode":"RANDOM","targetType":"TARGET","skills":[{"id":4049,"level":1,"triggerRate":100,"targetType":"TARGET"}]},` + "\n" +
			`{"id":1,"level":2,"slot":1,"duration":900,"delay":10,"maxCount":30,"useUp":1,"power":434,"selectionMode":"RANDOM","targetType":"TARGET","skills":[{"id":4049,"level":2,"triggerRate":100,"targetType":"TARGET"}]},` + "\n" +
			// The defaults: power 0, RANDOM, TARGET.
			`{"id":2,"level":1,"slot":2,"duration":900,"delay":15,"maxCount":20,"useUp":0,"power":0,"selectionMode":"RANDOM","targetType":"TARGET","skills":[]},` + "\n" +
			`{"id":3,"level":1,"slot":3,"duration":900,"delay":13,"maxCount":20,"useUp":1,"power":0,"selectionMode":"ORDERED","targetType":"HEAL","conditions":[{"kind":"range","range":900}],` +
			`"skills":[{"id":4051,"level":1,"triggerRate":50,"targetType":"MASTER","targetDebuff":true,"priority":true,"conditions":[{"kind":"hp","hpType":"LESSER","percent":60},{"kind":"healthPercent","min":10,"max":90}]}]}` + "\n]\n",
	} {
		if got := string(files[name]); got != want {
			t.Errorf("%s =\n%s\nwant\n%s", name, got, want)
		}
	}
	fsys := fstest.MapFS{}
	for _, name := range []string{data.PetsFile, data.PetSkillsFile, data.CubicsFile} {
		fsys[name] = &fstest.MapFile{Data: files[name]}
	}
	s, err := data.LoadSummonsFS(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := s.PetByItem(2375); !ok || p.NpcID != 12077 {
		t.Errorf("PetByItem(2375) = %+v, %v", p, ok)
	} else if l, ok := p.Level(2); !ok || l.HP != float64(float32(24.0458413461538)) {
		t.Errorf("wolf level 2 = %+v, %v", l, ok)
	}
	if c, ok := s.Cubic(3, 1); !ok || len(c.Skills) != 1 || len(c.Skills[0].Conditions) != 2 {
		t.Errorf("Cubic(3, 1) = %+v, %v", c, ok)
	}
	if s := "summons: 2 pets, 3 pet skill rows, 4 cubic levels"; !strings.Contains(log.String(), s) {
		t.Errorf("log %q does not mention %q", log.String(), s)
	}
}

func TestSummonsErrors(t *testing.T) {
	edit := func(name, old, new string) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, m fstest.MapFS) {
			t.Helper()
			for k, v := range summonSample(t) {
				m[k] = v
			}
			src := string(m[name].Data)
			if !strings.Contains(src, old) {
				t.Fatalf("%s does not contain %q", name, old)
			}
			m[name] = file(strings.Replace(src, old, new, 1))
		}
	}
	wolf := func(old, new string) func(*testing.T, fstest.MapFS) { return edit(sampleWolfFile, old, new) }
	cubic := func(old, new string) func(*testing.T, fstest.MapFS) { return edit(sampleCubicSub, old, new) }
	expectErrors(t, []string{"summons"}, []sampleCase{
		{"no pet folder", func(*testing.T, fstest.MapFS) {}, "read " + petXMLDir},
		{"pet twice", edit(sampleWyvernFile, `id="12621"`, `id="12077"`), "pet 12077 is listed twice"},
		{"pet root", func(t *testing.T, m fstest.MapFS) {
			edit(sampleWyvernFile, "<pets>", "<list>")(t, m)
			m[sampleWyvernFile] = file(strings.Replace(string(m[sampleWyvernFile].Data), "</pets>", "</list>", 1))
		}, "root element is <list>, want <pets>"},
		{"pet set", wolf(`name="load"`, `name="weight"`), "unknown pet set weight"},
		{"pet element", wolf("<stats>", "<stat2 />\n<stats>"), "<stat2>: unknown element"},
		{"pet food", wolf(`val="2515;2516;2515"`, `val="2515,2516"`), `food: "2515,2516" is not an int`},
		{"pet level twice", wolf(`<stat level="1">`, `<stat level="2">`), "level 2 is listed twice"},
		{"pet level gap", wolf(`<stat level="1">`, `<stat level="4">`), "level 4 does not follow level 2"},
		{"pet stat missing", wolf(`<set name="org_mp" val="23.64" />`, ""), "level 2: no org_mp"},
		{"pet stat unknown", wolf(`<set name="org_mp" val="23.64" />`, `<set name="org_mp" val="23.64" /><set name="org_cp" val="1" />`), "level 2: unknown set org_cp"},
		{"pet stat not a number", wolf(`val="23.64"`, `val="x"`), `org_mp: "x" is not a number`},
		{"pet shots not a short", wolf(`<set name="soulshot_count" val="1" />`, `<set name="soulshot_count" val="40000" />`), `soulshot_count: "40000" is not an integer`},
		{"pet item twice", edit(sampleWyvernFile, `itemId="-1"`, `itemId="2375"`), "item 2375 also summons pet 12077"},
		{"pet ride speed", edit(sampleWyvernFile, ` walk="250"`, ""), "missing attribute walk"},
		{"pet skill attribute", edit(petSkillXML, `skillLevel="2"`, `level="2"`), "missing attribute skillLevel"},
		{"cubic twice", edit(sampleCubicFile, `id="1" level="2"`, `id="1" level="1"`), "cubic 1 level 1 is listed twice"},
		{"cubic required", cubic(` delay="15"`, ""), "missing attribute delay"},
		{"cubic not decimal", cubic(`delay="15"`, `delay="0x15"`), `attribute delay: "0x15" is not an int`},
		{"cubic target type", cubic(`targetType="HEAL"`, `targetType="heal"`), `attribute targetType: "heal" is not one of`},
		{"cubic selection mode", cubic(`selectionMode="ORDERED"`, `selectionMode="FIRST"`), `attribute selectionMode: "FIRST" is not one of`},
		{"cubic condition", cubic(`<range value="900" />`, `<level value="900" />`), "<level>: unknown element"},
		{"cubic hp type", cubic(`type="LESSER"`, `type="LESS"`), `attribute type: "LESS" is not one of GREATER, LESSER`},
		{"cubic hp no type", cubic(`type="LESSER" `, ""), "missing attribute type"},
		{"cubic skill attribute", cubic(`priority="true"`, `priority="true" chance="5"`), "unknown attribute chance"},
		{"cubic skill target", cubic(`target="MASTER"`, `target="SELF"`), `attribute target: "SELF" is not one of`},
		{"cubic level", cubic(`level="1" slot="3"`, `level="0" slot="3"`), "cubic 3 level 0: ID and level must be positive"},
	})
}

// Spot checks of what datagen makes from the local Mobius clone, against
// the Mobius XML.
func TestSummonsMobius(t *testing.T) {
	if _, err := os.Stat(mobiusDir); err != nil {
		t.Skipf("no Mobius clone at %s", mobiusDir)
	}
	files, err := generate(os.DirFS(mobiusDir), false, []string{"summons"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{}
	for name, b := range files {
		fsys[name] = &fstest.MapFile{Data: b}
	}
	s, err := data.LoadSummonsFS(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Pets) != 12 || len(s.SummonData.PetSkills) != 1220 {
		t.Errorf("%d pets, %d pet skill rows; want 12, 1220", len(s.Pets), len(s.SummonData.PetSkills))
	}
	ids := map[int]bool{}
	for _, c := range s.Cubics {
		ids[c.ID] = true
	}
	if len(ids) != 9 {
		t.Errorf("%d cubic IDs, want 9", len(ids))
	}
	// 12077_Wolf.xml
	w, ok := s.Pet(12077)
	if !ok {
		t.Fatal("no wolf")
	}
	if w.ItemID != 2375 || len(w.Food) != 1 || w.Food[0] != 2515 || w.HungryLimit != 55 || w.Load != 54510 {
		t.Errorf("wolf = %+v", w)
	}
	if l, ok := w.Level(1); !ok || l.MaxMeal != 248 {
		t.Errorf("wolf level 1 = %+v, %v", l, ok)
	}
	if l, ok := w.Level(2); !ok || l.Exp != 636 {
		t.Errorf("wolf level 2 = %+v, %v", l, ok)
	}
	// cubic-01.xml: Storm Cubic.
	c, ok := s.Cubic(1, 1)
	if !ok || c.Duration != 900 || c.Delay != 10 || c.MaxCount != 30 || c.Power != 282 ||
		len(c.Skills) != 1 || c.Skills[0].ID != 4049 || c.Skills[0].Level != 1 {
		t.Errorf("Cubic(1, 1) = %+v, %v", c, ok)
	}
}

// The committed summon data is what datagen makes from Mobius.
func TestCommittedSummonsMatchMobius(t *testing.T) { committed(t, repoData, false, "summons") }
