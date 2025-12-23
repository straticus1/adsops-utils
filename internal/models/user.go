package models

import (
	"encoding/json"
	"net"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID                    uuid.UUID      `db:"id" json:"id"`
	OrganizationID        uuid.UUID      `db:"organization_id" json:"organization_id"`
	Email                 string         `db:"email" json:"email"`
	Username              *string        `db:"username" json:"username,omitempty"`
	FullName              string         `db:"full_name" json:"full_name"`
	PasswordHash          *string        `db:"password_hash" json:"-"`
	RequirePasswordChange bool           `db:"require_password_change" json:"require_password_change"`
	MFAEnabled            bool           `db:"mfa_enabled" json:"mfa_enabled"`
	MFASecret             *string        `db:"mfa_secret" json:"-"`
	BackupCodes           []string       `db:"backup_codes" json:"-"`
	WebAuthnCredentials   json.RawMessage `db:"webauthn_credentials" json:"-"`
	OAuthProvider         *string        `db:"oauth_provider" json:"oauth_provider,omitempty"`
	OAuthSubject          *string        `db:"oauth_subject" json:"-"`
	Roles                 []UserRole     `db:"roles" json:"roles"`
	IsApprover            bool           `db:"is_approver" json:"is_approver"`
	ApprovalTypes         []ApprovalType `db:"approval_types" json:"approval_types,omitempty"`
	ApprovalDelegateID    *uuid.UUID     `db:"approval_delegate_id" json:"approval_delegate_id,omitempty"`
	IsActive              bool           `db:"is_active" json:"is_active"`
	EmailVerified         bool           `db:"email_verified" json:"email_verified"`
	LastLoginAt           *time.Time     `db:"last_login_at" json:"last_login_at,omitempty"`
	LastLoginIP           *net.IP        `db:"last_login_ip" json:"last_login_ip,omitempty"`
	AcceptsTermsVersion   *string        `db:"accepts_terms_version" json:"accepts_terms_version,omitempty"`
	AcceptsTermsAt        *time.Time     `db:"accepts_terms_at" json:"accepts_terms_at,omitempty"`
	DataRetentionConsent  bool           `db:"data_retention_consent" json:"data_retention_consent"`
	CreatedAt             time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time      `db:"updated_at" json:"updated_at"`
	DeletedAt             *time.Time     `db:"deleted_at" json:"deleted_at,omitempty"`
}

// HasRole checks if the user has a specific role
func (u *User) HasRole(role UserRole) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsAdmin checks if the user is an admin
func (u *User) IsAdmin() bool {
	return u.HasRole(UserRoleAdmin)
}

// CanApprove checks if the user can approve a specific approval type
func (u *User) CanApprove(approvalType ApprovalType) bool {
	if !u.IsApprover || !u.IsActive {
		return false
	}
	for _, at := range u.ApprovalTypes {
		if at == approvalType {
			return true
		}
	}
	return false
}

// UserSummary represents a minimal user response for API
type UserSummary struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
}

// ToSummary converts a User to UserSummary
func (u *User) ToSummary() UserSummary {
	return UserSummary{
		ID:       u.ID,
		Email:    u.Email,
		FullName: u.FullName,
	}
}

// CreateUserInput represents input for creating a user
type CreateUserInput struct {
	Email         string         `json:"email" validate:"required,email"`
	FullName      string         `json:"full_name" validate:"required,min=2,max=255"`
	Username      *string        `json:"username,omitempty" validate:"omitempty,min=3,max=100,alphanum"`
	Password      *string        `json:"password,omitempty" validate:"omitempty,min=12"`
	Roles         []UserRole     `json:"roles" validate:"required,min=1,dive"`
	IsApprover    bool           `json:"is_approver"`
	ApprovalTypes []ApprovalType `json:"approval_types,omitempty" validate:"omitempty,dive"`
}

// UpdateUserInput represents input for updating a user
type UpdateUserInput struct {
	FullName              *string        `json:"full_name,omitempty" validate:"omitempty,min=2,max=255"`
	Username              *string        `json:"username,omitempty" validate:"omitempty,min=3,max=100,alphanum"`
	Roles                 []UserRole     `json:"roles,omitempty" validate:"omitempty,min=1,dive"`
	IsApprover            *bool          `json:"is_approver,omitempty"`
	ApprovalTypes         []ApprovalType `json:"approval_types,omitempty" validate:"omitempty,dive"`
	ApprovalDelegateID    *uuid.UUID     `json:"approval_delegate_id,omitempty"`
	IsActive              *bool          `json:"is_active,omitempty"`
	RequirePasswordChange *bool          `json:"require_password_change,omitempty"`
}

// Session represents an authenticated session
type Session struct {
	ID                uuid.UUID  `db:"id" json:"id"`
	UserID            uuid.UUID  `db:"user_id" json:"user_id"`
	OrganizationID    uuid.UUID  `db:"organization_id" json:"organization_id"`
	SessionToken      string     `db:"session_token" json:"-"`
	RefreshToken      *string    `db:"refresh_token" json:"-"`
	IPAddress         *net.IP    `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent         *string    `db:"user_agent" json:"user_agent,omitempty"`
	DeviceFingerprint *string    `db:"device_fingerprint" json:"-"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	ExpiresAt         time.Time  `db:"expires_at" json:"expires_at"`
	LastActiveAt      time.Time  `db:"last_active_at" json:"last_active_at"`
	Revoked           bool       `db:"revoked" json:"revoked"`
	RevokedAt         *time.Time `db:"revoked_at" json:"revoked_at,omitempty"`
	RevokeReason      *string    `db:"revoke_reason" json:"revoke_reason,omitempty"`
}

// IsExpired checks if the session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid checks if the session is valid (not revoked and not expired)
func (s *Session) IsValid() bool {
	return !s.Revoked && !s.IsExpired()
}
