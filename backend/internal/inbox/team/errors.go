package team

import "errors"

var (
	// ErrTeamInboxNotFound is returned when a team inbox does not exist.
	ErrTeamInboxNotFound = errors.New("team inbox not found")

	// ErrNotTeamAdmin is returned when a non-admin attempts an admin-only operation.
	ErrNotTeamAdmin = errors.New("only team inbox admins can perform this operation")

	// ErrNotTeamMember is returned when a non-member tries to access a team inbox.
	ErrNotTeamMember = errors.New("user is not a member of this team inbox")

	// ErrAlreadyMember is returned when trying to add a user who is already a member.
	ErrAlreadyMember = errors.New("user is already a member of this team inbox")

	// ErrCannotRemoveLastAdmin is returned when trying to remove the last admin from a team inbox.
	ErrCannotRemoveLastAdmin = errors.New("cannot remove the last admin from a team inbox")

	// ErrManualClaimInRoundRobin is returned when trying to manually claim in round-robin mode.
	ErrManualClaimInRoundRobin = errors.New("manual claiming is not allowed in round-robin assignment mode")
)
