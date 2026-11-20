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
