# Alerting

Scopri come la piattaforma My gestisce le regole di alerting e le notifiche per organizzazione tramite Grafana Mimir Alertmanager.

## Panoramica

La piattaforma My utilizza l'Alertmanager multi-tenant di [Grafana Mimir](https://grafana.com/oss/mimir/) per gestire regole di alert e inviare notifiche. Ogni organizzazione dispone di un proprio insieme isolato di regole di alerting e configurazioni di notifica: nessuna organizzazione può vedere o modificare le regole delle altre.

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
/api/services/mimir/alertmanager/api/v2/
```

È compatibile con l'[API standard di Alertmanager v2](https://github.com/prometheus/alertmanager/blob/main/api/v2/openapi.yaml).

### Esempi di utilizzo

**Recuperare gli alert attivi:**
```bash
curl https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/alerts \
  -u "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE:my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
```

**Recuperare i gruppi di alert:**
```bash
curl https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/alerts/groups \
  -u "<system_key>:<system_secret>"
```

**Creare un silenzio:**
```bash
curl -X POST https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/silences \
  -u "<system_key>:<system_secret>" \
  -H "Content-Type: application/json" \
  -d '{
    "matchers": [{"name": "alertname", "value": "HighCPU", "isRegex": false}],
    "startsAt": "2024-01-01T00:00:00Z",
    "endsAt": "2024-01-02T00:00:00Z",
    "createdBy": "admin",
    "comment": "Manutenzione pianificata"
  }'
```

**Recuperare i silenzi attivi:**
```bash
curl https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/silences \
  -u "<system_key>:<system_secret>"
```

**Eliminare un silenzio:**
```bash
curl -X DELETE https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/silence/<silence_id> \
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

## Documentazione Correlata

- [Registrazione Sistema](05-system-registration.md)
- [Inventario e Heartbeat](06-inventory-heartbeat.md)
- [Gestione Sistemi](04-systems.md)
