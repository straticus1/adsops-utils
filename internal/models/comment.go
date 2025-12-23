package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Comment represents a comment on a ticket
type Comment struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	TicketID       uuid.UUID       `db:"ticket_id" json:"ticket_id"`
	OrganizationID uuid.UUID       `db:"organization_id" json:"organization_id"`
	AuthorID       uuid.UUID       `db:"author_id" json:"author_id"`
	Comment        string          `db:"comment" json:"comment"`
	IsInternal     bool            `db:"is_internal" json:"is_internal"`
	MentionedUsers []uuid.UUID     `db:"mentioned_users" json:"mentioned_users,omitempty"`
	AttachmentURLs []string        `db:"attachment_urls" json:"attachment_urls,omitempty"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updated_at"`
	DeletedAt      *time.Time      `db:"deleted_at" json:"deleted_at,omitempty"`
	Edited         bool            `db:"edited" json:"edited"`
	EditHistory    json.RawMessage `db:"edit_history" json:"edit_history,omitempty"`

	// Relationships
	Author *UserSummary `db:"-" json:"author,omitempty"`
}

// CanEdit checks if a user can edit this comment
func (c *Comment) CanEdit(userID uuid.UUID) bool {
	// Only author can edit, and only within 15 minutes
	if c.AuthorID != userID {
		return false
	}
	editWindow := c.CreatedAt.Add(15 * time.Minute)
	return time.Now().Before(editWindow)
}

// CanDelete checks if a user can delete this comment
func (c *Comment) CanDelete(userID uuid.UUID, isAdmin bool) bool {
	return c.AuthorID == userID || isAdmin
}

// CreateCommentInput represents input for creating a comment
type CreateCommentInput struct {
	Comment        string      `json:"comment" validate:"required,min=1"`
	IsInternal     bool        `json:"is_internal"`
	MentionedUsers []uuid.UUID `json:"mentioned_users,omitempty"`
	AttachmentURLs []string    `json:"attachment_urls,omitempty"`
}

// UpdateCommentInput represents input for updating a comment
type UpdateCommentInput struct {
	Comment string `json:"comment" validate:"required,min=1"`
}

// CommentEditEntry represents an entry in the edit history
type CommentEditEntry struct {
	PreviousComment string    `json:"previous_comment"`
	EditedAt        time.Time `json:"edited_at"`
}
