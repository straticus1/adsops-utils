package models

import (
	"encoding/json"
	"net"
	"time"

	"github.com/google/uuid"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID                   uuid.UUID             `db:"id" json:"id"`
	OrganizationID       uuid.UUID             `db:"organization_id" json:"organization_id"`
	UserID               *uuid.UUID            `db:"user_id" json:"user_id,omitempty"`
	Username             *string               `db:"username" json:"username,omitempty"`
	Action               string                `db:"action" json:"action"`
	ResourceType         string                `db:"resource_type" json:"resource_type"`
	ResourceID           *uuid.UUID            `db:"resource_id" json:"resource_id,omitempty"`
	Description          string                `db:"description" json:"description"`
	Changes              json.RawMessage       `db:"changes" json:"changes,omitempty"`
	Metadata             json.RawMessage       `db:"metadata" json:"metadata,omitempty"`
	IPAddress            *net.IP               `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent            *string               `db:"user_agent" json:"user_agent,omitempty"`
	SessionID            *string               `db:"session_id" json:"session_id,omitempty"`
	ComplianceRelevant   bool                  `db:"compliance_relevant" json:"compliance_relevant"`
	ComplianceFrameworks []ComplianceFramework `db:"compliance_frameworks" json:"compliance_frameworks,omitempty"`
	CreatedAt            time.Time             `db:"created_at" json:"created_at"`
	Anonymized           bool                  `db:"anonymized" json:"anonymized"`
	AnonymizedAt         *time.Time            `db:"anonymized_at" json:"anonymized_at,omitempty"`
}

// AuditAction constants
const (
	AuditActionCreate          = "create"
	AuditActionUpdate          = "update"
	AuditActionDelete          = "delete"
	AuditActionView            = "view"
	AuditActionApprove         = "approve"
	AuditActionDeny            = "deny"
	AuditActionRequestUpdate   = "request_update"
	AuditActionSubmit          = "submit"
	AuditActionCancel          = "cancel"
	AuditActionClose           = "close"
	AuditActionReopen          = "reopen"
	AuditActionLogin           = "login"
	AuditActionLogout          = "logout"
	AuditActionLoginFailed     = "login_failed"
	AuditActionPasswordChange  = "password_change"
	AuditActionMFAEnable       = "mfa_enable"
	AuditActionMFADisable      = "mfa_disable"
	AuditActionExport          = "export"
	AuditActionDownload        = "download"
)

// AuditResourceType constants
const (
	AuditResourceTicket       = "ticket"
	AuditResourceApproval     = "approval"
	AuditResourceComment      = "comment"
	AuditResourceUser         = "user"
	AuditResourceOrganization = "organization"
	AuditResourceSession      = "session"
	AuditResourceReport       = "report"
)

// AuditChanges represents before/after changes
type AuditChanges struct {
	Before json.RawMessage `json:"before,omitempty"`
	After  json.RawMessage `json:"after,omitempty"`
}

// CreateAuditLogInput represents input for creating an audit log entry
type CreateAuditLogInput struct {
	UserID               *uuid.UUID
	Username             *string
	Action               string
	ResourceType         string
	ResourceID           *uuid.UUID
	Description          string
	Changes              json.RawMessage
	Metadata             json.RawMessage
	IPAddress            *net.IP
	UserAgent            *string
	SessionID            *string
	ComplianceRelevant   bool
	ComplianceFrameworks []ComplianceFramework
}

// AuditLogFilter represents filter options for querying audit logs
type AuditLogFilter struct {
	UserID               *uuid.UUID            `json:"user_id,omitempty"`
	Action               []string              `json:"action,omitempty"`
	ResourceType         []string              `json:"resource_type,omitempty"`
	ResourceID           *uuid.UUID            `json:"resource_id,omitempty"`
	ComplianceRelevant   *bool                 `json:"compliance_relevant,omitempty"`
	ComplianceFrameworks []ComplianceFramework `json:"compliance_frameworks,omitempty"`
	FromDate             *time.Time            `json:"from_date,omitempty"`
	ToDate               *time.Time            `json:"to_date,omitempty"`
	Search               string                `json:"search,omitempty"`
	Page                 int                   `json:"page" validate:"min=1"`
	PerPage              int                   `json:"per_page" validate:"min=1,max=1000"`
}

// SetDefaults sets default values for the filter
func (f *AuditLogFilter) SetDefaults() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 1000 {
		f.PerPage = 100
	}
}

// Offset returns the offset for pagination
func (f *AuditLogFilter) Offset() int {
	return (f.Page - 1) * f.PerPage
}

// NotificationQueue represents a pending notification
type NotificationQueue struct {
	ID               uuid.UUID  `db:"id" json:"id"`
	OrganizationID   uuid.UUID  `db:"organization_id" json:"organization_id"`
	UserID           *uuid.UUID `db:"user_id" json:"user_id,omitempty"`
	Email            string     `db:"email" json:"email"`
	NotificationType string     `db:"notification_type" json:"notification_type"`
	Subject          string     `db:"subject" json:"subject"`
	BodyHTML         string     `db:"body_html" json:"-"`
	BodyText         string     `db:"body_text" json:"-"`
	TicketID         *uuid.UUID `db:"ticket_id" json:"ticket_id,omitempty"`
	ApprovalID       *uuid.UUID `db:"approval_id" json:"approval_id,omitempty"`
	Status           string     `db:"status" json:"status"`
	Attempts         int        `db:"attempts" json:"attempts"`
	MaxAttempts      int        `db:"max_attempts" json:"max_attempts"`
	SentAt           *time.Time `db:"sent_at" json:"sent_at,omitempty"`
	FailedAt         *time.Time `db:"failed_at" json:"failed_at,omitempty"`
	ErrorMessage     *string    `db:"error_message" json:"error_message,omitempty"`
	SESMessageID     *string    `db:"ses_message_id" json:"ses_message_id,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	ScheduledFor     time.Time  `db:"scheduled_for" json:"scheduled_for"`
}

// NotificationStatus constants
const (
	NotificationStatusPending = "pending"
	NotificationStatusSent    = "sent"
	NotificationStatusFailed  = "failed"
	NotificationStatusBounced = "bounced"
)

// NotificationType constants
const (
	NotificationTypeApprovalRequest  = "approval_request"
	NotificationTypeApprovalDecision = "approval_decision"
	NotificationTypeApprovalReminder = "approval_reminder"
	NotificationTypeTicketCreated    = "ticket_created"
	NotificationTypeTicketUpdated    = "ticket_updated"
	NotificationTypeCommentAdded     = "comment_added"
	NotificationTypeMention          = "mention"
)
