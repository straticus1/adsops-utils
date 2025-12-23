package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Ticket represents a change management ticket
type Ticket struct {
	ID                           uuid.UUID             `db:"id" json:"id"`
	OrganizationID               uuid.UUID             `db:"organization_id" json:"organization_id"`
	TicketNumber                 string                `db:"ticket_number" json:"ticket_number"`
	CreatedBy                    uuid.UUID             `db:"created_by" json:"created_by"`
	AssignedTo                   *uuid.UUID            `db:"assigned_to" json:"assigned_to,omitempty"`
	Title                        string                `db:"title" json:"title"`
	Description                  string                `db:"description" json:"description"`
	Status                       TicketStatus          `db:"status" json:"status"`
	Priority                     TicketPriority        `db:"priority" json:"priority"`
	RiskLevel                    RiskLevel             `db:"risk_level" json:"risk_level"`
	Industry                     IndustryType          `db:"industry" json:"industry"`
	ComplianceFrameworks         []ComplianceFramework `db:"compliance_frameworks" json:"compliance_frameworks"`
	ComplianceNotes              *string               `db:"compliance_notes" json:"compliance_notes,omitempty"`
	ChangeType                   *string               `db:"change_type" json:"change_type,omitempty"`
	AffectedSystems              []string              `db:"affected_systems" json:"affected_systems,omitempty"`
	AffectedDataTypes            []string              `db:"affected_data_types" json:"affected_data_types,omitempty"`
	ImpactDescription            *string               `db:"impact_description" json:"impact_description,omitempty"`
	RollbackPlan                 *string               `db:"rollback_plan" json:"rollback_plan,omitempty"`
	TestingPlan                  *string               `db:"testing_plan" json:"testing_plan,omitempty"`
	RequestedImplementationDate  *time.Time            `db:"requested_implementation_date" json:"requested_implementation_date,omitempty"`
	ScheduledStart               *time.Time            `db:"scheduled_start" json:"scheduled_start,omitempty"`
	ScheduledEnd                 *time.Time            `db:"scheduled_end" json:"scheduled_end,omitempty"`
	ActualStart                  *time.Time            `db:"actual_start" json:"actual_start,omitempty"`
	ActualEnd                    *time.Time            `db:"actual_end" json:"actual_end,omitempty"`
	RequiresApprovalTypes        []ApprovalType        `db:"requires_approval_types" json:"requires_approval_types"`
	ApprovalDeadline             *time.Time            `db:"approval_deadline" json:"approval_deadline,omitempty"`
	AttachmentURLs               []string              `db:"attachment_urls" json:"attachment_urls,omitempty"`
	CustomFields                 json.RawMessage       `db:"custom_fields" json:"custom_fields,omitempty"`
	SubmittedAt                  *time.Time            `db:"submitted_at" json:"submitted_at,omitempty"`
	SubmittedSnapshot            json.RawMessage       `db:"submitted_snapshot" json:"submitted_snapshot,omitempty"`
	Version                      int                   `db:"version" json:"version"`
	CreatedAt                    time.Time             `db:"created_at" json:"created_at"`
	UpdatedAt                    time.Time             `db:"updated_at" json:"updated_at"`
	ClosedAt                     *time.Time            `db:"closed_at" json:"closed_at,omitempty"`
	DeletedAt                    *time.Time            `db:"deleted_at" json:"deleted_at,omitempty"`
	DeletionReason               *string               `db:"deletion_reason" json:"deletion_reason,omitempty"`

	// Relationships (populated via joins)
	Creator      *UserSummary `db:"-" json:"creator,omitempty"`
	Assignee     *UserSummary `db:"-" json:"assignee,omitempty"`
	Approvals    []Approval   `db:"-" json:"approvals,omitempty"`
	Comments     []Comment    `db:"-" json:"comments,omitempty"`
}

// IsDraft returns true if the ticket is in draft status
func (t *Ticket) IsDraft() bool {
	return t.Status == TicketStatusDraft
}

// IsSubmitted returns true if the ticket has been submitted
func (t *Ticket) IsSubmitted() bool {
	return t.SubmittedAt != nil
}

// CanEdit returns true if the ticket can be edited
func (t *Ticket) CanEdit() bool {
	return t.Status == TicketStatusDraft || t.Status == TicketStatusUpdateRequested
}

// CanSubmit returns true if the ticket can be submitted
func (t *Ticket) CanSubmit() bool {
	return t.Status == TicketStatusDraft || t.Status == TicketStatusUpdateRequested
}

// CanCancel returns true if the ticket can be cancelled
func (t *Ticket) CanCancel() bool {
	return t.Status == TicketStatusDraft || t.Status == TicketStatusSubmitted
}

// CanClose returns true if the ticket can be closed
func (t *Ticket) CanClose() bool {
	return t.Status == TicketStatusCompleted
}

// CanReopen returns true if the ticket can be reopened
func (t *Ticket) CanReopen() bool {
	return t.Status == TicketStatusClosed
}

// TicketSummary represents a minimal ticket for list views
type TicketSummary struct {
	ID           uuid.UUID      `json:"id"`
	TicketNumber string         `json:"ticket_number"`
	Title        string         `json:"title"`
	Status       TicketStatus   `json:"status"`
	Priority     TicketPriority `json:"priority"`
	RiskLevel    RiskLevel      `json:"risk_level"`
	CreatedBy    UserSummary    `json:"created_by"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// ToSummary converts a Ticket to TicketSummary
func (t *Ticket) ToSummary() TicketSummary {
	summary := TicketSummary{
		ID:           t.ID,
		TicketNumber: t.TicketNumber,
		Title:        t.Title,
		Status:       t.Status,
		Priority:     t.Priority,
		RiskLevel:    t.RiskLevel,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
	if t.Creator != nil {
		summary.CreatedBy = *t.Creator
	}
	return summary
}

// CreateTicketInput represents input for creating a ticket
type CreateTicketInput struct {
	Title                       string                `json:"title" validate:"required,min=5,max=500"`
	Description                 string                `json:"description" validate:"required,min=10"`
	Priority                    TicketPriority        `json:"priority" validate:"required"`
	RiskLevel                   RiskLevel             `json:"risk_level" validate:"required"`
	Industry                    IndustryType          `json:"industry" validate:"required"`
	ComplianceFrameworks        []ComplianceFramework `json:"compliance_frameworks" validate:"required,min=1,dive"`
	ComplianceNotes             *string               `json:"compliance_notes,omitempty"`
	ChangeType                  *string               `json:"change_type,omitempty"`
	AffectedSystems             []string              `json:"affected_systems,omitempty"`
	AffectedDataTypes           []string              `json:"affected_data_types,omitempty"`
	ImpactDescription           *string               `json:"impact_description,omitempty"`
	RollbackPlan                *string               `json:"rollback_plan,omitempty"`
	TestingPlan                 *string               `json:"testing_plan,omitempty"`
	RequestedImplementationDate *time.Time            `json:"requested_implementation_date,omitempty"`
	RequiresApprovalTypes       []ApprovalType        `json:"requires_approval_types" validate:"required,min=1,dive"`
	ApprovalDeadline            *time.Time            `json:"approval_deadline,omitempty"`
	CustomFields                json.RawMessage       `json:"custom_fields,omitempty"`
	Submit                      bool                  `json:"submit"` // true = submit immediately, false = save as draft
}

// UpdateTicketInput represents input for updating a ticket
type UpdateTicketInput struct {
	Title                       *string               `json:"title,omitempty" validate:"omitempty,min=5,max=500"`
	Description                 *string               `json:"description,omitempty" validate:"omitempty,min=10"`
	Priority                    *TicketPriority       `json:"priority,omitempty"`
	RiskLevel                   *RiskLevel            `json:"risk_level,omitempty"`
	ComplianceFrameworks        []ComplianceFramework `json:"compliance_frameworks,omitempty" validate:"omitempty,min=1,dive"`
	ComplianceNotes             *string               `json:"compliance_notes,omitempty"`
	ChangeType                  *string               `json:"change_type,omitempty"`
	AffectedSystems             []string              `json:"affected_systems,omitempty"`
	AffectedDataTypes           []string              `json:"affected_data_types,omitempty"`
	ImpactDescription           *string               `json:"impact_description,omitempty"`
	RollbackPlan                *string               `json:"rollback_plan,omitempty"`
	TestingPlan                 *string               `json:"testing_plan,omitempty"`
	RequestedImplementationDate *time.Time            `json:"requested_implementation_date,omitempty"`
	ScheduledStart              *time.Time            `json:"scheduled_start,omitempty"`
	ScheduledEnd                *time.Time            `json:"scheduled_end,omitempty"`
	RequiresApprovalTypes       []ApprovalType        `json:"requires_approval_types,omitempty" validate:"omitempty,min=1,dive"`
	ApprovalDeadline            *time.Time            `json:"approval_deadline,omitempty"`
	CustomFields                json.RawMessage       `json:"custom_fields,omitempty"`
	AssignedTo                  *uuid.UUID            `json:"assigned_to,omitempty"`
}

// TicketRevision represents a change history entry for a ticket
type TicketRevision struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	TicketID       uuid.UUID       `db:"ticket_id" json:"ticket_id"`
	OrganizationID uuid.UUID       `db:"organization_id" json:"organization_id"`
	RevisionNumber int             `db:"revision_number" json:"revision_number"`
	ChangedBy      uuid.UUID       `db:"changed_by" json:"changed_by"`
	ChangeReason   *string         `db:"change_reason" json:"change_reason,omitempty"`
	Changes        json.RawMessage `db:"changes" json:"changes"`
	TicketSnapshot json.RawMessage `db:"ticket_snapshot" json:"ticket_snapshot"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	IPAddress      *string         `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent      *string         `db:"user_agent" json:"user_agent,omitempty"`

	// Relationships
	ChangedByUser *UserSummary `db:"-" json:"changed_by_user,omitempty"`
}

// TicketListFilter represents filter options for listing tickets
type TicketListFilter struct {
	Status              []TicketStatus        `json:"status,omitempty"`
	Priority            []TicketPriority      `json:"priority,omitempty"`
	RiskLevel           []RiskLevel           `json:"risk_level,omitempty"`
	CreatedBy           *uuid.UUID            `json:"created_by,omitempty"`
	AssignedTo          *uuid.UUID            `json:"assigned_to,omitempty"`
	ComplianceFramework []ComplianceFramework `json:"compliance_framework,omitempty"`
	FromDate            *time.Time            `json:"from_date,omitempty"`
	ToDate              *time.Time            `json:"to_date,omitempty"`
	Search              string                `json:"search,omitempty"`
	Page                int                   `json:"page" validate:"min=1"`
	PerPage             int                   `json:"per_page" validate:"min=1,max=100"`
	SortBy              string                `json:"sort_by,omitempty"`
	SortOrder           string                `json:"sort_order,omitempty"` // asc or desc
}

// SetDefaults sets default values for the filter
func (f *TicketListFilter) SetDefaults() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 50
	}
	if f.SortBy == "" {
		f.SortBy = "created_at"
	}
	if f.SortOrder == "" {
		f.SortOrder = "desc"
	}
}

// Offset returns the offset for pagination
func (f *TicketListFilter) Offset() int {
	return (f.Page - 1) * f.PerPage
}
