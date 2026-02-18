# Alerting

Scopri come la piattaforma My gestisce le regole di alerting e le notifiche per organizzazione tramite Grafana Mimir Alertmanager.

## Panoramica

La piattaforma My utilizza l'Alertmanager multi-tenant di [Grafana Mimir](https://grafana.com/oss/mimir/) per gestire regole di alert e inviare notifiche. Ogni organizzazione dispone di un proprio insieme isolato di regole di alerting e configurazioni di notifica: nessuna organizzazione può vedere o modificare le regole delle altre.

Per la documentazione completa dell'API HTTP di Mimir, consulta:
- [Mimir HTTP API Documentation](https://grafana.com/docs/mimir/latest/references/http-api/)
- [Prometheus Alertmanager v2 OpenAPI Specification](https://github.com/prometheus/alertmanager/blob/main/api/v2/openapi.yaml)

## Come Funziona

### Multi-Tenancy

Ogni sistema appartiene a un'organizzazione. Il servizio collect risolve l'`organization_id` del sistema dalle sue credenziali e lo inietta come header `X-Scope-OrgID` prima di inoltrare la richiesta a Mimir. Questo garantisce che le regole di alert e le notifiche siano completamente isolate tra le organizzazioni — ogni organizzazione gestisce e riceve solo i propri alert.

## Autenticazione

L'accesso all'API Alertmanager usa le stesse credenziali della registrazione del sistema e dell'inventario:

| Campo | Valore |
|-------|--------|
| **Username** | `system_key` (es. `NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE`) |
| **Password** | `system_secret` (es. `my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0`) |
| **Metodo** | HTTP Basic Auth |

Non è necessaria una registrazione separata — qualsiasi sistema che ha completato la registrazione può interagire con l'API Alertmanager immediatamente. Consulta [Registrazione Sistema](05-system-registration.md) per come ottenere le credenziali.

## API Alertmanager

L'Alertmanager è esposto tramite il proxy della piattaforma al percorso:

```
/api/services/mimir/
```

È compatibile con l'[API standard di Alertmanager v2](https://github.com/prometheus/alertmanager/blob/main/api/v2/openapi.yaml).

## Esempi Comuni

### 1. Gestione Configurazione Alertmanager

#### Recuperare la configurazione corrente dell'organizzazione

```bash
curl -u "system_key:system_secret" \
  https://my.nethesis.it/api/services/mimir/api/v1/alerts
```

**Risposta (200 OK):**
```json
{
  "template_files": {},
  "alertmanager_config": "global:\n  resolve_timeout: 5m\n..."
}
```

#### Aggiornare la configurazione dell'alertmanager

```bash
curl -X POST \
  -u "system_key:system_secret" \
  -H "Content-Type: application/json" \
  https://my.nethesis.it/api/services/mimir/api/v1/alerts \
  -d '{
    "template_files": {},
    "alertmanager_config": "global:\n  resolve_timeout: 5m\nroute:\n  group_by: [alertname]\n  group_wait: 10s\n  group_interval: 1m\n  repeat_interval: 1h\n  receiver: \"default\"\nreceivers:\n  - name: \"default\"\n    webhook_configs:\n      - url: \"https://hooks.your-domain.com/alerts\"\n"
  }'
```

**Risposta (201 Created)** - Configurazione creata con successo

#### Eliminare la configurazione dell'alertmanager

```bash
curl -X DELETE \
  -u "system_key:system_secret" \
  https://my.nethesis.it/api/services/mimir/api/v1/alerts
```

**Risposta (200 OK)** - Configurazione eliminata

---

### 2. Gestione Alert

#### Aggiungere un alert direttamente (Injection API)

```bash
curl -X POST \
  -u "system_key:system_secret" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/alerts \
  -d '[{
    "labels": {
      "alertname": "HighCPU",
      "severity": "critical",
      "host": "server-01"
    },
    "annotations": {
      "summary": "CPU usage è troppo alto",
      "description": "CPU su server-01 è al 95%",
      "runbook": "https://wiki.your-domain.com/high-cpu"
    },
    "generatorURL": "https://prometheus.your-domain.com/graph",
    "startsAt": "2024-01-15T10:30:00Z",
    "endsAt": "0001-01-01T00:00:00Z"
  }]'
```

**Risposta (200 OK)** - Alert aggiunto con successo

**Nota sulla risoluzione:** Impostare `endsAt` su `0001-01-01T00:00:00Z` significa che l'alert rimane attivo indefinitamente finché non viene risolto esplicitamente.

#### Recuperare gli alert attivi

```bash
curl -u "system_key:system_secret" \
  -H "Accept: application/json" \
  https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/alerts
```

**Risposta (200 OK):**
```json
[
  {
    "labels": {
      "alertname": "HighCPU",
      "severity": "critical",
      "host": "server-01"
    },
    "annotations": {
      "summary": "CPU usage è troppo alto",
      "description": "CPU su server-01 è al 95%",
      "runbook": "https://wiki.your-domain.com/high-cpu"
    },
    "startsAt": "2024-01-15T10:30:00Z",
    "endsAt": "0001-01-01T00:00:00Z",
    "generatorURL": "https://prometheus.your-domain.com/graph",
    "status": {
      "state": "active",
      "silencedBy": [],
      "inhibitedBy": []
    }
  }
]
```

#### Recuperare i gruppi di alert

```bash
curl -u "system_key:system_secret" \
  -H "Accept: application/json" \
  https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/alerts/groups
```

**Risposta (200 OK):** Ritorna gli alert raggruppati per le etichette di raggruppamento configurate.

#### Risolvere un alert

Per risolvere un alert, invia lo stesso alert con `endsAt` impostato a un timestamp nel passato:

```bash
curl -X POST \
  -u "system_key:system_secret" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/alerts \
  -d '[{
    "labels": {
      "alertname": "HighCPU",
      "severity": "critical",
      "host": "server-01"
    },
    "annotations": {
      "summary": "CPU usage è tornato alla norma",
      "description": "Problema risolto"
    },
    "generatorURL": "https://prometheus.your-domain.com/graph",
    "startsAt": "2024-01-15T10:30:00Z",
    "endsAt": "2024-01-15T11:30:00Z"
  }]'
```

---

### 3. Gestione Silenzi

#### Creare un silenzio

```bash
curl -X POST \
  -u "system_key:system_secret" \
  -H "Content-Type: application/json" \
  https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/silences \
  -d '{
    "matchers": [
      {
        "name": "alertname",
        "value": "HighCPU",
        "isRegex": false
      },
      {
        "name": "host",
        "value": "server-01",
        "isRegex": false
      }
    ],
    "startsAt": "2024-01-15T10:00:00Z",
    "endsAt": "2024-01-15T18:00:00Z",
    "createdBy": "admin@your-domain.com",
    "comment": "Manutenzione pianificata su server-01"
  }'
```

**Risposta (200 OK):**
```json
{
  "silenceID": "2b05304b-a71e-48c0-a877-bb4824e84969"
}
```

#### Recuperare i silenzi attivi

```bash
curl -u "system_key:system_secret" \
  -H "Accept: application/json" \
  https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/silences
```

**Risposta (200 OK):** Lista di tutti i silenzi attivi e le loro configurazioni.

#### Eliminare un silenzio

```bash
curl -X DELETE \
  -u "system_key:system_secret" \
  https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/silence/2b05304b-a71e-48c0-a877-bb4824e84969
```

**Risposta (200 OK)** - Silenzio eliminato

---

### Esempi di Utilizzo Precedenti

**Recuperare i gruppi di alert:**
```bash
curl https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/alerts/groups \
  -u "<system_key>:<system_secret>"
```

## Risoluzione Problemi

### HTTP 401 Unauthorized

**Causa:** `system_key` o `system_secret` non corretti.

**Soluzioni:**
1. Verifica che le credenziali corrispondano a quelle memorizzate sul sistema
2. Assicurati che il sistema abbia completato la registrazione (vedi [Registrazione Sistema](05-system-registration.md))
3. Controlla eventuali spazi iniziali o finali nelle credenziali
4. Testa manualmente:
   ```bash
   curl https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/alerts \
     -u "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE:my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
   ```
   Una risposta `200 OK` o `404 Not Found` (non 401) conferma che l'autenticazione funziona.

### HTTP 500 Internal Server Error

**Causa:** Il backend Mimir non è raggiungibile o è configurato in modo errato.

**Soluzioni:**
1. Si tratta di un problema lato piattaforma — contatta il tuo amministratore
2. Controlla la pagina di stato della piattaforma o gli avvisi di monitoraggio
3. Riprova dopo qualche minuto; Mimir potrebbe essere in fase di riavvio

### HTTP 400 Bad Request

**Causa:** Il corpo della richiesta non è valido (JSON malformato, campi obbligatori mancanti, ecc.)

**Soluzioni:**
1. Verifica che il JSON sia valido usando uno strumento online come [jsonlint.com](https://www.jsonlint.com/)
2. Controlla che tutti i campi obbligatori siano presenti
3. Verifica il formato delle date ISO 8601 (es. `2024-01-15T10:30:00Z`)

## Documentazione Correlata

- [Registrazione Sistema](05-system-registration.md)
- [Inventario e Heartbeat](06-inventory-heartbeat.md)
- [Gestione Sistemi](04-systems.md)
- [Mimir HTTP API Documentation](https://grafana.com/docs/mimir/latest/references/http-api/)
- [Prometheus Alertmanager v2 OpenAPI](https://github.com/prometheus/alertmanager/blob/main/api/v2/openapi.yaml)
