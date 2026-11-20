package gameserver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IPeter/l2go/internal/game"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
)

// day is the unit of World.DeleteCharAfterDays.
const day = 24 * time.Hour

// sendCharacterList sends the account's character list (CharSelectionInfo)
// and remembers which character is in which slot, for CharacterDelete,
// CharacterRestore and CharacterSelect.
//
// Characters whose delete time has passed are deleted first and left out,
// as Mobius CharSelectionInfo does: there is no background job for it. The
// others show the seconds left until their deletion.
func (s *Server) sendCharacterList(ctx context.Context, c *client) error {
	a := c.auth.Load()
	if a == nil {
		return errors.New("character list before AuthLogin")
	}
	chars, err := s.chars.ListByAccount(ctx, a.account)
	if err != nil {
		return fmt.Errorf("list characters: %w", err)
	}
	now := s.now()
	kept := chars[:0]
	for _, ch := range chars {
		if ch.DeleteExpired(now) {
			s.purge(ctx, c, ch)
			continue
		}
		kept = append(kept, ch)
	}
	ids := make([]int32, len(kept))
	for i, ch := range kept {
		ids[i] = ch.ObjectID
	}
	c.setSlots(ids)
	c.Send(serverpackets.NewCharSelectionInfoPacket(a.account, a.keys.PlayOK1, kept, now))
	return nil
}

// purge deletes a character whose delete time has passed. If that fails,
// the character is still left out of the list, as in Mobius, and the next
// list tries again.
func (s *Server) purge(ctx context.Context, c *client, ch storage.Character) {
	logger := c.log().With("name", ch.Name, "objectId", ch.ObjectID, "deleteTime", ch.DeleteTime)
	switch err := s.chars.Delete(ctx, ch.ObjectID); {
	case err == nil:
		logger.Info("character deleted: its delete time has passed")
	case errors.Is(err, storage.ErrNotFound):
		// Another connection of the account deleted it first.
	default:
		logger.Error("cannot delete a character whose delete time has passed", "err", err)
	}
}

// characterDelete handles CharacterDelete (Mobius CharacterDelete and
// GameClient.markToDeleteChar): the character is marked for deletion in
// DeleteCharAfterDays days, or deleted at once when that is 0. Success gets
// CharDeleteSuccess. The list follows in every case; an empty slot or a
// storage error gets only the list, as in Mobius. A clan member or leader
// gets CharDeleteFail with its reason instead (markToDeleteChar answers 1
// and 2), when the game knows the clans (game.ClanDeletionChecker).
func (s *Server) characterDelete(ctx context.Context, c *client, data []byte) error {
	req, err := clientpackets.ParseCharacterDelete(data)
	if err != nil {
		s.hackAttempts.Add(1)
		return err
	}
	id, ok := c.slotObjectID(req.Slot)
	switch {
	case !ok:
		c.log().Info("CharacterDelete for an empty slot", "slot", req.Slot)
	case s.refuseClanDeletion(ctx, c, id):
	case s.markForDeletion(ctx, c, id):
		c.Send(serverpackets.NewCharDeleteSuccessPacket())
	}
	return s.sendCharacterList(ctx, c)
}

// refuseClanDeletion sends CharDeleteFail and reports true when the
// character is a clan member or leader. On a storage error it reports true
// without a packet: nothing is deleted, as with Mobius's answer -1.
func (s *Server) refuseClanDeletion(ctx context.Context, c *client, id int32) bool {
	checker, ok := s.game.(game.ClanDeletionChecker)
	if !ok {
		return false
	}
	ch, err := s.chars.Get(ctx, id)
	if err != nil {
		c.log().Warn("cannot read a character to delete", "objectId", id, "err", err)
		return true
	}
	answer, err := checker.CharacterClanDeletion(ctx, id, ch.ClanID)
	switch {
	case err != nil:
		c.log().Warn("cannot check the clan of a character to delete", "objectId", id, "err", err)
		return true
	case answer == game.ClanDeletionMember:
		c.Send(serverpackets.NewCharDeleteFailPacket(serverpackets.CharDeleteFailClanMember))
		return true
	case answer == game.ClanDeletionLeader:
		c.Send(serverpackets.NewCharDeleteFailPacket(serverpackets.CharDeleteFailClanLeader))
		return true
	}
	return false
}

// markForDeletion sets the delete time of character id, or deletes it when
// DeleteCharAfterDays is 0. It reports whether that worked.
func (s *Server) markForDeletion(ctx context.Context, c *client, id int32) bool {
	logger := c.log().With("objectId", id)
	days := s.gs.World.DeleteCharAfterDays
	if days == 0 {
		if err := s.chars.Delete(ctx, id); err != nil {
			logger.Warn("character deletion failed", "err", err)
			return false
		}
		logger.Info("character deleted")
		return true
	}
	at := s.now().Add(time.Duration(days) * day)
	if err := s.chars.SetDeleteTime(ctx, id, at); err != nil {
		logger.Warn("cannot mark a character for deletion", "err", err)
		return false
	}
	logger.Info("character marked for deletion", "deleteTime", at)
	return true
}

// characterRestore handles CharacterRestore (Mobius CharacterRestore and
// GameClient.restore): the delete time is cleared. The list follows in
// every case; it is the only answer.
func (s *Server) characterRestore(ctx context.Context, c *client, data []byte) error {
	req, err := clientpackets.ParseCharacterRestore(data)
	if err != nil {
		s.hackAttempts.Add(1)
		return err
	}
	id, ok := c.slotObjectID(req.Slot)
	if !ok {
		c.log().Info("CharacterRestore for an empty slot", "slot", req.Slot)
	} else if err := s.chars.SetDeleteTime(ctx, id, time.Time{}); err != nil {
		c.log().Warn("cannot restore a character", "objectId", id, "err", err)
	} else {
		c.log().Info("character restored", "objectId", id)
	}
	return s.sendCharacterList(ctx, c)
}

// characterSelect hands CharacterSelect to the game, which loads the
// character, sends CharSelected and sets StateEntering. As in Mobius, a
// slot that is not in the last list gets no answer, and neither does a
// character the game cannot load.
func (s *Server) characterSelect(ctx context.Context, c *client, data []byte) error {
	req, err := clientpackets.ParseCharacterSelect(data)
	if err != nil {
		s.hackAttempts.Add(1)
		return err
	}
	if _, ok := c.slotObjectID(req.Slot); !ok {
		c.log().Info("CharacterSelect for an empty slot, ignored", "slot", req.Slot)
		return nil
	}
	c.selected = true
	if err := s.game.SelectCharacter(ctx, c, int(req.Slot)); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		c.log().Warn("character selection failed", "slot", req.Slot, "err", err)
	}
	return nil
}
