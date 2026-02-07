package models

import (
	"time"

	"github.com/google/uuid"
)

// ChannelRole represents a user's role in a channel
type ChannelRole string

const (
	ChannelRoleOwner  ChannelRole = "owner"
	ChannelRoleAdmin  ChannelRole = "admin"
	ChannelRoleMember ChannelRole = "member"
)

// ValidChannelRoles for validation
var ValidChannelRoles = map[ChannelRole]bool{
	ChannelRoleOwner:  true,
	ChannelRoleAdmin:  true,
	ChannelRoleMember: true,
}

// IsValid checks if the channel role is valid
func (cr ChannelRole) IsValid() bool {
	return ValidChannelRoles[cr]
}

// String returns the string representation of the channel role
func (cr ChannelRole) String() string {
	return string(cr)
}

// CanModerate returns true if the role can moderate the channel (owner or admin)
func (cr ChannelRole) CanModerate() bool {
	return cr == ChannelRoleOwner || cr == ChannelRoleAdmin
}

// Channel represents a chat channel
type Channel struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	IsPrivate   bool       `json:"is_private"`
	IsDM        bool       `json:"is_dm"`
	IsArchived  bool       `json:"is_archived"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	DMUser1     *uuid.UUID `json:"dm_user1,omitempty"`
	DMUser2     *uuid.UUID `json:"dm_user2,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ChannelWithRelations includes membership and denormalized data for API responses
type ChannelWithRelations struct {
	Channel
	MemberCount int          `json:"member_count"`
	MyRole      *ChannelRole `json:"my_role,omitempty"`
	LastMessage *Message     `json:"last_message,omitempty"`
	UnreadCount int          `json:"unread_count"`
}

// ChannelMembership represents a user's membership in a channel
type ChannelMembership struct {
	ChannelID  uuid.UUID   `json:"channel_id"`
	UserID     uuid.UUID   `json:"user_id"`
	Role       ChannelRole `json:"role"`
	JoinedAt   time.Time   `json:"joined_at"`
	LastReadAt *time.Time  `json:"last_read_at,omitempty"`
}

// ChannelMember represents a channel member with user details
type ChannelMember struct {
	ChannelMembership
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}
