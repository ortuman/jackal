package xep0045

type roleType string

const (
	RoleModerator   = roleType("moderator")
	RoleNone        = roleType("none")
	RoleParticipant = roleType("participant")
	RoleVisitor     = roleType("visitor")
)

type rolePrivileges struct {
	presentInRoom                   bool
	receiveMessages                 bool
	receiveOccupantPresence         bool
	broadcastPresenceToAllOccupants bool
	changeAvailabilityStatus        bool
	changeRoomNickname              bool
	sendPrivateMessages             bool
	inviteOtherUsers                bool
	sendMessagesToAll               bool
	modifySubject                   bool
	kickParticipantsAndVisitors     bool
	grantVoice                      bool
	revokeVoice                     bool
}

var roles struct {
	moderator   rolePrivileges
	none        rolePrivileges
	participant rolePrivileges
	visitor     rolePrivileges
}

func init() {
	roles.moderator = newRolePrivileges(true, true, true, true, true, true, true, true, true, true, true, true, true)
	roles.participant = newRolePrivileges(true, true, true, true, true, true, true, true, true, true, false, false, false)
	roles.visitor = newRolePrivileges(true, true, true, true, true, true, true, true, false, false, false, false, false)
	roles.none = newRolePrivileges(false, false, false, false, false, false, false, false, false, false, false, false, false)
}

func newRolePrivileges(
	presentInRoom,
	receiveMessages,
	receiveOccupantPresence,
	broadcastPresenceToAllOccupants,
	changeAvailabilityStatus,
	changeRoomNickname,
	sendPrivateMessages,
	inviteOtherUsers,
	sendMessagesToAll,
	modifySubject,
	kickParticipantsAndVisitors,
	grantVoice,
	revokeVoice bool) rolePrivileges {
	x := rolePrivileges{
		presentInRoom:                   presentInRoom,
		receiveMessages:                 receiveMessages,
		receiveOccupantPresence:         receiveOccupantPresence,
		broadcastPresenceToAllOccupants: broadcastPresenceToAllOccupants,
		changeAvailabilityStatus:        changeAvailabilityStatus,
		changeRoomNickname:              changeRoomNickname,
		sendPrivateMessages:             sendPrivateMessages,
		inviteOtherUsers:                inviteOtherUsers,
		kickParticipantsAndVisitors:     kickParticipantsAndVisitors,
		modifySubject:                   modifySubject,
		sendMessagesToAll:               sendMessagesToAll,
		grantVoice:                      grantVoice,
		revokeVoice:                     revokeVoice,
	}

	return x
}

func (x roleType) String() string {
	return string(x)
}

func (x roleType) privileges() rolePrivileges {
	switch x {
	case RoleModerator:
		return roles.moderator
	case RoleNone:
		return roles.none
	case RoleParticipant:
		return roles.participant
	case RoleVisitor:
		return roles.visitor
	}
	return roles.none
}

func (x roleType) ChangeAvailabilityStatus() bool {
	return x.privileges().changeAvailabilityStatus
}

func (x roleType) ChangeRoomNickname() bool {
	return x.privileges().changeRoomNickname
}

func (x roleType) GrantVoice() bool {
	return x.privileges().grantVoice
}

func (x roleType) InviteOtherUsers() bool {
	return x.privileges().inviteOtherUsers
}

func (x roleType) KickParticipantsAndVisitors() bool {
	return x.privileges().kickParticipantsAndVisitors
}

func (x roleType) ModifySubject() bool {
	return x.privileges().modifySubject
}

func (x roleType) BroadcastPresenceToAllOccupants() bool {
	return x.privileges().broadcastPresenceToAllOccupants
}

func (x roleType) PresentInRoom() bool {
	return x.privileges().presentInRoom
}

func (x roleType) ReceiveMessages() bool {
	return x.privileges().receiveMessages
}

func (x roleType) ReceiveOccupantPresence() bool {
	return x.privileges().receiveOccupantPresence
}

func (x roleType) RevokeVoice() bool {
	return x.privileges().revokeVoice
}

func (x roleType) SendMessagesToAll() bool {
	return x.privileges().sendMessagesToAll
}

func (x roleType) SendPrivateMessages() bool {
	return x.privileges().sendPrivateMessages
}
