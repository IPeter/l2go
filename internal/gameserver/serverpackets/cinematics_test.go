package serverpackets

import "testing"

// Mobius SpecialCamera.writeImpl: c 0xc7; d objectId, force, angle1,
// angle2, time, duration, relYaw, relPitch, isWide, relAngle, unk. Antharas's
// first camera: new SpecialCamera(npc, 700, 13, -19, 0, 10000, 20000, 0, 0,
// 0, 0, 0), whose range (10000) is not written.
func TestSpecialCameraPacket(t *testing.T) {
	checkPacket(t, "SpecialCamera",
		NewSpecialCameraPacket(SpecialCamera{ObjectID: 0x60000001, Force: 700, Angle1: 13, Angle2: -19, Duration: 20000}),
		unhex(t, "c7 01000060 bc020000 0d000000 edffffff 00000000 204e0000 00000000 00000000 00000000 00000000 00000000"))
	checkPacket(t, "SpecialCamera every field",
		NewSpecialCameraPacket(SpecialCamera{
			ObjectID: 1, Force: 2, Angle1: 3, Angle2: 4, Time: 5, Duration: 6,
			RelYaw: 7, RelPitch: 8, Wide: 9, RelAngle: 10, Unknown: 11,
		}),
		unhex(t, "c7 01000000 02000000 03000000 04000000 05000000 06000000 07000000 08000000 09000000 0a000000 0b000000"))
}

// Mobius Earthquake.writeImpl: c 0xc4; d x, y, z, intensity, duration, 0.
func TestEarthquakePacket(t *testing.T) {
	checkPacket(t, "Earthquake",
		NewEarthquakePacket(Earthquake{X: 185708, Y: 114298, Z: -8221, Intensity: 20, Duration: 10}),
		unhex(t, "c4 6cd50200 7abe0100 e3dfffff 14000000 0a000000 00000000"))
}
