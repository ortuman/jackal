package xep0045

type affiliationType string

const (
	AffiliationAdmin   = affiliationType("admin")
	AffiliationMember  = affiliationType("member")
	AffiliationNone    = affiliationType("none")
	AffiliationOutcast = affiliationType("outcast")
	AffiliationOwner   = affiliationType("owner")
)

type affiliationPrivileges struct {
	enterOpenRoom                  bool
	registerWithOpenRoom           bool
	retrieveMemberList             bool
	enterMembersOnlyRoom           bool
	banMembersAndUnaffiliatedUsers bool
	editMemberList                 bool
	editModeratorList              bool
	editAdminList                  bool
	editOwnerList                  bool
	changeRoomDefinition           bool
	destroyRoom                    bool
}

var affiliations struct {
	admin   affiliationPrivileges
	member  affiliationPrivileges
	none    affiliationPrivileges
	outcast affiliationPrivileges
	owner   affiliationPrivileges
}

func init() {
	affiliations.admin = newAffiliationPrivileges(true, true, true, true, true, true, true, false, false, false, false)
	affiliations.member = newAffiliationPrivileges(true, true, true, true, false, false, false, false, false, false, false)
	affiliations.none = newAffiliationPrivileges(true, true, false, false, false, false, false, false, false, false, false)
	affiliations.outcast = newAffiliationPrivileges(false, false, false, false, false, false, false, false, false, false, false)
	affiliations.owner = newAffiliationPrivileges(true, true, true, true, true, true, true, true, true, true, true)
}

func newAffiliationPrivileges(
	enterOpenRoom,
	registerWithOpenRoom,
	retrieveMemberList,
	enterMembersOnlyRoom,
	banMembersAndUnaffiliatedUsers,
	editMemberList,
	editModeratorList,
	editAdminList,
	editOwnerList,
	changeRoomDefinition,
	destroyRoom bool) affiliationPrivileges {
	x := affiliationPrivileges{
		banMembersAndUnaffiliatedUsers: banMembersAndUnaffiliatedUsers,
		changeRoomDefinition:           changeRoomDefinition,
		destroyRoom:                    destroyRoom,
		editAdminList:                  editAdminList,
		editMemberList:                 editMemberList,
		editModeratorList:              editModeratorList,
		editOwnerList:                  editOwnerList,
		enterMembersOnlyRoom:           enterMembersOnlyRoom,
		enterOpenRoom:                  enterOpenRoom,
		registerWithOpenRoom:           registerWithOpenRoom,
		retrieveMemberList:             retrieveMemberList,
	}
	return x
}

func (x affiliationType) String() string {
	return string(x)
}

func (x affiliationType) privileges() affiliationPrivileges {
	switch x {
	case AffiliationAdmin:
		return affiliations.admin
	case AffiliationMember:
		return affiliations.member
	case AffiliationNone:
		return affiliations.none
	case AffiliationOutcast:
		return affiliations.outcast
	case AffiliationOwner:
		return affiliations.owner
	}
	return affiliations.none
}

func (x affiliationType) BanMembersAndUnaffiliatedUsers() bool {
	return x.privileges().banMembersAndUnaffiliatedUsers
}

func (x affiliationType) ChangeRoomDefinition() bool {
	return x.privileges().changeRoomDefinition
}

func (x affiliationType) DestroyRoom() bool {
	return x.privileges().destroyRoom
}

func (x affiliationType) EditAdminList() bool {
	return x.privileges().editAdminList
}

func (x affiliationType) EditMemberList() bool {
	return x.privileges().editMemberList
}

func (x affiliationType) EditModeratorList() bool {
	return x.privileges().editModeratorList
}

func (x affiliationType) EditOwnerList() bool {
	return x.privileges().editOwnerList
}

func (x affiliationType) EnterMembersOnlyRoom() bool {
	return x.privileges().enterMembersOnlyRoom
}

func (x affiliationType) EnterOpenRoom() bool {
	return x.privileges().enterOpenRoom
}

func (x affiliationType) RegisterWithOpenRoom() bool {
	return x.privileges().registerWithOpenRoom
}

func (x affiliationType) RetrieveMemberList() bool {
	return x.privileges().retrieveMemberList
}
