package game

import (
	"errors"
	"testing"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

func TestSelectCharacterSendsCharSelected(t *testing.T) {
	e := newEnv(t, nil)
	c := e.createCharacter("hero", "Hero", classHumanFighter, storage.AccessLevelPlayer)
	sess := newSession("hero")

	if err := e.srv.SelectCharacter(e.ctx, sess, 0); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	ps := sess.take(t)
	checkNames(t, "after CharacterSelect", ps, "CharSelected")
	got := ps[0].(testclient.CharSelected)
	switch {
	case got.ObjectID != c.ObjectID || got.Name != "Hero":
		t.Errorf("CharSelected is for %d %q, want %d Hero", got.ObjectID, got.Name, c.ObjectID)
	case got.SessionID != testPlayKey:
		t.Errorf("CharSelected session ID %#x, want %#x", got.SessionID, testPlayKey)
	case got.X != c.X || got.Y != c.Y || got.Z != c.Z:
		t.Errorf("CharSelected at (%d, %d, %d), want (%d, %d, %d)", got.X, got.Y, got.Z, c.X, c.Y, c.Z)
	case got.Level != 1 || got.ClassID != classHumanFighter || got.ClassID2 != classHumanFighter:
		t.Errorf("CharSelected level %d class %d/%d, want 1 and %d", got.Level, got.ClassID, got.ClassID2, classHumanFighter)
	case got.GameTime != testGameTime:
		t.Errorf("CharSelected game time %d, want %d", got.GameTime, testGameTime)
	// Human Fighter base stats (Mobius HumanFighter.xml).
	case got.STR != 40 || got.CON != 43 || got.DEX != 30 || got.INT != 21 || got.WIT != 11 || got.MEN != 25:
		t.Errorf("CharSelected stats STR %d CON %d DEX %d INT %d WIT %d MEN %d", got.STR, got.CON, got.DEX, got.INT, got.WIT, got.MEN)
	}
	if st := sess.State(); st != StateEntering {
		t.Errorf("state after CharacterSelect = %v, want StateEntering", st)
	}
}

// slotSession is a session that keeps the slots of its last character list
// (CharacterSlots).
type slotSession struct {
	*fakeSession
	slots []int32
}

func (s *slotSession) SlotObjectID(slot int32) (int32, bool) {
	if slot < 0 || int(slot) >= len(s.slots) {
		return 0, false
	}
	return s.slots[slot], true
}

func TestSelectCharacterUsesTheSessionsSlots(t *testing.T) {
	e := newEnv(t, nil)
	first := e.createCharacter("hero", "First", classHumanFighter, storage.AccessLevelPlayer)
	second := e.createCharacter("hero", "Second", classHumanMystic, storage.AccessLevelPlayer)
	// The session listed them the other way round.
	sess := &slotSession{fakeSession: newSession("hero"), slots: []int32{second.ObjectID, first.ObjectID}}

	if err := e.srv.SelectCharacter(e.ctx, sess, 0); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	ps := sess.take(t)
	checkNames(t, "after CharacterSelect", ps, "CharSelected")
	if got := ps[0].(testclient.CharSelected); got.ObjectID != second.ObjectID || got.Name != "Second" {
		t.Errorf("selected %d %q, want the session's slot 0: %d Second", got.ObjectID, got.Name, second.ObjectID)
	}
	if err := e.srv.SelectCharacter(e.ctx, &slotSession{fakeSession: newSession("hero")}, 0); !errors.Is(err, ErrNoCharacter) {
		t.Errorf("SelectCharacter of a slot the session does not have = %v, want ErrNoCharacter", err)
	}
}

func TestSelectCharacterRefusals(t *testing.T) {
	e := newEnv(t, nil)
	e.createCharacter("hero", "Hero", classHumanFighter, storage.AccessLevelPlayer)
	e.createCharacter("banned", "Outlaw", classHumanFighter, storage.AccessLevelBanned)

	t.Run("no such slot", func(t *testing.T) {
		sess := newSession("hero")
		for _, slot := range []int{-1, 1, 7} {
			if err := e.srv.SelectCharacter(e.ctx, sess, slot); !errors.Is(err, ErrNoCharacter) {
				t.Errorf("SelectCharacter(slot %d) = %v, want ErrNoCharacter", slot, err)
			}
		}
		if ps := sess.take(t); len(ps) != 0 {
			t.Errorf("sent %q for a slot without a character; Mobius sends nothing", testclient.Names(ps))
		}
	})

	t.Run("repeated", func(t *testing.T) {
		sess := newSession("hero")
		if err := e.srv.SelectCharacter(e.ctx, sess, 0); err != nil {
			t.Fatalf("SelectCharacter: %v", err)
		}
		sess.take(t)
		if err := e.srv.SelectCharacter(e.ctx, sess, 0); !errors.Is(err, ErrAlreadySelected) {
			t.Errorf("second SelectCharacter = %v, want ErrAlreadySelected", err)
		}
		if ps := sess.take(t); len(ps) != 0 {
			t.Errorf("a repeated CharacterSelect sent %q", testclient.Names(ps))
		}
		e.srv.Disconnected(sess)
	})

	t.Run("banned", func(t *testing.T) {
		sess := newSession("banned")
		if err := e.srv.SelectCharacter(e.ctx, sess, 0); !errors.Is(err, ErrCharacterBanned) {
			t.Errorf("SelectCharacter of a banned character = %v, want ErrCharacterBanned", err)
		}
		final, _, closed := sess.closedWith(t)
		if _, isClose := final.(testclient.ServerClose); !closed || !isClose {
			t.Errorf("banned character: closed %v with %v, want ServerClose", closed, final)
		}
	})
}

// TestEnterWorldSequence checks the packets of EnterWorld against Mobius's
// order (EnterWorld.runImpl) and their content for a new Human Fighter.
func TestEnterWorldSequence(t *testing.T) {
	e := newEnv(t, nil)
	c := e.createCharacter("hero", "Hero", classHumanFighter, storage.AccessLevelPlayer)
	sess := newSession("hero")
	if err := e.srv.SelectCharacter(e.ctx, sess, 0); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	sess.take(t)

	e.srv.EnterWorld(sess)
	ps := sess.take(t)
	checkNames(t, "EnterWorld", ps,
		"UserInfo", "SendMacroList", "ItemList", "ShortcutInit", "HennaInfo", "QuestList",
		"EtcStatusUpdate", "ExStorageMaxCount", "FriendList", "SystemMessage",
		"SkillCoolTime", "UserInfo", "ValidateLocation", "ActionFailed")
	if st := sess.State(); st != StateInGame {
		t.Errorf("state after EnterWorld = %v, want StateInGame", st)
	}

	e.advance(enterDelay - 1)
	if late := sess.take(t); len(late) != 0 {
		t.Fatalf("before 300 ms: %q", testclient.Names(late))
	}
	e.advance(1)
	checkNames(t, "300 ms after EnterWorld", sess.take(t), "AbnormalStatusUpdate", "SkillList")

	// Contents.
	ui := ps[0].(testclient.UserInfo)
	if ui.ObjectID != c.ObjectID || ui.Name != "Hero" || ui.Position() != (testclient.Position{X: c.X, Y: c.Y, Z: c.Z}) {
		t.Errorf("UserInfo for %d %q at %v", ui.ObjectID, ui.Name, ui.Position())
	}
	// The numbers of a level 1 Human Fighter with its starting gear, as the
	// stat engine's hand-checked tests and Mobius have them. The stored HP
	// and MP are the template's (80, 30), within the maximum.
	want := userInfoNumbers{
		MaxHP: 126, MaxMP: 38, MaxCP: 50, HP: 80, MP: 30,
		MaxLoad: 109020, InventoryLimit: 80,
		PAtk: 6, PAtkSpd: 417, PDef: 75, Evasion: 34, Accuracy: 34, Critical: 88,
		MAtk: 2, CastSpd: 213, MDef: 47, RunSpd: 115, WalkSpd: 80, SwimRunSpd: 50, SwimWalkSpd: 50,
		WeaponFlag: 40, Running: true, NameColor: defaultNameColor, TitleColor: defaultTitleColor,
	}
	if got := numbersOf(ui); got != want {
		t.Errorf("UserInfo numbers:\n got %+v\nwant %+v", got, want)
	}
	if ui.PAtkSpd2 != ui.PAtkSpd {
		t.Errorf("UserInfo repeats attack speed %d as %d", ui.PAtkSpd, ui.PAtkSpd2)
	}
	if ui.PaperdollItemIDs[testclient.SlotRHand] != itemSquireSword ||
		ui.PaperdollItemIDs[testclient.SlotRHand2] != itemSquireSword ||
		ui.PaperdollItemIDs[testclient.SlotChest] != itemSquireShirt ||
		ui.PaperdollItemIDs[testclient.SlotLegs] != itemSquirePants {
		t.Errorf("UserInfo paperdoll item IDs %v", ui.PaperdollItemIDs)
	}
	if ui.CurLoad <= 0 {
		t.Errorf("UserInfo current load %d, want the weight of the starting items", ui.CurLoad)
	}

	macros := ps[1].(testclient.SendMacroList)
	if macros.Revision != 2 || macros.Count != 0 || macros.Macro != nil {
		t.Errorf("SendMacroList revision %d count %d, want an empty list with revision 2", macros.Revision, macros.Count)
	}

	items := ps[2].(testclient.ItemList)
	if items.ShowWindow {
		t.Error("EnterWorld's ItemList opens the inventory window")
	}
	if len(items.Items) != len(c.Items) {
		t.Fatalf("ItemList has %d items, want %d", len(items.Items), len(c.Items))
	}
	for i, it := range items.Items {
		stored := c.Items[i]
		tmpl, _ := e.data.ItemTemplate(int(stored.ItemID))
		switch {
		case it.ObjectID != stored.ObjectID || it.ItemID != int32(tmpl.DisplayID):
			t.Errorf("item %d is %d/%d, want %d/%d", i, it.ObjectID, it.ItemID, stored.ObjectID, tmpl.DisplayID)
		case it.Equipped != stored.Slot.Equipped():
			t.Errorf("item %d (%d) equipped %v, want %v", i, stored.ItemID, it.Equipped, stored.Slot.Equipped())
		case int(it.Type1) != tmpl.Type1 || int(it.Type2) != tmpl.Type2 || int(it.BodyPart) != tmpl.BodyPartMask:
			t.Errorf("item %d (%d) types %d/%d body part %#x, want %d/%d %#x", i, stored.ItemID, it.Type1, it.Type2, it.BodyPart, tmpl.Type1, tmpl.Type2, tmpl.BodyPartMask)
		case it.Mana != -1:
			t.Errorf("item %d (%d) mana %d, want -1", i, stored.ItemID, it.Mana)
		}
	}

	limits := ps[7].(testclient.ExStorageMaxCount)
	if limits != (testclient.ExStorageMaxCount{Inventory: 80, Warehouse: 100, ClanWarehouse: 200, PrivateSell: 3, PrivateBuy: 4, DwarfRecipe: 50, CommonRecipe: 50}) {
		t.Errorf("ExStorageMaxCount = %+v", limits)
	}
	if msg := ps[9].(testclient.SystemMessage); msg.ID != int32(serverpackets.SysMsgWelcome) || len(msg.Params) != 0 {
		t.Errorf("SystemMessage %d %v, want the welcome message", msg.ID, msg.Params)
	}
	if loc := ps[12].(testclient.ValidateLocation); loc.ObjectID != c.ObjectID || loc.Position() != (testclient.Position{X: c.X, Y: c.Y, Z: c.Z}) {
		t.Errorf("ValidateLocation %+v", loc)
	}
}

// userInfoNumbers are the numbers of UserInfo that come from the stat engine
// and the player's state.
type userInfoNumbers struct {
	MaxHP, MaxMP, MaxCP, HP, MP                          int32
	MaxLoad                                              int32
	InventoryLimit                                       uint16
	PAtk, PAtkSpd, PDef, Evasion, Accuracy, Critical     int32
	MAtk, CastSpd, MDef                                  int32
	RunSpd, WalkSpd, SwimRunSpd, SwimWalkSpd, WeaponFlag int32
	Running                                              bool
	NameColor, TitleColor                                int32
}

func numbersOf(u testclient.UserInfo) userInfoNumbers {
	return userInfoNumbers{
		MaxHP: u.MaxHP, MaxMP: u.MaxMP, MaxCP: u.MaxCP, HP: u.HP, MP: u.MP,
		MaxLoad: u.MaxLoad, InventoryLimit: u.InventoryLimit,
		PAtk: u.PAtk, PAtkSpd: u.PAtkSpd, PDef: u.PDef, Evasion: u.Evasion, Accuracy: u.Accuracy, Critical: u.Critical,
		MAtk: u.MAtk, CastSpd: u.CastSpd, MDef: u.MDef,
		RunSpd: u.RunSpd, WalkSpd: u.WalkSpd, SwimRunSpd: u.SwimRunSpd, SwimWalkSpd: u.SwimWalkSpd, WeaponFlag: u.WeaponFlag,
		Running: u.Running, NameColor: u.NameColor, TitleColor: u.TitleColor,
	}
}

func TestEnterWorldWithoutSelection(t *testing.T) {
	e := newEnv(t, nil)
	sess := newSession("nobody")
	e.srv.EnterWorld(sess)
	final, _, closed := sess.closedWith(t)
	if _, ok := final.(testclient.LeaveWorld); !closed || !ok {
		t.Errorf("EnterWorld without a character: closed %v with %v, want LeaveWorld", closed, final)
	}
}

func TestEnterWorldRunsEnterHooks(t *testing.T) {
	var entered []string
	e := newEnv(t, nil, func(s *Server) {
		s.OnPlayerEnter(func(p *Player) {
			if !p.InWorld() {
				t.Errorf("OnPlayerEnter: %s is not in the world", p.Name())
			}
			entered = append(entered, p.Name())
		})
	})
	e.createCharacter("hero", "Hero", classHumanFighter, storage.AccessLevelPlayer)
	e.enter(newSession("hero"), 0)
	var got []string
	e.onLoop(func() { got = append(got, entered...) })
	if len(got) != 1 || got[0] != "Hero" {
		t.Errorf("OnPlayerEnter ran for %q, want [Hero]", got)
	}
}
