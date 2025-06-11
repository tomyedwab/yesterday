package state

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/tomyedwab/yesterday/database/events"
	"github.com/tomyedwab/yesterday/wasi/guest"
)

// RuleType represents the type of access rule (ACCEPT or DENY)
type RuleType string

const (
	RuleTypeAccept RuleType = "ACCEPT"
	RuleTypeDeny   RuleType = "DENY"
)

// SubjectType represents the subject type of the rule (USER or GROUP)
type SubjectType string

const (
	SubjectTypeUser  SubjectType = "USER"
	SubjectTypeGroup SubjectType = "GROUP"
)

// Group constants
const (
	GroupAllUsers = "all_users"
)

// UserAccessRule represents a user access rule in the system.
type UserAccessRule struct {
	ID             int         `db:"id" json:"id"`
	ApplicationID  string      `db:"application_id" json:"applicationId"`
	RuleType       RuleType    `db:"rule_type" json:"ruleType"`
	SubjectType    SubjectType `db:"subject_type" json:"subjectType"`
	SubjectID      string      `db:"subject_id" json:"subjectId"` // Either user ID or group name
	CreatedAt      string      `db:"created_at" json:"createdAt"`
}

// Event types for user access rules management
const CreateUserAccessRuleEventType string = "CreateUserAccessRule"
const DeleteUserAccessRuleEventType string = "DeleteUserAccessRule"

// Event structures for user access rules management
type CreateUserAccessRuleEvent struct {
	events.GenericEvent
	ApplicationID string      `json:"applicationId"`
	RuleType      RuleType    `json:"ruleType"`
	SubjectType   SubjectType `json:"subjectType"`
	SubjectID     string      `json:"subjectId"`
}

type DeleteUserAccessRuleEvent struct {
	events.GenericEvent
	RuleID int `json:"ruleId"`
}

// -- Event handlers --

func UserAccessRulesHandleInitEvent(tx *sqlx.Tx, event *events.DBInitEvent) (bool, error) {
	// Create user_access_rules table
	_, err := tx.Exec(`
		CREATE TABLE user_access_rules_v1 (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			application_id TEXT NOT NULL,
			rule_type TEXT NOT NULL CHECK (rule_type IN ('ACCEPT', 'DENY')),
			subject_type TEXT NOT NULL CHECK (subject_type IN ('USER', 'GROUP')),
			subject_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (application_id) REFERENCES applications_v1(instance_id)
		)`)
	if err != nil {
		return false, fmt.Errorf("failed to create user_access_rules table: %w", err)
	}

	// Create index for efficient lookups
	_, err = tx.Exec(`
		CREATE INDEX idx_user_access_rules_lookup 
		ON user_access_rules_v1(application_id, subject_type, subject_id)`)
	if err != nil {
		return false, fmt.Errorf("failed to create user_access_rules index: %w", err)
	}

	// Create some default rules - allow admin user access to all applications
	_, err = tx.Exec(`
		INSERT INTO user_access_rules_v1 (application_id, rule_type, subject_type, subject_id)
		SELECT '3bf3e3c0-6e51-482a-b180-00f6aa568ee9', 'ACCEPT', 'USER', '1'
	`)
	if err != nil {
		return false, fmt.Errorf("failed to create default login service access rule: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO user_access_rules_v1 (application_id, rule_type, subject_type, subject_id)
		SELECT '18736e4f-93f9-4606-a7be-863c7986ea5b', 'ACCEPT', 'USER', '1'
	`)
	if err != nil {
		return false, fmt.Errorf("failed to create default admin service access rule: %w", err)
	}

	fmt.Println("User access rules tables initialized.")
	return true, nil
}

func UserAccessRulesHandleCreateEvent(tx *sqlx.Tx, event *CreateUserAccessRuleEvent) (bool, error) {
	guest.WriteLog(fmt.Sprintf("Creating user access rule for application: %s", event.ApplicationID))
	
	_, err := tx.Exec(`
		INSERT INTO user_access_rules_v1 (application_id, rule_type, subject_type, subject_id)
		VALUES ($1, $2, $3, $4)`,
		event.ApplicationID, string(event.RuleType), string(event.SubjectType), event.SubjectID)
	
	if err != nil {
		return false, fmt.Errorf("failed to create user access rule: %w", err)
	}
	
	return true, nil
}

func UserAccessRulesHandleDeleteEvent(tx *sqlx.Tx, event *DeleteUserAccessRuleEvent) (bool, error) {
	guest.WriteLog(fmt.Sprintf("Deleting user access rule ID: %d", event.RuleID))
	
	result, err := tx.Exec(`DELETE FROM user_access_rules_v1 WHERE id = $1`, event.RuleID)
	if err != nil {
		return false, fmt.Errorf("failed to delete user access rule %d: %w", event.RuleID, err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return false, fmt.Errorf("no access rule found with ID %d", event.RuleID)
	}
	
	return true, nil
}

// -- DB Helpers --

// GetUserAccessRulesForApplication retrieves all access rules for a specific application.
func GetUserAccessRulesForApplication(db *sqlx.DB, applicationID string) ([]UserAccessRule, error) {
	var rules []UserAccessRule
	err := db.Select(&rules, `
		SELECT id, application_id, rule_type, subject_type, subject_id, created_at 
		FROM user_access_rules_v1 
		WHERE application_id = $1
		ORDER BY subject_type DESC, rule_type ASC`, applicationID)
	return rules, err
}

// GetAllUserAccessRules retrieves all access rules, optionally filtered by application ID.
func GetAllUserAccessRules(db *sqlx.DB) ([]UserAccessRule, error) {
	var rules []UserAccessRule
	err := db.Select(&rules, `
		SELECT id, application_id, rule_type, subject_type, subject_id, created_at 
		FROM user_access_rules_v1 
		ORDER BY application_id, subject_type DESC, rule_type ASC`)
	return rules, err
}

// CheckUserAccess determines if a user has access to an application based on access rules.
// Priority order: USER ACCEPT/DENY rules first, then GROUP ACCEPT/DENY rules.
// If no rule applies, access is denied by default.
func CheckUserAccess(db *sqlx.DB, applicationID string, userID int) (bool, error) {
	var rules []UserAccessRule
	err := db.Select(&rules, `
		SELECT rule_type, subject_type, subject_id 
		FROM user_access_rules_v1 
		WHERE application_id = $1 
		AND (
			(subject_type = 'USER' AND subject_id = $2) OR
			(subject_type = 'GROUP' AND subject_id = $3)
		)
		ORDER BY 
			CASE WHEN subject_type = 'USER' THEN 1 ELSE 2 END,
			CASE WHEN rule_type = 'ACCEPT' THEN 1 ELSE 2 END`, 
		applicationID, fmt.Sprintf("%d", userID), GroupAllUsers)
	
	if err != nil {
		return false, fmt.Errorf("failed to check user access: %w", err)
	}

	// Check USER rules first (highest priority)
	for _, rule := range rules {
		if rule.SubjectType == SubjectTypeUser && rule.SubjectID == fmt.Sprintf("%d", userID) {
			return rule.RuleType == RuleTypeAccept, nil
		}
	}

	// Check GROUP rules second
	for _, rule := range rules {
		if rule.SubjectType == SubjectTypeGroup && rule.SubjectID == GroupAllUsers {
			return rule.RuleType == RuleTypeAccept, nil
		}
	}

	// Default: deny access if no rule applies
	return false, nil
}

// AddUserAccessRule adds a new access rule.
func AddUserAccessRule(db *sqlx.DB, applicationID string, ruleType RuleType, subjectType SubjectType, subjectID string) error {
	_, err := db.Exec(`
		INSERT INTO user_access_rules_v1 (application_id, rule_type, subject_type, subject_id)
		VALUES ($1, $2, $3, $4)`,
		applicationID, string(ruleType), string(subjectType), subjectID)
	
	if err != nil {
		return fmt.Errorf("failed to add user access rule: %w", err)
	}
	return nil
}

// RemoveUserAccessRule removes an access rule by ID.
func RemoveUserAccessRule(db *sqlx.DB, ruleID int) error {
	result, err := db.Exec("DELETE FROM user_access_rules_v1 WHERE id = $1", ruleID)
	if err != nil {
		return fmt.Errorf("failed to remove user access rule: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("no access rule found with ID %d", ruleID)
	}
	
	return nil
}

// GetUserAccessRule retrieves a specific access rule by ID.
func GetUserAccessRule(db *sqlx.DB, ruleID int) (*UserAccessRule, error) {
	var rule UserAccessRule
	err := db.Get(&rule, `
		SELECT id, application_id, rule_type, subject_type, subject_id, created_at 
		FROM user_access_rules_v1 
		WHERE id = $1`, ruleID)
	return &rule, err
}