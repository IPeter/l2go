package main

import (
	"strings"
	"testing"
)

// mappingTestSource exercises the translator rules for Arrays.asList
// views, known regular expressions, ItemHolder checks, FuncSet stat
// overrides and static packets.
const mappingTestSource = `package quests.Q09999_MappingTest;

import java.util.Arrays;
import java.util.Collections;

public class Q09999_MappingTest extends Quest
{
	private static final ItemHolder MEAT = new ItemHolder(7209, 3);

	public Q09999_MappingTest()
	{
		super(9999);
		addStartNpc(30001);
	}

	@Override
	public String onEvent(String event, Npc npc, Player player)
	{
		final QuestState qs = getQuestState(player, false);
		final Integer[] arr = {1, 2, 3};
		Collections.shuffle(Arrays.asList(arr));
		qs.set("numbers", Arrays.asList(arr).toString().replaceAll("[^\\d ]", ""));
		final int[] prim = {1, 2};
		qs.set("count", String.valueOf(Arrays.asList(prim).size()));
		if (hasItem(player, MEAT))
		{
			takeItem(player, MEAT);
		}
		npc.addStatFunc(new FuncSet(Stat.MAGIC_SUCCESS_RES, 0x100, npc, 0, null));
		npc.getAI().setIntentionAttack(player);
		player.sendPacket(ExShowSlideshowKamael.STATIC_PACKET);
		if (SevenSigns.getInstance().getPlayerCabal(player.getObjectId()) == SevenSigns.CABAL_DAWN)
		{
			return null;
		}
		return String.valueOf(qs.get("selected").replaceAll("\\D", "").length());
	}
}
`

func TestMappingRules(t *testing.T) {
	lua, res := translateQuest("Q09999_MappingTest.java", mappingTestSource, "Q09999_MappingTest")
	if res.err != nil {
		t.Fatal(res.err)
	}
	for _, n := range res.todos {
		t.Errorf("unexpected todo at java:%d: %s", n.line, n.reason)
	}
	for _, want := range []string{
		// Arrays.asList of an object array is a view: the shuffle reaches arr.
		"J.shuffle(arr)",
		`(string.gsub(J.list_to_string(arr), "[^%d ]", ""))`,
		// Of a primitive array it is a list with one element, as in Java.
		"#({prim})",
		"q:get_quest_items_count(player, MEAT.id) >= MEAT.count",
		"q:take_items(player, MEAT)",
		"npc:set_stat(Stat.MAGIC_SUCCESS_RES, 0)",
		"npc:attack(player)",
		"player:show_slideshow_kamael()",
		"== seven_signs.CABAL_DAWN",
		`#(string.gsub(qs:get("selected"), "%D", ""))`,
	} {
		if !strings.Contains(lua, want) {
			t.Errorf("output lacks %q", want)
		}
	}
	if !strings.Contains(lua, `q:requires("seven_signs")`) {
		t.Errorf("output lacks the seven_signs requirement")
	}
	if t.Failed() {
		t.Log(lua)
	}
}

// typedHelpersSource uses the Java methods whose int and long, or literal
// and regex, forms map to different helpers.
const typedHelpersSource = `package quests.Q09998_TypedHelpers;

public class Q09998_TypedHelpers extends Quest
{
	public Q09998_TypedHelpers()
	{
		super(9998);
	}

	@Override
	public String onEvent(String event, Npc npc, Player player)
	{
		final double rate = 1.5;
		final int i = (int) (rate * 10);
		final long l = (long) (rate * 10);
		long total = 3;
		total *= rate;
		int count = 3;
		count *= rate;
		final int a = Integer.parseInt(event);
		final long b = Long.parseLong(event);
		final String[] words = event.split("\\s+");
		final String[] chars = event.split("\\s");
		final String[] dots = event.split("\\.");
		final String x = event.replace("a", "$");
		final String y = event.replaceAll("a", "\\$");
		return x + y + words.length + chars.length + dots.length + i + l + total + count + a + b;
	}
}
`

func TestTypedHelpers(t *testing.T) {
	lua, res := translateQuest("Q09998_TypedHelpers.java", typedHelpersSource, "Q09998_TypedHelpers")
	if res.err != nil {
		t.Fatal(res.err)
	}
	for _, n := range res.todos {
		t.Errorf("unexpected todo at java:%d: %s", n.line, n.reason)
	}
	for _, want := range []string{
		"local i = J.int(rate * 10)\n",
		`local l = J.int(rate * 10, "long")`,
		`total = J.int(total * rate, "long")`,
		"count = J.int(count * rate)\n",
		"J.parse_int(event)",
		"J.parse_long(event)",
		"J.split_ws(event)",
		"J.split_ws(event, true)",
		`J.split(event, ".")`,
		`J.replace(event, "a", "$")`,
		`J.replace_all(event, "a", "\\$")`,
	} {
		if !strings.Contains(lua, want) {
			t.Errorf("output lacks %q", want)
		}
	}
	if t.Failed() {
		t.Log(lua)
	}
}

// aiCallsSource uses the NPC AI, skill, item and player calls the Lua API
// has (npc:attack, skills.get, items.get, player:active_weapon, ...).
const aiCallsSource = `package quests.Q09997_AiCalls;

public class Q09997_AiCalls extends Quest
{
	private static final SkillHolder GOW = new SkillHolder(4284, 2);

	public Q09997_AiCalls()
	{
		super(9997);
		addStartNpc(30001);
		addAttackId(27001);
	}

	@Override
	public String onEvent(String event, Npc npc, Player player)
	{
		final Attackable mob = addSpawn(27001, npc, true, 0).asAttackable();
		mob.setRunning();
		mob.addDamageHate(player, 0, 999);
		mob.getAI().setIntentionAttack(player);
		npc.getAI().setIntentionMoveTo(new Location(1, 2, 3));
		npc.teleToLocation(4, 5, 6);
		npc.setTarget(player);
		npc.doCast(GOW.getSkill());
		npc.doCast(SkillData.getInstance().getSkill(5089, 1));
		npc.broadcastPacket(new MagicSkillUse(npc, player, 5073, 5, 1500, 0));
		npc.asMonster().dropItem(player, 6353, 1);
		npc.abortAttack();
		if ((player.getActiveWeaponItem() != null) && (player.getActiveWeaponItem().getId() == 3471) && player.isAffectedBySkill(4082))
		{
			player.setPkKills(player.getPkKills() - 1);
			player.updateUserInfo();
		}
		if (!player.isNoble())
		{
			player.setNoble(true);
		}
		if ((npc.getLevel() > 40) && (npc.getCurrentHp() < (npc.getMaxHp() / 3)))
		{
			npc.doDie(player);
		}
		if (World.getVisibleObjects().stream().anyMatch(object -> object.getId() == 25290))
		{
			return null;
		}
		player.getTarget().decayMe();
		final ItemTemplate item = ItemData.getInstance().getTemplate(57);
		if (item.isStackable())
		{
			return item.getName();
		}
		return String.valueOf(player.getAllActiveQuests().size());
	}

	@Override
	public void onAttack(Npc npc, Player attacker, int damage, boolean isSummon, Skill skill)
	{
		if ((skill != null) && (skill.getId() == 1177) && (skill.getLevel() == 1))
		{
			npc.setScriptValue(1);
		}
	}
}
`

func TestAICalls(t *testing.T) {
	lua, res := translateQuest("Q09997_AiCalls.java", aiCallsSource, "Q09997_AiCalls")
	if res.err != nil {
		t.Fatal(res.err)
	}
	for _, n := range res.todos {
		t.Errorf("unexpected todo at java:%d: %s", n.line, n.reason)
	}
	for _, want := range []string{
		"mob:set_running()",
		"mob:add_damage_hate(player, 0, 999)",
		"mob:attack(player)",
		"npc:move_to({x = 1, y = 2, z = 3})",
		"npc:teleport(4, 5, 6)",
		"npc:set_target(player)",
		// A holder's skill is the runtime Skill.
		"npc:cast(skills.get(GOW.id, GOW.level))",
		"npc:cast(skills.get(5089, 1))",
		"npc:magic_skill_use(player, 5073, 5, 1500, 0)",
		"npc:drop_item(player, 6353, 1)",
		"npc:abort_attack()",
		// player:active_weapon() is the item ID itself.
		"player:active_weapon() ~= nil and player:active_weapon() == 3471 and player:is_affected_by_skill(4082)",
		"player:set_pk_kills(player:pk_kills() - 1)",
		"player:update_user_info()",
		"if not player:is_noble() then",
		"player:set_noble(true)",
		// getMaxHp is an int: Java's integer division.
		"npc:level() > 40 and npc:hp() < J.idiv(npc:max_hp(), 3)",
		"npc:die(player)",
		"world.npc_exists(25290)",
		"player:target():delete()",
		"items.get(57)",
		"item.stackable",
		"item.name",
		"#player:active_quests()",
		// Skill id and level are methods.
		"skill:id() == 1177 and skill:level() == 1",
	} {
		if !strings.Contains(lua, want) {
			t.Errorf("output lacks %q", want)
		}
	}
	// Noblesse exists now: no requirement.
	if strings.Contains(lua, "q:requires(") {
		t.Errorf("output has a requirement")
	}
	if t.Failed() {
		t.Log(lua)
	}
}

func TestRegexReplaceUnknownPatternStaysTodo(t *testing.T) {
	src := strings.Replace(mappingTestSource, `replaceAll("\\D", "")`, `replaceAll("\\w+", "")`, 1)
	_, res := translateQuest("Q09999_MappingTest.java", src, "Q09999_MappingTest")
	if res.err != nil {
		t.Fatal(res.err)
	}
	found := false
	for _, n := range res.todos {
		if strings.Contains(n.reason, "regular expression") {
			found = true
		}
	}
	if !found {
		t.Errorf("replaceAll with an unknown regular expression should stay a todo")
	}
}
