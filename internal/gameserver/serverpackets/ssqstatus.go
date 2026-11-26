package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpSSQStatus is SSQStatus, the Record of the Seven Signs window (Mobius
// ServerPackets.SSQ_STATUS).
const OpSSQStatus = 0xf5

// SSQStatus is a page of the Record of the Seven Signs (Plan 14 M11l). The
// caller computes every value as Mobius SSQStatus does; the packet only
// writes the fields of its Page:
//
//   - page 1: the cycle, the period and its end, the player's cabal, seal,
//     stones and ancient adena, and each cabal's scores;
//   - page 2: the festivals' best scores and their parties;
//   - page 3: the seal votes;
//   - page 4: the predicted winner and the seals' predicted owners.
type SSQStatus struct {
	Page, Period int8

	// Page 1.
	Cycle                   int32
	PeriodMsg, PeriodEndMsg int32 // system message IDs
	Cabal, Seal             int8
	StonesTurnedIn          int32
	AncientAdenaToCollect   int32

	DuskStoneScore, DuskFestivalScore, DuskTotal int32
	DuskPercent                                  int8
	DawnStoneScore, DawnFestivalScore, DawnTotal int32
	DawnPercent                                  int8

	// Page 2.
	Festivals []SSQFestival
	// Page 3.
	Seals []SSQSealVote
	// Page 4.
	Winner      int8
	Predictions []SSQSealPrediction
}

// SSQFestival is a festival of page 2: its client-side ID (1-5), the score
// it is worth, and each cabal's best score with its party's names.
type SSQFestival struct {
	ID          int8
	LevelScore  int32
	DuskScore   int32
	DuskMembers []string
	DawnScore   int32
	DawnMembers []string
}

// SSQSealVote is a seal of page 3: its owner and the share of each cabal's
// members who chose it, in percent.
type SSQSealVote struct {
	Seal, Owner, DuskPercent, DawnPercent int8
}

// SSQSealPrediction is a seal of page 4: its predicted owner and the system
// message that explains it.
type SSQSealPrediction struct {
	Seal, Owner int8
	Msg         int16
}

// The fixed values Mobius writes on page 3: the vote share a winner needs
// to keep a seal and to claim one.
const (
	ssqKeepSealPercent  = 10
	ssqClaimSealPercent = 35
)

// NewSSQStatusPacket builds SSQStatus (Mobius SSQStatus.writeImpl):
//
//	c 0xf5, c page, c period, then
//	page 1: d cycle, d period msg, d period end msg, c cabal, c seal, d stones, d ancient adena,
//	        d dusk stone score, d dusk festival score, d dusk total, c dusk %,
//	        d dawn stone score, d dawn festival score, d dawn total, c dawn %;
//	page 2: h 1, c count, per festival c id, d level score, d dusk score, c n, n S names,
//	        d dawn score, c n, n S names;
//	page 3: c 10, c 35, c count, per seal c seal, c owner, c dusk %, c dawn %;
//	page 4: c winner, c count, per seal c seal, c owner, h message.
//
// Any other page has the header only.
func NewSSQStatusPacket(s SSQStatus) []byte {
	w := packets.NewWriter(64)
	w.WriteUint8(OpSSQStatus)
	putInt8(w, s.Page)
	putInt8(w, s.Period)
	switch s.Page {
	case 1:
		w.WriteInt32(s.Cycle)
		w.WriteInt32(s.PeriodMsg)
		w.WriteInt32(s.PeriodEndMsg)
		putInt8(w, s.Cabal)
		putInt8(w, s.Seal)
		w.WriteInt32(s.StonesTurnedIn)
		w.WriteInt32(s.AncientAdenaToCollect)
		w.WriteInt32(s.DuskStoneScore)
		w.WriteInt32(s.DuskFestivalScore)
		w.WriteInt32(s.DuskTotal)
		putInt8(w, s.DuskPercent)
		w.WriteInt32(s.DawnStoneScore)
		w.WriteInt32(s.DawnFestivalScore)
		w.WriteInt32(s.DawnTotal)
		putInt8(w, s.DawnPercent)
	case 2:
		w.WriteUint16(1)
		w.WriteUint8(uint8(len(s.Festivals))) //nolint:gosec // five festivals
		for _, f := range s.Festivals {
			putInt8(w, f.ID)
			w.WriteInt32(f.LevelScore)
			w.WriteInt32(f.DuskScore)
			putNames8(w, f.DuskMembers)
			w.WriteInt32(f.DawnScore)
			putNames8(w, f.DawnMembers)
		}
	case 3:
		w.WriteUint8(ssqKeepSealPercent)
		w.WriteUint8(ssqClaimSealPercent)
		w.WriteUint8(uint8(len(s.Seals))) //nolint:gosec // three seals
		for _, v := range s.Seals {
			putInt8(w, v.Seal)
			putInt8(w, v.Owner)
			putInt8(w, v.DuskPercent)
			putInt8(w, v.DawnPercent)
		}
	case 4:
		putInt8(w, s.Winner)
		w.WriteUint8(uint8(len(s.Predictions))) //nolint:gosec // three seals
		for _, p := range s.Predictions {
			putInt8(w, p.Seal)
			putInt8(w, p.Owner)
			w.WriteUint16(uint16(p.Msg)) //nolint:gosec // same bits
		}
	}
	return w.Bytes()
}

// putNames8 writes c count and the names (Mobius writeByte(length) then
// writeString each).
func putNames8(w *packets.Writer, names []string) {
	w.WriteUint8(uint8(len(names))) //nolint:gosec // a party
	for _, n := range names {
		w.WriteString(n)
	}
}
