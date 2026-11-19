// Package game runs the game world on the world loop: players and NPCs in
// the world, entering and leaving it, movement, chat, NPCs, dialogs and the
// rest of the in-world game logic.
//
// The network/session layer (internal/gameserver) talks to this package
// only through the Session and Service interfaces of contract.go. World
// state is owned by the world loop (internal/world, decision W1); code that
// touches it must run on the loop: in a packet handler, a timer, a tick
// hook, or a function given to Server.OnLoop.
//
// # The core
//
// Server (server.go) owns the loop, the world, the stat rules and the
// registries. Player (player.go) is a selected character: it embeds
// world.Base and implements world.Viewer and scripting.PlayerRef.
//
//   - Entering (enter.go): SelectCharacter loads a character off the loop,
//     kicks another session of the account (decision E7) and sends
//     CharSelected; EnterWorld sends Mobius's enter-world sequence
//     (decision E1, see enterWorld).
//   - Leaving (leave.go): Restart, Logout, Disconnected and the double-login
//     kick save the player and take it out of the world.
//   - Visibility (info.go): AddObject, RemoveObject and MoveObject change
//     the world and send what changed (SendDelta): DeleteObject to those
//     that lost sight, and the InfoFor packets of an InfoProvider to those
//     that gained it. UserInfo and CharInfo refreshes are coalesced per tick
//     (Player.BroadcastUserInfo, RefreshStats).
//   - Persistence (persist.go): Player.Snapshot on the loop, written by a
//     background saver in order (decision W13); autosave every
//     Settings.AutosaveInterval.
//   - Teleport (teleport.go): Server.Teleport and the Appearing round trip.
//   - Stats (adapter.go): the data turned into stat engine input.
//
// # Adding a feature
//
// A feature (movement, chat, NPCs, dialogs, zones, ...) lives in its own
// files of this package. It registers a setup function from init, which New
// runs after the core, in name order:
//
//	func init() { registerFeature("chat", setupChat) }
//
//	func setupChat(s *Server) error {
//		return errors.Join(
//			s.Register(clientpackets.OpSay2, s.handleSay2),
//			s.RegisterGMCommand("announce", storage.AccessLevelAdmin, s.gmAnnounce),
//		)
//	}
//
// The registries, all filled in setup:
//
//   - Register and RegisterEx: a Handler per client opcode or Ex sub-opcode.
//     Handlers run on the loop for players in the game (also while they
//     teleport), with the packet after its opcode; parse it with
//     internal/gameserver/clientpackets.
//   - RegisterGMCommand: "//name args" commands (SendBypassBuildCmd) with a
//     minimum access level; RunGMCommand gives Mobius's answers for unknown
//     commands and missing rights.
//   - RegisterUserCommand: "/loc" and friends (BypassUserCmd) by number.
//
// The hooks: OnPlayerEnter, OnPlayerLeave, OnPlayerMoved, OnForget (a pair
// of objects lost sight of each other: drop targets), OnTeleport (stop
// movement, drop the target) and OnTick. Features offer some of their own:
// OnPlayerArrived (movement: a walk reached its destination), OnNpcInteract
// (NPCs: a player talks to an NPC), and the RegisterActionUse registry
// (movement: the action window). Hooks run in the order they were added, so
// features run theirs in name order: on a teleport the movement feature's
// StopMove comes before the NPC feature's TargetUnselected, as in Mobius,
// and on a tick the walks advance before the walks up to NPCs are checked.
//
// Game objects other than players (NPCs) embed world.Base, implement
// InfoProvider (NpcInfo, plus MoveToLocation while walking) and, for
// coalesced refreshes, UpdateFlusher; they enter and leave the world with
// AddObject and RemoveObject, and move with MoveObject. Runtime object IDs
// come from World().IDs(). Data: Data (players, items), WorldData (NPCs,
// spawns, teleporters, zones, map regions), Settings (chat ranges, NPC
// interaction range, ...), Geo.
//
// Player keeps the plain state the features own: Target, Move (MoveState),
// Zones, HTML (HTMLCache, the dialog link cache), LastNpc, Sitting,
// CurHP/CurMP/CurCP. Running is part of the stat set (SetRunning). After
// changing what UserInfo shows, call RefreshStats or BroadcastUserInfo.
package game
