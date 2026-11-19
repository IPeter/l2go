package testclient

import (
	"bytes"
	"testing"
)

// The client packets of the world as Mobius reads them, transcribed from
// their readImpl.

// mobiusCharacterSelect: d slot, h, d, d, d (the last four ignored).
func mobiusCharacterSelect(slot int32) []byte {
	return msg(0x0d).i32(slot).u16(0).i32(0).i32(0).i32(0).b
}

// mobiusEnterWorld: 32 bytes, d, d, d, d, 32 bytes, d, then five trace
// route hops of four unsigned bytes.
func mobiusEnterWorld() []byte {
	w := msg(0x03).raw(make([]byte, 32)...).i32(0).i32(0).i32(0).i32(0).raw(make([]byte, 32)...).i32(0)
	for range 5 {
		w.raw(127, 0, 0, 1)
	}
	return w.b
}

// mobiusClientMove is the client MoveToLocation: target, origin, mode.
func mobiusClientMove(target, origin Position, mode int32) []byte {
	return msg(0x01).i32(target.X).i32(target.Y).i32(target.Z).i32(origin.X).i32(origin.Y).i32(origin.Z).i32(mode).b
}

// mobiusSay2: S text, d type, S target for whispers.
func mobiusSay2(text string, typ ChatType, target string) []byte {
	w := msg(0x38).str(text).i32(int32(typ))
	if typ == ChatWhisper {
		w.str(target)
	}
	return w.b
}

// mobiusAction is Action (0x04) and AttackRequest (0x0a): d object, d x,
// d y, d z, c shift.
func mobiusAction(op byte, objectID int32, origin Position, shift bool) []byte {
	return msg(op).i32(objectID).i32(origin.X).i32(origin.Y).i32(origin.Z).u8(b2u8(shift)).b
}

func TestEncodeWorldRequests(t *testing.T) {
	target := Position{-71000, 258000, -3100}
	origin := Position{-71338, 258271, -3104}
	tests := []struct {
		name string
		got  []byte
		want []byte
	}{
		{"CharacterSelect", encodeCharacterSelect(2), []byte{
			0x0d,
			2, 0, 0, 0, // slot
			0, 0, // h
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // d d d
		}},
		{"CharacterDelete", encodeCharacterDelete(1), []byte{0x0c, 1, 0, 0, 0}},
		{"CharacterRestore", encodeCharacterRestore(1), []byte{0x62, 1, 0, 0, 0}},
		{"EnterWorld", encodeEnterWorld(), mobiusEnterWorld()},
		{"MoveToLocation mouse", encodeMoveToLocation(MoveRequest{Target: target, Origin: origin}), mobiusClientMove(target, origin, 1)},
		{"MoveToLocation keyboard", encodeMoveToLocation(MoveRequest{Target: target, Origin: origin, Keyboard: true}), mobiusClientMove(target, origin, 0)},
		{"ValidatePosition", encodeValidatePosition(origin, 32768, 0), msg(0x48).i32(origin.X).i32(origin.Y).i32(origin.Z).i32(32768).i32(0).b},
		{"CannotMoveAnymore", encodeCannotMoveAnymore(origin, 100), msg(0x36).i32(origin.X).i32(origin.Y).i32(origin.Z).i32(100).b},
		{"Appearing", encodeOpcode(OpAppearing), []byte{0x30}},
		{"RequestActionUse sit", encodeRequestActionUse(ActionSitStand, false, false), []byte{0x45, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"RequestActionUse run ctrl shift", encodeRequestActionUse(ActionRunWalk, true, true), []byte{0x45, 1, 0, 0, 0, 1, 0, 0, 0, 1}},
		{"ChangeMoveType2", encodeChangeMoveType2(true), []byte{0x1c, 1, 0, 0, 0}},
		{"ChangeWaitType2", encodeChangeWaitType2(false), []byte{0x1d, 0, 0, 0, 0}},
		{"StartRotating", encodeStartRotating(16000, 1), msg(0x4a).i32(16000).i32(1).b},
		{"FinishRotating", encodeFinishRotating(16000), msg(0x4b).i32(16000).i32(0).b},
		{"Say2 general", encodeSay2("hi", ChatGeneral, "ignored"), []byte{0x38, 'h', 0, 'i', 0, 0, 0, 0, 0, 0, 0}},
		{"Say2 whisper", encodeSay2("hi", ChatWhisper, "Bob"), []byte{
			0x38,
			'h', 0, 'i', 0, 0, 0,
			2, 0, 0, 0,
			'B', 0, 'o', 0, 'b', 0, 0, 0,
		}},
		{"Say2 shout", encodeSay2("x", ChatShout, ""), mobiusSay2("x", ChatShout, "")},
		{"SendBypassBuildCmd", encodeString(OpSendBypassBuildCmd, "loc"), []byte{0x5b, 'l', 0, 'o', 0, 'c', 0, 0, 0}},
		{"BypassUserCmd", encodeInt(OpBypassUserCmd, UserCmdLoc), []byte{0xaa, 0, 0, 0, 0}},
		{"RequestRestart", encodeOpcode(OpRequestRestart), []byte{0x46}},
		{"Logout", encodeOpcode(OpLogout), []byte{0x09}},
		{"Action", encodeAction(OpAction, 0x60000001, origin, false), mobiusAction(0x04, 0x60000001, origin, false)},
		{"Action shift", encodeAction(OpAction, 0x60000001, origin, true), mobiusAction(0x04, 0x60000001, origin, true)},
		{"AttackRequest", encodeAction(OpAttackRequest, 0x60000001, origin, false), mobiusAction(0x0a, 0x60000001, origin, false)},
		{"RequestTargetCanceld", encodeRequestTargetCanceld(0), []byte{0x37, 0, 0}},
		{"RequestBypassToServer", encodeString(OpRequestBypassToServer, "npc_1_Chat 1"), msg(0x21).str("npc_1_Chat 1").b},
		{"RequestLinkHtml", encodeString(OpRequestLinkHtml, "default/30006-1.htm"), msg(0x20).str("default/30006-1.htm").b},
		{"RequestSocialAction", encodeInt(OpRequestSocialAction, 2), []byte{0x1b, 2, 0, 0, 0}},
		{"RequestQuestList", encodeOpcode(OpRequestQuestList), []byte{0x63}},
		{"RequestSkillCoolTime", encodeOpcode(OpRequestSkillCoolTime), []byte{0x9d}},
		{"RequestSkillList", encodeOpcode(OpRequestSkillList), []byte{0x3f}},
		{"RequestShowMiniMap", encodeOpcode(OpRequestShowMiniMap), []byte{0xcd}},
		// Ex sub-opcodes are little-endian shorts.
		{"RequestManorList", encodeEx(ExOpRequestManorList), []byte{0xd0, 0x08, 0x00}},
	}
	for _, tt := range tests {
		if !bytes.Equal(tt.got, tt.want) {
			t.Errorf("%s:\n got % x\nwant % x", tt.name, tt.got, tt.want)
		}
	}

	// Mobius drops an EnterWorld shorter than what readImpl reads.
	ew := encodeEnterWorld()
	if len(ew) != 1+104 {
		t.Errorf("EnterWorld body is %d bytes, Mobius reads 104", len(ew)-1)
	}
	if !bytes.Equal(ew[1:85], make([]byte, 84)) || !bytes.Equal(ew[85:89], []byte{127, 0, 0, 1}) {
		t.Errorf("EnterWorld % x", ew)
	}
}

func TestClientPacketNames(t *testing.T) {
	tests := []struct {
		raw  []byte
		want string
	}{
		{encodeProtocolVersion(InterludeProtocol), "ProtocolVersion"},
		{encodeEnterWorld(), "EnterWorld"},
		{encodeMoveToLocation(MoveRequest{}), "MoveToLocation"},
		{encodeSay2("x", ChatGeneral, ""), "Say2"},
		{encodeCharacterSelect(0), "CharacterSelect"},
		{encodeEx(ExOpRequestManorList), "RequestManorList"},
		{encodeEx(0x42), "UnknownEx(0x0042)"},
		{[]byte{0xee}, "Unknown(0xee)"},
	}
	for _, tt := range tests {
		if got := clientPacketName(tt.raw); got != tt.want {
			t.Errorf("clientPacketName(% x) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}
