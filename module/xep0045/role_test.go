package xep0045

import (
	"testing"
	"github.com/stretchr/testify/require"
)

func TestXEP0045_Role(t *testing.T) {
	moderator := RoleModerator
	participant := RoleParticipant
	visitor := RoleVisitor
	none := RoleNone

	// Present in Room
	require.Equal(t, true, moderator.PresentInRoom())
	require.Equal(t, true, participant.PresentInRoom())
	require.Equal(t, true, visitor.PresentInRoom())
	require.Equal(t, false, none.PresentInRoom())

	// Receive Messages
	require.Equal(t, true, moderator.ReceiveMessages())
	require.Equal(t, true, participant.ReceiveMessages())
	require.Equal(t, true, visitor.ReceiveMessages())
	require.Equal(t, false, none.ReceiveMessages())

	// Receive Occupant Presence
	require.Equal(t, true, moderator.ReceiveOccupantPresence())
	require.Equal(t, true, participant.ReceiveOccupantPresence())
	require.Equal(t, true, visitor.ReceiveOccupantPresence())
	require.Equal(t, false, none.ReceiveOccupantPresence())

	// Broadcast Presence to All Occupants
	require.Equal(t, true, moderator.BroadcastPresenceToAllOccupants())
	require.Equal(t, true, participant.BroadcastPresenceToAllOccupants())
	require.Equal(t, true, visitor.BroadcastPresenceToAllOccupants())
	require.Equal(t, false, none.BroadcastPresenceToAllOccupants())

	// Change Availability Status
	require.Equal(t, true, moderator.ChangeAvailabilityStatus())
	require.Equal(t, true, participant.ChangeAvailabilityStatus())
	require.Equal(t, true, visitor.ChangeAvailabilityStatus())
	require.Equal(t, false, none.ChangeAvailabilityStatus())

	// Change Room Nickname
	require.Equal(t, true, moderator.ChangeRoomNickname())
	require.Equal(t, true, participant.ChangeRoomNickname())
	require.Equal(t, true, visitor.ChangeRoomNickname())
	require.Equal(t, false, none.ChangeRoomNickname())

	// Send Private Messages
	require.Equal(t, true, moderator.SendPrivateMessages())
	require.Equal(t, true, participant.SendPrivateMessages())
	require.Equal(t, true, visitor.SendPrivateMessages())
	require.Equal(t, false, none.SendPrivateMessages())

	// Invite Other Users
	require.Equal(t, true, moderator.InviteOtherUsers())
	require.Equal(t, true, participant.InviteOtherUsers())
	require.Equal(t, true, visitor.InviteOtherUsers())
	require.Equal(t, false, none.InviteOtherUsers())

	// Send Messages to All
	require.Equal(t, true, moderator.SendMessagesToAll())
	require.Equal(t, true, participant.SendMessagesToAll())
	require.Equal(t, false, visitor.SendMessagesToAll())
	require.Equal(t, false, none.SendMessagesToAll())

	// Send Messages to All
	require.Equal(t, true, moderator.SendMessagesToAll())
	require.Equal(t, true, participant.SendMessagesToAll())
	require.Equal(t, false, visitor.SendMessagesToAll())
	require.Equal(t, false, none.SendMessagesToAll())

	// Modify Subject
	require.Equal(t, true, moderator.ModifySubject())
	require.Equal(t, true, participant.ModifySubject())
	require.Equal(t, false, visitor.ModifySubject())
	require.Equal(t, false, none.ModifySubject())

	// Kick Participants and Visitors
	require.Equal(t, true, moderator.KickParticipantsAndVisitors())
	require.Equal(t, false, participant.KickParticipantsAndVisitors())
	require.Equal(t, false, visitor.KickParticipantsAndVisitors())
	require.Equal(t, false, none.KickParticipantsAndVisitors())

	// Grant Voice
	require.Equal(t, true, moderator.GrantVoice())
	require.Equal(t, false, participant.GrantVoice())
	require.Equal(t, false, visitor.GrantVoice())
	require.Equal(t, false, none.GrantVoice())

	// Revoke Voice
	require.Equal(t, true, moderator.RevokeVoice())
	require.Equal(t, false, participant.RevokeVoice())
	require.Equal(t, false, visitor.RevokeVoice())
	require.Equal(t, false, none.RevokeVoice())

}
