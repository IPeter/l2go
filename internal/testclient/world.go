package testclient

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"sync"
)

// logLimit is how many chat lines, NPC speeches and system messages the
// world model keeps; older ones are dropped.
const logLimit = 1000

// DistanceTo returns the 3D distance between p and q.
func (p Position) DistanceTo(q Position) float64 {
	dx := float64(q.X) - float64(p.X)
	dy := float64(q.Y) - float64(p.Y)
	dz := float64(q.Z) - float64(p.Z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// Motion is the movement state the world model keeps for the character
// and for every visible object.
type Motion struct {
	// Pos is the last position the server reported.
	Pos     Position
	Heading int32
	// Moving is set by MoveToLocation and MoveToPawn, with Dest where the
	// movement ends, and cleared by StopMove, TeleportToLocation and
	// ChangeWaitType. The server does not report a normal arrival, so
	// Moving stays set after a walk that ended at Dest.
	Moving  bool
	Dest    Position
	Running bool
	Sitting bool
	// TargetID is the object's target (TargetSelected, or MyTargetSelected
	// for the character), 0 for none.
	TargetID int32
}

// ObjectKind tells which packet showed an object.
type ObjectKind uint8

// Object kinds.
const (
	// KindPlayer is a player shown by CharInfo.
	KindPlayer ObjectKind = iota + 1
	// KindNpc is an NPC, monster or summon shown by NpcInfo.
	KindNpc
	// KindStatic is an NPC shown by ServerObjectInfo.
	KindStatic
)

func (k ObjectKind) String() string {
	switch k {
	case KindPlayer:
		return "player"
	case KindNpc:
		return "npc"
	case KindStatic:
		return "static"
	}
	return fmt.Sprintf("ObjectKind(%d)", uint8(k))
}

// Object is what the client knows about an object it can see.
type Object struct {
	Motion
	ObjectID int32
	Kind     ObjectKind
	Name     string
	Title    string
	// NpcID is the NPC template's display ID, 0 for players.
	NpcID      int32
	Attackable bool
	// HP and MaxHP come from StatusUpdate (and ServerObjectInfo), 0 while
	// unknown.
	HP, MaxHP int32
	// Relation and AutoAttackable come from RelationChanged.
	Relation       int32
	AutoAttackable bool
	// Info is the last CharInfoPacket, NpcInfo or ServerObjectInfo of the
	// object.
	Info Packet
}

// Me is what the client knows about its own character.
type Me struct {
	Motion
	ObjectID  int32
	Name      string
	HP, MaxHP int32
	MP, MaxMP int32
	CP, MaxCP int32
	// UserInfo is the last UserInfo; HasUserInfo tells whether one came.
	UserInfo    UserInfo
	HasUserInfo bool
}

// World is the client's view of the game world, kept up to date from every
// server packet the connection reads: the character (CharSelected,
// UserInfo), the visible objects (CharInfo, NpcInfo, ServerObjectInfo,
// DeleteObject) with their movement, targets and HP, the inventory, the
// chat and system message logs and the last dialog page. CharSelected and
// a successful RestartResponse clear it.
//
// It is safe for concurrent use. Its methods return copies.
type World struct {
	mu      sync.Mutex
	me      Me
	objects map[int32]*Object
	items   []Item
	chat    []CreatureSay
	npcSays []NpcSay
	sysMsgs []SystemMessage
	html    NpcHtmlMessage
	hasHTML bool
}

func newWorld() *World {
	return &World{objects: make(map[int32]*Object)}
}

// Me returns the character's state.
func (w *World) Me() Me {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.me
}

// Object returns the visible object with the given ID.
func (w *World) Object(id int32) (Object, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	o, ok := w.objects[id]
	if !ok {
		return Object{}, false
	}
	return *o, true
}

// Objects returns every visible object, sorted by object ID.
func (w *World) Objects() []Object {
	return w.selectObjects(func(*Object) bool { return true })
}

// Players returns the visible players, sorted by object ID.
func (w *World) Players() []Object {
	return w.selectObjects(func(o *Object) bool { return o.Kind == KindPlayer })
}

// Npcs returns the visible NPCs (NpcInfo and ServerObjectInfo), sorted by
// object ID.
func (w *World) Npcs() []Object {
	return w.selectObjects(func(o *Object) bool { return o.Kind != KindPlayer })
}

// FindPlayer returns the visible player called name.
func (w *World) FindPlayer(name string) (Object, bool) {
	ps := w.selectObjects(func(o *Object) bool { return o.Kind == KindPlayer && o.Name == name })
	if len(ps) == 0 {
		return Object{}, false
	}
	return ps[0], true
}

// FindNpc returns the visible NPC of template npcID nearest to the
// character.
func (w *World) FindNpc(npcID int32) (Object, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	var best *Object
	bestDist := math.Inf(1)
	for _, o := range w.objects {
		if o.Kind == KindPlayer || o.NpcID != npcID {
			continue
		}
		d := o.Pos.DistanceTo(w.me.Pos)
		if best == nil || d < bestDist || (d == bestDist && o.ObjectID < best.ObjectID) {
			best, bestDist = o, d
		}
	}
	if best == nil {
		return Object{}, false
	}
	return *best, true
}

// Items returns the inventory: the last ItemList with the InventoryUpdates
// after it applied.
func (w *World) Items() []Item {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.items)
}

// ItemCount returns the total count of the items of itemID in the
// inventory (57 is adena).
func (w *World) ItemCount(itemID int32) int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	var n int64
	for _, it := range w.items {
		if it.ItemID == itemID {
			n += int64(it.Count)
		}
	}
	return n
}

// Chat returns the chat log (CreatureSay), oldest first.
func (w *World) Chat() []CreatureSay {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.chat)
}

// NpcSays returns the NPC speech log (NpcSay), oldest first.
func (w *World) NpcSays() []NpcSay {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.npcSays)
}

// SystemMessages returns the system message log, oldest first.
func (w *World) SystemMessages() []SystemMessage {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.sysMsgs)
}

// HTML returns the last dialog page (NpcHtmlMessage).
func (w *World) HTML() (NpcHtmlMessage, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.html, w.hasHTML
}

func (w *World) selectObjects(keep func(*Object) bool) []Object {
	w.mu.Lock()
	defer w.mu.Unlock()
	var out []Object
	for _, o := range w.objects {
		if keep(o) {
			out = append(out, *o)
		}
	}
	slices.SortFunc(out, func(a, b Object) int { return cmp.Compare(a.ObjectID, b.ObjectID) })
	return out
}

// reset forgets everything; the caller holds w.mu.
func (w *World) reset() {
	w.me = Me{}
	clear(w.objects)
	w.items = nil
	w.chat, w.npcSays, w.sysMsgs = nil, nil, nil
	w.html, w.hasHTML = NpcHtmlMessage{}, false
}

// object returns the object id, adding it when it is new; the caller holds
// w.mu.
func (w *World) object(id int32, kind ObjectKind) *Object {
	o, ok := w.objects[id]
	if !ok {
		o = &Object{ObjectID: id}
		w.objects[id] = o
	}
	o.Kind = kind
	return o
}

// motion returns the movement state of the character or of a visible
// object, or nil for an object the client does not know; the caller holds
// w.mu.
func (w *World) motion(id int32) *Motion {
	if id != 0 && id == w.me.ObjectID {
		return &w.me.Motion
	}
	if o, ok := w.objects[id]; ok {
		return &o.Motion
	}
	return nil
}

// apply updates the world from one server packet.
func (w *World) apply(p Packet) {
	w.mu.Lock()
	defer w.mu.Unlock()
	switch v := p.(type) {
	case CharSelected:
		w.reset()
		w.me.ObjectID = v.ObjectID
		w.me.Name = v.Name
		w.me.Pos = Position{v.X, v.Y, v.Z}
	case RestartResponse:
		if v.OK {
			w.reset()
		}
	case UserInfo:
		w.me.ObjectID = v.ObjectID
		w.me.Name = v.Name
		w.me.Pos = v.Position()
		w.me.Running = v.Running
		w.me.HP, w.me.MaxHP = v.HP, v.MaxHP
		w.me.MP, w.me.MaxMP = v.MP, v.MaxMP
		w.me.CP, w.me.MaxCP = v.CP, v.MaxCP
		w.me.UserInfo = v
		w.me.HasUserInfo = true
	case CharInfoPacket:
		o := w.object(v.ObjectID, KindPlayer)
		o.Name, o.Title = v.Name, v.Title
		o.Pos, o.Heading = v.Position(), v.Heading
		o.Running, o.Sitting = v.Running, !v.Standing
		o.Info = v
	case NpcInfo:
		o := w.object(v.ObjectID, KindNpc)
		o.Name, o.Title, o.NpcID = v.Name, v.Title, v.NpcID()
		o.Pos, o.Heading = v.Position(), v.Heading
		o.Running, o.Attackable = v.Running, v.Attackable
		o.Info = v
	case ServerObjectInfo:
		o := w.object(v.ObjectID, KindStatic)
		o.Name, o.NpcID = v.Name, v.NpcID()
		o.Pos, o.Heading = v.Position(), v.Heading
		o.Attackable = v.Attackable
		o.HP, o.MaxHP = v.HP, v.MaxHP
		o.Info = v
	case DeleteObject:
		delete(w.objects, v.ObjectID)
	case RelationChanged:
		if o, ok := w.objects[v.ObjectID]; ok {
			o.Relation, o.AutoAttackable = v.Relation, v.AutoAttackable
		}
	default:
		w.applyMotion(p)
		w.applyOther(p)
	}
}

// applyMotion handles the packets that move, turn or target an object.
func (w *World) applyMotion(p Packet) {
	switch v := p.(type) {
	case MoveToLocation:
		if m := w.motion(v.ObjectID); m != nil {
			m.Pos, m.Dest, m.Moving = v.Origin(), v.Destination(), true
		}
	case MoveToPawn:
		if m := w.motion(v.ObjectID); m != nil {
			m.Pos = Position{v.X, v.Y, v.Z}
			m.Dest = Position{v.TargetX, v.TargetY, v.TargetZ}
			m.Moving = true
		}
	case StopMove:
		if m := w.motion(v.ObjectID); m != nil {
			m.Pos, m.Heading, m.Moving = v.Position(), v.Heading, false
		}
	case ValidateLocation:
		if m := w.motion(v.ObjectID); m != nil {
			m.Pos, m.Heading = v.Position(), v.Heading
		}
	case TeleportToLocation:
		if m := w.motion(v.ObjectID); m != nil {
			m.Pos, m.Heading, m.Moving = v.Position(), v.Heading, false
		}
	case ChangeMoveType:
		if m := w.motion(v.ObjectID); m != nil {
			m.Running = v.Running
		}
	case ChangeWaitType:
		if m := w.motion(v.ObjectID); m != nil {
			m.Pos = Position{v.X, v.Y, v.Z}
			m.Sitting = v.WaitType == WaitSitting
			m.Moving = false
		}
	case StopRotation:
		if m := w.motion(v.ObjectID); m != nil {
			m.Heading = v.Degree
		}
	case MyTargetSelected:
		w.me.TargetID = v.ObjectID
	case TargetSelected:
		if m := w.motion(v.ObjectID); m != nil {
			m.TargetID = v.TargetID
		}
	case TargetUnselected:
		if m := w.motion(v.ObjectID); m != nil {
			m.TargetID = 0
		}
	}
}

// applyOther handles status, inventory, logs and dialogs.
func (w *World) applyOther(p Packet) {
	switch v := p.(type) {
	case StatusUpdate:
		w.status(v)
	case ItemList:
		w.items = slices.Clone(v.Items)
	case InventoryUpdate:
		w.updateItems(v)
	case CreatureSay:
		w.chat = appendLog(w.chat, v)
	case NpcSay:
		w.npcSays = appendLog(w.npcSays, v)
	case SystemMessage:
		w.sysMsgs = appendLog(w.sysMsgs, v)
	case NpcHtmlMessage:
		w.html, w.hasHTML = v, true
	}
}

// status applies a StatusUpdate; the caller holds w.mu.
func (w *World) status(s StatusUpdate) {
	if s.ObjectID != 0 && s.ObjectID == w.me.ObjectID {
		for _, a := range s.Attributes {
			switch a.ID {
			case StatusCurHP:
				w.me.HP = a.Value
			case StatusMaxHP:
				w.me.MaxHP = a.Value
			case StatusCurMP:
				w.me.MP = a.Value
			case StatusMaxMP:
				w.me.MaxMP = a.Value
			case StatusCurCP:
				w.me.CP = a.Value
			case StatusMaxCP:
				w.me.MaxCP = a.Value
			}
		}
		return
	}
	o, ok := w.objects[s.ObjectID]
	if !ok {
		return
	}
	for _, a := range s.Attributes {
		switch a.ID {
		case StatusCurHP:
			o.HP = a.Value
		case StatusMaxHP:
			o.MaxHP = a.Value
		}
	}
}

// updateItems applies an InventoryUpdate; the caller holds w.mu.
func (w *World) updateItems(u InventoryUpdate) {
	for _, ch := range u.Items {
		i := slices.IndexFunc(w.items, func(it Item) bool { return it.ObjectID == ch.ObjectID })
		switch {
		case ch.Change == ItemRemoved:
			if i >= 0 {
				w.items = slices.Delete(w.items, i, i+1)
			}
		case i >= 0:
			w.items[i] = ch.Item
		default:
			w.items = append(w.items, ch.Item)
		}
	}
}

// appendLog appends v, dropping the oldest entry when the log is full.
func appendLog[T any](log []T, v T) []T {
	if len(log) >= logLimit {
		copy(log, log[len(log)-logLimit+1:])
		log = log[:logLimit-1]
	}
	return append(log, v)
}
