package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// The cinematic packets of the boss scripts (Plan 13, Phase 10.6):
// SpecialCamera and Earthquake. PlaySound and ExShowScreenMessage have
// their own files.

// Opcodes of the cinematic packets.
const (
	OpEarthquake    = 0xc4
	OpSpecialCamera = 0xc7
)

// SpecialCamera moves the client's camera around a creature (Mobius
// SpecialCamera). Range is the constructor argument Mobius never writes.
type SpecialCamera struct {
	ObjectID                                  int32
	Force, Angle1, Angle2, Time, Duration     int32
	RelYaw, RelPitch, Wide, RelAngle, Unknown int32
}

// NewSpecialCameraPacket builds SpecialCamera (Mobius
// SpecialCamera.writeImpl): c 0xc7; d object, force, angle1, angle2, time,
// duration, relYaw, relPitch, isWide, relAngle, unk.
func NewSpecialCameraPacket(c SpecialCamera) []byte {
	w := packets.NewWriter(1 + 11*4)
	w.WriteUint8(OpSpecialCamera)
	w.WriteInt32(c.ObjectID)
	w.WriteInt32(c.Force)
	w.WriteInt32(c.Angle1)
	w.WriteInt32(c.Angle2)
	w.WriteInt32(c.Time)
	w.WriteInt32(c.Duration)
	w.WriteInt32(c.RelYaw)
	w.WriteInt32(c.RelPitch)
	w.WriteInt32(c.Wide)
	w.WriteInt32(c.RelAngle)
	w.WriteInt32(c.Unknown)
	return w.Bytes()
}

// Earthquake shakes the screens around a point (Mobius Earthquake).
type Earthquake struct {
	X, Y, Z             int32
	Intensity, Duration int32
}

// NewEarthquakePacket builds Earthquake (Mobius Earthquake.writeImpl): c
// 0xc4; d x, y, z, intensity, duration, and a d Mobius always writes as 0.
func NewEarthquakePacket(e Earthquake) []byte {
	w := packets.NewWriter(1 + 6*4)
	w.WriteUint8(OpEarthquake)
	w.WriteInt32(e.X)
	w.WriteInt32(e.Y)
	w.WriteInt32(e.Z)
	w.WriteInt32(e.Intensity)
	w.WriteInt32(e.Duration)
	w.WriteInt32(0)
	return w.Bytes()
}
