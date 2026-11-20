package recording

import (
	"cmp"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/cmd/refdiff/scenario"
	"github.com/IPeter/l2go/internal/testclient"
)

// minTextID is the smallest number taken for an object ID inside a string
// or inside the bytes of an unparsed packet. Mobius and L2Go hand out
// object IDs from 0x10000000 on, so small numbers such as prices, counts
// and template IDs are never mistaken for one.
const minTextID = 1 << 24

// Placeholders.
const (
	phSessionID = "<session-id>"
	phServerID  = "<server-id>"
	phGameTime  = "<game-time>"
)

// fixedPlaceholders replace fields whose values are random or depend on
// the time or the deployment, keyed by packet name and generic path.
var fixedPlaceholders = map[string]string{
	"Init.SessionID":                      phSessionID,
	"Init.ScrambledModulus":               "<rsa-modulus>",
	"Init.BlowfishKey":                    "<blowfish-key>",
	"LoginOk.LoginOK1":                    "<login-ok1>",
	"LoginOk.LoginOK2":                    "<login-ok2>",
	"PlayOk.PlayOK1":                      "<play-ok1>",
	"PlayOk.PlayOK2":                      "<play-ok2>",
	"KeyPacket.KeyHalf":                   "<xor-key>",
	"CharSelected.GameTime":               phGameTime,
	"ServerList.Servers[].IP":             "<ip>",
	"ServerList.Servers[].Port":           "<port>",
	"ServerList.Servers[].CurrentPlayers": "<players>",
}

// serverIDFields hold a game server ID: it becomes <server-id> when it is
// the target's (L2Go is server 1, the reference server 2).
var serverIDFields = map[string]bool{
	"KeyPacket.ServerID":               true,
	"ServerList.Servers[].ID":          true,
	"ServerList.Characters[].ServerID": true,
}

// keyFields hold a session key or the session ID: the value becomes the
// placeholder of the key it equals.
var keyFields = map[string]bool{
	"GGAuth.Response":                          true,
	"CharSelectionInfo.Characters[].SessionID": true,
	"CharSelected.SessionID":                   true,
}

// idFields are the struct fields that hold object IDs.
var idFields = map[string]bool{
	"ObjectID":           true,
	"TargetID":           true,
	"VehicleID":          true,
	"OnlineObjectID":     true,
	"ClanID":             true,
	"AllyID":             true,
	"ClanCrestID":        true,
	"AllyCrestID":        true,
	"ClanCrestLargeID":   true,
	"PaperdollObjectIDs": true,
	"EquipObjectIDs":     true,
	// Milestone 5–7 packets (ground items, combat, casting).
	"AttackerID": true,
	"CasterID":   true,
	"DropperID":  true,
	"PlayerID":   true,
	"Targets":    true,
	"ObjectID2":  true,
}

// idKind returns the kind of object an ID field names, for its canonical
// name.
func idKind(packet, path, field string) string {
	switch field {
	case "PaperdollObjectIDs", "EquipObjectIDs":
		return "item"
	case "ClanID", "AllyID":
		return "clan"
	case "ClanCrestID", "AllyCrestID", "ClanCrestLargeID":
		return "crest"
	case "VehicleID":
		return "vehicle"
	case "OnlineObjectID":
		return "player"
	case "PlayerID":
		return "player"
	case "ObjectID":
		switch {
		case strings.HasPrefix(path, "Items["):
			return "item"
		case strings.HasPrefix(path, "Friends["):
			return "player"
		}
		switch packet {
		case "CharSelectionInfo", "CharSelected", "UserInfo", "CharInfo":
			return "player"
		case "NpcInfo", "ServerObjectInfo":
			return "npc"
		case "DropItem", "SpawnItem", "GetItem", "EquipUpdate":
			return "item"
		}
	}
	return "obj"
}

// mask hides n bytes at off of an unparsed packet's body behind name.
type mask struct {
	off, n int
	name   string
}

// rawMasks are the time values inside packets the client does not parse.
var rawMasks = map[string][]mask{
	// ClientSetTime: d game time in minutes, d days per real day.
	"ClientSetTime": {{off: 0, n: 4, name: "game-time"}},
}

// Visibility bursts. A burst is a run of consecutive server packets whose
// names are all in groupHeads or groupFollowers. A head packet starts a
// unit; a follower about the same object as the unit's head joins it
// (CharInfo then RelationChanged, DoorInfo then DoorStatusUpdate). The
// units of a burst are sorted by content, because the order in which a
// server shows the objects around a player comes from its own iteration
// order.
var (
	groupHeads = map[string]bool{
		"NpcInfo": true, "ServerObjectInfo": true, "CharInfo": true, "StaticObjectInfo": true,
		"SpawnItem": true, "DoorInfo": true, "DeleteObject": true,
	}
	groupFollowers = map[string]bool{
		"RelationChanged": true, "DoorStatusUpdate": true, "MoveToLocation": true,
	}
)

// item is a packet on its way through normalization.
type item struct {
	dir    testclient.Direction
	name   string
	opcode byte
	ex     uint16
	leaves []leaf
	// raw is the body of an unparsed or malformed server packet.
	raw       []byte
	hasRaw    bool
	masks     []mask
	malformed bool
	errText   string
	// objID is the raw object the packet is about, for bursts: its
	// ObjectID field, or the first word of an unparsed body.
	objID int32
}

func (it *item) grouped() bool {
	return it.dir == testclient.FromServer && (groupHeads[it.name] || groupFollowers[it.name])
}

// replacement is a name to replace in strings.
type replacement struct{ from, to string }

// idMap gives object IDs their canonical names.
type idMap struct {
	names  map[int32]string
	counts map[string]int
}

func newIDMap() idMap {
	return idMap{names: make(map[int32]string), counts: make(map[string]int)}
}

// assign names id when it is new; 0 never gets a name.
func (m idMap) assign(id int32, kind string) {
	if id == 0 {
		return
	}
	if _, ok := m.names[id]; ok {
		return
	}
	m.counts[kind]++
	m.names[id] = fmt.Sprintf("#%s%d", kind, m.counts[kind])
}

func (m idMap) name(id int32) (string, bool) {
	n, ok := m.names[id]
	return n, ok
}

// render returns the canonical name of id, "0" for 0, or the number for an
// ID without a name.
func (m idMap) render(id int32) string {
	if id == 0 {
		return "0"
	}
	if n, ok := m.names[id]; ok {
		return n
	}
	return strconv.Itoa(int(id))
}

// normalizer holds what one capture needs for normalization.
type normalizer struct {
	names    []replacement
	secrets  map[int32]string
	serverID string
	anchors  map[int32]testclient.Position
	ids      idMap
	// monsters holds the raw IDs of the NPCs some player could attack
	// (NpcInfo's attackable flag), droppers the raw dropper of every item
	// a DropItem showed falling.
	monsters map[int32]bool
	droppers map[int32]int32
	// adenaStart is the adena of each role in its first ItemList.
	// startAdena is the adena of the role of the stream being normalized,
	// hasAdena whether it is known.
	adenaStart map[string]int64
	startAdena int64
	hasAdena   bool
}

func newNormalizer(c *scenario.Capture) *normalizer {
	n := &normalizer{
		secrets:    make(map[int32]string),
		serverID:   strconv.Itoa(int(c.Target.ServerID)),
		anchors:    make(map[int32]testclient.Position),
		ids:        newIDMap(),
		monsters:   make(map[int32]bool),
		droppers:   make(map[int32]int32),
		adenaStart: make(map[string]int64),
	}
	n.scan(c)
	for _, p := range c.Players {
		if p.Account != "" {
			n.names = append(n.names, replacement{p.Account, "<account:" + p.Role + ">"})
		}
		if p.Char != "" {
			n.names = append(n.names, replacement{p.Char, "<char:" + p.Role + ">"})
		}
	}
	// Longest first, so that a name inside a longer one does not break it.
	slices.SortStableFunc(n.names, func(a, b replacement) int { return cmp.Compare(len(b.from), len(a.from)) })
	for _, id := range c.SessionIDs {
		n.secret(id, phSessionID)
	}
	for _, k := range c.Keys {
		n.secret(k.LoginOK1, "<login-ok1>")
		n.secret(k.LoginOK2, "<login-ok2>")
		n.secret(k.PlayOK1, "<play-ok1>")
		n.secret(k.PlayOK2, "<play-ok2>")
	}
	return n
}

// scan collects what the rules need from the whole capture before any
// packet is normalized: the start points (anchors), the monsters, who
// dropped which item, and each role's adena at the start.
func (n *normalizer) scan(c *scenario.Capture) {
	for _, st := range c.Streams {
		role, isGame := streamRole(st.Name)
		for _, p := range st.Packets {
			switch v := p.Value.(type) {
			case testclient.CharSelected:
				if _, seen := n.anchors[v.ObjectID]; !seen {
					n.anchors[v.ObjectID] = testclient.Position{X: v.X, Y: v.Y, Z: v.Z}
				}
			case testclient.NpcInfo:
				if v.Attackable {
					n.monsters[v.ObjectID] = true
				}
			case testclient.ServerObjectInfo:
				if v.Attackable {
					n.monsters[v.ObjectID] = true
				}
			case testclient.DropItem:
				n.droppers[v.ObjectID] = v.DropperID
			case testclient.ItemList:
				if _, seen := n.adenaStart[role]; isGame && !seen {
					n.adenaStart[role] = adenaIn(v.Items)
				}
			}
		}
	}
}

// streamRole returns the role of a game stream ("game alice 2" gives
// alice).
func streamRole(name string) (string, bool) {
	f := strings.Fields(name)
	if len(f) < 2 || f[0] != "game" {
		return "", false
	}
	return f[1], true
}

// adenaIn returns the adena among items.
func adenaIn(items []testclient.Item) int64 {
	var n int64
	for _, it := range items {
		if it.ItemID == itemAdena {
			n += int64(it.Count)
		}
	}
	return n
}

// secret records a random value with its placeholder. 0 is never one.
func (n *normalizer) secret(v int32, placeholder string) {
	if v != 0 {
		n.secrets[v] = placeholder
	}
}

// Normalize turns a capture into a recording that does not depend on the
// server or the run (see the package documentation for the rules).
func Normalize(c *scenario.Capture) Recording {
	n := newNormalizer(c)
	rec := Recording{
		Version:  Version,
		Scenario: c.Scenario,
		Server:   c.Target.Label,
		Source:   c.Target.Source(),
	}
	streams := make([][]*item, len(c.Streams))
	for si, st := range c.Streams {
		role, _ := streamRole(st.Name)
		n.startAdena, n.hasAdena = n.adenaStart[role]
		items := make([]*item, len(st.Packets))
		for i, p := range st.Packets {
			items[i] = n.item(p)
		}
		n.order(items)
		streams[si] = items
	}
	for si, st := range c.Streams {
		out := Stream{Name: st.Name, Entries: make([]Entry, len(streams[si]))}
		for i, it := range streams[si] {
			out.Entries[i] = n.entry(it)
		}
		out.Entries = collapseFights(out.Entries)
		rec.Streams = append(rec.Streams, out)
	}
	rec.Anchors = n.anchorList()
	if c.Err != nil {
		rec.Errors = []string{n.text(c.Err.Error())}
	}
	for _, note := range c.Notes {
		rec.Notes = append(rec.Notes, n.text(note))
	}
	return rec
}

// item flattens one packet and applies the field rules.
func (n *normalizer) item(p scenario.Packet) *item {
	it := &item{dir: p.Dir, name: p.Name, opcode: p.Opcode, ex: p.ExOpcode}
	if p.Dir != testclient.FromServer {
		return it
	}
	switch v := p.Value.(type) {
	case nil:
		it.malformed = true
		if p.Err != nil {
			it.errText = p.Err.Error()
		}
		if len(p.Raw) > 0 {
			it.raw, it.hasRaw = p.Raw[1:], true
		}
	case testclient.Unknown:
		it.raw, it.hasRaw = v.Body, true
		it.masks = rawMasks[p.Name]
		it.objID = firstWord(v)
	default:
		it.leaves = n.leaves(p.Name, unordered(v))
		for _, l := range it.leaves {
			if l.path == "ObjectID" && l.kind != "" {
				it.objID = l.id
				break
			}
		}
	}
	return it
}

// unordered sorts the lists whose order is the server's iteration order,
// keeping the order of equal entries: Mobius keeps the inventory and the
// warehouse in hash sets, so the items of ItemList, SellList and the
// warehouse windows are sorted by item ID, and the changes of
// InventoryUpdate (which Mobius gathers in a map) by item ID and change.
func unordered(v any) any {
	byItemID := func(a, b testclient.Item) int { return cmp.Compare(a.ItemID, b.ItemID) }
	switch l := v.(type) {
	case testclient.ItemList:
		l.Items = slices.Clone(l.Items)
		slices.SortStableFunc(l.Items, byItemID)
		return l
	case testclient.InventoryUpdate:
		l.Items = slices.Clone(l.Items)
		slices.SortStableFunc(l.Items, func(a, b testclient.ItemChange) int {
			return cmp.Or(byItemID(a.Item, b.Item), cmp.Compare(a.Change, b.Change))
		})
		return l
	case testclient.SellList:
		l.Items = slices.Clone(l.Items)
		slices.SortStableFunc(l.Items, func(a, b testclient.SellListItem) int { return cmp.Compare(a.ItemID, b.ItemID) })
		return l
	case testclient.WareHouseDepositList:
		l.WarehouseList = sortedWarehouse(l.WarehouseList)
		return l
	case testclient.WareHouseWithdrawalList:
		l.WarehouseList = sortedWarehouse(l.WarehouseList)
		return l
	}
	return v
}

func sortedWarehouse(l testclient.WarehouseList) testclient.WarehouseList {
	l.Items = slices.Clone(l.Items)
	slices.SortStableFunc(l.Items, func(a, b testclient.WarehouseListItem) int { return cmp.Compare(a.ItemID, b.ItemID) })
	return l
}

// firstWord returns the first word of an unparsed body (after the
// sub-opcode of an Ex packet), which is the object ID in most packets
// about an object.
func firstWord(u testclient.Unknown) int32 {
	b := u.Body
	if u.Opcode == testclient.OpExServer {
		if len(b) < 2 {
			return 0
		}
		b = b[2:]
	}
	if len(b) < 4 {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(b)) //nolint:gosec // same bits as the Java int
}

// leaves flattens a decoded packet and applies the placeholder, ID and
// coordinate rules.
func (n *normalizer) leaves(name string, v any) []leaf {
	ls := flatten(v)
	for i := range ls {
		l := &ls[i]
		key := name + "." + genericPath(l.path)
		field := fieldName(l.path)
		switch {
		case fixedPlaceholders[key] != "":
			l.val, l.text = fixedPlaceholders[key], false
		case serverIDFields[key]:
			if l.val == n.serverID {
				l.val = phServerID
			}
		case keyFields[key]:
			if id, err := strconv.ParseInt(l.val, 10, 32); err == nil {
				if ph, ok := n.secrets[int32(id)]; ok {
					l.val = ph
				}
			}
		case idFields[field]:
			if id, err := strconv.ParseInt(l.val, 10, 32); err == nil {
				l.id, l.kind = int32(id), idKind(name, l.path, field)
			}
		}
	}
	if name == "ShortcutInit" {
		markShortcutItems(ls)
	}
	n.relativeCoords(name, ls)
	n.randomValues(name, ls)
	return ls
}

// markShortcutItems marks the IDs of item shortcuts as item object IDs.
// The other shortcut types carry skill, action, macro or recipe IDs.
func markShortcutItems(ls []leaf) {
	itemType := strconv.Itoa(int(testclient.ShortcutItem))
	for i := range ls {
		p := ls[i].path
		if !strings.HasPrefix(p, "Shortcuts[") || !strings.HasSuffix(p, ".Type") || ls[i].val != itemType {
			continue
		}
		idPath := strings.TrimSuffix(p, ".Type") + ".ID"
		for j := i + 1; j < len(ls); j++ {
			if ls[j].path != idPath {
				continue
			}
			if id, err := strconv.ParseInt(ls[j].val, 10, 32); err == nil {
				ls[j].id, ls[j].kind = int32(id), "item"
			}
			break
		}
	}
}

// Placeholders of the coordinate rules.
const (
	phMonster = "<monster>"
	phDrop    = "<drop>"
)

// relativeCoords rewrites the top-level coordinates of a packet by whose
// they are:
//   - X, Y and Z (where the object is) and DestX, DestY and DestZ (where it
//     walks to) of one of the scenario's characters become relative to its
//     anchor; those of an item a character dropped, relative to the
//     dropper's anchor. The object is the packet's ObjectID, else its
//     AttackerID.
//   - TargetX, TargetY and TargetZ (MoveToPawn, Attack) follow the same
//     rule for the target (TargetID, or the first hit's target).
//   - A monster's coordinates and heading become <monster>: it stands at
//     a random point of its territory and wanders. Those of an item a
//     monster dropped become <drop>: loot falls at random offsets.
//
// Other coordinates, such as where a town NPC stands, stay.
func (n *normalizer) relativeCoords(name string, ls []leaf) {
	var subject, target int32
	for _, l := range ls {
		switch {
		case l.kind == "":
		case l.path == "ObjectID" && subject == 0:
			subject = l.id
		case l.path == "AttackerID" && subject == 0:
			subject = l.id
		case l.path == "TargetID" || l.path == "Hits[0].TargetID":
			if target == 0 {
				target = l.id
			}
		}
	}
	if subject != 0 {
		n.rewriteCoords(ls, subject, "", true)
	}
	if target != 0 {
		n.rewriteCoords(ls, target, "Target", false)
	}
}

// rewriteCoords applies the coordinate rule of object id to the fields
// prefix+X, prefix+Y and prefix+Z (and, when own is set, to DestX, DestY,
// DestZ and a monster's Heading).
func (n *normalizer) rewriteCoords(ls []leaf, id int32, prefix string, own bool) {
	anchor, ok := n.anchors[id]
	placeholder := ""
	if !ok {
		if dropper, dropped := n.droppers[id]; dropped {
			anchor, ok = n.anchors[dropper]
			if !ok {
				placeholder = phDrop
			}
		} else if n.monsters[id] {
			placeholder = phMonster
		}
	}
	if !ok && placeholder == "" {
		return
	}
	for i := range ls {
		var base int32
		switch p := ls[i].path; p {
		case prefix + "X", prefix + "Y", prefix + "Z":
			base = axis(anchor, p[len(prefix):])
		case "DestX", "DestY", "DestZ":
			if !own {
				continue
			}
			base = axis(anchor, strings.TrimPrefix(p, "Dest"))
		case "Heading":
			if own && placeholder == phMonster {
				ls[i].val = placeholder
			}
			continue
		default:
			continue
		}
		if placeholder != "" {
			ls[i].val = placeholder
			continue
		}
		v, err := strconv.ParseInt(ls[i].val, 10, 64)
		if err != nil {
			continue
		}
		ls[i].val = relative(v - int64(base))
	}
}

// axis returns the coordinate of p named "X", "Y" or "Z".
func axis(p testclient.Position, name string) int32 {
	switch name {
	case "X":
		return p.X
	case "Y":
		return p.Y
	}
	return p.Z
}

// relative formats an offset from a start point.
func relative(d int64) string {
	switch {
	case d == 0:
		return "start"
	case d > 0:
		return "start+" + strconv.FormatInt(d, 10)
	default:
		return "start" + strconv.FormatInt(d, 10)
	}
}

// order sorts the visibility bursts of a stream and gives the stream's new
// object IDs their names, in one pass: a burst's sort keys use the names
// given before it.
func (n *normalizer) order(items []*item) {
	for i := 0; i < len(items); {
		if !items[i].grouped() {
			n.assignIDs(items[i])
			i++
			continue
		}
		j := i + 1
		for j < len(items) && items[j].grouped() {
			j++
		}
		n.sortBurst(items[i:j])
		for _, it := range items[i:j] {
			n.assignIDs(it)
		}
		i = j
	}
}

func (n *normalizer) assignIDs(it *item) {
	for _, l := range it.leaves {
		if l.kind != "" {
			n.ids.assign(l.id, l.kind)
		}
	}
}

// sortBurst sorts the units of a visibility burst by their head's name and
// content, keeping the order of equal units.
func (n *normalizer) sortBurst(burst []*item) {
	var units [][]*item
	for _, it := range burst {
		if k := len(units) - 1; k >= 0 && groupFollowers[it.name] && it.objID != 0 && it.objID == units[k][0].objID {
			units[k] = append(units[k], it)
			continue
		}
		units = append(units, []*item{it})
	}
	if len(units) < 2 {
		return
	}
	keys := make([]string, len(units))
	for i, u := range units {
		keys[i] = u[0].name + "\x00" + n.sortKey(u[0])
	}
	order := make([]int, len(units))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return strings.Compare(keys[a], keys[b]) })
	sorted := make([]*item, 0, len(burst))
	for _, i := range order {
		sorted = append(sorted, units[i]...)
	}
	copy(burst, sorted)
}

// sortKey is the content of a packet for sorting: its fields, with object
// IDs that have no name yet left out.
func (n *normalizer) sortKey(it *item) string {
	if it.leaves == nil {
		return n.rawText(it.raw, it.masks, true)
	}
	var b strings.Builder
	for _, l := range it.leaves {
		b.WriteString(l.path)
		b.WriteByte('=')
		switch {
		case l.kind != "":
			if _, ok := n.ids.name(l.id); ok || l.id == 0 {
				b.WriteString(n.ids.render(l.id))
			} else {
				b.WriteByte('?')
			}
		case l.text:
			b.WriteString(n.text(l.val))
		default:
			b.WriteString(l.val)
		}
		b.WriteByte(';')
	}
	return b.String()
}

// entry renders a normalized packet.
func (n *normalizer) entry(it *item) Entry {
	e := Entry{Dir: DirServer, Name: it.name, Opcode: it.opcode, ExOpcode: it.ex}
	if it.dir != testclient.FromServer {
		e.Dir = DirClient
		return e
	}
	for _, l := range it.leaves {
		e.Fields = append(e.Fields, Field{Path: l.path, Value: n.value(l)})
	}
	if it.hasRaw {
		e.Raw = n.rawText(it.raw, it.masks, false)
	}
	e.Malformed = it.malformed
	if it.errText != "" {
		e.Error = n.text(it.errText)
	}
	return e
}

// value renders one field.
func (n *normalizer) value(l leaf) string {
	switch {
	case l.kind != "":
		return n.ids.render(l.id)
	case l.text:
		return strconv.Quote(n.text(l.val))
	}
	return l.val
}

// text replaces the scenario's names and the known object IDs in s.
func (n *normalizer) text(s string) string {
	for _, r := range n.names {
		s = strings.ReplaceAll(s, r.from, r.to)
	}
	return n.replaceIDs(s)
}

// replaceIDs replaces every run of digits in s that is a known object ID
// by its canonical name.
func (n *normalizer) replaceIDs(s string) string {
	var (
		b    strings.Builder
		last int
	)
	for i := 0; i < len(s); {
		if !isDigit(s[i]) {
			i++
			continue
		}
		j := i
		for j < len(s) && isDigit(s[j]) {
			j++
		}
		if v, err := strconv.ParseInt(s[i:j], 10, 32); err == nil && v >= minTextID {
			if name, ok := n.ids.name(int32(v)); ok {
				b.WriteString(s[last:i])
				b.WriteString(name)
				last = j
			}
		}
		i = j
	}
	if last == 0 {
		return s
	}
	b.WriteString(s[last:])
	return b.String()
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// rawText writes an unparsed body in hex, with masked values and known
// object IDs as {name}. For a sort key (key set), words that look like
// object IDs without a name yet become {?}, so that the key does not
// depend on the server's IDs.
func (n *normalizer) rawText(b []byte, masks []mask, key bool) string {
	var sb strings.Builder
	for i := 0; i < len(b); {
		if m, ok := maskAt(masks, i, len(b)); ok {
			sb.WriteString("{" + m.name + "}")
			i += m.n
			continue
		}
		if i+4 <= len(b) {
			id := int32(binary.LittleEndian.Uint32(b[i:])) //nolint:gosec // same bits as the Java int
			if id >= minTextID {
				name, ok := n.ids.name(id)
				if !ok && key {
					name, ok = "?", true
				}
				if ok {
					sb.WriteString("{" + name + "}")
					i += 4
					continue
				}
			}
		}
		sb.WriteString(hex.EncodeToString(b[i : i+1]))
		i++
	}
	return sb.String()
}

// maskAt returns the mask that starts at offset i and fits in size bytes.
func maskAt(masks []mask, i, size int) (mask, bool) {
	for _, m := range masks {
		if m.off == i && m.off+m.n <= size {
			return m, true
		}
	}
	return mask{}, false
}

// anchorList returns the start points by canonical name.
func (n *normalizer) anchorList() []Anchor {
	var out []Anchor
	for id, p := range n.anchors {
		out = append(out, Anchor{Object: n.ids.render(id), X: p.X, Y: p.Y, Z: p.Z})
	}
	slices.SortFunc(out, func(a, b Anchor) int { return strings.Compare(a.Object, b.Object) })
	return out
}
