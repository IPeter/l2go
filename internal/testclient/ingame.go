package testclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrRestartRefused is returned by Restart when the server answers
// RestartResponse with result 0 (Mobius refuses during combat, trading or
// a class change).
var ErrRestartRefused = errors.New("testclient: server refused the restart")

// SelectCharacter sends CharacterSelect for the character in slot (its
// index in the last character list) and returns CharSelected. The world
// model starts over with that character. Mobius sends nothing back for a
// slot that does not exist, so such a call ends with its context.
func (c *GameConn) SelectCharacter(ctx context.Context, slot int32) (CharSelected, error) {
	const want = "CharSelected"
	p, body, err := c.request(ctx, encodeCharacterSelect(slot), want)
	if err != nil {
		return CharSelected{}, err
	}
	if v, ok := p.(CharSelected); ok {
		return v, nil
	}
	return CharSelected{}, unexpectedGame(body, want)
}

// EnterWorldResult holds what the server sent from EnterWorld until the
// world was ready.
type EnterWorldResult struct {
	// Packets holds every packet, in the order the server sent them, so
	// tests can compare it with Mobius's order.
	Packets []Packet
	// UserInfo is the last UserInfo (Mobius sends two), Location the
	// character's last ValidateLocation, and the others the last packet of
	// their type. Fields of packets that did not come are zero.
	UserInfo  UserInfo
	Items     ItemList
	Shortcuts ShortcutInit
	Quests    QuestList
	Skills    SkillList
	EtcStatus EtcStatusUpdate
	Location  ValidateLocation
}

// Names returns the packet names, in order.
func (r EnterWorldResult) Names() []string { return Names(r.Packets) }

func (r *EnterWorldResult) add(p Packet) {
	r.Packets = append(r.Packets, p)
	switch v := p.(type) {
	case UserInfo:
		r.UserInfo = v
	case ItemList:
		r.Items = v
	case ShortcutInit:
		r.Shortcuts = v
	case QuestList:
		r.Quests = v
	case SkillList:
		r.Skills = v
	case EtcStatusUpdate:
		r.EtcStatus = v
	case ValidateLocation:
		if r.UserInfo.ObjectID == 0 || v.ObjectID == r.UserInfo.ObjectID {
			r.Location = v
		}
	}
}

// WorldReady reports whether seen ends Mobius's enter-world sequence:
// EnterWorld.runImpl ends with ValidateLocation for the character and
// ActionFailed, and the SkillList comes 300 ms later
// (Player.sendSkillList). The character is the one of the first UserInfo
// in seen; without one, any ValidateLocation counts.
func WorldReady(seen []Packet) bool {
	var me int32
	if u, ok := Find[UserInfo](seen); ok {
		me = u.ObjectID
	}
	located, done, skills := false, false, false
	for _, p := range seen {
		switch v := p.(type) {
		case ValidateLocation:
			if me == 0 || v.ObjectID == me {
				located = true
			}
		case ActionFailed:
			if located {
				done = true
			}
		case SkillList:
			skills = true
		}
	}
	return done && skills
}

// EnterWorld enters the world with the selected character: it starts the
// background reader, sends EnterWorld with the full body Mobius reads and
// collects the server's packets until WorldReady. Packets after that stay
// queued for Next and WaitFor.
func (c *GameConn) EnterWorld(ctx context.Context) (EnterWorldResult, error) {
	return c.EnterWorldUntil(ctx, WorldReady)
}

// EnterWorldUntil is EnterWorld with another end condition: it collects
// packets until ready(seen) is true. LeaveWorld or ServerClose ends it with
// an error. On an error the result holds the packets collected so far.
func (c *GameConn) EnterWorldUntil(ctx context.Context, ready func(seen []Packet) bool) (EnterWorldResult, error) {
	c.StartReader()
	var res EnterWorldResult
	if err := c.send(ctx, encodeEnterWorld()); err != nil {
		return res, err
	}
	for !ready(res.Packets) {
		r, err := c.recv(ctx)
		if err != nil {
			we := &WaitError{Want: "the end of the enter-world sequence", Err: err}
			for _, p := range res.Packets {
				we.see(p.PacketName())
			}
			return res, we
		}
		if r.err != nil {
			return res, r.err
		}
		res.add(r.p)
		switch r.p.(type) {
		case LeaveWorld, ServerClose:
			return res, fmt.Errorf("testclient: server sent %s while entering the world (packets: %s)",
				r.p.PacketName(), strings.Join(res.Names(), ", "))
		}
	}
	return res, nil
}

// SendMove sends the client MoveToLocation without waiting for an answer.
func (c *GameConn) SendMove(ctx context.Context, req MoveRequest) error {
	return c.send(ctx, encodeMoveToLocation(req))
}

// MoveTo walks the character to (x, y, z) like a mouse click, from the
// position the world model knows, and returns the server's MoveToLocation
// for the character. Mobius refuses a move with ActionFailed alone; since
// ActionFailed names no request, MoveTo does not take it as the answer and
// a refused move ends with the context. Negative tests use SendMove and
// Await[ActionFailed] instead.
func (c *GameConn) MoveTo(ctx context.Context, x, y, z int32) (MoveToLocation, error) {
	me := c.world.Me()
	if err := c.SendMove(ctx, MoveRequest{Target: Position{x, y, z}, Origin: me.Pos}); err != nil {
		return MoveToLocation{}, err
	}
	return Await(ctx, c, func(m MoveToLocation) bool { return m.ObjectID == me.ObjectID })
}

// ValidatePosition reports where the client thinks the character is
// (ValidatePosition, vehicle 0). The server answers with ValidateLocation
// only when it disagrees, so nothing is awaited.
func (c *GameConn) ValidatePosition(ctx context.Context, pos Position, heading int32) error {
	return c.send(ctx, encodeValidatePosition(pos, heading, 0))
}

// CannotMoveAnymore tells the server the character got stuck at pos.
func (c *GameConn) CannotMoveAnymore(ctx context.Context, pos Position, heading int32) error {
	return c.send(ctx, encodeCannotMoveAnymore(pos, heading))
}

// Appearing tells the server the client has loaded the area after a
// teleport, and returns the UserInfo the server sends back.
func (c *GameConn) Appearing(ctx context.Context) (UserInfo, error) {
	if err := c.send(ctx, encodeOpcode(OpAppearing)); err != nil {
		return UserInfo{}, err
	}
	me := c.world.Me().ObjectID
	return Await(ctx, c, func(u UserInfo) bool { return me == 0 || u.ObjectID == me })
}

// TeleportResult is a finished teleport of the character.
type TeleportResult struct {
	Teleport TeleportToLocation
	UserInfo UserInfo
}

// FinishTeleport waits for the character's TeleportToLocation (after a GM
// command or a gatekeeper bypass), answers with Appearing as the client
// does once it has loaded the area, and returns both with the UserInfo
// that follows.
func (c *GameConn) FinishTeleport(ctx context.Context) (TeleportResult, error) {
	me := c.world.Me().ObjectID
	t, err := Await(ctx, c, func(t TeleportToLocation) bool { return t.ObjectID == me })
	if err != nil {
		return TeleportResult{}, err
	}
	u, err := c.Appearing(ctx)
	if err != nil {
		return TeleportResult{}, err
	}
	return TeleportResult{Teleport: t, UserInfo: u}, nil
}

// RequestActionUse sends RequestActionUse: ActionSitStand toggles sitting
// (ChangeWaitType comes back), ActionRunWalk toggles running
// (ChangeMoveType).
func (c *GameConn) RequestActionUse(ctx context.Context, actionID int32, ctrl, shift bool) error {
	return c.send(ctx, encodeRequestActionUse(actionID, ctrl, shift))
}

// ChangeMoveType2 asks to run (true) or walk.
func (c *GameConn) ChangeMoveType2(ctx context.Context, run bool) error {
	return c.send(ctx, encodeChangeMoveType2(run))
}

// ChangeWaitType2 asks to stand (true) or sit.
func (c *GameConn) ChangeWaitType2(ctx context.Context, stand bool) error {
	return c.send(ctx, encodeChangeWaitType2(stand))
}

// StartRotating starts turning the character (keyboard movement); the
// server broadcasts StartRotation.
func (c *GameConn) StartRotating(ctx context.Context, degree, side int32) error {
	return c.send(ctx, encodeStartRotating(degree, side))
}

// FinishRotating stops turning at degree; the server broadcasts
// StopRotation.
func (c *GameConn) FinishRotating(ctx context.Context, degree int32) error {
	return c.send(ctx, encodeFinishRotating(degree))
}

// Say sends text to a chat channel (Say2). It does not check the type or
// the text, so it also serves negative tests. The server's CreatureSay
// comes back for the channels that echo to the sender.
func (c *GameConn) Say(ctx context.Context, chatType ChatType, text string) error {
	return c.send(ctx, encodeSay2(text, chatType, ""))
}

// Whisper sends text to the player called target (Say2, ChatWhisper).
func (c *GameConn) Whisper(ctx context.Context, target, text string) error {
	return c.send(ctx, encodeSay2(text, ChatWhisper, target))
}

// GMCommand sends a GM command as the client sends "//cmd"
// (SendBypassBuildCmd): cmd is the command without the slashes, such as
// "teleport 0 0 0". A leading "//" is removed.
func (c *GameConn) GMCommand(ctx context.Context, cmd string) error {
	return c.send(ctx, encodeString(OpSendBypassBuildCmd, strings.TrimPrefix(cmd, "//")))
}

// UserCommand sends a user command such as /loc (BypassUserCmd with
// UserCmdLoc).
func (c *GameConn) UserCommand(ctx context.Context, id int32) error {
	return c.send(ctx, encodeInt(OpBypassUserCmd, id))
}

// RequestQuestList asks for the quest list (QuestList comes back).
func (c *GameConn) RequestQuestList(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpRequestQuestList))
}

// RequestSkillCoolTime asks for the skill cooldowns (SkillCoolTime comes
// back).
func (c *GameConn) RequestSkillCoolTime(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpRequestSkillCoolTime))
}

// RequestSkillList asks for the skill list (SkillList comes back).
func (c *GameConn) RequestSkillList(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpRequestSkillList))
}

// RequestManorList sends the Ex packet RequestManorList (D0 08 00), which
// the client sends after entering the world.
func (c *GameConn) RequestManorList(ctx context.Context) error {
	return c.send(ctx, encodeEx(ExOpRequestManorList))
}

// RequestShowMiniMap opens the map (ShowMiniMap comes back).
func (c *GameConn) RequestShowMiniMap(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpRequestShowMiniMap))
}

// Restart leaves the world for the character screen: RequestRestart, then
// RestartResponse and the character list. Packets before RestartResponse
// are skipped. A RestartResponse with result 0 gives ErrRestartRefused.
func (c *GameConn) Restart(ctx context.Context) (CharacterList, error) {
	if err := c.send(ctx, encodeOpcode(OpRequestRestart)); err != nil {
		return CharacterList{}, err
	}
	rr, err := Await[RestartResponse](ctx, c, nil)
	if err != nil {
		return CharacterList{}, err
	}
	if !rr.OK {
		return CharacterList{}, ErrRestartRefused
	}
	return Await[CharacterList](ctx, c, nil)
}

// Logout sends Logout, waits for LeaveWorld (skipping the packets before
// it) and then for the server to close the connection; a packet after
// LeaveWorld gives an *UnexpectedPacketError. Mobius refuses a logout
// during combat with ActionFailed alone, so such a call ends with its
// context.
func (c *GameConn) Logout(ctx context.Context) error {
	if err := c.send(ctx, encodeOpcode(OpLogout)); err != nil {
		return err
	}
	if _, err := Await[LeaveWorld](ctx, c, nil); err != nil {
		return err
	}
	return c.ExpectClosed(ctx)
}

// DeleteResult is the answer to CharacterDelete.
type DeleteResult struct {
	// Deleted is true when the server sent CharDeleteSuccess: the character
	// is marked for deletion and its DeleteTimer counts down.
	Deleted bool
	// Failed is true when the server sent CharDeleteFail, with Reason.
	// Neither is set when Mobius hit an error and sent only the list.
	Failed bool
	Reason DeleteFailReason
	// Characters is the updated character list.
	Characters CharacterList
}

// DeleteCharacter sends CharacterDelete for slot and reads the answer
// (CharDeleteSuccess, CharDeleteFail or nothing) and the character list
// Mobius always sends after it.
func (c *GameConn) DeleteCharacter(ctx context.Context, slot int32) (DeleteResult, error) {
	const want = "CharDeleteSuccess, CharDeleteFail or CharSelectionInfo"
	p, body, err := c.request(ctx, encodeCharacterDelete(slot), want)
	if err != nil {
		return DeleteResult{}, err
	}
	var res DeleteResult
	switch v := p.(type) {
	case CharacterList:
		res.Characters = v
		return res, nil
	case CharDeleteSuccess:
		res.Deleted = true
	case CharDeleteFail:
		res.Failed, res.Reason = true, v.Reason
	default:
		return DeleteResult{}, unexpectedGame(body, want)
	}
	const next = "CharSelectionInfo after the delete answer"
	p, body, err = c.read(ctx, next)
	if err != nil {
		return DeleteResult{}, err
	}
	list, ok := p.(CharacterList)
	if !ok {
		return DeleteResult{}, unexpectedGame(body, next)
	}
	res.Characters = list
	return res, nil
}

// RestoreCharacter sends CharacterRestore for slot and returns the
// character list Mobius sends back (it has no other answer).
func (c *GameConn) RestoreCharacter(ctx context.Context, slot int32) (CharacterList, error) {
	const want = "CharSelectionInfo"
	p, body, err := c.request(ctx, encodeCharacterRestore(slot), want)
	if err != nil {
		return CharacterList{}, err
	}
	if v, ok := p.(CharacterList); ok {
		return v, nil
	}
	return CharacterList{}, unexpectedGame(body, want)
}

// Action clicks objectID (Action), from the position the world model
// knows; shift sends a shift click. The first click targets an object, the
// second talks to an NPC or attacks.
func (c *GameConn) Action(ctx context.Context, objectID int32, shift bool) error {
	return c.send(ctx, encodeAction(OpAction, objectID, c.world.Me().Pos, shift))
}

// Target clicks objectID once and returns the MyTargetSelected that
// confirms it as the target.
func (c *GameConn) Target(ctx context.Context, objectID int32) (MyTargetSelected, error) {
	if err := c.Action(ctx, objectID, false); err != nil {
		return MyTargetSelected{}, err
	}
	return Await(ctx, c, func(m MyTargetSelected) bool { return m.ObjectID == objectID })
}

// Talk opens the dialog of the NPC objectID the way a double click does:
// it targets the NPC unless it is the target already, clicks it again and
// returns the NpcHtmlMessage. Mobius first walks the character into
// interaction range (MoveToPawn); the ActionFailed after the page stays
// queued.
func (c *GameConn) Talk(ctx context.Context, objectID int32) (NpcHtmlMessage, error) {
	if c.world.Me().TargetID != objectID {
		if _, err := c.Target(ctx, objectID); err != nil {
			return NpcHtmlMessage{}, err
		}
	}
	if err := c.Action(ctx, objectID, false); err != nil {
		return NpcHtmlMessage{}, err
	}
	return Await[NpcHtmlMessage](ctx, c, nil)
}

// CancelTarget drops the target like the Esc key (RequestTargetCanceld
// with 0). The server answers with TargetUnselected when there was a
// target.
func (c *GameConn) CancelTarget(ctx context.Context) error {
	return c.send(ctx, encodeRequestTargetCanceld(0))
}

// Bypass sends a dialog link (RequestBypassToServer), such as
// "npc_268441600_Chat 1" or "Script MyNpc hello".
func (c *GameConn) Bypass(ctx context.Context, cmd string) error {
	return c.send(ctx, encodeString(OpRequestBypassToServer, cmd))
}

// LinkHtml sends a "link" dialog link (RequestLinkHtml): link is the page
// path below data/html.
func (c *GameConn) LinkHtml(ctx context.Context, link string) error {
	return c.send(ctx, encodeString(OpRequestLinkHtml, link))
}

// RequestSocialAction plays a social action (2-13); the server broadcasts
// SocialAction.
func (c *GameConn) RequestSocialAction(ctx context.Context, actionID int32) error {
	return c.send(ctx, encodeInt(OpRequestSocialAction, actionID))
}

// AttackRequest sends AttackRequest for objectID and waits for the
// ActionFailed the server answers with when it refuses the attack, for
// example on an object that cannot be attacked. Packets queued before the
// request are dropped first, so a stale ActionFailed does not count.
// Attack waits for the attack itself, SendAttackRequest for nothing.
func (c *GameConn) AttackRequest(ctx context.Context, objectID int32, shift bool) error {
	_, _ = c.Drain()
	if err := c.send(ctx, encodeAction(OpAttackRequest, objectID, c.world.Me().Pos, shift)); err != nil {
		return err
	}
	_, err := Await[ActionFailed](ctx, c, nil)
	return err
}
