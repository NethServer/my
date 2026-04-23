---
sidebar_position: 4
---

# Backup di configurazione

Ogni sistema registrato — NethServer 8 (NS8) e NethSecurity — carica ogni giorno un'istantanea cifrata della propria configurazione su MY. Questa pagina descrive che cosa viene conservato, come i dati sono protetti e come gli operatori interagiscono col sottosistema di backup.

## Panoramica

I backup sono **cifrati end-to-end sull'appliance** prima dell'upload. MY memorizza il ciphertext insieme a un insieme ridotto di metadati (dimensione, SHA-256, timestamp di upload, versione dell'appliance) su un object store S3-compatible; il server non vede mai il plaintext e non è in grado di decifrare il contenuto.

```
┌──────────────┐   blob GPG-cifrato       ┌──────────────┐    PutObject    ┌──────────────┐
│  Appliance   │ ───────────────────────► │   collect    │ ───────────────►│   Bucket S3  │
│ (NS8 / NSEC) │   POST /systems/backups  │  (ingest)    │                 │ (DO Spaces,  │
└──────────────┘   HTTP Basic auth        └──────────────┘                 │  AWS S3, …)  │
                                                                           └──────▲───────┘
                                                                                  │
                                                                                  │ presigned URL
                                                                                  │
                                                                          ┌──────────────┐
                                                                          │   backend    │
                                                                          │  (list/read) │
                                                                          └──────────────┘
```

## Autenticazione e controllo accessi

Gli upload usano **HTTP Basic auth** con la stessa coppia `system_key:system_secret` che l'appliance usa già per inventario e heartbeat (vedi [registrazione del sistema](registration)). Un sistema può scrivere o leggere solo il proprio prefisso sul bucket — l'accesso cross-tenant viene rifiutato lato server.

Le letture da parte degli utenti passano per `backend` con il normale JWT Logto e obbediscono alle stesse regole RBAC di `GET /systems/:id`: un utente vede i backup di un sistema soltanto se l'organizzazione a cui appartiene ne è proprietaria.

## Layout dello storage

Ogni oggetto di backup ha la chiave:

```
{org_id}/{system_key}/{backup_id}.{ext}
```

- `org_id` è l'organizzazione che possiede il sistema.
- `system_key` è l'identificatore utente stabile (`NETH-…`) con cui l'appliance si autentica — scelto rispetto all'UUID interno per permettere agli operatori di riconoscere a colpo d'occhio ogni sistema sfogliando il bucket.
- `backup_id` è uno UUIDv7 ordinabile nel tempo, generato al momento dell'upload.
- `ext` riflette il formato di compressione/cifratura rilevato dal filename che l'appliance invia (`.tar.gz`, `.tar.xz`, `.gpg`, `.bin`).

I metadati per oggetto viaggiano come header `x-amz-meta-*` standard:

| Header                   | Significato                                                              |
|--------------------------|--------------------------------------------------------------------------|
| `x-amz-meta-sha256`      | SHA-256 del blob cifrato, calcolato da `collect` durante lo streaming.   |
| `x-amz-meta-filename`    | Nome user-facing fornito dall'appliance tramite l'header `X-Filename`.   |
| `x-amz-meta-uploader-ip` | Indirizzo del peer visto da `collect` — non falsificabile via proxy.     |

## Retention e quote

L'ingest applica tre limiti indipendenti per sistema:

| Parametro                          | Default     | Significato                                        |
|------------------------------------|-------------|----------------------------------------------------|
| `BACKUP_MAX_PER_SYSTEM`            | 10          | Numero massimo di backup mantenuti per sistema.    |
| `BACKUP_MAX_SIZE_PER_SYSTEM`       | 500&nbsp;MB | Totale byte massimo per sistema.                   |
| `BACKUP_MAX_UPLOAD_SIZE`           | 2&nbsp;GB   | Limite duro per singolo upload.                    |

È attivo anche un tetto per-organizzazione:

| Parametro                      | Default    | Significato                                                   |
|--------------------------------|------------|---------------------------------------------------------------|
| `BACKUP_MAX_SIZE_PER_ORG`      | 100&nbsp;GB | Totale byte aggregato su tutti i sistemi della stessa organizzazione. Impostare a `0` per disabilitare (viene loggato un warning all'avvio). |

Quando una delle soglie (count o size) viene superata, l'oggetto più vecchio sotto il prefisso del sistema viene eliminato finché il backup non rientra nei limiti. La pruning è serializzata da un lock Redis: upload concorrenti dallo stesso appliance non possono mai correre sullo stesso oggetto vittima.

Gli upload dell'appliance sono inoltre sottoposti a rate limit per sistema (default 6 al minuto, 60 all'ora). NS8 e NethSecurity in produzione caricano su timer giornaliero e non raggiungono mai queste soglie; il limite serve a contenere abusi di tipo flood.

## Eliminazione e GDPR

Un **soft delete** lascia i backup al loro posto: la riga del sistema viene marcata con `deleted_at` ma può ancora essere ripristinata dalla UI, e i suoi backup devono sopravvivere perché il ripristino sia utile. Un **hard destroy** è irreversibile ed esegue un'erasure GDPR-compliant: ogni oggetto sotto il prefisso `{org_id}/{system_key}/` del sistema viene rimosso dal bucket prima che la riga nel database venga cancellata, sia che il sistema sia stato soft-deleted in precedenza sia che no. Se il cleanup dello storage fallisce, il destroy viene rifiutato in modo che l'operatore possa riprovare; nessun ciphertext orfano rimane mai sotto il prefisso di un sistema distrutto. Le modifiche delle credenziali (rotazione del secret, soft delete) invalidano ogni cache di auth su `collect` entro un secondo, tramite un bus Redis pub/sub cross-service.

## Gestione dei backup

Endpoint API esposti dal `backend` per gli amministratori:

| Metodo | Path                                                      | Scopo                                              |
|--------|-----------------------------------------------------------|----------------------------------------------------|
| `GET`  | `/api/systems/:id/backups`                                | Elenca tutti i backup del sistema, con contatori d'uso. |
| `GET`  | `/api/systems/:id/backups/:backup_id/download`            | Restituisce una URL di download presigned (TTL 5 min). |
| `DELETE` | `/api/systems/:id/backups/:backup_id`                   | Elimina un singolo backup.                         |

Le URL presigned vengono generate lato server e non veicolano alcuna autenticazione — trattarle come bearer token a vita breve e non condividerle.

La UI per listare, scaricare ed eliminare i backup si trova nella vista di dettaglio del sistema.

## Riferimenti

- [Registrazione del sistema](registration) — come un'appliance ottiene le credenziali usate per gli upload di backup.
- [`collect/README.md`](https://github.com/NethServer/my/blob/main/collect/README.md) — configurazione dello storage (`BACKUP_S3_*`) e una ricetta `curl` per simulare un upload appliance.
- [`backend/README.md`](https://github.com/NethServer/my/blob/main/backend/README.md) — stessa configurazione lato letture.
