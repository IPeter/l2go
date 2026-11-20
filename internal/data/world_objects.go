package data

// World objects: the doors, static objects (town maps, thrones, signboards)
// and fences that cmd/datagen converts from Mobius Doors.xml,
// StaticObjects.xml and FenceData.xml. They are loaded with LoadObjects.

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"slices"
)

// Paths of the world object files, relative to the data folder.
const (
	DoorsFile         = "objects/doors.json"
	StaticObjectsFile = "objects/staticobjects.json"
	FencesFile        = "objects/fences.json"
)

// The ways a door opens (Door.OpenMethod is a set of them), Mobius
// Door.OPEN_BY_*.
const (
	DoorOpenByClick = 1
	DoorOpenByTime  = 2
	DoorOpenByItem  = 4
	DoorOpenBySkill = 8
	DoorOpenByCycle = 16
	doorOpenMethods = DoorOpenByClick | DoorOpenByTime | DoorOpenByItem | DoorOpenBySkill | DoorOpenByCycle
)

// DoorMinCollisionRadius is the smallest collision radius Mobius gives a
// door (DoorData.insertCollisionData).
const DoorMinCollisionRadius = 20

// DoorAction is what a child door does when its master door opens or
// closes (Door.MasterOpen, Door.MasterClose); empty: nothing (Mobius
// act_nothing, the default).
type DoorAction string

// The door actions.
const (
	DoorActOpen  DoorAction = "act_open"
	DoorActClose DoorAction = "act_close"
)

// Door is one <door> of Doors.xml, with Mobius's defaults filled in
// (DoorTemplate). Times are in seconds.
type Door struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// X, Y, Z are the pos attribute. Mobius spawns the door at
	// Z = min(Z, NodeZ) (see SpawnZ).
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
	// Nodes are the 4 corners of the door's outline, in order; the edges
	// join each node with the next and the last with the first.
	Nodes [4]Point `json:"nodes"`
	// NodeZ is the bottom of the door; Height its height. A line crosses the
	// door between NodeZ and NodeZ+Height (exclusive, see ZMin, ZMax).
	NodeZ  int `json:"nodeZ"`
	Height int `json:"height"`
	// HP, PDef and MDef are baseHpMax (Mobius default 1), basePDef and
	// baseMDef (default 0).
	HP    float64 `json:"hp"`
	PDef  int     `json:"pDef"`
	MDef  int     `json:"mDef"`
	Level int     `json:"level,omitempty"`
	// OpenMethod is the open_method bit set of DoorOpenBy* values.
	OpenMethod int `json:"openMethod,omitempty"`
	// DefaultOpen: default_status="open"; doors start closed otherwise.
	DefaultOpen bool `json:"defaultOpen,omitempty"`
	// ShowHP is hp_showable (default true).
	ShowHP bool `json:"showHp"`
	Wall   bool `json:"wall,omitempty"`
	// Group names the door group: opening or closing one door of a group
	// opens or closes the others (Mobius manageGroupOpen).
	Group string `json:"group,omitempty"`
	// ChildID is child_id_event: the door that follows this one, by its
	// MasterOpen and MasterClose actions. 0: none.
	ChildID     int        `json:"childId,omitempty"`
	MasterOpen  DoorAction `json:"masterOpen,omitempty"`
	MasterClose DoorAction `json:"masterClose,omitempty"`
	// Targetable is targetable (default true).
	Targetable bool `json:"targetable"`
	// CloseTime is close_time: an open door closes itself after it, unless it
	// opens by time or belongs to a clan hall (see AutoCloses). A door that
	// opens by time stays open that long. 0: none (Mobius -1).
	CloseTime int `json:"closeTime,omitempty"`
	// OpenTime and RandomTime are read only for doors that open by time: a
	// closed door opens after OpenTime + Rnd(RandomTime) seconds, an open
	// one closes after CloseTime + Rnd(RandomTime). RandomTime 0: none.
	OpenTime   int `json:"openTime,omitempty"`
	RandomTime int `json:"randomTime,omitempty"`
	// CheckCollision is check_collision (default true): whether the door
	// blocks movement and sight.
	CheckCollision bool `json:"checkCollision"`
	// Attackable is is_attackable (default false).
	Attackable bool `json:"attackable,omitempty"`
	ClanHallID int  `json:"clanHallId,omitempty"`
	// Emitter is emitter_id (unused in Interlude, where no door has one).
	Emitter int  `json:"emitter,omitempty"`
	Stealth bool `json:"stealth,omitempty"`
}

// SpawnZ is the Z Mobius spawns the door at: the lower of the pos Z and
// NodeZ.
func (d *Door) SpawnZ() int { return min(d.Z, d.NodeZ) }

// ZMin is the bottom of the door (Mobius Door.getZMin).
func (d *Door) ZMin() int { return d.NodeZ }

// ZMax is the top of the door (Mobius Door.getZMax).
func (d *Door) ZMax() int { return d.NodeZ + d.Height }

// CollisionRadius is the door's collision radius as Mobius computes it:
// the smaller extent between the first two nodes, at least
// DoorMinCollisionRadius.
func (d *Door) CollisionRadius() int {
	r := min(doorAbs(d.Nodes[0].X-d.Nodes[1].X), doorAbs(d.Nodes[0].Y-d.Nodes[1].Y))
	return max(r, DoorMinCollisionRadius)
}

// OpensBy reports whether the door opens by method, a DoorOpenBy* value.
func (d *Door) OpensBy(method int) bool { return d.OpenMethod&method == method }

// AutoCloses reports whether an opened door closes itself after CloseTime
// (Mobius startAutoCloseTask): it has a close time, does not open by time
// and belongs to no clan hall.
func (d *Door) AutoCloses() bool {
	return d.CloseTime > 0 && !d.OpensBy(DoorOpenByTime) && d.ClanHallID == 0
}

// The static object types (StaticObject.Type).
const (
	StaticTownMap   = 0
	StaticThrone    = 1
	StaticSignboard = 2
)

// StaticObject is one <object> of StaticObjects.xml.
type StaticObject struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Type is a Static* value: a town map shows the map on click, a throne
	// (or chair) can be sat on, a signboard shows its page.
	Type int `json:"type"`
	X    int `json:"x"`
	Y    int `json:"y"`
	Z    int `json:"z"`
	// Texture, MapX and MapY are the town map picture and the object's
	// place on it. An empty Texture is Mobius's "none".
	Texture string `json:"texture,omitempty"`
	MapX    int    `json:"mapX,omitempty"`
	MapY    int    `json:"mapY,omitempty"`
}

// The fence states (Fence.State), Mobius FenceState.
const (
	FenceHidden       = "HIDDEN"
	FenceOpened       = "OPENED"
	FenceClosed       = "CLOSED"
	FenceClosedHidden = "CLOSED_HIDDEN"
)

var fenceStates = setOf(FenceHidden, FenceOpened, FenceClosed, FenceClosedHidden)

// Fence is one <fence> of FenceData.xml.
type Fence struct {
	Name   string `json:"name"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Z      int    `json:"z"`
	Width  int    `json:"width"`
	Length int    `json:"length"`
	Height int    `json:"height"`
	// State is a Fence* value (Mobius default CLOSED).
	State string `json:"state"`
}

// FenceClientState returns the state number the client gets for a Fence*
// state (Mobius FenceState.getClientId).
func FenceClientState(state string) int {
	switch state {
	case FenceOpened:
		return 1
	case FenceClosed:
		return 2
	}
	return 0
}

// ObjectsData is the content of the world object files.
type ObjectsData struct {
	Doors         []Door         // DoorsFile, sorted by ID
	StaticObjects []StaticObject // StaticObjectsFile, sorted by ID
	Fences        []Fence        // FencesFile, in file order
}

// Objects is the checked world object data with its lookups. It must not
// be modified after NewObjects or LoadObjects returns, and is then safe for
// concurrent use; the pointers its lookups return point into it.
type Objects struct {
	ObjectsData

	doors  map[int]int      // door ID -> index in Doors
	groups map[string][]int // group name -> door IDs, sorted
}

// LoadObjects reads and checks the world object files in dir.
func LoadObjects(dir string) (*Objects, error) {
	o, err := LoadObjectsFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return o, nil
}

// LoadObjectsFS reads and checks the world object files in fsys.
func LoadObjectsFS(fsys fs.FS) (*Objects, error) {
	var d ObjectsData
	for _, f := range []struct {
		name string
		v    any
	}{
		{DoorsFile, &d.Doors},
		{StaticObjectsFile, &d.StaticObjects},
		{FencesFile, &d.Fences},
	} {
		if err := readJSON(fsys, f.name, f.v); err != nil {
			return nil, err
		}
	}
	return NewObjects(d)
}

// NewObjects checks the world objects and indexes them. The slices are used
// as they are. Doors: IDs positive, unique and sorted; a name; a positive
// height and HP; known open methods and master actions; non-negative times,
// and an open time only on doors that open by time; a child door that
// exists. Static objects: IDs unique and sorted, a known type. Fences: a
// known state and a positive size.
func NewObjects(d ObjectsData) (*Objects, error) {
	o := &Objects{
		ObjectsData: d,
		doors:       make(map[int]int, len(d.Doors)),
		groups:      make(map[string][]int),
	}
	var p problems
	for i := range d.Doors {
		o.checkDoor(&p, i)
	}
	for i := range d.Doors {
		dr := &d.Doors[i]
		if _, ok := o.doors[dr.ChildID]; dr.ChildID != 0 && !ok {
			p.addf("%s: door %d: child door %d does not exist", DoorsFile, dr.ID, dr.ChildID)
		}
	}
	for i, s := range d.StaticObjects {
		where := fmt.Sprintf("%s: static object %d", StaticObjectsFile, s.ID)
		if i > 0 && d.StaticObjects[i-1].ID >= s.ID {
			p.addf("%s: IDs are not sorted and unique", where)
		}
		if s.Type < StaticTownMap || s.Type > StaticSignboard {
			p.addf("%s: unknown type %d", where, s.Type)
		}
		if s.Name == "" {
			p.addf("%s: no name", where)
		}
	}
	for i, f := range d.Fences {
		where := fmt.Sprintf("%s: fence %d (%s)", FencesFile, i, f.Name)
		if !fenceStates[f.State] {
			p.addf("%s: unknown state %q", where, f.State)
		}
		if f.Width <= 0 || f.Length <= 0 || f.Height <= 0 {
			p.addf("%s: size %dx%dx%d must be positive", where, f.Width, f.Length, f.Height)
		}
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return o, nil
}

func (o *Objects) checkDoor(p *problems, i int) {
	d := &o.Doors[i]
	where := fmt.Sprintf("%s: door %d", DoorsFile, d.ID)
	if d.ID <= 0 {
		p.addf("%s: ID must be positive", where)
	}
	if _, dup := o.doors[d.ID]; dup {
		p.addf("%s: duplicate ID", where)
		return
	}
	o.doors[d.ID] = i
	if i > 0 && o.Doors[i-1].ID > d.ID {
		p.addf("%s: doors are not sorted by ID", where)
	}
	if d.Name == "" {
		p.addf("%s: no name", where)
	}
	if d.Height <= 0 {
		p.addf("%s: height %d must be positive", where, d.Height)
	}
	if d.HP <= 0 {
		p.addf("%s: HP %v must be positive", where, d.HP)
	}
	if d.PDef < 0 || d.MDef < 0 || d.Level < 0 {
		p.addf("%s: negative defence or level", where)
	}
	if d.OpenMethod&^doorOpenMethods != 0 || d.OpenMethod < 0 {
		p.addf("%s: unknown open method bits %d", where, d.OpenMethod)
	}
	for _, a := range []DoorAction{d.MasterOpen, d.MasterClose} {
		if a != "" && a != DoorActOpen && a != DoorActClose {
			p.addf("%s: unknown master action %q", where, a)
		}
	}
	if d.CloseTime < 0 || d.OpenTime < 0 || d.RandomTime < 0 {
		p.addf("%s: negative time", where)
	}
	if !d.OpensBy(DoorOpenByTime) && (d.OpenTime != 0 || d.RandomTime != 0) {
		p.addf("%s: open or random time on a door that does not open by time", where)
	}
	if d.ChildID == d.ID {
		p.addf("%s: the door is its own child", where)
	}
	if d.Group != "" {
		o.groups[d.Group] = append(o.groups[d.Group], d.ID)
	}
}

// Door returns the door with the given ID.
func (o *Objects) Door(id int) (*Door, bool) {
	i, ok := o.doors[id]
	if !ok {
		return nil, false
	}
	return &o.Doors[i], true
}

// DoorGroup returns the IDs of the doors of a group, sorted, or nil. The
// slice must not be modified.
func (o *Objects) DoorGroup(name string) []int { return o.groups[name] }

// DoorGroups returns the group names, sorted.
func (o *Objects) DoorGroups() []string {
	names := make([]string, 0, len(o.groups))
	for name := range o.groups {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// StaticObject returns the static object with the given ID.
func (o *Objects) StaticObject(id int) (*StaticObject, bool) {
	i, ok := slices.BinarySearchFunc(o.StaticObjects, id, func(s StaticObject, id int) int { return cmp.Compare(s.ID, id) })
	if !ok {
		return nil, false
	}
	return &o.StaticObjects[i], true
}

func doorAbs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
