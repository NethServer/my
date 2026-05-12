# Recap — Hierarchical Alerting Config

## TL;DR

Ogni org ha un proprio "layer" di config. La config che Alertmanager usa per ogni tenant è il **merge** di tutti i layer dalla cima (Owner) al tenant (Customer). Mimir resta agnostico: vede un YAML flat per tenant come prima.

```
Owner.layer ─→ Distributor.layer ─→ Reseller.layer ─→ Customer.layer
                                                           │
                                                           ▼
                                              merged config → Mimir tenant YAML
```

## Le 3 regole chiave

### 1. Liste = additive union

```
Owner: mail_addresses=[alerts@msp.it]
Reseller: mail_addresses=[res@reseller.it]
Customer: mail_addresses=[cust@x.it]

→ Effective per Customer's tenant: [alerts@msp.it, res@reseller.it, cust@x.it]
```

I discendenti possono solo **aggiungere** recipient. Non possono mai rimuovere quelli ereditati.

### 2. Bool channel toggles = additivo (OR) per global, "più specifico vince" per severity/system

```
Owner: mail_enabled=true
Customer: mail_enabled=false  → SILENT NORMALIZED a null (non può disabilitare)

Effective: mail_enabled=true (Owner wins)
```

Ma:

```
Owner: severities=[{severity:info, mail_enabled:false}]
→ Per alert severity=info, route a blackhole (Owner ha facoltà di disabilitare per scope)
```

**Decisione UX correlata**: per non-Owner la UI **non mostra** i global toggle (`mail_enabled`, `webhook_enabled`, `telegram_enabled`). Il flusso "aggiungi mail + disabilita canale" è un anti-pattern: se non vuoi ricevere, non aggiungere la mail. Punto. Solo l'Owner ha senso che gestisca i toggle globali (può anche spegnerli per severity/system specifiche). Questo allinea UI e backend (che già normalizza `false → nil` per non-Owner).

### 3. System/severity override liste = ADDITIVE al render (Opzione 2)

```
Owner: mail_addresses=[alerts@msp.it]
Customer: systems=[{system_key:X, mail_addresses:[mobile@cust.it]}]

→ Per alert su X: receiver = [alerts@msp.it, mobile@cust.it]
                                          ↑                    ↑
                                       Owner unioned         Customer added
```

Prima dell'Opzione 2 il system override era REPLACE → Customer accidentalmente blindava Owner. **Risolto**.

## API

```
POST   /api/alerts/config              → salva il proprio layer + propaga ai descendants
GET    /api/alerts/config              → proprio layer + inherited[] (read-only ancestors)
GET    /api/alerts/config/effective    → merge effettivo di un tenant
DELETE /api/alerts/config              → rimuovi proprio layer + propaga
```

`POST` e `DELETE` rispondono con `{affected_tenants, propagated_to, warnings[]}` — fan-out concorrente con timeout.

## Esempio fine: edit di Owner preserva downstream

```
Stato:
  Owner: [alerts@msp.it]
  Reseller: [res@reseller.it]
  Customer: [cust@x.it]

Owner re-save:
  POST {mail_addresses:[noc@msp.it]}    # cambia la propria email

Risultato:
  Owner.layer:   [noc@msp.it]
  Reseller.layer: [res@reseller.it]    ← intoccato
  Customer.layer: [cust@x.it]          ← intoccato

  Effective Customer's tenant: [noc@msp.it, res@reseller.it, cust@x.it]
```

Il save di Owner riscrive **solo** la propria riga in `alert_config_layers`. Il merge ricalcolato preserva tutto il resto. Niente downstream wipe.

## File principali

| File | Cosa |
|---|---|
| `database/migrations/024_add_alert_config_layers.sql` | Tabella per i layer |
| `models/alerting.go` (`AlertingConfigLayer`) | Model con `*bool` channel toggles per distinguere "non set" da "false esplicito" |
| `entities/local_alert_config_layers.go` | Repo CRUD + `GetByOrgIDs` per merge bulk |
| `services/alerting/merge.go` | `MergeLayers` (tristate bool, union dedup), `NormalizeLayerForRole` (strip false per non-Owner) |
| `services/alerting/effective.go` | `ResolveAncestorChain`, `ComputeEffectiveConfig`, `RenderAndPushEffective` |
| `services/alerting/template.go` (`effectiveSettings`) | Render UNION (Opzione 2) per system/severity lists |
| `methods/alerting.go` | Handler refactored: `ConfigureAlerts`, `DisableAlerts`, `GetAlertingConfig`, `GetAlertingConfigEffective` |

## Limiti attuali (backlog)

1. **Mute rules per-recipient**: "Owner muta intero customer X solo per il mio occhio". Non implementato. Workaround attuale: silences (time-bounded, rinnovabili).

2. **Lingua per-recipient**: il IT manager inglese di un Customer italiano riceve in italiano (è la lingua del tenant). Non c'è preferenza per-email. Backlog.

3. **`email_template_lang`**: unico campo che usa "deepest non-empty wins" (i discendenti possono override per il proprio sub-tree). Tutti gli altri sono additive.

4. **Customer non può "togliere" recipient ereditati**: per design (anti-blinding del MSP). Per mute permanente di alcuni allarmi → mute rules quando le faremo.

5. **Provisioning di nuove org**: usano `ComputeEffectiveConfig` quindi ereditano automaticamente i layer ancestor esistenti al momento della creazione.

6. **Owner pigro**: se Owner non salva mai, Distributor/Reseller/Customer operano comunque indipendentemente. Niente "Owner privilege" hard-coded — è solo il livello più alto della catena.

## Sicurezza

- `NormalizeLayerForRole` strippa `*bool=&false` da non-Owner al write time → contratto additive enforced sul boundary di storage
- Validation server-side su email format, max items, system_key charset (`^[A-Za-z0-9_:.-]+$` anti-injection)
- Webhook URL denylist per host privati/loopback (anti-SSRF)

## RBAC sulle rotte di config

Permission split: **operations** restano sul resource `systems`, **policy** sta su un resource dedicato `alerts`.

| Endpoint | Resource gate(s) | Effective permission |
|---|---|---|
| `GET /alerts/config` | `systems` (parent) + `alerts` | `read:systems` AND `read:alerts` |
| `GET /alerts/config/effective` | `systems` (parent) + `alerts` | `read:systems` AND `read:alerts` |
| `POST /alerts/config` | `systems` (parent) + `alerts` | `manage:systems` AND `manage:alerts` |
| `DELETE /alerts/config` | `systems` (parent) + `alerts` | `manage:systems` AND `manage:alerts` |
| `GET /alerts`, `/totals`, `/trend`, `/stats`, `/history`, `/:fingerprint/activity` | `systems` | `read:systems` |
| `POST/PUT/DELETE /systems/:id/alerts/silences/...` | `systems` | `manage:systems` |
| `GET /systems/:id/alerts/silences` | `systems` | `read:systems` |

Assegnazione ai ruoli (`sync/configs/config*.yml`):

| Ruolo | `read:alerts` | `manage:alerts` | Note |
|---|:---:|:---:|---|
| `admin` | ✅ | ✅ | Gestisce policy alerting per la propria org |
| `super` | ✅ | ✅ | Stesso scope di admin + privilegi destroy |
| `support` | ❌ | ❌ | Può silenziare alert e vedere activity, ma non riscrivere policy |
| `backoffice` | ❌ | ❌ | Account management, niente policy alerting |
| `reader` | ❌ | ❌ | Vede alert list/aggregati, non la lista email/recipient del MSP |

L'org-scoping è enforced dal handler: `ConfigureAlerts`/`DisableAlerts` operano sempre su `user.OrganizationID`. Nessuna possibilità di scrivere il layer di un'altra org.

Note di implementazione:

- Sub-group nested in `main.go`: `alerts/config` eredita `RequireResourcePermission("systems")` del parent + aggiunge `RequireResourcePermission("alerts")`. Doppia gate intentional — Admin/Super hanno entrambe, gli altri ruoli sono bloccati al secondo gate.
- Le silences (`/systems/:id/alerts/silences/*`) e l'activity timeline (`/alerts/:fingerprint/activity`) restano sotto `systems`: Support può fare manutenzione operativa senza poter cambiare la policy globale.

---

# Mappa completa dei campi

## 1. Tabella globale (radice del layer)

| Campo | Tipo | Owner | Distributor / Reseller / Customer | Merge fra layer | Effective al render |
|---|---|---|---|---|---|
| `mail_addresses` | `[]string` | RW | RW (additivo) | Union + dedup | Lista finale |
| `mail_enabled` | `*bool` | RW (`true` / `false`) | RW solo `true`; `false` → silently `nil` | OR (qualsiasi `true` vince) | `true` se almeno un layer `true` |
| `webhook_receivers` | `[]WebhookReceiver` | RW | RW (additivo) | Union + dedup per `url` | Lista finale |
| `webhook_enabled` | `*bool` | RW (`true` / `false`) | RW solo `true`; `false` → `nil` | OR | come sopra |
| `telegram_receivers` | `[]TelegramReceiver` | RW | RW (additivo) | Union + dedup per `chat_id` | Lista finale |
| `telegram_enabled` | `*bool` | RW (`true` / `false`) | RW solo `true`; `false` → `nil` | OR | come sopra |
| `email_template_lang` | `string` (es. `"it"` / `"en"`) | RW | RW (override per sub-tree) | **Deepest non-empty wins** | Lingua del layer più vicino al tenant |

## 2. Override per severity (`severities[]`) e per system (`systems[]`)

Sono **liste** di entry. Ogni entry ha una **chiave** + gli stessi campi recipient/toggle del global.

```
severities[]: { severity_key, mail_addresses[], mail_enabled*, webhook_receivers[], webhook_enabled*, telegram_receivers[], telegram_enabled* }
systems[]:    { system_key,   mail_addresses[], mail_enabled*, webhook_receivers[], webhook_enabled*, telegram_receivers[], telegram_enabled* }
```

Regole merge per ogni **entry** con stessa chiave (es. due layer hanno `severities[critical]`):

| Sub-campo dell'entry | Owner | Non-Owner | Merge | Render (Opzione 2) |
|---|---|---|---|---|
| `mail_addresses` | RW | RW (additivo) | Union dedup fra entry omonimi | **Union con global** |
| `webhook_receivers` | RW | RW (additivo) | Union dedup per url | Union con global |
| `telegram_receivers` | RW | RW (additivo) | Union dedup per chat_id | Union con global |
| `mail_enabled` | RW (`true`/`false`) | solo `true`; `false` → `nil` | **Tristate**: any `true` → `true`; else any `false` → `false`; else `nil` (eredita global) | Se `false`: route a **blackhole** per quella severity/system |
| `webhook_enabled` | RW (`true`/`false`) | solo `true` | tristate | come sopra |
| `telegram_enabled` | RW (`true`/`false`) | solo `true` | tristate | come sopra |

**Tristate spiegato**: per ogni boolean, lo merge accumula così:

- almeno un layer dice `true` → effective `true`
- nessuno `true`, almeno un layer dice `false` → effective `false`
- nessun layer ha settato → `nil` → eredita dal global

## 3. Cosa succede al SAVE

```
POST /alerts/config (layer di org L)

├── 1. NormalizeLayerForRole(layer, role)
│     ├── se non-Owner: per ogni *bool=&false → setta a nil
│     └── se Owner: niente strip
│
├── 2. UPSERT in alert_config_layers (PK: organization_id)
│     └── tocca SOLO la riga di L. Mai gli altri layer.
│
├── 3. ResolveDescendants(L) → [L, child1, child2, ..., grandchild...]
│
├── 4. Fan-out parallelo (max 10 worker, timeout 30s per tenant):
│     per ogni descendant D:
│       a) chain = ResolveAncestorChain(D)         // [Owner, ..., D]
│       b) effective = MergeLayers(chain)          // applica le regole
│       c) yaml = RenderConfig(effective, D)       // template Alertmanager
│       d) PUSH /alerts/config (Mimir tenant=D)
│
└── 5. Response: { affected_tenants, propagated_to, warnings[] }
      warnings[] = elenco tenant per cui il push Mimir è fallito (non blocca gli altri)
```

`DELETE` è identico, ma allo step 2 fa `DELETE` invece di `UPSERT` e i descendant ricevono YAML calcolato senza il contributo di L.

## 4. Mappa visiva

```
Layer di un'org (riga in alert_config_layers, JSONB)
│
├── global
│   ├── mail_addresses[]            ─ additivo, sempre
│   ├── mail_enabled                ─ Owner W full / Non-Owner solo true
│   ├── webhook_receivers[]         ─ additivo
│   ├── webhook_enabled             ─ Owner W full / Non-Owner solo true
│   ├── telegram_receivers[]        ─ additivo
│   ├── telegram_enabled            ─ Owner W full / Non-Owner solo true
│   └── email_template_lang         ─ deepest non-empty wins (per-sub-tree)
│
├── severities[]                    ─ entries union per severity_key
│   └── { severity_key,             ─ chiave di merge
│         mail_addresses[],         ─ additivo, UNION con global a render
│         mail_enabled,             ─ tristate; false di Owner = blackhole su quella severity
│         webhook_receivers[], webhook_enabled,
│         telegram_receivers[], telegram_enabled }
│
└── systems[]                       ─ entries union per system_key
    └── { system_key,
          mail_addresses[], mail_enabled,
          webhook_receivers[], webhook_enabled,
          telegram_receivers[], telegram_enabled }
```

## 5. Quick-reference per ruolo

**Owner** (può tutto):

- accendere/spegnere canali a livello globale
- definire i recipient di base
- spegnere canale per severity/system specifica (es: "niente mail per `info`")
- impostare la lingua di default

**Distributor / Reseller / Customer** (solo additivo):

- aggiungere i propri recipient (mail, webhook, telegram)
- aggiungere recipient per severity/system specifici → si **uniscono** a quelli ereditati
- impostare lingua per il proprio sub-tree (vince per i propri descendant)
- **non possono**: rimuovere recipient ereditati, spegnere canali, mascherare alert agli ascendenti

## 6. Edge case importanti da ricordare

1. **`email_template_lang`** è l'unico campo "subtractive-by-default": ogni layer override la lingua per sé e i suoi discendenti. Non è una violazione del principio additive: è preferenza locale, non un canale di silenziamento.

2. **Tristate per i bool di severity/system di Owner** è la sola "leva di disable" nel sistema. Es: Owner setta `severities=[{severity:info, mail_enabled:false}]` → tutti i descendant si beccano route blackhole per `info`. Non-Owner non può fare lo stesso (il `false` viene strippato).

3. **Provisioning di nuove org**: alla creazione di un Customer figlio di un Reseller esistente, il primo render Mimir applica già il merge di tutta la chain ancestor. Niente "buco" iniziale.

4. **Layer vuoto è valido**: una org può non avere riga in `alert_config_layers`. Significa che eredita tutto dagli ancestor. È lo stato di default delle org appena create che non hanno mai chiamato POST.

## 7. Sintesi semantica (cheat sheet)

Riassunto operativo di cosa può fare chi:

| Cosa | Comportamento |
|---|---|
| **Flag bool global** (`mail_enabled`, `webhook_enabled`, `telegram_enabled`) | Solo **"acceso"** è propagabile (OR fra layer). Spento non si può imporre. |
| **Endpoint/recipient** (`mail_addresses`, `webhook_receivers`, `telegram_receivers`) | Additivi: union dedup di tutti i layer della chain. |
| **`email_template_lang`** | Vince il layer più profondo non vuoto (= il più vicino al tenant). |

**Sfumatura sui flag** — l'OR fa sì che un flag sia acceso per il **sub-tree di chi lo accende**, non necessariamente "per tutti":

```
Owner: telegram_enabled=nil
Distributor A: telegram_enabled=true     → effective: TRUE per A e suoi figli
Distributor B: telegram_enabled=nil      → effective: FALSE/NIL per B e suoi figli
```

Più precisamente:

- **Owner accende il flag** → acceso per **tutti** i descendant (nessun opt-out possibile).
- **Livello intermedio accende il flag** → acceso solo per il **suo sub-tree**.
- **Spegnere targettando un sub-tree dall'alto** → ❌ non possibile (richiede mute rules / owner_overrides — backlog).
- **Spegnere il proprio canale (non-Owner)** → ❌ non possibile (`false` normalizzato a `nil`).

---

# Cheat Sheet — Recap multi-level config

Riferimento sintetico per presentazioni e onboarding rapido.

## 1. Gerarchia (4 livelli)

```
Owner → Distributor → Reseller → Customer
(top)                              (tenant)

merge dall'alto verso il basso → effective config per tenant
```

## 2. Regole di merge per campo

| Campo | Rule | Cosa può cambiare descendant |
|---|---|---|
| `mail_addresses[]`, `webhook_receivers[]`, `telegram_receivers[]` (globali) | **UNION dedup** | Solo aggiungere |
| `mail_enabled`, `webhook_enabled`, `telegram_enabled` (globali) | **OR** (any true → true) | Solo accendere (false strippato) |
| `email_template_lang` | **deepest non-empty wins** | Override per il proprio sub-tree |
| `severities[X].mail_enabled` (e webhook/telegram) | **TRISTATE**: any true → true; else any false → false; else null | Owner può spegnere; descendant può ri-accendere |
| `severities[X].mail_addresses[]` | **UNION** + UNION con globale al render | Solo aggiungere |
| `systems[X].*` (toggles + recipients) | **stesso pattern di severities** | Idem |
| `systems[X].severities[]` (narrowing) | **any "all" wins; else union** | Solo aggiungere |

## 3. Cosa può fare ogni ruolo

| | Owner | Distributor / Reseller / Customer |
|---|---|---|
| Aggiungere recipient (mail/webhook/telegram) | ✅ | ✅ |
| Disabilitare canale globale | ❌ no-op (OR ignora false) | ❌ false strippato |
| Disabilitare canale per severity/system | ✅ | ❌ false strippato |
| **Ri-abilitare** un canale che ancestor ha spento | ✅ | ✅ (per il proprio sub-tree) |
| Cambiare lingua email | ✅ | ✅ (per proprio sub-tree) |
| Override per sistema specifico | ✅ | ✅ |
| Narrowing su severity per system | ✅ | ✅ |

## 4. Cosa vede ogni ruolo

| Vista | Owner | Distributor | Reseller | Customer |
|---|---|---|---|---|
| **Inherited (read-only)** | nessuno | Owner | Owner + Distributor | Owner + Distributor + Reseller |
| **Your settings** (proprio layer, editabile) | proprio | proprio | proprio | proprio |
| **Effective preview** (merge per un tenant) | qualunque org | proprio sub-tree | proprio sub-tree | solo se stesso |
| Secrets degli ancestor (bot_token, webhook URL) | ✅ in chiaro | `[REDACTED]` | `[REDACTED]` | `[REDACTED]` |
| Propri secret nelle proprie viste | ✅ in chiaro | ✅ in chiaro | ✅ in chiaro | ✅ in chiaro |

## 5. Propagazione (eager fan-out)

```
[POST /alerts/config]
    ↓
Save proprio layer
    ↓
Per ogni tenant nei discendenti (max 10 in parallelo):
  effective = merge(Owner.layer, ..., tenant.layer)
  yaml = render(effective, tenant)
  PUSH /api/v1/alerts → Mimir tenant
    ↓
Response: {affected_tenants, propagated_to, warnings[]}
```

## 6. Render finale (per un tenant) — ordine delle route Alertmanager

```
1. system_key="X" AND severity="..."   → system-X-receiver        (system override + narrowing)
2. severity="critical"                  → severity-critical-receiver
3. severity="warning"                   → severity-warning-receiver
4. severity="info"                      → severity-info-receiver
5. (nessun matcher)                     → global-receiver
```

`continue: false`: prima route che matcha vince (più specifica). Recipient di ciascun receiver = UNION (Opzione 2) di global + per-severity + per-system applicabili.

## 7. Esempio concreto end-to-end

```
Owner:        mail [a@msp]; severities[info].mail_enabled=false
Distributor:  mail [b@dist]
Reseller:     mail [c@res]; lang=it
Customer:     mail [d@cust]; systems[X].severities=[critical]; mail [+sys-crit@]

Effective per Customer's tenant:
  mail_enabled=true, lang=it (Reseller deepest wins)
  Recipients globali: [a, b, c, d]
  Per severity=info: blackhole (Owner false propagato; Customer non override)
  Per system=X AND severity=critical: [a, b, c, d, +sys-crit@]
```

## 8. Vincoli

- Customer/Reseller/Distributor non possono mai **rimuovere** un recipient ereditato
- Solo Owner può "spegnere" un canale a livello globale (anche se l'OR-merge lo rende un no-op pratico) e per scope (severity / system, dove invece è effettivo)
- Per-recipient language e per-recipient muting: backlog (richiede `owner_overrides` con matchers)
