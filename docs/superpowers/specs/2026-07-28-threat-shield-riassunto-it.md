# Threat Shield — Riassunto in italiano

> **Nota.** La lingua ufficiale del progetto è l'inglese. Questo documento è una
> traduzione di sintesi: in caso di divergenza vale
> `2026-07-28-threat-shield-design.md`.

**Stato:** design approvato · **Data:** 2026-07-28
**Repository coinvolti:** `my`, `nethsecurity`, `ns8-crowdsec`, `ns8-loki`

## 1. Obiettivo

Trasformare la flotta NethSecurity e NethServer 8 in una rete di sonde di sicurezza, le
cui osservazioni diventano un prodotto: una blocklist di IP ad alta affidabilità
distribuita da un endpoint autenticato su `my`.

La proposta iniziale (Appendice A del design) descrive cinque prodotti insieme. Il lavoro
viene scomposto:

| # | Sotto-progetto | Dipende da | Coperto qui |
|---|---|---|---|
| 0 | Contratto di ingest dei segnali delle sonde (envelope versionato, sanitizer, archivio eventi) | — | sì |
| 1 | Threat Shield (consenso → feed IP autenticato) | 0 | sì |
| 2 | Ingest metriche di flotta (dati di capacità P85) | 0 | no |
| 3 | Health score e configuration drift | 2 + inventario | no |
| 4 | Consulenza di sizing hardware | 2 | no |

Fuori ambito: nessuna pipeline di metriche di flotta, nessun TimescaleDB, nessuna
sostituzione di CrowdSec CAPI, nessuna dashboard di flotta nella v1.

## 2. Decisioni principali

| # | Decisione | Motivazione |
|---|---|---|
| D1 | Il feed si scarica con le credenziali dell'appliance (`system_key:system_secret`, HTTP Basic); contenuto globale, identico per tutti | Riusa l'autenticazione già presente su `collect` per inventario e heartbeat. Nessuna gestione di chiavi lato umano. |
| D2 | Invio dati: telegraf `outputs.http` su NethSecurity, servizio Python su NS8 | Entrambi gli agent sono già installati. Nessun nuovo demone su NethSecurity. |
| D3 | Categorie v1: `port_scan`, `ssh_bruteforce`, `http_exploit`, `sip_probe` | La disponibilità varia per prodotto; il contratto è generico rispetto alla categoria. |
| D4 | Partecipazione attiva per default, opt-out per organizzazione. Nessun vincolo di reciprocità | Percorso più rapido alla massa critica di sonde. Base giuridica: legittimo interesse (GDPR art. 6(1)(f)), esplicitamente richiamato dal Considerando 49 per la sicurezza delle reti e delle informazioni — non consenso. Opt-out documentato e sanitizzazione come misure compensative. |
| D5 | Regola di promozione: ≥3 sistemi distinti su ≥2 organizzazioni distinte in 60 minuti mobili. Scadenza 24 h dall'ultima segnalazione, rinnovata dalle nuove | Il requisito cross-organizzazione elimina il caso della singola flotta mal configurata. Il TTL breve evita che indirizzi in affitto o sotto NAT restino elencati dopo la riassegnazione. |
| D6 | Postgres partizionato sull'istanza esistente, nessun nuovo datastore | Il consenso è una `GROUP BY` su 60 minuti sopra ≤7 giorni di righe. |
| D7 | Ingest e feed su `collect`; API di lettura e gestione su `backend` | `collect` possiede Basic auth appliance, code Redis, worker e cron; `backend` è il piano JWT/RBAC. Stessa separazione di inventario, heartbeat e backup. |
| D8 | Il reporter NS8 vive in `ns8-crowdsec` (primario) e `ns8-loki` (fallback) | Gli alert CrowdSec contengono l'IP attaccante; le serie Prometheus raccolte da `ns8-metrics` sono aggregate e l'hanno già scartato. |
| D9 | Il feed parte spento e viene pubblicato solo al superamento di una soglia di adozione | Vedi §6. |

## 3. Architettura in breve

```
NethSecurity (telegraf)  ─┐
                          ├─ HTTPS Basic ─> collect (:8081)
NS8 (ns8-crowdsec         ─┘                  POST /api/systems/threat-events
     o ns8-loki)                              sanitizer -> coda Redis -> worker
                                              cron 5 min: promozione/scadenza
                                              GET  /api/systems/threat-shield/blocklist
                                                v
                          Postgres: threat_events (partizionato, 7 giorni)
                                    threat_blocklist, threat_allowlist,
                                    threat_daily_stats
                          Redis:    snapshot del feed (body/gz/etag)
                                                v
                          backend (:8080) JWT+RBAC: API tenant/Owner,
                                    allowlist, toggle di partecipazione
```

## 4. Modello dati e contratto

Migrazione `039_add_threat_shield.sql` con rollback e aggiornamento di `schema.sql`.
Tabelle: `threat_events` (partizionata per giorno, ritenzione 7 giorni),
`threat_blocklist`, `threat_allowlist`, `threat_daily_stats`,
`threat_shield_participation`.

Tre scelte da sottolineare:

- **`listing_reason`** conserva le prove al momento della promozione. Gli eventi grezzi
  durano sette giorni: senza questo snapshot, «perché questo IP è in lista?» diventa una
  domanda senza risposta dopo una settimana — e la porrà proprio chi ha un cliente
  bloccato.
- **`threat_daily_stats`** è il vero asset analitico di lungo periodo: aggregare prima che
  le partizioni vengano eliminate è ciò che trasforma una blocklist in dati di tendenza
  sulle minacce, al costo di poche righe al giorno.
- **Riga di partecipazione assente = partecipa**, così l'attivazione per default non
  richiede di popolare tutte le organizzazioni.

L'envelope di ingest è versionato e generico rispetto alla categoria: i sotto-progetti 2–4
aggiungeranno un array fratello, non un nuovo endpoint né una nuova autenticazione.
`system_id` e `organization_id` sono risolti dal sistema autenticato e **mai** letti dal
corpo della richiesta; una richiesta che li contiene viene rifiutata.

Il sanitizer scarta gli indirizzi non pubblici già in ingresso (RFC1918, loopback, CGNAT,
link-local, IMDS, multicast, ULA IPv6, IP di uscita della flotta), filtra `metadata` con
una allowlist di chiavi per categoria, elimina del tutto gli username (senza hash: un hash
di `root` è banalmente reversibile e privo di valore analitico), riduce le URI alla firma
corrispondente e sostituisce gli interni SIP con una classe di lunghezza. Le richieste
oltre i limiti vengono **troncate con un contatore, non rifiutate**: una sonda sotto
attacco non deve perdere l'intero batch.

## 5. Consenso, feed e client

La allowlist è applicata **in promozione, non in lettura**, così un inserimento delista
retroattivamente al passaggio successivo. Gli IP di uscita della flotta sono esclusi
automaticamente: `collect` registra l'IP sorgente osservato di ogni sistema, chiudendo il
caso peggiore — l'appliance mal configurata di un cliente che manda in lista l'indirizzo
WAN di tutta la flotta.

Il feed è testo semplice con `ETag`, `304` e gzip, servito da Redis e non da Postgres: il
costo è costante al crescere degli abbonati e un problema del database non può svuotare la
lista. Tetto massimo 50 000 voci.

La partecipazione è disaccoppiata dal consumo: chi fa opt-out smette di segnalare e
continua a ricevere il feed.

Lato client:

- **NethSecurity** — solo configurazione telegraf più catena di scrubbing. Il consumo
  riusa la funzione di prodotto già chiamata Threat Shield (`banip`): `ns-plug` scarica con
  cache ETag e scrive `/etc/banip/nethesis-threat-shield.txt`, quindi le credenziali
  restano fuori dalla configurazione di banip.
- **ns8-crowdsec (primario)** — `threat-shield-forwarder.service` modellato su
  `cloud-log-manager-forwarder` di ns8-loki, con sorgente `cscli alerts list -o json`.
  **Solo origine locale:** re-inviare la lista community CAPI nel nostro consenso
  fabbricherebbe un accordo tra organizzazioni e avvelenerebbe il feed. Il consumo importa
  le decisioni con `--scenario nethesis/threat-shield`, così i ban di origine Nethesis
  restano distinguibili e rimovibili.
- **ns8-loki (fallback)** — stessa forma, query LogQL su `{category="security"}`, per i
  cluster senza CrowdSec.

## 6. Avvio a freddo

Con la regola di promozione il feed è vuoto finché non segnalano abbastanza sonde. Per
questo il feed parte **spento**: le fasi 1–2 raccolgono e promuovono normalmente, mentre
`GET /blocklist` resta dietro un flag di configurazione fino al superamento di una soglia
(proposta: ≥50 sistemi che segnalano su ≥10 organizzazioni, più una settimana di
statistiche verificate). Pubblicare una lista vuota o di cinque voci su firewall in
produzione brucerebbe la credibilità della funzione il primo giorno.

## 7. Gestione degli errori

- Un client **non deve mai scrivere una lista vuota in caso di errore**: file vuoto
  significa «nessuna minaccia» e disattiva silenziosamente la protezione. Solo un `200`
  con corpo ben formato sostituisce il file locale.
- I watermark dei reporter avanzano solo dopo un `2xx`: un'indisponibilità di `my` produce
  eventi in ritardo, non eventi perduti.
- L'ingest è fail-closed sull'autenticazione e fail-open sul contenuto.
- Se la generazione del consenso fallisce, resta servito lo snapshot precedente con il suo
  `generated_at` originale: lista vecchia, mai vuota.

## 8. Piano di implementazione

**Fase 1 — `my` (in ordine):** migrazione 039 → sanitizer con la sua tabella di test →
endpoint di ingest → worker idempotente → controllo di opt-out con cache Redis → cron di
consenso (promozione, scadenza, rollup, materializzazione) → endpoint del feed con flag di
lancio spento → manutenzione partizioni e ritenzione → risorsa RBAC `threat_shield` in
`sync/configs/config.yml` → API `backend` con `resolveOrgID`, allowlist e partecipazione,
`openapi.yaml` aggiornato → scenario end-to-end con `apitool` → `make pre-commit` su
`collect`, `backend`, `sync`.

**Fase 2 — sonde (in parallelo, una PR per repository):** `ns8-crowdsec`, `ns8-loki`,
`nethsecurity`. Il lavoro lato client è specificato qui come contratto; ogni repository ha
il proprio ciclo di review e di build delle immagini, quindi unirli in un unico piano li
bloccherebbe tutti sul più lento.

**Fase 3 — `my`:** frontend minimo (toggle di partecipazione, riepilogo minacce, allowlist
Owner con lookup della provenienza), statistiche, documentazione e accensione del feed al
superamento della soglia.

## 9. Test

La superficie di test a più alto valore è il sanitizer: un suo difetto è un incidente di
protezione dei dati. Il consenso richiede un test di integrazione su Postgres reale
(`make dev-up`) con fixture sui casi limite — 3 sistemi in 1 organizzazione non devono
promuovere, 3 sistemi su 2 organizzazioni sì — perché la logica *è* l'SQL e `go-sqlmock`
non può validarla.
