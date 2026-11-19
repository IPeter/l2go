package gameserver

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// humanFighter is a valid request for a male Human Fighter (class 0).
func humanFighter(name string) testclient.CreateRequest {
	return testclient.CreateRequest{
		Name: name, Race: 0, Sex: 0, ClassID: 0,
		INT: 21, STR: 40, CON: 43, MEN: 25, DEX: 30, WIT: 11,
		HairStyle: 1, HairColor: 2, Face: 1,
	}
}

// elvenFighter is a valid request for a female Elven Fighter (class 18).
func elvenFighter(name string) testclient.CreateRequest {
	return testclient.CreateRequest{
		Name: name, Race: 1, Sex: 1, ClassID: 18,
		INT: 23, STR: 36, CON: 36, MEN: 26, DEX: 35, WIT: 14,
		HairStyle: 2, HairColor: 3, Face: 1,
	}
}

// create sends CharacterCreate and fails the test on a protocol error.
func create(t *testing.T, c *testclient.GameConn, req testclient.CreateRequest) testclient.CreateResult {
	t.Helper()
	res, err := c.CreateCharacter(testCtx(t), req)
	if err != nil {
		t.Fatalf("CreateCharacter(%q): %v", req.Name, err)
	}
	return res
}

// wantCreated checks that req is accepted with the list before
// CharCreateOk, as Mobius sends them, and that the list has the character.
func wantCreated(t *testing.T, c *testclient.GameConn, req testclient.CreateRequest) testclient.CreateResult {
	t.Helper()
	res := create(t, c, req)
	if !res.Created {
		t.Fatalf("%q: refused with %v, want created", req.Name, res.Reason)
	}
	if !res.ListBeforeOk {
		t.Errorf("%q: CharCreateOk came before the list; Mobius sends the list first", req.Name)
	}
	if _, ok := res.Characters.Find(req.Name); !ok {
		t.Errorf("%q: not in the updated list", req.Name)
	}
	return res
}

// wantRefused checks that req gets CharCreateFail with reason.
func wantRefused(t *testing.T, c *testclient.GameConn, req testclient.CreateRequest, reason testclient.CreateFailReason) {
	t.Helper()
	res := create(t, c, req)
	if res.Created {
		t.Fatalf("%q: created, want %v", req.Name, reason)
	}
	if res.Reason != reason {
		t.Fatalf("%q: reason %v, want %v", req.Name, res.Reason, reason)
	}
}

func TestNewCharacterTemplates(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := e.login(t, "player1")
	templates, err := c.NewCharacter(testCtx(t))
	if err != nil {
		t.Fatalf("NewCharacter: %v", err)
	}
	d := loadData(t)
	if len(templates) != len(startingClasses) {
		t.Fatalf("%d templates, want %d", len(templates), len(startingClasses))
	}
	for i, got := range templates {
		want, ok := d.Template(startingClasses[i])
		if !ok {
			t.Fatalf("data has no template for class %d", startingClasses[i])
		}
		if got.ClassID != int32(want.ClassID) || got.Race != int32(want.Race) {
			t.Errorf("template %d: class %d race %d, want class %d race %d", i, got.ClassID, got.Race, want.ClassID, want.Race)
		}
		s := want.Stats
		for name, pair := range map[string]struct {
			got  testclient.StatRange
			base int
		}{
			"STR": {got.STR, s.STR}, "DEX": {got.DEX, s.DEX}, "CON": {got.CON, s.CON},
			"INT": {got.INT, s.INT}, "WIT": {got.WIT, s.WIT}, "MEN": {got.MEN, s.MEN},
		} {
			wantRange := testclient.StatRange{Max: 0x46, Base: int32(pair.base), Min: 0x0a}
			if pair.got != wantRange {
				t.Errorf("class %d %s = %+v, want %+v", want.ClassID, name, pair.got, wantRange)
			}
		}
	}
	// Spot check against the Mobius files (plan Phase 4).
	hf := templates[0]
	if hf.ClassID != 0 || hf.STR.Base != 40 || hf.DEX.Base != 30 || hf.CON.Base != 43 ||
		hf.INT.Base != 21 || hf.WIT.Base != 11 || hf.MEN.Base != 25 {
		t.Errorf("Human Fighter = %+v", hf)
	}
}

func TestCharacterCreate(t *testing.T) {
	e := newEnv(t, envOptions{})
	ctx := testCtx(t)
	keys := e.issue(t, "player1")
	c := e.dial(t)
	if _, err := c.AuthLogin(ctx, "player1", keys); err != nil {
		t.Fatal(err)
	}
	res := wantCreated(t, c, humanFighter("Frostwind"))

	// What was saved.
	stored, err := e.chars.ListByAccount(ctx, "player1")
	if err != nil || len(stored) != 1 {
		t.Fatalf("stored characters: %v, %v", stored, err)
	}
	ch := stored[0]
	tmpl, _ := loadData(t).Template(0)
	if ch.Name != "Frostwind" || ch.Account != "player1" || ch.Race != 0 || ch.Sex != 0 ||
		ch.ClassID != 0 || ch.BaseClassID != 0 || ch.HairStyle != 1 || ch.HairColor != 2 || ch.Face != 1 {
		t.Errorf("identity and appearance: %+v", ch)
	}
	if ch.Level != 1 || ch.Exp != 0 || ch.SP != 0 || ch.Karma != 0 || ch.AccessLevel != storage.AccessLevelPlayer {
		t.Errorf("level 1 values: %+v", ch)
	}
	// Level 1 HP, MP and CP from the template; HP and MP full, CP 0.
	if ch.MaxHP != 80 || ch.CurHP != 80 || ch.MaxMP != 30 || ch.CurMP != 30 || ch.MaxCP != 32 || ch.CurCP != 0 {
		t.Errorf("vitals: HP %v/%v MP %v/%v CP %v/%v", ch.CurHP, ch.MaxHP, ch.CurMP, ch.MaxMP, ch.CurCP, ch.MaxCP)
	}
	if !slices.Contains(tmpl.CreationPoints, data.Location{X: int(ch.X), Y: int(ch.Y), Z: int(ch.Z)}) {
		t.Errorf("location %d,%d,%d is not a creation point of class 0", ch.X, ch.Y, ch.Z)
	}
	if ch.ObjectID < storage.FirstObjectID {
		t.Errorf("object ID %#x is below %#x", ch.ObjectID, storage.FirstObjectID)
	}

	// Items: the starting equipment of class 0, worn ones in their slots.
	wantSlots := map[int32]storage.PaperdollSlot{
		10:   storage.PaperdollNone,  // Dagger, carried
		1146: storage.PaperdollChest, // Squire's Shirt
		1147: storage.PaperdollLegs,  // Squire's Pants
		2369: storage.PaperdollRHand, // Squire's Sword
		5588: storage.PaperdollNone,  // Tutorial Guide
	}
	if len(ch.Items) != len(wantSlots) {
		t.Fatalf("%d items, want %d: %+v", len(ch.Items), len(wantSlots), ch.Items)
	}
	ids := map[int32]bool{ch.ObjectID: true}
	byItem := map[int32]storage.Item{}
	for _, it := range ch.Items {
		want, ok := wantSlots[it.ItemID]
		if !ok {
			t.Errorf("unexpected item %d", it.ItemID)
			continue
		}
		if it.Slot != want {
			t.Errorf("item %d in slot %d, want %d", it.ItemID, it.Slot, want)
		}
		if it.Count != 1 {
			t.Errorf("item %d count %d, want 1", it.ItemID, it.Count)
		}
		if ids[it.ObjectID] {
			t.Errorf("object ID %#x used twice", it.ObjectID)
		}
		ids[it.ObjectID] = true
		byItem[it.ItemID] = it
	}

	// What the client sees.
	info, ok := res.Characters.Find("Frostwind")
	if !ok {
		t.Fatal("the new character is not listed")
	}
	if info.ObjectID != ch.ObjectID || info.Account != "player1" || info.SessionID != keys.PlayOK1 {
		t.Errorf("object ID %#x account %q session %d; want %#x player1 %d",
			info.ObjectID, info.Account, info.SessionID, ch.ObjectID, keys.PlayOK1)
	}
	if info.Level != 1 || info.ClassID != 0 || info.BaseClassID != 0 || info.Race != 0 || info.Sex != 0 ||
		info.CurrentHP != 80 || info.MaxHP != 80 || info.CurrentMP != 30 || info.MaxMP != 30 ||
		info.HairStyle != 1 || info.HairColor != 2 || info.Face != 1 || info.GameServer != 1 ||
		info.DeleteTimer != 0 || info.Active || info.Enchant != 0 {
		t.Errorf("listed character: %+v", info)
	}
	var wantObj, wantItem [testclient.NumEquipSlots]int32
	for slot, itemID := range map[int]int32{
		testclient.SlotChest:  1146,
		testclient.SlotLegs:   1147,
		testclient.SlotRHand:  2369,
		testclient.SlotRHand2: 2369,
	} {
		wantItem[slot] = itemID
		wantObj[slot] = byItem[itemID].ObjectID
	}
	if info.EquipItemIDs != wantItem {
		t.Errorf("equipment item IDs %v, want %v", info.EquipItemIDs, wantItem)
	}
	if info.EquipObjectIDs != wantObj {
		t.Errorf("equipment object IDs %v, want %v", info.EquipObjectIDs, wantObj)
	}

	// A new connection lists it too.
	_, list := e.login(t, "player1")
	if _, ok := list.Find("Frostwind"); !ok || len(list.Characters) != 1 {
		t.Errorf("list after reconnecting: %+v", list.Characters)
	}
}

// Every starting class can be created and wears its starting equipment.
func TestCharacterCreateEveryClass(t *testing.T) {
	e := newEnv(t, envOptions{configure: func(gs *config.GameServer) { gs.MaxCharactersPerAccount = 0 }})
	c, _ := e.login(t, "player1")
	d := loadData(t)
	for i, class := range startingClasses {
		tmpl, _ := d.Template(class)
		req := humanFighter(fmt.Sprintf("Class%d", class))
		req.ClassID, req.Race = int32(class), int32(tmpl.Race)
		res := wantCreated(t, c, req)
		if len(res.Characters.Characters) != i+1 {
			t.Fatalf("list has %d characters after %d creations", len(res.Characters.Characters), i+1)
		}
		info, _ := res.Characters.Find(req.Name)
		if info.Race != int32(tmpl.Race) || info.MaxHP != tmpl.Level1.HP || info.MaxMP != tmpl.Level1.MP {
			t.Errorf("class %d: race %d HP %v MP %v; want %d %v %v", class, info.Race, info.MaxHP, info.MaxMP,
				tmpl.Race, tmpl.Level1.HP, tmpl.Level1.MP)
		}
		var worn []int32
		for _, it := range d.InitialItems(class) {
			if it.Equipped {
				worn = append(worn, int32(it.ItemID))
			}
		}
		for _, id := range worn {
			if !slices.Contains(info.EquipItemIDs[:], id) {
				t.Errorf("class %d: item %d is not shown as worn: %v", class, id, info.EquipItemIDs)
			}
		}
	}
	// Orc Fighter's Training Gloves are two-handed: right hand.
	orc, _ := e.chars.ListByAccount(testCtx(t), "player1")
	for _, ch := range orc {
		if ch.ClassID != 44 {
			continue
		}
		for _, it := range ch.Items {
			if it.ItemID == 2368 && it.Slot != storage.PaperdollRHand {
				t.Errorf("Training Gloves in slot %d, want right hand", it.Slot)
			}
		}
	}
}

// Every CharCreateFail reason the server's own checks give, in Mobius order.
func TestCharacterCreateFailReasons(t *testing.T) {
	e := newEnv(t, envOptions{configure: func(gs *config.GameServer) { gs.MaxCharactersPerAccount = 2 }})
	ctx := testCtx(t)
	if err := e.chars.Create(ctx, &storage.Character{ObjectID: 1, Account: "other", Name: "Taken"}); err != nil {
		t.Fatal(err)
	}
	c, _ := e.login(t, "player1")

	tests := []struct {
		name   string
		change func(*testclient.CreateRequest)
		want   testclient.CreateFailReason
	}{
		{"empty name", func(r *testclient.CreateRequest) { r.Name = "" }, testclient.CreateFailNameTooLong},
		{"17 characters", func(r *testclient.CreateRequest) { r.Name = strings.Repeat("a", 17) }, testclient.CreateFailNameTooLong},
		{"space", func(r *testclient.CreateRequest) { r.Name = "Bad Name" }, testclient.CreateFailIncorrectName},
		{"underscore", func(r *testclient.CreateRequest) { r.Name = "Bad_Name" }, testclient.CreateFailIncorrectName},
		{"emoji", func(r *testclient.CreateRequest) { r.Name = "Smile😀" }, testclient.CreateFailIncorrectName},
		{"face 3", func(r *testclient.CreateRequest) { r.Face = 3 }, testclient.CreateFailCreationFailed},
		{"face -1", func(r *testclient.CreateRequest) { r.Face = -1 }, testclient.CreateFailCreationFailed},
		{"male hair style 5", func(r *testclient.CreateRequest) { r.HairStyle = 5 }, testclient.CreateFailCreationFailed},
		{"female hair style 7", func(r *testclient.CreateRequest) { r.Sex, r.HairStyle = 1, 7 }, testclient.CreateFailCreationFailed},
		{"hair color 4", func(r *testclient.CreateRequest) { r.HairColor = 4 }, testclient.CreateFailCreationFailed},
		{"not a starting class", func(r *testclient.CreateRequest) { r.ClassID = 1 }, testclient.CreateFailCreationFailed},
		{"unknown class", func(r *testclient.CreateRequest) { r.ClassID = 999 }, testclient.CreateFailCreationFailed},
		{"name taken, other case", func(r *testclient.CreateRequest) { r.Name = "tAKEN" }, testclient.CreateFailNameExists},
		// The name check comes before the appearance check.
		{"bad name and bad face", func(r *testclient.CreateRequest) { r.Name, r.Face = "a b", 9 }, testclient.CreateFailIncorrectName},
		// The appearance check comes before the name-taken check.
		{"taken name and bad face", func(r *testclient.CreateRequest) { r.Name, r.Face = "Taken", 9 }, testclient.CreateFailCreationFailed},
	}
	for _, tt := range tests {
		req := humanFighter("Valid1")
		tt.change(&req)
		res := create(t, c, req)
		if res.Created || res.Reason != tt.want {
			t.Errorf("%s: created=%v reason %v, want %v", tt.name, res.Created, res.Reason, tt.want)
		}
	}
	if e.srv.HackAttempts() == 0 {
		t.Error("impossible appearance values were not counted as hack attempts")
	}

	// Female hair style 6 is fine; the account then reaches its limit of 2.
	lady := humanFighter("Lady")
	lady.Sex, lady.HairStyle = 1, 6
	wantCreated(t, c, lady)
	wantCreated(t, c, humanFighter("Second"))
	wantRefused(t, c, humanFighter("Third"), testclient.CreateFailTooManyCharacters)

	// The limit is checked before the name: a taken name also gets "too many".
	wantRefused(t, c, humanFighter("Taken"), testclient.CreateFailTooManyCharacters)

	n, err := e.chars.CountByAccount(ctx, "player1")
	if err != nil || n != 2 {
		t.Errorf("player1 has %d characters (%v), want 2: refused requests must not save anything", n, err)
	}
	// The connection is still usable after all the refusals.
	if _, err := c.NewCharacter(ctx); err != nil {
		t.Fatalf("NewCharacter: %v", err)
	}
}

// A taken name is refused for the same account too.
func TestCharacterCreateNameExistsSameAccount(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := e.login(t, "player1")
	wantCreated(t, c, humanFighter("Twin"))
	wantRefused(t, c, humanFighter("TWIN"), testclient.CreateFailNameExists)
}

func TestCharacterCreateMalformedClosesConnection(t *testing.T) {
	e := newEnv(t, envOptions{})
	ctx := testCtx(t)
	c, _ := e.login(t, "player1")
	if err := c.SendRaw(ctx, clientpackets.OpCharacterCreate, []byte{'A', 0, 0, 0, 1, 0}); err != nil {
		t.Fatal(err)
	}
	if err := c.ExpectClosed(ctx); err != nil {
		t.Fatal(err)
	}
}

// Two clients create the same name at once: one wins, the others get
// "name exists".
func TestCharacterCreateConcurrentSameName(t *testing.T) {
	e := newEnv(t, envOptions{})
	const n = 8
	conns := make([]*testclient.GameConn, n)
	for i := range conns {
		conns[i], _ = e.login(t, fmt.Sprintf("racer%d", i))
	}
	ctx := testCtx(t)
	results := make([]testclient.CreateResult, n)
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, c := range conns {
		wg.Go(func() {
			<-start
			results[i], errs[i] = c.CreateCharacter(ctx, humanFighter("Samename"))
		})
	}
	close(start)
	wg.Wait()

	created := 0
	for i, res := range results {
		switch {
		case errs[i] != nil:
			t.Errorf("client %d: %v", i, errs[i])
		case res.Created:
			created++
		case res.Reason != testclient.CreateFailNameExists:
			t.Errorf("client %d: reason %v, want name exists", i, res.Reason)
		}
	}
	if created != 1 {
		t.Errorf("%d clients created the name, want 1", created)
	}
	if ok, _ := e.chars.NameExists(ctx, "samename"); !ok {
		t.Error("the name was not saved")
	}
}

// Two connections of one account cannot pass the character limit together.
func TestCharacterCreateConcurrentLimit(t *testing.T) {
	e := newEnv(t, envOptions{configure: func(gs *config.GameServer) { gs.MaxCharactersPerAccount = 1 }})
	const n = 4
	conns := make([]*testclient.GameConn, n)
	for i := range conns {
		conns[i], _ = e.login(t, "player1") // fresh keys for each connection
	}
	ctx := testCtx(t)
	results := make([]testclient.CreateResult, n)
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, c := range conns {
		wg.Go(func() {
			<-start
			results[i], errs[i] = c.CreateCharacter(ctx, humanFighter(fmt.Sprintf("Twin%d", i)))
		})
	}
	close(start)
	wg.Wait()

	created := 0
	for i, res := range results {
		switch {
		case errs[i] != nil:
			t.Errorf("client %d: %v", i, errs[i])
		case res.Created:
			created++
		case res.Reason != testclient.CreateFailTooManyCharacters:
			t.Errorf("client %d: reason %v, want too many characters", i, res.Reason)
		}
	}
	if created != 1 {
		t.Errorf("%d characters created, want 1", created)
	}
}

// A created character is still listed after a server restart.
func TestCharacterSurvivesRestart(t *testing.T) {
	first := newEnv(t, envOptions{})
	c, _ := first.login(t, "player1")
	wantCreated(t, c, humanFighter("Keeper"))
	if err := first.stop(); err != nil {
		t.Fatal(err)
	}

	second := newEnv(t, envOptions{chars: first.chars, ids: first.ids})
	c2, list := second.login(t, "player1")
	info, ok := list.Find("Keeper")
	if !ok {
		t.Fatalf("character missing after the restart: %+v", list.Characters)
	}
	if info.EquipItemIDs[testclient.SlotRHand] != 2369 {
		t.Errorf("equipment lost: %v", info.EquipItemIDs)
	}
	// New object IDs do not clash with the old ones.
	res := wantCreated(t, c2, humanFighter("Newcomer"))
	if a, _ := res.Characters.Find("Newcomer"); a.ObjectID == info.ObjectID {
		t.Error("the new character reused an object ID")
	}
}

func TestCharacterCreateRejectedByScript(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"events/rules.lua": `
events.on_character_create(function(ev)
  if ev.name == "Badname" then
    return events.reject("incorrect_name")
  end
  if ev.account == "player1" and ev.name == "Taken" then
    return events.reject("name_exists")
  end
  assert(ev.race == 1 and ev.sex == 1 and ev.class_id == 18, "race, sex, class")
  assert(ev.hair_style == 2 and ev.hair_color == 3 and ev.face == 1, "appearance")
end)
`})
	e := newEnv(t, envOptions{scripts: h.Engine()})
	c, _ := e.login(t, "player1")

	wantRefused(t, c, elvenFighter("Badname"), testclient.CreateFailIncorrectName)
	// The script sees the template's race, not the one the client sent.
	odd := elvenFighter("Goodname")
	odd.Race = 4
	wantCreated(t, c, odd)
	wantRefused(t, c, elvenFighter("Taken"), testclient.CreateFailNameExists)
	for _, l := range h.Logs() {
		if l.Msg == "script handler failed" {
			t.Errorf("a handler failed: %v", l.Attrs)
		}
	}
	if n, _ := e.chars.CountByAccount(testCtx(t), "player1"); n != 1 {
		t.Errorf("%d characters saved, want 1", n)
	}
}

// The shipped default script refuses the Mobius ForbiddenNames.
func TestCharacterCreateDefaultScript(t *testing.T) {
	h := scripttest.Load(t, filepath.Join("..", "..", "scripts"))
	e := newEnv(t, envOptions{scripts: h.Engine()})
	c, _ := e.login(t, "player1")

	wantRefused(t, c, humanFighter("Announcer"), testclient.CreateFailIncorrectName)
	wantCreated(t, c, humanFighter("Frostwind"))
}

// Every reason a script can give arrives as its own Interlude code, and the
// server's checks run before the script.
func TestCharacterCreateEveryReason(t *testing.T) {
	reasons := scripting.Reasons(scripting.EventCharacterCreate)
	var src strings.Builder
	src.WriteString("local reasons = {\n")
	for i, r := range reasons {
		fmt.Fprintf(&src, "  Reason%d = %q,\n", i, r)
	}
	src.WriteString("}\nevents.on_character_create(function(ev)\n  return events.reject(reasons[ev.name] or \"creation_not_allowed\")\nend)\n")
	h := scripttest.LoadSource(t, map[string]string{"events/reason.lua": src.String()})
	e := newEnv(t, envOptions{scripts: h.Engine()})
	c, _ := e.login(t, "player1")

	want := map[string]testclient.CreateFailReason{
		scripting.ReasonCreationFailed:     testclient.CreateFailCreationFailed,
		scripting.ReasonTooManyCharacters:  testclient.CreateFailTooManyCharacters,
		scripting.ReasonNameExists:         testclient.CreateFailNameExists,
		scripting.ReasonNameTooLong:        testclient.CreateFailNameTooLong,
		scripting.ReasonIncorrectName:      testclient.CreateFailIncorrectName,
		scripting.ReasonCreationNotAllowed: testclient.CreateFailNotAllowed,
	}
	if len(want) != len(reasons) {
		t.Fatalf("scripting has %d reasons, the test knows %d", len(reasons), len(want))
	}
	for i, r := range reasons {
		code, ok := want[r]
		if !ok {
			t.Fatalf("reason %q has no expected code", r)
		}
		wantRefused(t, c, humanFighter(fmt.Sprintf("Reason%d", i)), code)
	}
	// A name the server refuses never reaches the script.
	wantRefused(t, c, humanFighter("No_Script"), testclient.CreateFailIncorrectName)
}

// Changing the script and reloading changes behaviour without a restart.
func TestCharacterCreateReload(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"events/rules.lua": `
events.on_character_create(function(ev)
  if ev.name == "First" then return events.reject("incorrect_name") end
end)
`})
	e := newEnv(t, envOptions{scripts: h.Engine()})
	c, _ := e.login(t, "player1")
	wantRefused(t, c, humanFighter("First"), testclient.CreateFailIncorrectName)
	wantCreated(t, c, humanFighter("Second"))

	scripttest.WriteScript(t, h.Dir(), "events/rules.lua", `
events.on_character_create(function(ev)
  if ev.name == "Third" then return events.reject("creation_not_allowed") end
end)
`)
	// As the server reloads: prepare off the loop, swap on the loop.
	ctx := testCtx(t)
	p, err := h.Engine().Prepare(ctx)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if err := e.game.OnLoop(ctx, func() { h.Engine().Swap(p) }); err != nil {
		t.Fatalf("swapping on the loop: %v", err)
	}
	wantCreated(t, c, humanFighter("First"))
	wantRefused(t, c, humanFighter("Third"), testclient.CreateFailNotAllowed)
}

// A handler error is logged and does not block creation.
func TestCharacterCreateHandlerError(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"events/broken.lua": `
events.on_character_create(function(ev)
  local x = nil
  return x.field
end)
`})
	e := newEnv(t, envOptions{scripts: h.Engine()})
	c, _ := e.login(t, "player1")
	wantCreated(t, c, humanFighter("Anyone"))
}

func TestCharacterCreateWithoutScripts(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := e.login(t, "player1")
	wantCreated(t, c, humanFighter("Announcer"))
}

// If the scripts engine is closed, creation fails safely.
func TestCharacterCreateEngineStopped(t *testing.T) {
	dir := scripttest.WriteScripts(t, map[string]string{"a.lua": ""})
	eng, err := scripting.New(context.Background(), scripting.Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	eng.Close()

	e := newEnv(t, envOptions{scripts: eng})
	c, _ := e.login(t, "player1")
	wantRefused(t, c, humanFighter("Anyone"), testclient.CreateFailCreationFailed)
	if n, _ := e.chars.CountByAccount(testCtx(t), "player1"); n != 0 {
		t.Errorf("%d characters saved, want 0", n)
	}
}

// If the world loop does not run the hook, creation fails safely.
func TestCharacterCreateLoopStopped(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"events/ok.lua": `
events.on_character_create(function(ev) end)
`})
	e := newEnv(t, envOptions{scripts: h.Engine()})
	c, _ := e.login(t, "player1")
	e.game.stopLoop()
	wantRefused(t, c, humanFighter("Anyone"), testclient.CreateFailCreationFailed)
	if n, _ := e.chars.CountByAccount(testCtx(t), "player1"); n != 0 {
		t.Errorf("%d characters saved, want 0", n)
	}
}

// The hook runs on the loop once per character that passes the server's
// own checks; refused requests never reach the loop.
func TestCharacterCreateHookRunsOnTheLoop(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"events/ok.lua": `
events.on_character_create(function(ev) end)
`})
	e := newEnv(t, envOptions{scripts: h.Engine()})
	c, _ := e.login(t, "player1")
	wantRefused(t, c, humanFighter("Bad Name"), testclient.CreateFailIncorrectName)
	if n := e.game.onLoopCalls(); n != 0 {
		t.Errorf("%d OnLoop calls for a refused name, want 0", n)
	}
	wantCreated(t, c, humanFighter("Looped"))
	if n := e.game.onLoopCalls(); n != 1 {
		t.Errorf("%d OnLoop calls after one creation, want 1", n)
	}
}
