package models

import (
	"net"
	"time"

	"github.com/google/uuid"
)

// Approval represents an approval record for a ticket
type Approval struct {
	ID                 uuid.UUID      `db:"id" json:"id"`
	TicketID           uuid.UUID      `db:"ticket_id" json:"ticket_id"`
	OrganizationID     uuid.UUID      `db:"organization_id" json:"organization_id"`
	ApprovalType       ApprovalType   `db:"approval_type" json:"approval_type"`
	SequenceOrder      int            `db:"sequence_order" json:"sequence_order"`
	ApproverID         uuid.UUID      `db:"approver_id" json:"approver_id"`
	DelegatedFrom      *uuid.UUID     `db:"delegated_from" json:"delegated_from,omitempty"`
	Status             ApprovalStatus `db:"status" json:"status"`
	ApprovedAt         *time.Time     `db:"approved_at" json:"approved_at,omitempty"`
	DeniedAt           *time.Time     `db:"denied_at" json:"denied_at,omitempty"`
	DecisionComment    *string        `db:"decision_comment" json:"decision_comment,omitempty"`
	Conditions         *string        `db:"conditions" json:"conditions,omitempty"`
	ApprovalToken      *string        `db:"approval_token" json:"-"`
	TokenExpiresAt     *time.Time     `db:"token_expires_at" json:"token_expires_at,omitempty"`
	ApprovalIP         *net.IP        `db:"approval_ip" json:"approval_ip,omitempty"`
	ApprovalUserAgent  *string        `db:"approval_user_agent" json:"-"`
	NotificationSentAt *time.Time     `db:"notification_sent_at" json:"notification_sent_at,omitempty"`
	NotificationReadAt *time.Time     `db:"notification_read_at" json:"notification_read_at,omitempty"`
	ReminderSentAt     *time.Time     `db:"reminder_sent_at" json:"reminder_sent_at,omitempty"`
	CreatedAt          time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time      `db:"updated_at" json:"updated_at"`

	// Relationships (populated via joins)
	Approver      *UserSummary   `db:"-" json:"approver,omitempty"`
	DelegatedUser *UserSummary   `db:"-" json:"delegated_user,omitempty"`
	Ticket        *TicketSummary `db:"-" json:"ticket,omitempty"`
}

// IsPending returns true if the approval is pending
func (a *Approval) IsPending() bool {
	return a.Status == ApprovalStatusPending
}

// IsDecided returns true if a decision has been made
func (a *Approval) IsDecided() bool {
	return a.Status == ApprovalStatusApproved || a.Status == ApprovalStatusDenied || a.Status == ApprovalStatusUpdateRequested
}

// IsTokenValid checks if the approval token is still valid
func (a *Approval) IsTokenValid() bool {
	if a.ApprovalToken == nil || a.TokenExpiresAt == nil {
		return false
	}
	return time.Now().Before(*a.TokenExpiresAt)
}

// CanApprove returns true if this approval can be approved
func (a *Approval) CanApprove() bool {
	return a.Status == ApprovalStatusPending
}

// ApprovalSummary represents a minimal approval for list views
type ApprovalSummary struct {
	ID             uuid.UUID      `json:"id"`
	TicketID       uuid.UUID      `json:"ticket_id"`
	TicketNumber   string         `json:"ticket_number"`
	TicketTitle    string         `json:"ticket_title"`
	ApprovalType   ApprovalType   `json:"approval_type"`
	Status         ApprovalStatus `json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
	TokenExpiresAt *time.Time     `json:"token_expires_at,omitempty"`
}

// ApproveInput represents input for approving a ticket
type ApproveInput struct {
	Comment    *string `json:"comment,omitempty"`
	Conditions *string `json:"conditions,omitempty"`
}

// DenyInput represents input for denying a ticket
type DenyInput struct {
	Comment string `json:"comment" validate:"required,min=10"`
	Reason  string `json:"reason" validate:"required,min=10"`
}

// RequestUpdateInput represents input for requesting an update
type RequestUpdateInput struct {
	Comment         string `json:"comment" validate:"required,min=10"`
	RequiredChanges string `json:"required_changes" validate:"required,min=10"`
}

// ApprovalListFilter represents filter options for listing approvals
type ApprovalListFilter struct {
	Status       []ApprovalStatus `json:"status,omitempty"`
	ApprovalType []ApprovalType   `json:"approval_type,omitempty"`
	TicketID     *uuid.UUID       `json:"ticket_id,omitempty"`
	ApproverID   *uuid.UUID       `json:"approver_id,omitempty"`
	Page         int              `json:"page" validate:"min=1"`
	PerPage      int              `json:"per_page" validate:"min=1,max=100"`
}

// SetDefaults sets default values for the filter
func (f *ApprovalListFilter) SetDefaults() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 50
	}
}

// Offset returns the offset for pagination
func (f *ApprovalListFilter) Offset() int {
	return (f.Page - 1) * f.PerPage
}

// PendingApprovalView represents a pending approval with full context
type PendingApprovalView struct {
	Approval
	TicketNumber string         `db:"ticket_number" json:"ticket_number"`
	TicketTitle  string         `db:"ticket_title" json:"ticket_title"`
	Priority     TicketPriority `db:"priority" json:"priority"`
	RiskLevel    RiskLevel      `db:"risk_level" json:"risk_level"`
	ApproverName string         `db:"approver_name" json:"approver_name"`
	ApproverEmail string        `db:"approver_email" json:"approver_email"`
}
