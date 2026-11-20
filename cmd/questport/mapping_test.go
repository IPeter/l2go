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
