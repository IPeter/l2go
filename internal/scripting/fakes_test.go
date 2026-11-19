package scripting

// fakePlayer is a PlayerRef for tests. onCall, if set, runs in every
// method.
type fakePlayer struct {
	id      int32
	name    string
	level   int
	classID int
	race    int
	onCall  func()
}

func (p *fakePlayer) touch() {
	if p.onCall != nil {
		p.onCall()
	}
}

func (p *fakePlayer) ObjectID() int32 { p.touch(); return p.id }
func (p *fakePlayer) Name() string    { p.touch(); return p.name }
func (p *fakePlayer) Level() int      { p.touch(); return p.level }
func (p *fakePlayer) ClassID() int    { p.touch(); return p.classID }
func (p *fakePlayer) Race() int       { p.touch(); return p.race }

// fakeNpc is an NpcRef for tests.
type fakeNpc struct {
	id    int32 // object ID
	npcID int   // template ID
	name  string
}

func (n *fakeNpc) ObjectID() int32 { return n.id }
func (n *fakeNpc) ID() int         { return n.npcID }
func (n *fakeNpc) Name() string    { return n.name }

// speech is one Output.NpcSay call.
type speech struct {
	npc  NpcRef
	text string
}

// fakeOutput records NpcSay calls. onSay, if set, runs inside NpcSay, on
// the engine's goroutine.
type fakeOutput struct {
	said  []speech
	onSay func(npc NpcRef, text string)
}

func (o *fakeOutput) NpcSay(npc NpcRef, text string) {
	o.said = append(o.said, speech{npc: npc, text: text})
	if o.onSay != nil {
		o.onSay(npc, text)
	}
}

// hero is a level 5 Human Fighter.
func hero() *fakePlayer {
	return &fakePlayer{id: 0x10000001, name: "Hero", level: 5}
}

// guardNpc is a Talking Island guard.
func guardNpc() *fakeNpc {
	return &fakeNpc{id: 0x60000001, npcID: 30733, name: "Guard"}
}
