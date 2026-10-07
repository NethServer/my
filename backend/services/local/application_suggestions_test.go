/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package local

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/backend/models"
)

func suggestionApp(displayName string, fqdns []string, userDomains []string) *models.Application {
	inv := map[string]interface{}{"fqdns": fqdns}
	if userDomains != nil {
		var domains []map[string]string
		for _, d := range userDomains {
			domains = append(domains, map[string]string{"name": d})
		}
		inv["user_domains"] = domains
	}
	raw, _ := json.Marshal(inv)
	app := &models.Application{InventoryData: raw}
	if displayName != "" {
		app.DisplayName = &displayName
	}
	return app
}

func candidate(id, name string) suggestionCandidate {
	return suggestionCandidate{dbID: "db-" + id, logtoID: id, name: name, words: distinctiveWords(name), whole: normalizeToken(name)}
}

func TestPickSuggestionMatchesCustomerNamedInHostname(t *testing.T) {
	cases := []struct {
		name      string
		app       *models.Application
		hostFQDN  string
		partner   string
		pool      []suggestionCandidate
		want      string // expected logto id, "" for no suggestion
		wantMatch string
	}{
		{
			name: "label compacted with legal form equals the whole customer name",
			app:  suggestionApp("", []string{"cti.rei-srl.it", "centralino.rei-srl.it"}, nil),
			pool: []suggestionCandidate{candidate("rei", "Rei Srl"), candidate("other", "Bianchi Spa")},
			want: "rei", wantMatch: "cti.rei-srl.it",
		},
		{
			name:     "customer word inside a longer label",
			app:      suggestionApp("nethvoice_oliveto", []string{"residenceolivetonvapp.lcamedia.eu"}, nil),
			hostFQDN: "nethsrv.lcamedia.cloud",
			pool:     []suggestionCandidate{candidate("oliveto", "Oliveto srl"), candidate("lca", "LCA Media")},
			want:     "oliveto", wantMatch: "residenceolivetonvapp.lcamedia.eu",
		},
		{
			name:     "display name alone identifies the customer",
			app:      suggestionApp("DOMUSNOVA", []string{"npbx007.partner.net"}, nil),
			hostFQDN: "npbx-cluster.partner.net",
			pool:     []suggestionCandidate{candidate("domus", "Domusnova"), candidate("x", "Xenia Srl")},
			want:     "domus", wantMatch: "DOMUSNOVA",
		},
		{
			name:     "ldap user domain identifies the customer",
			app:      suggestionApp("", []string{"npbx009.partner.net"}, []string{"ranocchisrl.cloud.neth.eu"}),
			hostFQDN: "node1.partner.net",
			pool:     []suggestionCandidate{candidate("ran", "Ranocchi S.r.l."), candidate("x", "Xenia Srl")},
			want:     "ran", wantMatch: "ranocchisrl.cloud.neth.eu",
		},
		{
			name: "generic word never matches",
			app:  suggestionApp("voicepigozzosystemit", []string{"voice.pigozzosystem.it"}, nil),
			pool: []suggestionCandidate{candidate("syn", "Synergy System")},
			want: "",
		},
		{
			name:     "partner name and cluster hostname are not customer signals",
			app:      suggestionApp("Fies", []string{"centralinofies.nicoratech.it", "cti.npbx002.erre-elle.net"}, nil),
			hostFQDN: "ncluster.erre-elle.net",
			partner:  "Nicora Tech Srl",
			pool:     []suggestionCandidate{candidate("nic", "Nicora Alberto"), candidate("erre", "Erre Elle Informatica")},
			want:     "",
		},
		{
			name: "a customer named test is never suggested",
			app:  suggestionApp("Test Teams", []string{"cti-test-teams.cloud.neth.eu"}, nil),
			pool: []suggestionCandidate{candidate("test", "test")},
			want: "",
		},
		{
			name: "tie between two customers gives no suggestion",
			app:  suggestionApp("", []string{"chat.acme.example.it"}, nil),
			pool: []suggestionCandidate{candidate("hq", "ACME HQ Srl"), candidate("fil", "ACME Filiale Srl")},
			want: "",
		},
		{
			name: "the customer with more matching tokens wins",
			app:  suggestionApp("", []string{"voice.acme-filiale.example.it", "cti.acme-filiale.example.it"}, nil),
			pool: []suggestionCandidate{candidate("hq", "ACME HQ Srl"), candidate("fil", "ACME Filiale Srl")},
			want: "fil",
		},
		{
			name: "no signals at all",
			app:  suggestionApp("", nil, nil),
			pool: []suggestionCandidate{candidate("x", "Xenia Srl")},
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			signals := applicationNameSignals(tc.app, tc.hostFQDN, tc.partner)
			got := pickSuggestion(signals, tc.pool)
			if tc.want == "" {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.want, got.LogtoID)
			assert.Equal(t, "customer", got.Type)
			if tc.wantMatch != "" {
				assert.Equal(t, tc.wantMatch, got.Matched)
			}
		})
	}
}

func TestInventoryStringsAcceptsStringsAndObjects(t *testing.T) {
	raw := json.RawMessage(`{"fqdns":["a.example.it",""],"user_domains":[{"name":"ad.example.it"},"plain.local",{"foo":1}]}`)
	assert.Equal(t, []string{"a.example.it"}, inventoryStrings(raw, "fqdns"))
	assert.Equal(t, []string{"ad.example.it", "plain.local"}, inventoryStrings(raw, "user_domains"))
	assert.Nil(t, inventoryStrings(raw, "missing"))
	assert.Nil(t, inventoryStrings(nil, "fqdns"))
}

func TestHostSuggestionOnlyForCustomerOwnedSystems(t *testing.T) {
	customer := suggestionHost{orgID: "cust", orgType: "customer", orgDBID: "db-cust", orgName: "Rossi Srl", fqdn: "ns8.rossi.it"}
	got := hostSuggestion(customer, "Cluster Rossi")
	require.NotNil(t, got)
	assert.Equal(t, "cust", got.LogtoID)
	assert.Equal(t, "Rossi Srl", got.Name)
	assert.Equal(t, "hosting_system", got.Source)
	assert.Equal(t, "Cluster Rossi", got.Matched)

	assert.Nil(t, hostSuggestion(suggestionHost{orgID: "res", orgType: "reseller", orgName: "Partner"}, "x"))
	assert.Nil(t, hostSuggestion(suggestionHost{orgType: "customer"}, "x"))
}
