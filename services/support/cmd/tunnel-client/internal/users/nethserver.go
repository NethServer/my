/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package users

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// NethServerProvisioner creates ephemeral users on NS8 clusters:
//   - A cluster-admin user (Redis-based, only from the leader node)
//   - A domain user per local LDAP/Samba provider (skips remote/external providers)
//
// On worker nodes, user provisioning is skipped entirely because cluster tasks
// can only be submitted from the leader. The leader's tunnel-client handles
// user creation for all domains in the cluster.
type NethServerProvisioner struct {
	RedisAddr string
}

// userDomainsResponse is the top-level response from cluster/list-user-domains
type userDomainsResponse struct {
	Domains []userDomain `json:"domains"`
}

// userDomain represents a single user domain from the NS8 API
type userDomain struct {
	Name      string         `json:"name"`
	Location  string         `json:"location"` // "internal" or "external"
	Schema    string         `json:"schema"`   // "ad" or "rfc2307"
	Providers []userProvider `json:"providers"`
}

// userProvider identifies a module providing the LDAP/Samba service
type userProvider struct {
	ID string `json:"id"` // module ID (e.g., "openldap3", "samba1")
}

func (p *NethServerProvisioner) Create(sessionID string) (*SessionUsers, error) {
	username := generateUsername(sessionID)
	result := &SessionUsers{
		SessionID: sessionID,
		Platform:  "ns8",
		CreatedAt: time.Now(),
	}

	// Skip user provisioning on worker nodes — only the leader can submit cluster tasks
	if !p.isLeaderNode() {
		log.Println("NS8 user provisioning: skipping (not the leader node)")
		return result, nil
	}

	// 1. Create cluster-admin user (remove first if exists, to ensure password matches)
	_ = p.deleteClusterAdmin(username)
	adminPwd := generatePassword()
	if err := p.createClusterAdmin(username, adminPwd); err != nil {
		log.Printf("NS8 user provisioning: cluster-admin creation failed: %v", err)
	} else {
		result.ClusterAdmin = &UserCredential{Username: username, Password: adminPwd}
		log.Printf("NS8 user provisioning: cluster-admin %q created", username)
	}

	// 2. List user domains and create users on local providers
	domains, err := p.listUserDomains()
	if err != nil {
		log.Printf("NS8 user provisioning: cannot list user domains: %v", err)
		return result, nil
	}

	for _, domain := range domains {
		if domain.Location == "external" {
			log.Printf("NS8 user provisioning: skipping external domain %q (read-only)", domain.Name)
			continue
		}
		if len(domain.Providers) == 0 {
			log.Printf("NS8 user provisioning: skipping domain %q (no providers)", domain.Name)
			continue
		}

		// Use the first provider for the domain
		provider := domain.Providers[0].ID
		_ = p.deleteDomainUser(provider, username)
		domainPwd := generatePassword()
		if err := p.createDomainUser(provider, username, domainPwd); err != nil {
			log.Printf("NS8 user provisioning: domain user creation on %q failed: %v", provider, err)
			continue
		}

		result.DomainUsers = append(result.DomainUsers, DomainUser{
			Domain:   domain.Name,
			Module:   provider,
			Username: username,
			Password: domainPwd,
		})
		log.Printf("NS8 user provisioning: domain user %q created on %s (%s)", username, provider, domain.Name)
	}

	return result, nil
}

func (p *NethServerProvisioner) Delete(users *SessionUsers) error {
	if users == nil {
		return nil
	}

	// Only the leader node can remove cluster/domain users
	if !p.isLeaderNode() {
		log.Println("NS8 user cleanup: skipping (not the leader node)")
		return nil
	}

	// Remove domain users (reverse order)
	for i := len(users.DomainUsers) - 1; i >= 0; i-- {
		du := users.DomainUsers[i]
		if err := p.deleteDomainUser(du.Module, du.Username); err != nil {
			log.Printf("NS8 user cleanup: failed to remove domain user %q from %s: %v", du.Username, du.Module, err)
		} else {
			log.Printf("NS8 user cleanup: domain user %q removed from %s", du.Username, du.Module)
		}
	}

	// Remove cluster-admin
	if users.ClusterAdmin != nil {
		if err := p.deleteClusterAdmin(users.ClusterAdmin.Username); err != nil {
			log.Printf("NS8 user cleanup: failed to remove cluster-admin %q: %v", users.ClusterAdmin.Username, err)
		} else {
			log.Printf("NS8 user cleanup: cluster-admin %q removed", users.ClusterAdmin.Username)
		}
	}

	return nil
}

func (p *NethServerProvisioner) createClusterAdmin(username, password string) error {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(password)))

	payload := map[string]interface{}{
		"user":          username,
		"password_hash": hash,
		"set": map[string]interface{}{
			"display_name": "Support Session",
		},
		"grant": []map[string]interface{}{
			{"role": "owner", "on": "*"},
		},
	}

	data, _ := json.Marshal(payload)
	return runAgentTask("cluster", "add-user", string(data))
}

func (p *NethServerProvisioner) deleteClusterAdmin(username string) error {
	data, _ := json.Marshal(map[string]string{"user": username})
	return runAgentTask("cluster", "remove-user", string(data))
}

func (p *NethServerProvisioner) listUserDomains() ([]userDomain, error) {
	output, err := runAgentTaskOutput("cluster", "list-user-domains", "{}")
	if err != nil {
		return nil, fmt.Errorf("list-user-domains failed: %w", err)
	}

	var resp userDomainsResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("cannot parse user domains: %w", err)
	}
	return resp.Domains, nil
}

func (p *NethServerProvisioner) createDomainUser(provider, username, password string) error {
	payload := map[string]interface{}{
		"user":         username,
		"display_name": "Support Session",
		"password":     password,
		"locked":       false,
	}
	data, _ := json.Marshal(payload)
	return runAgentTask(fmt.Sprintf("module/%s", provider), "add-user", string(data))
}

func (p *NethServerProvisioner) deleteDomainUser(provider, username string) error {
	data, _ := json.Marshal(map[string]string{"user": username})
	return runAgentTask(fmt.Sprintf("module/%s", provider), "remove-user", string(data))
}

// isLeaderNode checks if the local Redis instance is the cluster leader (master).
// Worker nodes have a read-only replica and cannot submit cluster tasks.
// Uses redis-cli ROLE which works with the default user's permissions on NS8.
func (p *NethServerProvisioner) isLeaderNode() bool {
	cmd := exec.Command("redis-cli", "ROLE") //nolint:gosec // redis-cli is a trusted system binary
	output, err := cmd.Output()
	if err != nil {
		log.Printf("NS8 user provisioning: cannot check Redis role: %v", err)
		return false
	}
	firstLine := strings.TrimSpace(strings.SplitN(string(output), "\n", 2)[0])
	return firstLine == "master"
}

// runAgentTask executes an NS8 agent task via a Python helper using runagent.
// This uses the agent.tasks framework with redis://127.0.0.1 endpoint,
// which bypasses the API server and runs tasks directly via Redis.
func runAgentTask(agentID, action, data string) error {
	_, err := runAgentTaskOutput(agentID, action, data)
	return err
}

// runAgentTaskOutput executes an NS8 agent task and returns the output JSON.
func runAgentTaskOutput(agentID, action, data string) ([]byte, error) {
	script := fmt.Sprintf(`
import agent.tasks, json, sys
result = agent.tasks.run(
    agent_id=%q,
    action=%q,
    data=json.loads(%q),
    extra={"isNotificationHidden": True},
    endpoint="redis://cluster-leader",
)
if result["exit_code"] != 0:
    print(json.dumps(result.get("error", "task failed")), file=sys.stderr)
    sys.exit(1)
print(json.dumps(result["output"]))
`, agentID, action, data)

	cmd := exec.Command("runagent", "python3", "-c", script) //nolint:gosec // runagent is a trusted NS8 system binary
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Truncate output to avoid dumping full Python stack traces in logs
		msg := strings.TrimSpace(string(output))
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		return nil, fmt.Errorf("agent task %s/%s failed: %w: %s", agentID, action, err, msg)
	}
	return output, nil
}
