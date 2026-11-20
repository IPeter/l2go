package game

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// scriptOutputOf returns a ScriptOutput bound to the test server.
func scriptOutputOf(e *testEnv) *ScriptOutput {
	o := NewScriptOutput()
	o.bind(e.srv)
	return o
}

// The scripts' NPC AI calls run the monster AI as Mobius quests do:
// setRunning, addDamageHate, setIntention(ATTACK), then MOVE_TO; the status
// reads the NPC; doDie kills it.
func TestScriptNpcAI(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	_, p := statusPlayer(t, e, "alice", "Alice")
	k := aiSpawn(e, aiKeltir, start, 200, 0)
	o := scriptOutputOf(e)
	ref := k.ScriptRef()

	var level int
	var hp, maxHP float64
	e.onLoop(func() {
		st := ref.(scripting.NpcStatus)
		level, hp, maxHP = st.Level(), st.CurrentHP(), st.MaxHP()
		o.NpcSetRunning(ref)
		o.NpcAddDamageHate(ref, p, 0, 999)
		o.NpcAttack(ref, p)
	})
	if level != 1 || hp <= 0 || hp != maxHP {
		t.Errorf("status = level %d, HP %v/%v; want level 1, full HP", level, hp, maxHP)
	}
	st := aiStateOf(e, k)
	if !st.running || st.intention != NpcAttack || st.target != Creature(p) {
		t.Errorf("after attack: %+v; want running, ATTACK on Alice", st)
	}
	if len(st.hate) != 1 || st.hate[0].Hate != 999 {
		t.Errorf("hate list %+v; want Alice with 999", st.hate)
	}

	var attackBy int
	e.onLoop(func() {
		k.AddAttackBy(p)
		o.NpcRemoveAttacker(ref, p)
		attackBy = len(k.AttackBy())
		o.NpcAbortAttack(ref)
		o.NpcMoveTo(ref, scripting.Location{X: start.X + 600, Y: start.Y, Z: start.Z})
	})
	if attackBy != 0 {
		t.Errorf("attack-by list has %d after remove_attacker", attackBy)
	}
	st = aiStateOf(e, k)
	if st.intention != NpcMoveTo || !st.moving || st.dest.X != int32(start.X+600) {
		t.Errorf("after move_to: %+v; want MOVE_TO, walking to x %d", st, start.X+600)
	}

	var exists, other, dead bool
	e.onLoop(func() {
		o.NpcDie(ref, nil)
		dead = k.Dead
		exists, other = o.NpcExists(aiKeltir), o.NpcExists(aiOrc)
	})
	if !dead || !exists || other {
		t.Errorf("die: dead %v, keltir exists %v, orc exists %v; want true, true (the corpse), false", dead, exists, other)
	}
}

// q:add_spawn with heading -1 spawns with a random heading; start_respawn
// brings a script spawn back where it was spawned, stop_respawn does not.
func TestScriptNpcRespawn(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	o := scriptOutputOf(e)
	loc := scripting.Location{X: start.X + 300, Y: start.Y, Z: start.Z, Heading: -1}

	var ref scripting.NpcRef
	var heading int32
	e.onLoop(func() {
		ref = o.AddSpawn(aiWolf, loc, false, 0, false)
		if n := npcOfRef(ref); n != nil {
			heading = n.Position().Heading
		}
	})
	n := npcOfRef(ref)
	if n == nil || heading < 0 || heading >= spawnRandomHeading {
		t.Fatalf("add_spawn: %v, heading %d; want an NPC with a heading in [0, %d)", ref, heading, spawnRandomHeading)
	}

	e.onLoop(func() {
		o.NpcSetRespawn(ref, true)
		e.srv.DespawnNpc(n)
	})
	e.advance(time.Second)
	var back bool
	var pos world.Position
	e.onLoop(func() { back, pos = n.InWorld(), n.Position() })
	if !back || pos.X != int32(loc.X) {
		t.Fatalf("after start_respawn and a despawn: in world %v at %v; want back at x %d", back, pos, loc.X)
	}

	e.onLoop(func() {
		o.NpcSetRespawn(ref, false)
		e.srv.DespawnNpc(n)
	})
	e.advance(time.Second)
	e.onLoop(func() { back = n.InWorld() })
	if back {
		t.Error("the NPC came back after stop_respawn")
	}
}

// skills.get finds the skill levels of the data; skills.apply_effects and
// npc:cast (at a target, or at the target set_target chose) put the buffs on
// the player.
func TestScriptSkills(t *testing.T) {
	e := newCastEnv(t)
	_, p := castEnter(t, e, "newbie", classHumanFighter)
	keltir := npcSpawnedOne(t, e, attackKeltir)
	o := scriptOutputOf(e)
	ref := keltir.ScriptRef()

	var exists, missing, before, applied, cast, targeted bool
	e.onLoop(func() {
		exists, missing = o.SkillExists(4322, 1), o.SkillExists(4322, 99)
		before = o.IsAffectedBySkill(p, 4322)
		o.ApplyEffects(scripting.Skill{ID: 4322, Level: 1}, ref, p)
		applied = o.IsAffectedBySkill(p, 4322)
		o.NpcCast(ref, scripting.Skill{ID: 4323, Level: 1}, p)
		cast = o.IsAffectedBySkill(p, 4323)
		o.NpcSetTarget(ref, p)
		o.NpcCast(ref, scripting.Skill{ID: 4324, Level: 1}, nil)
		targeted = o.IsAffectedBySkill(p, 4324)
	})
	if !exists || missing {
		t.Errorf("SkillExists(4322, 1) = %v, (4322, 99) = %v", exists, missing)
	}
	if before || !applied || !cast || !targeted {
		t.Errorf("affected before %v, after apply_effects %v, cast at the player %v, cast at the set target %v", before, applied, cast, targeted)
	}
}

// The player refs: the weapon, the target, PKs, noblesse and the quests;
// MagicSkillUse reaches the player; drop_item puts items on the ground;
// items.get reads the item data.
func TestScriptPlayerAndItems(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	sess, p := statusPlayer(t, e, "alice", "Alice")
	k := aiSpawn(e, aiKeltir, start, 100, 0)
	o := scriptOutputOf(e)

	var weapon, rhand, pks int
	var target any
	var noble bool
	var quests []int
	e.onLoop(func() {
		weapon, rhand = p.ActiveWeaponID(), o.EquippedItemID(p, 7)
		e.srv.SetTarget(p, k)
		target = p.TargetObject()
		p.SetPkKills(4)
		pks = p.PkKills()
		p.SetNoble(true)
		noble = p.IsNoble()
		quests = p.ActiveQuests()
	})
	if weapon == 0 || weapon != rhand || pks != 4 || !noble || len(quests) != 0 {
		t.Errorf("weapon %d (right hand %d), pk %d, noble %v, quests %v; want the starter weapon, 4, true, none", weapon, rhand, pks, noble, quests)
	}
	if r, ok := target.(scripting.NpcRef); !ok || r.ObjectID() != int32(k.ID()) {
		t.Errorf("target = %v; want the Keltir", target)
	}

	sess.take(t)
	e.onLoop(func() { o.MagicSkillUse(p, k.ScriptRef(), 4339, 1, 6*time.Second, 0) })
	var msu *testclient.MagicSkillUse
	for _, pk := range sess.take(t) {
		if m, ok := pk.(testclient.MagicSkillUse); ok {
			msu = &m
		}
	}
	if msu == nil || msu.CasterID != p.ObjectID() || msu.TargetID != int32(k.ID()) || msu.SkillID != 4339 {
		t.Errorf("MagicSkillUse = %+v; want Alice on the Keltir, skill 4339", msu)
	}

	var drops int
	var tmpl scripting.ItemTemplate
	var found, unknown bool
	e.onLoop(func() {
		o.NpcDropItem(k.ScriptRef(), p, adenaID, 10)
		for obj := range e.srv.world.Objects() {
			if g, ok := obj.(*GroundItem); ok && g.Item().ItemID() == adenaID {
				drops++
			}
		}
		tmpl, found = o.ItemTemplate(adenaID)
		_, unknown = o.ItemTemplate(-5)
	})
	if drops != 1 {
		t.Errorf("%d adena drops on the ground; want 1", drops)
	}
	if !found || tmpl.Name != "Adena" || !tmpl.Stackable || unknown {
		t.Errorf("ItemTemplate(57) = %+v, %v; unknown found %v", tmpl, found, unknown)
	}
}
