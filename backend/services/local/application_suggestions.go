/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package local

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/lib/pq"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// Organization suggestions for unassigned applications.
//
// A partner's multi-tenant cluster is registered under one organization while
// each module on it belongs to one of the partner's customers, and partners
// tend to name the module after that customer: in the hostnames Traefik
// publishes it on (cti.rei-srl.it), in the instance label ("DOMUSNOVA") or in
// its LDAP user domain. SuggestOrganizations looks for exactly that, a
// customer of the same partner whose name shares a distinctive word with the
// application. When nothing stands out and the system itself belongs to a
// customer, that customer is proposed: a cluster registered to a customer
// normally runs that customer's applications. Either way the list is only
// annotated so the user can confirm with one click; nothing is assigned.

// nameStopwords are labels and words that name a product, a role, a legal
// form or a generic business, never a specific customer.
var nameStopwords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`
		cloud mail webmail chat voice voce nethvoice nethserver nethesis nethspot neth
		autoconfig autodiscover webtop nextcloud mattermost roundcube collabora piler
		posta email files drive office portal intranet proxy controller server cluster
		host voip centralino phone telefonia local online test demo prova backup nuovo
		nuova vecchio sede filiale ufficio azienda societa cooperativa studio gruppo
		impresa consorzio ditta fratelli eredi figli service services servizi group
		italia italy comune associazione fondazione hotel agenzia farmacia onlus
		unipersonale international holding energy immobiliare costruzioni commerciale
		tecnologie technology technologies solutions soluzioni consulting trasporti
		logistica autotrasporti system systems sistemi informatica digital software
		computer computers network networks telecom telecomunicazioni dottore
		commercialista ragioniere avvocato ingegnere geometra architetto notaio medico
		legale srls limited company`) {
		nameStopwords[w] = true
	}
}

// nameSignal is one distinctive token taken from an application, with the
// string it was taken from so the UI can say where the suggestion comes from.
type nameSignal struct {
	token  string
	source string
	origin string
}

type suggestionCandidate struct {
	dbID    string
	logtoID string
	name    string
	words   []string
	whole   string
}

// normalizeToken lowercases s and keeps letters and digits only.
func normalizeToken(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// splitWords splits s on every run of non-alphanumeric characters.
func splitWords(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func isNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return s != ""
}

// distinctiveWords keeps the words of a name that can identify a customer:
// at least four characters, not a number, not a stopword.
func distinctiveWords(s string) []string {
	var out []string
	for _, w := range splitWords(s) {
		if len(w) >= 4 && !isNumeric(w) && !nameStopwords[w] {
			out = append(out, w)
		}
	}
	return out
}

// hostnameLabels returns the tokens of a hostname: every label as a whole
// (ranocchisrl-voice -> ranocchisrlvoice) plus its dash-separated parts.
func hostnameLabels(fqdn string) []string {
	var out []string
	for _, label := range strings.Split(strings.ToLower(fqdn), ".") {
		if whole := normalizeToken(label); whole != "" {
			out = append(out, whole)
		}
		out = append(out, splitWords(label)...)
	}
	return out
}

// applicationNameSignals extracts the distinctive tokens of an application.
// Tokens that belong to the cluster's own hostname or to the partner's name
// are dropped: they identify the host, not the customer.
func applicationNameSignals(app *models.Application, hostFQDN, partnerName string) []nameSignal {
	var excluded []string
	for _, w := range hostnameLabels(hostFQDN) {
		if len(w) >= 4 && !isNumeric(w) {
			excluded = append(excluded, w)
		}
	}
	excluded = append(excluded, distinctiveWords(partnerName)...)

	seen := map[string]bool{}
	var signals []nameSignal
	add := func(token, source, origin string) {
		if len(token) < 4 || isNumeric(token) || nameStopwords[token] || seen[token] {
			return
		}
		for _, ex := range excluded {
			if strings.Contains(token, ex) {
				return
			}
		}
		seen[token] = true
		signals = append(signals, nameSignal{token: token, source: source, origin: origin})
	}

	for _, fqdn := range inventoryStrings(app.InventoryData, "fqdns") {
		for _, t := range hostnameLabels(fqdn) {
			add(t, "fqdn", fqdn)
		}
	}
	if app.DisplayName != nil && *app.DisplayName != "" {
		for _, w := range splitWords(*app.DisplayName) {
			add(w, "display_name", *app.DisplayName)
		}
		add(normalizeToken(*app.DisplayName), "display_name", *app.DisplayName)
	}
	for _, domain := range inventoryStrings(app.InventoryData, "user_domains") {
		for _, t := range hostnameLabels(domain) {
			add(t, "user_domain", domain)
		}
	}
	return signals
}

// inventoryStrings reads a list of strings from inventory_data[key]; entries
// may be plain strings or objects carrying the value in "name".
func inventoryStrings(inventory json.RawMessage, key string) []string {
	if len(inventory) == 0 {
		return nil
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(inventory, &data); err != nil {
		return nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(data[key], &items); err != nil {
		return nil
	}
	var out []string
	for _, item := range items {
		var s string
		if err := json.Unmarshal(item, &s); err == nil {
			if s != "" {
				out = append(out, s)
			}
			continue
		}
		var obj struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(item, &obj); err == nil && obj.Name != "" {
			out = append(out, obj.Name)
		}
	}
	return out
}

// matchScore counts the application tokens that name the candidate: a token
// containing a distinctive word of the candidate (residenceolivetonvapp /
// Oliveto), a longer word containing the token (plastica / plast), or the
// whole compacted name matching the token (reisrl / Rei Srl). It also returns
// the origin of the first match for the UI.
func matchScore(signals []nameSignal, c suggestionCandidate) (int, nameSignal) {
	score := 0
	var first nameSignal
	for _, s := range signals {
		matched := false
		for _, w := range c.words {
			if s.token == w || (len(w) >= 5 && strings.Contains(s.token, w)) || (len(s.token) >= 5 && strings.Contains(w, s.token)) {
				matched = true
				break
			}
		}
		if !matched && len(s.token) >= 6 && len(c.whole) >= 6 && (strings.Contains(c.whole, s.token) || strings.Contains(s.token, c.whole)) {
			matched = true
		}
		if matched {
			if score == 0 {
				first = s
			}
			score++
		}
	}
	return score, first
}

// pickSuggestion returns the candidate that outscores every other one, or
// nil when no candidate matches or two of them tie.
func pickSuggestion(signals []nameSignal, candidates []suggestionCandidate) *models.SuggestedOrganization {
	best, second := 0, 0
	var bestCand suggestionCandidate
	var bestSignal nameSignal
	for _, c := range candidates {
		score, sig := matchScore(signals, c)
		switch {
		case score > best:
			second = best
			best, bestCand, bestSignal = score, c, sig
		case score > second:
			second = score
		}
	}
	if best == 0 || best == second {
		return nil
	}
	return &models.SuggestedOrganization{
		OrganizationSummary: models.OrganizationSummary{
			ID:      bestCand.dbID,
			LogtoID: bestCand.logtoID,
			Name:    bestCand.name,
			Type:    "customer",
		},
		Source:  bestSignal.source,
		Matched: bestSignal.origin,
	}
}

type suggestionHost struct {
	fqdn       string
	candParent string // organization whose customers are the candidates
	orgID      string // organization the system belongs to
	orgType    string
	orgDBID    string
	orgName    string
}

// hostSuggestion proposes the customer the system belongs to, the fallback
// when no customer is named by the application.
func hostSuggestion(host suggestionHost, systemName string) *models.SuggestedOrganization {
	if host.orgType != "customer" || host.orgID == "" {
		return nil
	}
	return &models.SuggestedOrganization{
		OrganizationSummary: models.OrganizationSummary{
			ID:      host.orgDBID,
			LogtoID: host.orgID,
			Name:    host.orgName,
			Type:    "customer",
		},
		Source:  "hosting_system",
		Matched: systemName,
	}
}

// SuggestOrganizations fills SuggestedOrganization on the unassigned
// applications in apps. Candidates are the customers of the partner the
// hosting system belongs to (the siblings of a hosting customer, the children
// of a hosting reseller or distributor), restricted to the organizations the
// caller is allowed to assign to.
func (s *LocalApplicationsService) SuggestOrganizations(ctx context.Context, apps []*models.Application, userOrgRole, userOrgID string) error {
	var unassigned []*models.Application
	systemIDs := map[string]bool{}
	for _, app := range apps {
		if app.DeletedAt != nil || (app.OrganizationID != nil && *app.OrganizationID != "") {
			continue
		}
		unassigned = append(unassigned, app)
		systemIDs[app.SystemID] = true
	}
	if len(unassigned) == 0 {
		return nil
	}

	hosts, err := loadSuggestionHosts(ctx, keys(systemIDs))
	if err != nil {
		return err
	}
	parents := map[string]bool{}
	for _, h := range hosts {
		if h.candParent != "" {
			parents[h.candParent] = true
		}
	}
	allowedIDs, err := s.getAllowedOrganizationIDs(userOrgRole, userOrgID)
	if err != nil {
		return fmt.Errorf("failed to resolve assignable organizations: %w", err)
	}
	allowed := map[string]bool{}
	for _, id := range allowedIDs {
		allowed[id] = true
	}

	candidates, parentNames, err := loadSuggestionCandidates(ctx, keys(parents))
	if err != nil {
		return err
	}

	for _, app := range unassigned {
		host, ok := hosts[app.SystemID]
		if !ok {
			continue
		}
		var pool []suggestionCandidate
		for _, c := range candidates[host.candParent] {
			if allowed[c.logtoID] {
				pool = append(pool, c)
			}
		}
		if len(pool) > 0 {
			signals := applicationNameSignals(app, host.fqdn, parentNames[host.candParent])
			app.SuggestedOrganization = pickSuggestion(signals, pool)
		}
		if app.SuggestedOrganization == nil && allowed[host.orgID] {
			systemName := host.fqdn
			if app.System != nil && app.System.Name != "" {
				systemName = app.System.Name
			}
			app.SuggestedOrganization = hostSuggestion(host, systemName)
		}
	}
	return nil
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func loadSuggestionHosts(ctx context.Context, systemIDs []string) (map[string]suggestionHost, error) {
	rows, err := database.DB.QueryContext(ctx, `
		SELECT s.id, COALESCE(s.fqdn, ''), COALESCE(s.organization_id, ''), COALESCE(uo.org_type, ''),
		       COALESCE(uo.db_id, ''), COALESCE(uo.name, ''), COALESCE(c.custom_data->>'createdBy', '')
		FROM systems s
		LEFT JOIN unified_organizations uo ON uo.logto_id = s.organization_id
		LEFT JOIN customers c ON c.logto_id = s.organization_id AND c.deleted_at IS NULL
		WHERE s.id = ANY($1::text[])`, pq.Array(systemIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to load hosting systems: %w", err)
	}
	defer func() { _ = rows.Close() }()

	hosts := map[string]suggestionHost{}
	for rows.Next() {
		var id, fqdn, orgID, orgType, orgDBID, orgName, createdBy string
		if err := rows.Scan(&id, &fqdn, &orgID, &orgType, &orgDBID, &orgName, &createdBy); err != nil {
			return nil, fmt.Errorf("failed to scan hosting system: %w", err)
		}
		h := suggestionHost{fqdn: fqdn, orgID: orgID, orgType: orgType, orgDBID: orgDBID, orgName: orgName}
		switch orgType {
		case "customer":
			h.candParent = createdBy
		case "reseller", "distributor":
			h.candParent = orgID
		}
		hosts[id] = h
	}
	return hosts, rows.Err()
}

// loadSuggestionCandidates returns, per parent organization, its live
// customers prepared for matching, plus the parents' names.
func loadSuggestionCandidates(ctx context.Context, parents []string) (map[string][]suggestionCandidate, map[string]string, error) {
	if len(parents) == 0 {
		return map[string][]suggestionCandidate{}, map[string]string{}, nil
	}
	rows, err := database.DB.QueryContext(ctx, `
		SELECT c.id, c.logto_id, c.name, c.custom_data->>'createdBy'
		FROM customers c
		WHERE c.deleted_at IS NULL AND c.logto_id IS NOT NULL AND c.custom_data->>'createdBy' = ANY($1::text[])`, pq.Array(parents))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load candidate customers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	candidates := map[string][]suggestionCandidate{}
	for rows.Next() {
		var c suggestionCandidate
		var parent string
		if err := rows.Scan(&c.dbID, &c.logtoID, &c.name, &parent); err != nil {
			return nil, nil, fmt.Errorf("failed to scan candidate customer: %w", err)
		}
		c.words = distinctiveWords(c.name)
		c.whole = normalizeToken(c.name)
		candidates[parent] = append(candidates[parent], c)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	nameRows, err := database.DB.QueryContext(ctx, `
		SELECT logto_id, name FROM unified_organizations WHERE logto_id = ANY($1::text[])`, pq.Array(parents))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load parent organizations: %w", err)
	}
	defer func() { _ = nameRows.Close() }()
	names := map[string]string{}
	for nameRows.Next() {
		var id, name string
		if err := nameRows.Scan(&id, &name); err != nil {
			return nil, nil, fmt.Errorf("failed to scan parent organization: %w", err)
		}
		names[id] = name
	}
	return candidates, names, nameRows.Err()
}
