/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package sync

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math/big"
	"net/smtp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/database"
	"github.com/nethesis/my/sync/internal/logger"
)

//go:embed templates/welcome_en.html
var welcomeEnHTML string

//go:embed templates/welcome_en.txt
var welcomeEnTXT string

//go:embed templates/welcome_it.html
var welcomeItHTML string

//go:embed templates/welcome_it.txt
var welcomeItTXT string

// welcomeEmailData mirrors the backend's WelcomeEmailData struct for template rendering
type welcomeEmailData struct {
	UserName         string
	UserEmail        string
	OrganizationName string
	OrganizationType string
	UserRoles        []string
	TempPassword     string
	LoginURL         string
	SupportEmail     string
	CompanyName      string
}

// PushEngine handles the forward synchronization process (from local to Logto)
type PushEngine struct {
	client  *client.LogtoClient
	options *PushOptions
}

// PushOptions contains push operation options
type PushOptions struct {
	DryRun            bool
	Verbose           bool
	OrganizationsOnly bool
	UsersOnly         bool
	SendEmail         bool
	SMTPHost          string
	SMTPPort          int
	SMTPUser          string
	SMTPPassword      string
	SMTPFrom          string
	SMTPUseTLS        bool
	SMTPFromName      string
	FrontendURL       string
	APIBaseURL        string
	Language          string
}

// PushResult contains the results of a push operation
type PushResult struct {
	StartTime     time.Time          `json:"start_time" yaml:"start_time"`
	EndTime       time.Time          `json:"end_time" yaml:"end_time"`
	Duration      time.Duration      `json:"duration" yaml:"duration"`
	DryRun        bool               `json:"dry_run" yaml:"dry_run"`
	Success       bool               `json:"success" yaml:"success"`
	Summary       *PushSummary       `json:"summary" yaml:"summary"`
	Operations    []PushOperation    `json:"operations" yaml:"operations"`
	UserPasswords []UserPasswordInfo `json:"user_passwords,omitempty" yaml:"user_passwords,omitempty"`
	Errors        []string           `json:"errors,omitempty" yaml:"errors,omitempty"`
}

// PushSummary contains a summary of push changes
type PushSummary struct {
	OrganizationsCreated int `json:"organizations_created" yaml:"organizations_created"`
	OrganizationsSkipped int `json:"organizations_skipped" yaml:"organizations_skipped"`
	UsersCreated         int `json:"users_created" yaml:"users_created"`
	UsersSkipped         int `json:"users_skipped" yaml:"users_skipped"`
	EmailsSent           int `json:"emails_sent" yaml:"emails_sent"`
	EmailsFailed         int `json:"emails_failed" yaml:"emails_failed"`
}

// PushOperation represents a single push operation performed
type PushOperation struct {
	Type        string    `json:"type" yaml:"type"`
	Action      string    `json:"action" yaml:"action"`
	Resource    string    `json:"resource" yaml:"resource"`
	Description string    `json:"description" yaml:"description"`
	Success     bool      `json:"success" yaml:"success"`
	Error       string    `json:"error,omitempty" yaml:"error,omitempty"`
	Timestamp   time.Time `json:"timestamp" yaml:"timestamp"`
}

// UserPasswordInfo holds the temporary password generated for a pushed user
type UserPasswordInfo struct {
	Username     string `json:"username" yaml:"username"`
	Email        string `json:"email" yaml:"email"`
	TempPassword string `json:"temp_password" yaml:"temp_password"`
	LogtoID      string `json:"logto_id" yaml:"logto_id"`
	EmailSent    bool   `json:"email_sent" yaml:"email_sent"`
}

// localOrgRow represents a row from any organization table
type localOrgRow struct {
	ID          string
	LogtoID     *string
	Name        string
	Description string
	CustomData  []byte
	OrgType     string
}

// localUserRow represents a row from the users table
type localUserRow struct {
	ID             string
	LogtoID        *string
	Username       string
	Email          string
	Name           string
	Phone          *string
	OrganizationID *string
	UserRoleIDs    []byte
	OrgName        string
	OrgType        string
}

// NewPushEngine creates a new push synchronization engine
func NewPushEngine(c *client.LogtoClient, options *PushOptions) *PushEngine {
	if options == nil {
		options = &PushOptions{}
	}
	return &PushEngine{client: c, options: options}
}

// Push performs the forward synchronization (from local to Logto)
func (e *PushEngine) Push() (*PushResult, error) {
	result := &PushResult{
		StartTime:  time.Now(),
		DryRun:     e.options.DryRun,
		Summary:    &PushSummary{},
		Operations: []PushOperation{},
		Errors:     []string{},
	}

	logger.Info("Starting push from local database to Logto")

	if e.options.DryRun {
		logger.Info("Running in dry-run mode - no changes will be made")
	}

	if !e.options.UsersOnly {
		if err := e.pushOrganizations(result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("organizations push failed: %v", err))
		}
	}

	if !e.options.OrganizationsOnly {
		if err := e.pushUsers(result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("users push failed: %v", err))
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = len(result.Errors) == 0

	if result.Success {
		logger.Info("Push completed successfully in %v", result.Duration)
	} else {
		logger.Error("Push completed with %d errors in %v", len(result.Errors), result.Duration)
	}

	return result, nil
}

// addPushOperation records a push operation in the result
func (e *PushEngine) addPushOperation(result *PushResult, opType, action, resource, description string, opErr error) {
	op := PushOperation{
		Type:        opType,
		Action:      action,
		Resource:    resource,
		Description: description,
		Success:     opErr == nil,
		Timestamp:   time.Now(),
	}
	if opErr != nil {
		op.Error = opErr.Error()
		logger.LogSyncOperation(opType, resource, action, false, opErr)
	} else {
		logger.LogSyncOperation(opType, resource, action, true, nil)
	}
	result.Operations = append(result.Operations, op)
}

// pushOrganizations creates in Logto any organizations that exist locally but are missing
func (e *PushEngine) pushOrganizations(result *PushResult) error {
	logger.Info("Pushing organizations to Logto...")

	logtoOrgs, err := e.client.GetAllOrganizations()
	if err != nil {
		return fmt.Errorf("failed to fetch organizations from Logto: %w", err)
	}

	logtoOrgByName := make(map[string]client.LogtoOrganization, len(logtoOrgs))
	for _, o := range logtoOrgs {
		logtoOrgByName[o.Name] = o
	}

	tables := []struct {
		name    string
		orgType string
	}{
		{"distributors", "distributor"},
		{"resellers", "reseller"},
		{"customers", "customer"},
	}

	for _, tbl := range tables {
		rows, err := e.fetchLocalOrgs(tbl.name, tbl.orgType)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to fetch local %s: %v", tbl.name, err))
			continue
		}
		for _, org := range rows {
			if err := e.pushOrganization(org, logtoOrgByName, result); err != nil {
				logger.Error("Failed to push %s '%s': %v", tbl.orgType, org.Name, err)
				e.addPushOperation(result, tbl.orgType, "create", org.Name, fmt.Sprintf("push %s %s", tbl.orgType, org.Name), err)
			}
		}
	}

	return nil
}

// fetchLocalOrgs reads active orgs from a given table
func (e *PushEngine) fetchLocalOrgs(tableName, orgType string) ([]localOrgRow, error) {
	//nolint:gosec // tableName is hardcoded, not user-supplied
	rows, err := database.DB.Query(fmt.Sprintf(
		`SELECT id, logto_id, name, COALESCE(description, ''), COALESCE(custom_data::text, '{}')
		 FROM %s WHERE deleted_at IS NULL`, tableName))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []localOrgRow
	for rows.Next() {
		var r localOrgRow
		r.OrgType = orgType
		if err := rows.Scan(&r.ID, &r.LogtoID, &r.Name, &r.Description, &r.CustomData); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// pushOrganization creates a single org in Logto if missing, then stores logto_id locally
func (e *PushEngine) pushOrganization(org localOrgRow, logtoOrgByName map[string]client.LogtoOrganization, result *PushResult) error {
	if org.LogtoID != nil && *org.LogtoID != "" {
		logger.Debug("Skipping %s '%s' - already has logto_id", org.OrgType, org.Name)
		result.Summary.OrganizationsSkipped++
		e.addPushOperation(result, org.OrgType, "skip", org.Name, fmt.Sprintf("%s already in Logto", org.OrgType), nil)
		return nil
	}

	// Link to existing Logto org if name matches
	if existing, found := logtoOrgByName[org.Name]; found {
		logger.Info("Linking %s '%s' to existing Logto org %s", org.OrgType, org.Name, existing.ID)
		if !e.options.DryRun {
			if err := e.updateOrgLogtoID(tableForType(org.OrgType), org.ID, existing.ID); err != nil {
				return fmt.Errorf("failed to link org: %w", err)
			}
		}
		result.Summary.OrganizationsSkipped++
		e.addPushOperation(result, org.OrgType, "link", org.Name, fmt.Sprintf("linked %s to existing Logto org", org.OrgType), nil)
		return nil
	}

	if e.options.DryRun {
		logger.Info("DRY RUN: Would create %s '%s' in Logto", org.OrgType, org.Name)
		result.Summary.OrganizationsCreated++
		e.addPushOperation(result, org.OrgType, "create", org.Name, fmt.Sprintf("would create %s in Logto", org.OrgType), nil)
		return nil
	}

	var customData map[string]interface{}
	if err := json.Unmarshal(org.CustomData, &customData); err != nil {
		customData = map[string]interface{}{}
	}
	customData["type"] = org.OrgType

	newOrg := client.LogtoOrganization{
		Name:        org.Name,
		Description: org.Description,
		CustomData:  customData,
	}

	created, err := e.client.CreateOrganization(newOrg)
	if err != nil {
		return fmt.Errorf("failed to create org in Logto: %w", err)
	}

	if err := e.updateOrgLogtoID(tableForType(org.OrgType), org.ID, created.ID); err != nil {
		return fmt.Errorf("failed to store logto_id: %w", err)
	}

	// Keep in-memory map up to date for user processing
	logtoOrgByName[org.Name] = *created

	logger.Info("Created %s '%s' in Logto (ID: %s)", org.OrgType, org.Name, created.ID)
	result.Summary.OrganizationsCreated++
	e.addPushOperation(result, org.OrgType, "create", org.Name, fmt.Sprintf("created %s in Logto (ID: %s)", org.OrgType, created.ID), nil)
	return nil
}

func tableForType(orgType string) string {
	switch orgType {
	case "distributor":
		return "distributors"
	case "reseller":
		return "resellers"
	default:
		return "customers"
	}
}

func (e *PushEngine) updateOrgLogtoID(tableName, localID, logtoID string) error {
	//nolint:gosec // tableName is hardcoded, not user-supplied
	_, err := database.DB.Exec(
		fmt.Sprintf(`UPDATE %s SET logto_id = $1, logto_synced_at = $2, updated_at = $2 WHERE id = $3`, tableName),
		logtoID, time.Now(), localID,
	)
	return err
}

// pushUsers creates in Logto any users that exist locally but are missing
func (e *PushEngine) pushUsers(result *PushResult) error {
	logger.Info("Pushing users to Logto...")

	logtoRoles, err := e.client.GetRoles()
	if err != nil {
		return fmt.Errorf("failed to fetch roles from Logto: %w", err)
	}
	roleByID := make(map[string]client.LogtoRole, len(logtoRoles))
	for _, r := range logtoRoles {
		roleByID[r.ID] = r
	}

	logtoOrgRoles, err := e.client.GetOrganizationRoles()
	if err != nil {
		return fmt.Errorf("failed to fetch organization roles from Logto: %w", err)
	}
	orgRoleByName := make(map[string]client.LogtoOrganizationRole, len(logtoOrgRoles))
	for _, r := range logtoOrgRoles {
		orgRoleByName[strings.ToLower(r.Name)] = r
	}

	logtoOrgs, err := e.client.GetAllOrganizations()
	if err != nil {
		return fmt.Errorf("failed to fetch organizations from Logto: %w", err)
	}
	logtoOrgByID := make(map[string]client.LogtoOrganization, len(logtoOrgs))
	for _, o := range logtoOrgs {
		logtoOrgByID[o.ID] = o
	}

	users, err := e.fetchLocalUsers()
	if err != nil {
		return fmt.Errorf("failed to fetch local users: %w", err)
	}

	logger.Info("Found %d local users to evaluate", len(users))

	for _, user := range users {
		if err := e.pushUser(user, roleByID, orgRoleByName, logtoOrgByID, result); err != nil {
			logger.Error("Failed to push user '%s': %v", user.Username, err)
			e.addPushOperation(result, "user", "create", user.Username, fmt.Sprintf("push user %s", user.Username), err)
		}
	}

	return nil
}

// fetchLocalUsers returns all active, non-suspended local users with their organization info
func (e *PushEngine) fetchLocalUsers() ([]localUserRow, error) {
	rows, err := database.DB.Query(`
		SELECT u.id, u.logto_id, u.username, COALESCE(u.email, ''), COALESCE(u.name, u.username),
		       u.phone, u.organization_id, COALESCE(u.user_role_ids::text, '[]'),
		       COALESCE(orgs.org_name, ''), COALESCE(orgs.org_type, '')
		FROM users u
		LEFT JOIN (
		    SELECT logto_id, name AS org_name, 'distributor' AS org_type FROM distributors WHERE deleted_at IS NULL
		    UNION ALL
		    SELECT logto_id, name AS org_name, 'reseller' AS org_type FROM resellers WHERE deleted_at IS NULL
		    UNION ALL
		    SELECT logto_id, name AS org_name, 'customer' AS org_type FROM customers WHERE deleted_at IS NULL
		) orgs ON orgs.logto_id = u.organization_id
		WHERE u.deleted_at IS NULL AND u.suspended_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []localUserRow
	for rows.Next() {
		var u localUserRow
		if err := rows.Scan(&u.ID, &u.LogtoID, &u.Username, &u.Email, &u.Name, &u.Phone, &u.OrganizationID, &u.UserRoleIDs, &u.OrgName, &u.OrgType); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

// pushUser creates a single user in Logto if missing
func (e *PushEngine) pushUser(
	user localUserRow,
	roleByID map[string]client.LogtoRole,
	orgRoleByName map[string]client.LogtoOrganizationRole,
	logtoOrgByID map[string]client.LogtoOrganization,
	result *PushResult,
) error {
	if user.LogtoID != nil && *user.LogtoID != "" {
		// Verify the user actually exists in Logto (logto_id set but may have been deleted)
		if e.logtoUserExists(*user.LogtoID) {
			logger.Debug("Skipping user '%s' - already in Logto", user.Username)
			result.Summary.UsersSkipped++
			e.addPushOperation(result, "user", "skip", user.Username, "user already in Logto", nil)
			return nil
		}
		// logto_id is stale - clear it so we recreate below
		logger.Info("User '%s' has stale logto_id %s - will recreate", user.Username, *user.LogtoID)
		user.LogtoID = nil
	}

	// Check if a user with this username already exists in Logto
	existing, findErr := e.client.GetUserByUsername(user.Username)
	if findErr == nil && existing != nil {
		existingID, _ := existing["id"].(string)
		if existingID != "" {
			logger.Info("Linking user '%s' to existing Logto user %s", user.Username, existingID)
			if !e.options.DryRun {
				if err := e.updateUserLogtoID(user.ID, existingID); err != nil {
					return fmt.Errorf("failed to link user: %w", err)
				}
			}
			result.Summary.UsersSkipped++
			e.addPushOperation(result, "user", "link", user.Username, "linked to existing Logto user", nil)
			return nil
		}
	}

	if e.options.DryRun {
		logger.Info("DRY RUN: Would create user '%s' (%s) in Logto with temp password", user.Username, user.Email)
		result.Summary.UsersCreated++
		e.addPushOperation(result, "user", "create", user.Username, "would create user in Logto with temp password", nil)
		return nil
	}

	tempPassword, err := generateTempPassword()
	if err != nil {
		return fmt.Errorf("failed to generate password: %w", err)
	}

	userData := map[string]interface{}{
		"username": user.Username,
		"name":     user.Name,
	}
	if user.Email != "" {
		userData["primaryEmail"] = user.Email
	}
	if user.Phone != nil && *user.Phone != "" {
		userData["primaryPhone"] = *user.Phone
	}

	created, err := e.client.CreateUser(userData)
	if err != nil {
		return fmt.Errorf("failed to create user in Logto: %w", err)
	}

	createdID, _ := created["id"].(string)

	if err := e.client.SetUserPassword(createdID, tempPassword); err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}

	// Assign user roles
	var roleIDs []string
	if jsonErr := json.Unmarshal(user.UserRoleIDs, &roleIDs); jsonErr != nil {
		logger.Warn("Failed to parse user_role_ids for user '%s': %v", user.Username, jsonErr)
	}
	var roleNames []string
	for _, roleID := range roleIDs {
		r, ok := roleByID[roleID]
		if !ok {
			logger.Warn("Role ID %s not found in Logto, skipping for user '%s'", roleID, user.Username)
			continue
		}
		roleNames = append(roleNames, r.Name)
		if assignErr := e.client.AssignRoleToUser(createdID, roleID); assignErr != nil {
			logger.Warn("Failed to assign role %s to user '%s': %v", roleID, user.Username, assignErr)
		}
	}

	// Add user to organization and assign org role
	if user.OrganizationID != nil && *user.OrganizationID != "" {
		e.assignUserToOrg(createdID, user.Username, *user.OrganizationID, logtoOrgByID, orgRoleByName)
	}

	if err := e.updateUserLogtoID(user.ID, createdID); err != nil {
		return fmt.Errorf("failed to store logto_id for user: %w", err)
	}

	logger.Info("Created user '%s' in Logto (ID: %s)", user.Username, createdID)
	result.Summary.UsersCreated++
	e.addPushOperation(result, "user", "create", user.Username, fmt.Sprintf("created user in Logto (ID: %s)", createdID), nil)

	pwInfo := UserPasswordInfo{
		Username:     user.Username,
		Email:        user.Email,
		TempPassword: tempPassword,
		LogtoID:      createdID,
	}

	if e.options.SendEmail && user.Email != "" {
		if emailErr := e.sendWelcomeEmail(user.Email, user.Name, tempPassword, user.OrgName, user.OrgType, roleNames); emailErr != nil {
			logger.Warn("Failed to send welcome email to '%s': %v", user.Email, emailErr)
			result.Summary.EmailsFailed++
		} else {
			logger.Info("Welcome email sent to '%s'", user.Email)
			result.Summary.EmailsSent++
			pwInfo.EmailSent = true
		}
	}

	result.UserPasswords = append(result.UserPasswords, pwInfo)
	return nil
}

// logtoUserExists checks if a user with the given logto_id actually exists in Logto
func (e *PushEngine) logtoUserExists(logtoID string) bool {
	_, err := e.client.GetUserByID(logtoID)
	return err == nil
}

// assignUserToOrg adds the user to their organization in Logto and assigns org role
func (e *PushEngine) assignUserToOrg(
	userLogtoID, username, orgID string,
	logtoOrgByID map[string]client.LogtoOrganization,
	orgRoleByName map[string]client.LogtoOrganizationRole,
) {
	logtoOrg, ok := logtoOrgByID[orgID]
	if !ok {
		logger.Warn("Organization %s not found in Logto for user '%s'", orgID, username)
		return
	}

	if err := e.client.AddUserToOrganization(logtoOrg.ID, userLogtoID); err != nil {
		logger.Warn("Failed to add user '%s' to org '%s': %v", username, logtoOrg.Name, err)
		return
	}

	orgType := ""
	if t, ok := logtoOrg.CustomData["type"].(string); ok {
		orgType = t
	}
	if orgType == "" {
		return
	}

	orgRole, ok := orgRoleByName[orgType]
	if !ok {
		logger.Warn("No org role found for type '%s' when assigning user '%s'", orgType, username)
		return
	}

	if err := e.client.AssignOrganizationRoleToUser(logtoOrg.ID, userLogtoID, orgRole.ID); err != nil {
		logger.Warn("Failed to assign org role '%s' to user '%s': %v", orgRole.Name, username, err)
	}
}

func (e *PushEngine) updateUserLogtoID(localID, logtoID string) error {
	var existing sql.NullString
	err := database.DB.QueryRow(`SELECT logto_id FROM users WHERE id = $1`, localID).Scan(&existing)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	_, err = database.DB.Exec(
		`UPDATE users SET logto_id = $1, logto_synced_at = $2, updated_at = $2 WHERE id = $3`,
		logtoID, time.Now(), localID,
	)
	return err
}

// sendWelcomeEmail sends a welcome email using the same HTML/text templates as the backend.
func (e *PushEngine) sendWelcomeEmail(to, userName, tempPassword, orgName, orgType string, roles []string) error {
	host := e.options.SMTPHost
	port := e.options.SMTPPort
	if port == 0 {
		port = 587
	}

	frontendURL := e.options.FrontendURL
	if frontendURL == "" {
		frontendURL = e.options.APIBaseURL
	}

	companyName := e.options.SMTPFromName
	if companyName == "" {
		companyName = "My Nethesis"
	}

	lang := e.options.Language
	if lang == "" {
		lang = "en"
	}

	data := welcomeEmailData{
		UserName:         userName,
		UserEmail:        to,
		OrganizationName: orgName,
		OrganizationType: orgType,
		UserRoles:        roles,
		TempPassword:     tempPassword,
		LoginURL:         fmt.Sprintf("%s/account?changePassword=true", frontendURL),
		SupportEmail:     e.options.SMTPFrom,
		CompanyName:      companyName,
	}

	htmlBody, err := renderEmailTemplate(selectTemplate(lang, "html"), data)
	if err != nil {
		return fmt.Errorf("failed to render HTML template: %w", err)
	}
	textBody, err := renderEmailTemplate(selectTemplate(lang, "txt"), data)
	if err != nil {
		return fmt.Errorf("failed to render text template: %w", err)
	}

	subject := localizedSubject(orgName, lang)
	from := fmt.Sprintf("%s <%s>", companyName, e.options.SMTPFrom)
	boundary := "boundary-nethesis-email"

	var msg strings.Builder
	fmt.Fprintf(&msg, "From: %s\r\n", from)
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", subject)
	msg.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&msg, "Content-Type: multipart/alternative; boundary=%s\r\n", boundary)
	msg.WriteString("\r\n")
	fmt.Fprintf(&msg, "--%s\r\n", boundary)
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	msg.WriteString(textBody + "\r\n\r\n")
	fmt.Fprintf(&msg, "--%s\r\n", boundary)
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	msg.WriteString(htmlBody + "\r\n\r\n")
	fmt.Fprintf(&msg, "--%s--\r\n", boundary)

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if e.options.SMTPUseTLS {
		if err := conn.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	if e.options.SMTPUser != "" && e.options.SMTPPassword != "" {
		auth := smtp.PlainAuth("", e.options.SMTPUser, e.options.SMTPPassword, host)
		if err := conn.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	if err := conn.Mail(e.options.SMTPFrom); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	if err := conn.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	w, err := conn.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	if _, err := w.Write([]byte(msg.String())); err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}
	return w.Close()
}

// selectTemplate returns the embedded template string for the given language and format.
// Falls back to English if the language is not supported.
func selectTemplate(lang, format string) string {
	switch lang {
	case "it":
		if format == "html" {
			return welcomeItHTML
		}
		return welcomeItTXT
	default:
		if format == "html" {
			return welcomeEnHTML
		}
		return welcomeEnTXT
	}
}

// renderEmailTemplate renders an email template string with the given data.
func renderEmailTemplate(tmplStr string, data welcomeEmailData) (string, error) {
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
	}
	tmpl, err := template.New("email").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// localizedSubject returns the email subject in the given language.
func localizedSubject(orgName, lang string) string {
	switch lang {
	case "it":
		return fmt.Sprintf("Benvenuto su %s - Account Creato", orgName)
	default:
		return fmt.Sprintf("Welcome to %s - Account Created", orgName)
	}
}

// generateTempPassword generates a cryptographically secure 16-character password
func generateTempPassword() (string, error) {
	const (
		lowerCase = "abcdefghijklmnopqrstuvwxyz"
		upperCase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits    = "0123456789"
		symbols   = "!@#$%^&*"
		length    = 16
	)

	charset := lowerCase + upperCase + digits + symbols
	password := make([]byte, length)

	// Guarantee at least one char from each character class
	sets := []string{lowerCase, upperCase, digits, symbols}
	for i, set := range sets {
		idx, err := secureRandInt(len(set))
		if err != nil {
			return "", fmt.Errorf("failed to generate password character: %w", err)
		}
		password[i] = set[idx]
	}

	for i := 4; i < length; i++ {
		idx, err := secureRandInt(len(charset))
		if err != nil {
			return "", fmt.Errorf("failed to generate password character: %w", err)
		}
		password[i] = charset[idx]
	}

	// Shuffle
	for i := length - 1; i > 0; i-- {
		j, err := secureRandInt(i + 1)
		if err != nil {
			return "", fmt.Errorf("failed to shuffle password: %w", err)
		}
		password[i], password[j] = password[j], password[i]
	}

	return string(password), nil
}

// secureRandInt returns a cryptographically secure random int in [0, max)
func secureRandInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, fmt.Errorf("failed to generate random integer: %w", err)
	}
	return int(n.Int64()), nil
}

// OutputText outputs the result in text format
func (r *PushResult) OutputText(w io.Writer) error {
	_, _ = fmt.Fprintf(w, "Push Operation Results\n")
	_, _ = fmt.Fprintf(w, "======================\n\n")
	_, _ = fmt.Fprintf(w, "Status: %s\n", map[bool]string{true: "SUCCESS", false: "FAILED"}[r.Success])
	_, _ = fmt.Fprintf(w, "Duration: %v\n", r.Duration)
	_, _ = fmt.Fprintf(w, "Dry Run: %v\n\n", r.DryRun)

	_, _ = fmt.Fprintf(w, "Summary:\n")
	_, _ = fmt.Fprintf(w, "  Organizations: %d created, %d skipped\n",
		r.Summary.OrganizationsCreated, r.Summary.OrganizationsSkipped)
	_, _ = fmt.Fprintf(w, "  Users: %d created, %d skipped\n",
		r.Summary.UsersCreated, r.Summary.UsersSkipped)
	_, _ = fmt.Fprintf(w, "  Emails: %d sent, %d failed\n\n",
		r.Summary.EmailsSent, r.Summary.EmailsFailed)

	if len(r.UserPasswords) > 0 {
		_, _ = fmt.Fprintf(w, "User Temporary Passwords (store securely):\n")
		_, _ = fmt.Fprintf(w, "%-30s %-40s %-20s %s\n", "Username", "Email", "Temp Password", "Email Sent")
		_, _ = fmt.Fprintf(w, "%s\n", strings.Repeat("-", 100))
		for _, up := range r.UserPasswords {
			sent := "no"
			if up.EmailSent {
				sent = "yes"
			}
			_, _ = fmt.Fprintf(w, "%-30s %-40s %-20s %s\n", up.Username, up.Email, up.TempPassword, sent)
		}
		_, _ = fmt.Fprintf(w, "\n")
	}

	if len(r.Errors) > 0 {
		_, _ = fmt.Fprintf(w, "Errors:\n")
		for _, err := range r.Errors {
			_, _ = fmt.Fprintf(w, "  - %s\n", err)
		}
		_, _ = fmt.Fprintf(w, "\n")
	}

	if len(r.Operations) > 0 {
		_, _ = fmt.Fprintf(w, "Operations:\n")
		for _, op := range r.Operations {
			status := "✓"
			if !op.Success {
				status = "✗"
			}
			_, _ = fmt.Fprintf(w, "  %s %s %s %s - %s\n", status, op.Type, op.Action, op.Resource, op.Description)
		}
	}

	return nil
}

// OutputJSON outputs the result in JSON format
func (r *PushResult) OutputJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}

// OutputYAML outputs the result in YAML format
func (r *PushResult) OutputYAML(w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(r)
}
