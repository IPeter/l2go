package testclient

import "context"

// The Seven Signs packets (Plan 14 M11l) in the Mobius Interlude layouts:
// RequestSSQStatus and the Record of the Seven Signs window, SSQStatus.

// Opcodes of the Seven Signs packets.
const (
	// Client: RequestSSQStatus.
	OpRequestSSQStatus byte = 0xc7
	// Server: SSQStatus.
	OpSSQStatus byte = 0xf5
)

// The decoder entry and name are added here rather than in the shared
// tables, so the Seven Signs packets stay in one file.
func init() {
	serverPackets[OpSSQStatus] = serverPacket{"SSQStatus", readSSQStatus}
	clientPacketNames[OpRequestSSQStatus] = "RequestSSQStatus"
}

// RequestSSQStatus asks for a page (1-4) of the Record of the Seven Signs;
// SSQStatus comes back (not page 4 in the results and seal validation
// periods).
func (c *GameConn) RequestSSQStatus(ctx context.Context, page int) error {
	return c.send(ctx, []byte{OpRequestSSQStatus, byte(page)}) //nolint:gosec // a page number
}

// SSQStatus is a page of the Record of the Seven Signs (0xF5). Only the
// fields of its Page are set.
type SSQStatus struct {
	Page, Period int8

	// Page 1: the cycle, the period's and its end's system messages, the
	// player's cabal, seal, stones and ancient adena, the cabals' scores.
	Cycle                   int32
	PeriodMsg, PeriodEndMsg int32
	Cabal, Seal             int8
	StonesTurnedIn          int32
	AncientAdenaToCollect   int32

	DuskStoneScore, DuskFestivalScore, DuskTotal int32
	DuskPercent                                  int8
	DawnStoneScore, DawnFestivalScore, DawnTotal int32
	DawnPercent                                  int8

	// Page 2: the festivals' best scores.
	Festivals []SSQFestival
	// Page 3: the vote shares a winner needs to keep (10) and claim (35) a
	// seal, and the seals' votes.
	KeepPercent, ClaimPercent int8
	Seals                     []SSQSealVote
	// Page 4: the predicted winner and the seals' predicted owners.
	Winner      int8
	Predictions []SSQSealPrediction
}

// SSQFestival is a festival of page 2.
type SSQFestival struct {
	ID          int8
	LevelScore  int32
	DuskScore   int32
	DuskMembers []string
	DawnScore   int32
	DawnMembers []string
}

// SSQSealVote is a seal of page 3.
type SSQSealVote struct {
	Seal, Owner, DuskPercent, DawnPercent int8
}

// SSQSealPrediction is a seal of page 4: the predicted owner and the system
// message explaining it.
type SSQSealPrediction struct {
	Seal, Owner int8
	Msg         int16
}

// PacketName implements Packet.
func (SSQStatus) PacketName() string { return "SSQStatus" }

func readSSQStatus(f *fields) Packet {
	var p SSQStatus
	p.Page, p.Period = f.i8(), f.i8()
	switch p.Page {
	case 1:
		p.Cycle, p.PeriodMsg, p.PeriodEndMsg = f.i32(), f.i32(), f.i32()
		p.Cabal, p.Seal = f.i8(), f.i8()
		p.StonesTurnedIn, p.AncientAdenaToCollect = f.i32(), f.i32()
		p.DuskStoneScore, p.DuskFestivalScore, p.DuskTotal, p.DuskPercent = f.i32(), f.i32(), f.i32(), f.i8()
		p.DawnStoneScore, p.DawnFestivalScore, p.DawnTotal, p.DawnPercent = f.i32(), f.i32(), f.i32(), f.i8()
	case 2:
		if v := f.u16(); f.ok() && v != 1 {
			f.failf("festival page starts with %d, want 1", v)
		}
		n := f.count8()
		for i := 0; i < n && f.ok(); i++ {
			var fe SSQFestival
			fe.ID, fe.LevelScore, fe.DuskScore = f.i8(), f.i32(), f.i32()
			fe.DuskMembers = f.names8()
			fe.DawnScore = f.i32()
			fe.DawnMembers = f.names8()
			p.Festivals = append(p.Festivals, fe)
		}
	case 3:
		p.KeepPercent, p.ClaimPercent = f.i8(), f.i8()
		n := f.count8()
		for i := 0; i < n && f.ok(); i++ {
			p.Seals = append(p.Seals, SSQSealVote{Seal: f.i8(), Owner: f.i8(), DuskPercent: f.i8(), DawnPercent: f.i8()})
		}
	case 4:
		p.Winner = f.i8()
		n := f.count8()
		for i := 0; i < n && f.ok(); i++ {
			p.Predictions = append(p.Predictions, SSQSealPrediction{Seal: f.i8(), Owner: f.i8(), Msg: f.i16()})
		}
	}
	return p
}

// names8 reads c count and that many names.
func (f *fields) names8() []string {
	n := f.count8()
	out := make([]string, 0, n)
	for i := 0; i < n && f.ok(); i++ {
		out = append(out, f.str())
	}
	return out
}
