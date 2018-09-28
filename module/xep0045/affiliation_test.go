package xep0045

import (
	"testing"
	"github.com/stretchr/testify/require"
)

func TestXEP0045_Affiliation(t *testing.T) {
	owner := AffiliationOwner
	admin := AffiliationAdmin
	member := AffiliationMember
	none := AffiliationNone
	outcast := AffiliationOutcast

	// Enter Open Room
	require.Equal(t, true, owner.EnterOpenRoom())
	require.Equal(t, true, admin.EnterOpenRoom())
	require.Equal(t, true, member.EnterOpenRoom())
	require.Equal(t, true, none.EnterOpenRoom())
	require.Equal(t, false, outcast.EnterOpenRoom())

	// Register with Open Room
	require.Equal(t, true, owner.RegisterWithOpenRoom())
	require.Equal(t, true, admin.RegisterWithOpenRoom())
	require.Equal(t, true, member.RegisterWithOpenRoom())
	require.Equal(t, true, none.RegisterWithOpenRoom())
	require.Equal(t, false, outcast.RegisterWithOpenRoom())

	// Retrieve Member List
	require.Equal(t, true, owner.RetrieveMemberList())
	require.Equal(t, true, admin.RetrieveMemberList())
	require.Equal(t, true, member.RetrieveMemberList())
	require.Equal(t, false, none.RetrieveMemberList())
	require.Equal(t, false, outcast.RetrieveMemberList())

	// Enter Members-Only Room
	require.Equal(t, true, owner.EnterMembersOnlyRoom())
	require.Equal(t, true, admin.EnterMembersOnlyRoom())
	require.Equal(t, true, member.EnterMembersOnlyRoom())
	require.Equal(t, false, none.EnterMembersOnlyRoom())
	require.Equal(t, false, outcast.EnterMembersOnlyRoom())

	// Ban Members and Unaffiliated Users
	require.Equal(t, true, owner.BanMembersAndUnaffiliatedUsers())
	require.Equal(t, true, admin.BanMembersAndUnaffiliatedUsers())
	require.Equal(t, false, member.BanMembersAndUnaffiliatedUsers())
	require.Equal(t, false, none.BanMembersAndUnaffiliatedUsers())
	require.Equal(t, false, outcast.BanMembersAndUnaffiliatedUsers())

	// Edit Member List
	require.Equal(t, true, owner.EditMemberList())
	require.Equal(t, true, admin.EditMemberList())
	require.Equal(t, false, member.EditMemberList())
	require.Equal(t, false, none.EditMemberList())
	require.Equal(t, false, outcast.EditMemberList())

	// Assign and Remove Moderator Role
	require.Equal(t, true, owner.EditModeratorList())
	require.Equal(t, true, admin.EditModeratorList())
	require.Equal(t, false, member.EditModeratorList())
	require.Equal(t, false, none.EditModeratorList())
	require.Equal(t, false, outcast.EditModeratorList())

	// Edit Admin List
	require.Equal(t, true, owner.EditAdminList())
	require.Equal(t, false, admin.EditAdminList())
	require.Equal(t, false, member.EditAdminList())
	require.Equal(t, false, none.EditAdminList())
	require.Equal(t, false, outcast.EditAdminList())

	// Edit Owner List
	require.Equal(t, true, owner.EditOwnerList())
	require.Equal(t, false, admin.EditOwnerList())
	require.Equal(t, false, member.EditOwnerList())
	require.Equal(t, false, none.EditOwnerList())
	require.Equal(t, false, outcast.EditOwnerList())

	// Change Room Configuration
	require.Equal(t, true, owner.ChangeRoomDefinition())
	require.Equal(t, false, admin.ChangeRoomDefinition())
	require.Equal(t, false, member.ChangeRoomDefinition())
	require.Equal(t, false, none.ChangeRoomDefinition())
	require.Equal(t, false, outcast.ChangeRoomDefinition())

	// Destroy Room
	require.Equal(t, true, owner.DestroyRoom())
	require.Equal(t, false, admin.DestroyRoom())
	require.Equal(t, false, member.DestroyRoom())
	require.Equal(t, false, none.DestroyRoom())
	require.Equal(t, false, outcast.DestroyRoom())

}
