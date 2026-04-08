# Alerting

Scopri come la piattaforma My gestisce alert e silenzi per organizzazione tramite Grafana Mimir Alertmanager.

## Panoramica

La piattaforma My utilizza l'Alertmanager multi-tenant di [Grafana Mimir](https://grafana.com/oss/mimir/) per gestire alert e silenzi. Ogni sistema appartiene a un'organizzazione: il servizio collect risolve l'`organization_id` dalle credenziali del sistema e lo inietta come header `X-Scope-OrgID` prima di inoltrare la richiesta a Mimir. Nessuna organizzazione può vedere o modificare i dati delle altre.

## Convenzione Label e Annotazioni

Tutti gli alert inviati alla piattaforma My devono seguire questa convenzione per garantire il corretto instradamento, la visualizzazione e le notifiche.

### Label

| Label | Obbligatoria | Descrizione |
|-------|--------------|-------------|
| `alertname` | ✅ Obbligatoria | Identificatore CamelCase del tipo di alert. Usa nomi descrittivi, es. `DiskFull`, `WanDown`, `BackupFailed`. |
| `severity` | ✅ Obbligatoria | Livello di gravità dell'alert. Valori consentiti: `critical`, `warning`, `info`. Default: `info` se non impostato. |
| `system_key` | ✅ Auto-aggiunta | Iniettata automaticamente dal servizio collect dalle credenziali del sistema. Non impostare manualmente. |
| `service` | Opzionale | Sotto-servizio del sistema che ha generato l'alert, es. `backup`, `storage`, `ha`, `network`. |

**Formato `alertname`:** Usa UpperCamelCase (PascalCase). Ogni parola inizia con la maiuscola senza separatori. Esempi: `DiskFull`, `WanDown`, `HaSyncFailed`, `CertExpired`.

**Livelli di `severity`:**

| Livello | Quando usarlo |
|---------|---------------|
| `critical` | Il sistema è down o la perdita di dati è imminente. Richiede azione immediata. |
| `warning` | Stato degradato o soglia in avvicinamento. Azione necessaria a breve. |
| `info` | Evento informativo. Nessuna azione immediata richiesta. |

### Annotazioni

| Annotazione | Obbligatoria | Descrizione |
|-------------|--------------|-------------|
| `description_en` | ✅ Obbligatoria | Descrizione in inglese della condizione di alert. |
| `description_it` | Opzionale | Traduzione italiana della descrizione. |

### Esempio di payload alert

```json
[{
  "labels": {
    "alertname": "DiskFull",
    "severity": "critical",
    "service": "storage"
  },
  "annotations": {
    "description_en": "Disk usage on /data is above 90%. Free space: 4 GB.",
    "description_it": "Utilizzo del disco su /data superiore al 90%. Spazio libero: 4 GB."
  },
  "startsAt": "2026-01-15T10:30:00Z",
  "endsAt": "0001-01-01T00:00:00Z"
}]
```

> `system_key` viene iniettata automaticamente — non includerla nel payload.

---

## Catalogo Alert

Tipi di alert standard definiti per i sistemi NethServer e NethSecurity. Gli alert sono unificati tra i prodotti per semplificare la manutenzione e l'integrazione.

### Storage & Sistema

| `alertname` | Severità | Servizio | `description_en` |
|-------------|----------|----------|-----------------|
| `DiskSpaceLow` | `warning` | `storage` | Disk usage on `{{ $labels.mountpoint }}` is above 80%. Free space is running low. |
| `DiskSpaceCritical` | `critical` | `storage` | Disk usage on `{{ $labels.mountpoint }}` is above 90%. Immediate action required. |
| `SwapFull` | `warning` | — | Swap space is filling up (usage above 80%). Current value: `{{ $value }}`. |
| `SwapNotPresent` | `critical` | — | Swap is not configured on this host. |
| `RaidDiskFailed` | `critical` | `storage` | Software RAID array `{{ $labels.device }}` has a failed disk. Immediate attention required. |
| `RaidDriveMissing` | `critical` | `storage` | Software RAID array `{{ $labels.device }}` has insufficient active drives. |

### Infrastruttura & Cluster

| `alertname` | Severità | Servizio | `description_en` |
|-------------|----------|----------|-----------------|
| `NodeOffline` | `critical` | — | Cluster node `{{ $labels.node }}` is offline. This may be caused by a network outage or a crashed metric exporter. |
| `LokiOffline` | `warning` | `loki` | Loki instance `{{ $labels.instance }}` is down or not running properly. |
| `WanDown` | `critical` | `network` | WAN interface `{{ $labels.interface }}` is down. Internet connectivity lost. |
| `HaSyncFailed` | `critical` | `ha` | High-availability synchronization between primary and secondary node has failed. |
| `HaPrimaryFailed` | `critical` | `ha` | High-availability primary node has failed. Failover may have occurred. |

### Certificati & Protezione Dati

| `alertname` | Severità | Servizio | `description_en` |
|-------------|----------|----------|-----------------|
| `CertExpiringSoon` | `warning` | — | TLS certificate `{{ $labels.cn }}` expires in less than 28 days (`{{ $value \| humanizeDuration }}`). |
| `CertExpiringCritical` | `critical` | — | TLS certificate `{{ $labels.cn }}` expires in less than 7 days (`{{ $value \| humanizeDuration }}`). |
| `CertExpired` | `critical` | — | TLS certificate `{{ $labels.cn }}` has expired (`{{ $value \| humanizeDuration }}` ago). |
| `BackupFailed` | `critical` | `backup` | Backup job `{{ $labels.name }}` (ID: `{{ $labels.id }}`) has failed. |
| `ConfigBackupNotEncrypted` | `warning` | `backup` | Configuration backup is not encrypted. Sensitive data may be exposed. |

## Autenticazione

L'accesso all'API Alertmanager usa le stesse credenziali della registrazione del sistema e dell'inventario:

| Campo | Valore |
|-------|--------|
| **Username** | `system_key` (es. `NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE`) |
| **Password** | `system_secret` (es. `my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0`) |
| **Metodo** | HTTP Basic Auth |

Non è necessaria una registrazione separata — qualsiasi sistema che ha completato la registrazione può interagire con l'API Alertmanager immediatamente. Consulta [Registrazione Sistema](05-system-registration.md) per come ottenere le credenziali.

## API Alertmanager

I sistemi possono accedere solo ai seguenti endpoint:

| Risorsa | Percorso |
|---------|----------|
| Alert | `/api/services/mimir/alertmanager/api/v2/alerts` |
| Silenzi | `/api/services/mimir/alertmanager/api/v2/silences[/{silence_id}]` |

## Esempi Comuni

### 1. Gestione Alert

#### Aggiungere un alert direttamente (Injection API)

```bash
curl -X POST \
  -u "system_key:system_secret" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/alerts \
  -d '[{
    "labels": {
      "alertname": "DiskFull",
      "severity": "critical",
      "service": "storage"
    },
    "annotations": {
      "description_en": "Disk usage on /data is above 90%. Free space: 4 GB.",
      "description_it": "Utilizzo del disco su /data superiore al 90%. Spazio libero: 4 GB."
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
      "alertname": "DiskFull",
      "severity": "critical",
      "service": "storage",
      "system_key": "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE"
    },
    "annotations": {
      "description_en": "Disk usage on /data is above 90%. Free space: 4 GB.",
      "description_it": "Utilizzo del disco su /data superiore al 90%. Spazio libero: 4 GB."
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
      "alertname": "DiskFull",
      "severity": "critical",
      "service": "storage"
    },
    "annotations": {
      "description_en": "Disk usage on /data is back to normal."
    },
    "generatorURL": "https://prometheus.your-domain.com/graph",
    "startsAt": "2024-01-15T10:30:00Z",
    "endsAt": "2024-01-15T11:30:00Z"
  }]'
```

---

### 2. Gestione Silenzi

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
        "value": "DiskFull",
        "isRegex": false
      },
      {
        "name": "service",
        "value": "storage",
        "isRegex": false
      }
    ],
    "startsAt": "2024-01-15T10:00:00Z",
    "endsAt": "2024-01-15T18:00:00Z",
    "createdBy": "admin@your-domain.com",
    "comment": "Finestra di manutenzione pianificata"
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
