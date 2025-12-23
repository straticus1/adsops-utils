package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Organization represents a tenant in the multi-tenant system
type Organization struct {
	ID                        uuid.UUID             `db:"id" json:"id"`
	Name                      string                `db:"name" json:"name"`
	Slug                      string                `db:"slug" json:"slug"`
	Industry                  IndustryType          `db:"industry" json:"industry"`
	ComplianceFrameworks      []ComplianceFramework `db:"compliance_frameworks" json:"compliance_frameworks"`
	CustomComplianceSpec      json.RawMessage       `db:"custom_compliance_spec" json:"custom_compliance_spec,omitempty"`
	PrimaryRegion             string                `db:"primary_region" json:"primary_region"`
	DataResidencyRequirements json.RawMessage       `db:"data_residency_requirements" json:"data_residency_requirements,omitempty"`
	RequireMFA                bool                  `db:"require_mfa" json:"require_mfa"`
	SessionTimeoutMinutes     int                   `db:"session_timeout_minutes" json:"session_timeout_minutes"`
	PasswordPolicy            json.RawMessage       `db:"password_policy" json:"password_policy,omitempty"`
	AdminEmail                string                `db:"admin_email" json:"admin_email"`
	SupportEmail              string                `db:"support_email" json:"support_email,omitempty"`
	CreatedAt                 time.Time             `db:"created_at" json:"created_at"`
	UpdatedAt                 time.Time             `db:"updated_at" json:"updated_at"`
	DeletedAt                 *time.Time            `db:"deleted_at" json:"deleted_at,omitempty"`
	CreatedBy                 *uuid.UUID            `db:"created_by" json:"created_by,omitempty"`
	UpdatedBy                 *uuid.UUID            `db:"updated_by" json:"updated_by,omitempty"`
}

// PasswordPolicyConfig represents password policy settings
type PasswordPolicyConfig struct {
	MinLength        int  `json:"min_length"`
	RequireUppercase bool `json:"require_uppercase"`
	RequireLowercase bool `json:"require_lowercase"`
	RequireNumbers   bool `json:"require_numbers"`
	RequireSymbols   bool `json:"require_symbols"`
	MaxAgeDays       int  `json:"max_age_days"`
	HistoryCount     int  `json:"history_count"` // Number of previous passwords to remember
}

// DefaultPasswordPolicy returns the default password policy
func DefaultPasswordPolicy() PasswordPolicyConfig {
	return PasswordPolicyConfig{
		MinLength:        12,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireNumbers:   true,
		RequireSymbols:   true,
		MaxAgeDays:       90,
		HistoryCount:     5,
	}
}

// CreateOrganizationInput represents input for creating an organization
type CreateOrganizationInput struct {
	Name                 string                `json:"name" validate:"required,min=2,max=255"`
	Slug                 string                `json:"slug" validate:"required,min=2,max=100,alphanum"`
	Industry             IndustryType          `json:"industry" validate:"required"`
	ComplianceFrameworks []ComplianceFramework `json:"compliance_frameworks" validate:"required,min=1,dive"`
	CustomComplianceSpec json.RawMessage       `json:"custom_compliance_spec,omitempty"`
	PrimaryRegion        string                `json:"primary_region" validate:"required"`
	RequireMFA           bool                  `json:"require_mfa"`
	AdminEmail           string                `json:"admin_email" validate:"required,email"`
	SupportEmail         string                `json:"support_email,omitempty" validate:"omitempty,email"`
}

// UpdateOrganizationInput represents input for updating an organization
type UpdateOrganizationInput struct {
	Name                 *string                `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Industry             *IndustryType          `json:"industry,omitempty"`
	ComplianceFrameworks []ComplianceFramework  `json:"compliance_frameworks,omitempty" validate:"omitempty,min=1,dive"`
	CustomComplianceSpec json.RawMessage        `json:"custom_compliance_spec,omitempty"`
	RequireMFA           *bool                  `json:"require_mfa,omitempty"`
	SessionTimeoutMinutes *int                  `json:"session_timeout_minutes,omitempty" validate:"omitempty,min=5,max=1440"`
	PasswordPolicy       *PasswordPolicyConfig  `json:"password_policy,omitempty"`
	SupportEmail         *string                `json:"support_email,omitempty" validate:"omitempty,email"`
}
