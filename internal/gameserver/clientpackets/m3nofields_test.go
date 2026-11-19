package clientpackets

import (
	"bytes"
	"testing"
)

// The packets without fields: Appearing, RequestRestart, Logout,
// RequestQuestList, RequestSkillCoolTime and RequestManorList. Their Mobius
// readImpl reads nothing, so any body parses, including garbage.

var m3NoFieldBodies = [][]byte{
	nil,
	{},
	{0x00},
	{0xff, 0xff, 0xff},
	bytes.Repeat([]byte{0x5a}, 64),
}

func TestParseNoFieldPackets(t *testing.T) {
	for _, body := range m3NoFieldBodies {
		if got, err := ParseAppearing(body); err != nil || got != (Appearing{}) {
			t.Errorf("ParseAppearing(% x) = %+v, %v", body, got, err)
		}
		if got, err := ParseRequestRestart(body); err != nil || got != (RequestRestart{}) {
			t.Errorf("ParseRequestRestart(% x) = %+v, %v", body, got, err)
		}
		if got, err := ParseLogout(body); err != nil || got != (Logout{}) {
			t.Errorf("ParseLogout(% x) = %+v, %v", body, got, err)
		}
		if got, err := ParseRequestQuestList(body); err != nil || got != (RequestQuestList{}) {
			t.Errorf("ParseRequestQuestList(% x) = %+v, %v", body, got, err)
		}
		if got, err := ParseRequestSkillCoolTime(body); err != nil || got != (RequestSkillCoolTime{}) {
			t.Errorf("ParseRequestSkillCoolTime(% x) = %+v, %v", body, got, err)
		}
		if got, err := ParseRequestManorList(body); err != nil || got != (RequestManorList{}) {
			t.Errorf("ParseRequestManorList(% x) = %+v, %v", body, got, err)
		}
	}
}

// The opcodes are those of Mobius ClientPackets and ExClientPackets.
func TestM3ClientOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"MoveToLocation", OpMoveToLocation, 0x01},
		{"EnterWorld", OpEnterWorld, 0x03},
		{"Logout", OpLogout, 0x09},
		{"CharacterDelete", OpCharacterDelete, 0x0c},
		{"CharacterSelect", OpCharacterSelect, 0x0d},
		{"Appearing", OpAppearing, 0x30},
		{"CannotMoveAnymore", OpCannotMoveAnymore, 0x36},
		{"Say2", OpSay2, 0x38},
		{"RequestActionUse", OpRequestActionUse, 0x45},
		{"RequestRestart", OpRequestRestart, 0x46},
		{"ValidatePosition", OpValidatePosition, 0x48},
		{"StartRotating", OpStartRotating, 0x4a},
		{"FinishRotating", OpFinishRotating, 0x4b},
		{"SendBypassBuildCmd", OpSendBypassBuildCmd, 0x5b},
		{"CharacterRestore", OpCharacterRestore, 0x62},
		{"RequestQuestList", OpRequestQuestList, 0x63},
		{"RequestSkillCoolTime", OpRequestSkillCoolTime, 0x9d},
		{"BypassUserCmd", OpBypassUserCmd, 0xaa},
		{"Ex", OpEx, 0xd0},
		{"Ex RequestManorList", ExOpRequestManorList, 0x08},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s opcode = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

func FuzzParseAppearing(f *testing.F) {
	m3Fuzz(f, ParseAppearing, 0, m3NoFieldBodies...)
}

func FuzzParseRequestRestart(f *testing.F) {
	m3Fuzz(f, ParseRequestRestart, 0, m3NoFieldBodies...)
}

func FuzzParseLogout(f *testing.F) {
	m3Fuzz(f, ParseLogout, 0, m3NoFieldBodies...)
}

func FuzzParseRequestQuestList(f *testing.F) {
	m3Fuzz(f, ParseRequestQuestList, 0, m3NoFieldBodies...)
}

func FuzzParseRequestSkillCoolTime(f *testing.F) {
	m3Fuzz(f, ParseRequestSkillCoolTime, 0, m3NoFieldBodies...)
}

func FuzzParseRequestManorList(f *testing.F) {
	m3Fuzz(f, ParseRequestManorList, 0, m3NoFieldBodies...)
}
