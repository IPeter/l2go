package main

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

// Cut-down Mobius sources of the script constants.
const (
	sampleQuestSound = `package org.l2jmobius.gameserver.mechanics.script;
public enum QuestSound
{
	ITEMSOUND_QUEST_ACCEPT(new PlaySound("ItemSound.quest_accept")),
	// Quest 648 and treasure chests.
	ITEMSOUND_BROKEN_KEY(new PlaySound("ItemSound2.broken_key")),
	BS07_D(new PlaySound("BS07_D"));

	private final PlaySound _playSound;
	private QuestSound(PlaySound playSound)
	{
		_playSound = playSound;
	}
}
`
	sampleQuestState = `package org.l2jmobius.gameserver.mechanics.script;
public class State
{
	public static final byte CREATED = 0;
	public static final byte STARTED = 1;
	// public static final byte ABORTED = 3;
	public static final byte COMPLETED = 2;
}
`
	sampleQuestType = `package org.l2jmobius.gameserver.mechanics.script;
public enum QuestType
{
	REPEATABLE,
	ONE_TIME,
	DAILY
}
`
	sampleChatType = `package org.l2jmobius.gameserver.network.enums;
public enum ChatType
{
	GENERAL(0),
	SHOUT(1),
	MPCC_ROOM(21),
	NPC_GENERAL(0), // Epilogue adjustment
	NPC_SHOUT(1); // Epilogue adjustment

	private final int _clientId;
	private ChatType(int clientId)
	{
		_clientId = clientId;
	}
}
`
	sampleSex = `package org.l2jmobius.gameserver.entity.actor.enums.player;
public enum Sex
{
	MALE,
	FEMALE,
	ETC;
}
`
	sampleSystemMessageID = `package org.l2jmobius.gameserver.network;
public class SystemMessageId
{
	@ClientString(id = 0, message = "You have been disconnected from the server.")
	public static SystemMessageId YOU_HAVE_BEEN_DISCONNECTED_FROM_THE_SERVER;

	@ClientString(id = 54, message = "You have earned $s1.")
	public static SystemMessageId YOU_HAVE_EARNED_S1;

	// @ClientString(id = 55, message = "Commented out.")
	@ClientString(id = 1210, message = "Say \"hello\" // not a comment; it's fine.")
	public static SystemMessageId SAY_HELLO;

	private static void buildFastLookupTable()
	{
	}
}
`
)

func constantsSample(t *testing.T) fstest.MapFS {
	t.Helper()
	m := sampleFS(t)
	m[questSoundFile] = file(sampleQuestSound)
	m[questStateFile] = file(sampleQuestState)
	m[questTypeFile] = file(sampleQuestType)
	m[chatTypeFile] = file(sampleChatType)
	m[sexFile] = file(sampleSex)
	m[systemMessageIDFile] = file(sampleSystemMessageID)
	return m
}

func TestConstants(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(constantsSample(t), false, []string{"constants"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	c, err := data.LoadConstantsFS(fstest.MapFS{data.ConstantsFile: {Data: files[data.ConstantsFile]}})
	if err != nil {
		t.Fatal(err)
	}
	ints := []struct {
		table, name string
		want        int
	}{
		{data.ConstState, "CREATED", 0},
		{data.ConstState, "STARTED", 1},
		{data.ConstState, "COMPLETED", 2},
		{data.ConstQuestType, "REPEATABLE", 0},
		{data.ConstQuestType, "DAILY", 2},
		{data.ConstChatType, "MPCC_ROOM", 21},
		{data.ConstChatType, "NPC_GENERAL", 0},
		{data.ConstChatType, "NPC_SHOUT", 1},
		{data.ConstSex, "FEMALE", 1},
		{data.ConstPaperdoll, "UNDER", 0},
		{data.ConstPaperdoll, "RHAND", 7},
		{data.ConstSystemMessageID, "YOU_HAVE_EARNED_S1", 54},
		{data.ConstSystemMessageID, "SAY_HELLO", 1210},
		{data.ConstClassID, "FIGHTER", 0},
		{data.ConstClassID, "WARRIOR", 1},
		{data.ConstClassID, "DOMINATOR", 115},
		{data.ConstRace, "HUMAN", 0},
		{data.ConstRace, "DARK_ELF", 2},
		{data.ConstCategory, "FIGHTER_GROUP", int(data.CategoryFighterGroup)},
	}
	for _, tt := range ints {
		if got, ok := c.Int(tt.table, tt.name); !ok || got != tt.want {
			t.Errorf("Int(%s, %s) = %d, %v; want %d", tt.table, tt.name, got, ok, tt.want)
		}
	}
	strs := []struct{ name, want string }{
		{"ITEMSOUND_QUEST_ACCEPT", "ItemSound.quest_accept"},
		{"ITEMSOUND_BROKEN_KEY", "ItemSound2.broken_key"},
		{"BS07_D", "BS07_D"},
	}
	for _, tt := range strs {
		if got, ok := c.String(data.ConstQuestSound, tt.name); !ok || got != tt.want {
			t.Errorf("String(QuestSound, %s) = %q, %v; want %q", tt.name, got, ok, tt.want)
		}
	}
	missing := []struct{ table, name string }{
		{data.ConstState, "ABORTED"},                     // commented out
		{data.ConstPaperdoll, "TOTALSLOTS"},              // not a slot
		{data.ConstPaperdoll, "PAPERDOLL_RHAND"},         // the prefix is dropped
		{data.ConstSystemMessageID, "COMMENTED_OUT"},     // no such constant
		{data.ConstQuestSound, "ITEMSOUND_QUEST_ACCEPT"}, // a string, not an int
	}
	for _, tt := range missing {
		if v, ok := c.Int(tt.table, tt.name); ok {
			t.Errorf("Int(%s, %s) = %d, want none", tt.table, tt.name, v)
		}
	}
	if n := len(c.Tables); n != len(data.ConstantTableNames()) {
		t.Errorf("%d tables, want %d", n, len(data.ConstantTableNames()))
	}
	if tbl, ok := c.Table(data.ConstSystemMessageID); !ok || tbl.Len() != 3 {
		t.Errorf("SystemMessageId table = %+v, %v; want 3 constants", tbl, ok)
	}
	for _, s := range []string{"constants: Category 160, ChatType 5, ClassId 89, Paperdoll 6", "SystemMessageId 3"} {
		if !strings.Contains(log.String(), s) {
			t.Errorf("log %q does not mention %q", log.String(), s)
		}
	}
}

func TestConstantsErrors(t *testing.T) {
	edit := func(name, content, old, new string) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, m fstest.MapFS) {
			t.Helper()
			for f, src := range map[string]string{
				questSoundFile: sampleQuestSound, questStateFile: sampleQuestState, questTypeFile: sampleQuestType,
				chatTypeFile: sampleChatType, sexFile: sampleSex, systemMessageIDFile: sampleSystemMessageID,
			} {
				m[f] = file(src)
			}
			m[name] = file(strings.Replace(content, old, new, 1))
		}
	}
	expectErrors(t, []string{"constants"}, []sampleCase{
		{"no files", func(*testing.T, fstest.MapFS) {}, "read " + chatTypeFile},
		{"sound not a PlaySound", edit(questSoundFile, sampleQuestSound, `new PlaySound("BS07_D")`, `"BS07_D"`),
			`QuestSound.BS07_D: argument "\"BS07_D\"" is not new PlaySound("name")`},
		{"chat type not an int", edit(chatTypeFile, sampleChatType, "MPCC_ROOM(21)", "MPCC_ROOM(X)"), `ChatType.MPCC_ROOM: not an int literal: "X"`},
		{"state missing", edit(questStateFile, sampleQuestState, "STARTED = 1", "BEGUN = 1"), "State.STARTED not found"},
		{"message not read", edit(systemMessageIDFile, sampleSystemMessageID, "public static SystemMessageId SAY_HELLO;", "public static SystemMessageId SAY_HELLO = null;"),
			"read 2 of 3 @ClientString declarations"},
		{"message twice", edit(systemMessageIDFile, sampleSystemMessageID, "YOU_HAVE_EARNED_S1;", "SAY_HELLO;"),
			"SystemMessageId.SAY_HELLO is declared twice"},
		{"lowercase name", edit(sexFile, sampleSex, "ETC;", "etc;"), `constant "etc" is not an upper case Lua name`},
	})
}

// The committed constants are what datagen makes from Mobius.
func TestCommittedConstantsMatchMobius(t *testing.T) { committed(t, repoData, false, "constants") }
