# Metriche

Scopri come i sistemi esterni inviano metriche Prometheus alla piattaforma My e come visualizzarle in Grafana.

## Panoramica

La piattaforma My supporta la raccolta di metriche Prometheus tramite [Grafana Mimir](https://grafana.com/oss/mimir/). Qualsiasi sistema NethServer o NethSecurity registrato può inviare metriche usando il protocollo standard Prometheus `remote_write`. Le metriche sono isolate per organizzazione e visibili nelle dashboard Grafana.

## Come Funziona

### Flusso di Acquisizione Metriche

```
┌─────────────────┐                                       ┌──────────────┐
│  NethServer /   │  POST /services/mimir/api/v1/push     │              │
│  NethSecurity   │ ─────────────────────────────────>    │    nginx     │
│                 │  Basic Auth: system_key:system_secret │              │
└─────────────────┘                                       └──────┬───────┘
                                                                │
                                                                │ /api/services/mimir/
                                                                v
                                                         ┌──────────────┐
                                                         │   Collect    │
                                                         │              │
                                                         │  1. Validate │
                                                         │     Basic    │
                                                         │     Auth     │
                                                         │  2. Set      │
                                                         │  X-Scope-    │
                                                         │  OrgID:      │
                                                         │  <org_id>    │
                                                         └──────┬───────┘
                                                                │
                                                                v
                                                         ┌──────────────┐
                                                         │    Mimir     │
                                                         │  (private)   │
                                                         └──────────────┘
```

### Flusso di Accesso Grafana

```
┌─────────────┐                   ┌──────────────┐     ┌──────────────┐
│   Browser   │  /grafana/        │    nginx     │     │   Grafana    │
│             │ ─────────────>    │              │ --> │              │
│             │                   └──────────────┘     └──────┬───────┘
└─────────────┘                                               │
                                                              │ queries Mimir
                                                              │ X-Scope-OrgID
                                                              v
                                                       ┌──────────────┐
                                                       │    Mimir     │
                                                       │  (private)   │
                                                       └──────────────┘
```

### Multi-Tenancy

Ogni sistema appartiene a un'organizzazione. Il servizio collect risolve l'`organization_id` del sistema dalle sue credenziali e lo inietta come header `X-Scope-OrgID` prima di inoltrare la richiesta a Mimir. Questo garantisce che le metriche siano completamente isolate tra le organizzazioni — ogni organizzazione vede solo i propri dati.

## Autenticazione

L'invio delle metriche usa le stesse credenziali della registrazione del sistema e dell'inventario:

| Campo | Valore |
|-------|--------|
| **Username** | `system_key` (es. `NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE`) |
| **Password** | `system_secret` (es. `my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0`) |
| **Metodo** | HTTP Basic Auth |

Non è necessaria una registrazione separata — qualsiasi sistema che ha completato la registrazione può inviare metriche immediatamente. Consulta [Registrazione Sistema](05-system-registration.md) per come ottenere le credenziali.

## Configurazione di Prometheus `remote_write`

Aggiungi il seguente blocco alla configurazione di Prometheus (`/etc/prometheus/prometheus.yml` o equivalente):

```yaml
remote_write:
  - url: https://my.nethesis.it/services/mimir/api/v1/push
    basic_auth:
      username: <system_key>
      password: <system_secret>
```

Sostituisci `<system_key>` e `<system_secret>` con le credenziali effettive memorizzate sul sistema.

**Esempio con valori realistici:**
```yaml
remote_write:
  - url: https://my.nethesis.it/services/mimir/api/v1/push
    basic_auth:
      username: NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE
      password: my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0
```

Dopo aver aggiornato la configurazione, ricarica Prometheus:
```bash
systemctl reload prometheus
# oppure invia SIGHUP
kill -HUP $(pidof prometheus)
```

!!! tip
    Prometheus inizierà a inoltrare tutte le metriche raccolte a My. Usa `remote_write_queue_samples_total` nel tuo Prometheus locale per verificare che le metriche vengano inviate.

## Accesso a Grafana

Grafana è disponibile all'indirizzo:

```
https://my.nethesis.it/grafana/
```

Le dashboard sono **per organizzazione**: gli utenti di ciascuna organizzazione possono vedere solo le metriche raccolte dai sistemi appartenenti alla propria organizzazione. L'isolamento del tenant è applicato automaticamente tramite `X-Scope-OrgID`.

!!! note
    L'accesso a Grafana è gestito dall'amministratore della piattaforma. Contattalo per ottenere l'accesso o per richiedere dashboard personalizzate per la tua organizzazione.

## Risoluzione Problemi

### HTTP 401 Unauthorized

**Causa:** `system_key` o `system_secret` non corretti.

**Soluzioni:**
1. Verifica che le credenziali corrispondano a quelle memorizzate sul sistema
2. Assicurati che il sistema abbia completato la registrazione (vedi [Registrazione Sistema](05-system-registration.md))
3. Controlla eventuali spazi iniziali o finali nelle credenziali
4. Testa manualmente:
   ```bash
   curl -X POST https://my.nethesis.it/services/mimir/api/v1/push \
     -u "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE:my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0" \
     -H "Content-Type: application/x-protobuf" \
     --data-binary @/dev/null
   ```
   Una risposta `400 Bad Request` (non 401) conferma che l'autenticazione funziona.

### HTTP 500 Internal Server Error

**Causa:** Il backend Mimir non è raggiungibile o è configurato in modo errato.

**Soluzioni:**
1. Si tratta di un problema lato piattaforma — contatta il tuo amministratore
2. Controlla la pagina di stato della piattaforma o gli avvisi di monitoraggio
3. Riprova dopo qualche minuto; Mimir potrebbe essere in fase di riavvio

### Metriche Non Visibili in Grafana

**Causa:** Le metriche vengono inviate ma non sono ancora visibili.

**Soluzioni:**
1. Attendi 1–2 minuti — Mimir ha un ritardo di acquisizione
2. Verifica che `remote_write` sia abilitato in Prometheus e che la configurazione sia corretta
3. Controlla i log di Prometheus per errori di remote write:
   ```bash
   journalctl -u prometheus -n 50 | grep remote_write
   ```
4. Conferma di essere connesso a Grafana con un account appartenente all'organizzazione corretta

## Documentazione Correlata

- [Registrazione Sistema](05-system-registration.md)
- [Inventario e Heartbeat](06-inventory-heartbeat.md)
- [Gestione Sistemi](04-systems.md)
