---
sidebar_position: 7
---

# Alerting

Configura le notifiche di allarme e monitora i problemi attivi su tutte le tue organizzazioni e sistemi.

:::info ALPHA
L'interfaccia di alerting è attualmente in alpha. Alcune funzionalità e schermate potrebbero cambiare nelle versioni future.
:::

## Panoramica

La funzionalità di Alerting fornisce una vista centralizzata di tutti gli allarmi attivi dai tuoi sistemi gestiti e ti permette di configurare come vengono consegnate le notifiche. Si integra con Grafana Mimir Alertmanager per una gestione degli allarmi affidabile e multi-tenant.

Dalla pagina Alerting puoi:

- Visualizzare gli allarmi attivi filtrati per stato, severità o sistema specifico
- Configurare notifiche email e webhook per ogni organizzazione
- Definire regole di notifica personalizzate per severità e per sistema
- Consultare lo storico degli allarmi di ciascun sistema

## Accesso

La pagina Alerting è accessibile dal menu laterale alla voce **Alerting**. L'accesso richiede il permesso `read:systems` per visualizzare gli allarmi e `manage:systems` per modificare la configurazione.

## Selezione organizzazione

Poiché l'alerting si configura per organizzazione, la pagina include un selettore in alto. Scegli l'organizzazione cliente di cui vuoi gestire allarmi e configurazione. Il selettore elenca solo le organizzazioni non-owner disponibili nella tua gerarchia.

## Allarmi attivi

Il tab **Allarmi** mostra tutti gli allarmi attualmente attivi per l'organizzazione selezionata, recuperati in tempo reale da Alertmanager.

### Campi dell'allarme

Ogni allarme mostra:

| Campo | Descrizione |
|-------|-------------|
| **Nome allarme** | Identificatore del tipo di allarme (es. `DiskFull`, `BackupFailed`) |
| **Severità** | Badge colorato: `critical` (rosso), `warning` (arancione), `info` (blu) |
| **Stato** | Stato corrente: `active`, `suppressed` o `unprocessed` |
| **Sistema** | Sistema che ha generato l'allarme |
| **Iniziato il** | Timestamp di quando l'allarme è stato scatenato |
| **Riepilogo** | Descrizione leggibile dalle annotazioni dell'allarme |
| **Label** | Metadata aggiuntivi in formato chiave-valore |

### Livelli di severità

| Livello | Significato | Uso tipico |
|---------|-------------|------------|
| **Critical** | Sistema down o perdita dati imminente | Azione immediata richiesta |
| **Warning** | Stato degradato o soglia in avvicinamento | Azione necessaria a breve |
| **Info** | Evento informativo | Nessuna azione immediata |

### Filtri

Puoi restringere la lista degli allarmi usando i filtri in cima alla pagina:

- **Filtro stato**: mostra solo allarmi in uno stato specifico (active, suppressed, unprocessed)
- **Filtro severità**: mostra solo allarmi che corrispondono ai livelli di severità selezionati
- **Ricerca system key**: campo di ricerca libera per filtrare gli allarmi di un sistema specifico

Clicca **Reset filtri** per rimuovere tutti i filtri attivi, oppure **Aggiorna** per ricaricare manualmente la lista.

## Configurazione alerting

Il tab **Configurazione** ti permette di definire come vengono instradati gli allarmi ai destinatari. La configurazione viene pushata ad Alertmanager e persiste finché non la cambi.

### Visualizzare la configurazione

La configurazione viene mostrata in due modalità:

- **Vista strutturata**: sezioni organizzate che mostrano le impostazioni mail e webhook correnti, le override per severità e per sistema in formato leggibile
- **Vista YAML raw**: la configurazione Alertmanager completa in YAML, con i campi sensibili (credenziali SMTP, token webhook) automaticamente offuscati. Clicca **Copia YAML** per copiare l'intera configurazione negli appunti.

Se non esiste ancora alcuna configurazione, la pagina mostra un messaggio "Nessuna configurazione trovata" con un pulsante **Modifica configurazione** per creare l'impostazione iniziale.

### Campi di configurazione

La configurazione viene modificata come oggetto JSON con i seguenti campi:

#### Impostazioni globali

| Campo | Tipo | Descrizione |
|-------|------|-------------|
| `mail_enabled` | boolean | Abilita o disabilita le notifiche email globalmente |
| `mail_addresses` | string[] | Lista di indirizzi email che ricevono tutti gli allarmi |
| `webhook_enabled` | boolean | Abilita o disabilita le notifiche webhook globalmente |
| `webhook_receivers` | object[] | Lista di endpoint webhook, ciascuno con `name` e `url` |
| `email_template_lang` | string | Lingua dei template email: `en` o `it` (default: `en`) |

#### Override per severità

Il campo `severities` ti permette di personalizzare il comportamento delle notifiche per ciascun livello di severità. Utile quando vuoi che gli allarmi critici raggiungano destinatari diversi rispetto a quelli informativi.

Ogni override include:

- `severity`: uno tra `critical`, `warning`, `info`
- `mail_enabled` (opzionale): override dell'impostazione email globale per questa severità
- `webhook_enabled` (opzionale): override dell'impostazione webhook globale
- `mail_addresses` (opzionale): lista di indirizzi email per questa severità
- `webhook_receivers` (opzionale): lista di webhook receiver per questa severità

Se la lista degli indirizzi di un override è vuota, vengono usati gli indirizzi globali come fallback.

#### Override per sistema

Il campo `systems` ti permette di personalizzare il comportamento per sistemi specifici. Utile quando sistemi diversi devono notificare team diversi.

Ogni override include:

- `system_key`: l'identificatore del sistema target
- `mail_enabled` (opzionale): override per questo sistema
- `webhook_enabled` (opzionale): override per questo sistema
- `mail_addresses` (opzionale): destinatari aggiuntivi per gli allarmi di questo sistema
- `webhook_receivers` (opzionale): webhook aggiuntivi per gli allarmi di questo sistema

### Priorità delle override

Quando si instrada un allarme, la priorità è:

1. **Override per sistema** (la più specifica)
2. **Override per severità**
3. **Impostazioni globali** (fallback)

### Esempio di configurazione

```json
{
  "mail_enabled": true,
  "webhook_enabled": false,
  "mail_addresses": ["ops@example.com"],
  "webhook_receivers": [],
  "email_template_lang": "it",
  "severities": [
    {
      "severity": "critical",
      "mail_addresses": ["oncall@example.com", "ops@example.com"]
    },
    {
      "severity": "info",
      "mail_enabled": false
    }
  ],
  "systems": [
    {
      "system_key": "NETH-ABCD-1234",
      "mail_addresses": ["platform-team@example.com"]
    }
  ]
}
```

In questo esempio:

- Tutti gli allarmi warning vanno a `ops@example.com`
- Gli allarmi critical vanno sia a `oncall@example.com` che a `ops@example.com`
- Gli allarmi info sono soppressi
- Gli allarmi dal sistema `NETH-ABCD-1234` vanno anche a `platform-team@example.com`
- I template email vengono renderizzati in italiano

### Modificare la configurazione

1. Clicca **Modifica configurazione** nella vista strutturata
2. Modifica il JSON nell'editor
3. Clicca **Salva configurazione** — il JSON non valido viene rifiutato con un errore di validazione
4. Al successo, la vista si aggiorna e appare una notifica di conferma

Per annullare senza salvare, clicca **Annulla**.

### Disabilitare tutti gli allarmi

In fondo alla pagina di configurazione trovi l'azione **Disabilita tutti gli allarmi**. Questa sostituisce la configurazione corrente con un routing "blackhole" che silenzia tutte le notifiche per l'organizzazione, senza perdere permanentemente la configurazione precedente — puoi ricrearla modificando di nuovo la configurazione.

Cliccando appare un passaggio di conferma prima dell'esecuzione.

## Allarmi a livello di sistema

Nella pagina di dettaglio di ciascun sistema trovi due widget aggiuntivi:

### Card Allarmi attivi

Mostra gli allarmi attualmente attivi per quel sistema specifico, filtrati per la system key. Ogni voce mostra nome allarme, severità, stato, riepilogo e ora di inizio. Se il sistema non ha allarmi attivi, viene mostrato un messaggio di stato vuoto.

### Pannello Storico allarmi

Mostra una tabella paginata degli allarmi risolti per il sistema, con colonne per nome, severità, stato, riepilogo, inizio e fine. Lo storico viene recuperato dal database locale dove gli allarmi risolti vengono salvati tramite i webhook di Alertmanager.

Puoi cambiare la dimensione della pagina (5, 10, 25, 50, 100) e navigare tra le pagine usando i controlli di paginazione in fondo alla tabella.

## Notifiche email

Quando le notifiche email sono abilitate, gli allarmi vengono consegnati da Alertmanager usando template personalizzati per la piattaforma. Ogni email include:

- Il nome e la severità dell'allarme
- La system key e l'eventuale label service
- Riepilogo e descrizione localizzati (in base al `email_template_lang` configurato)
- Il timestamp di firing o risoluzione
- Un pulsante **Visualizza sistema** che linka direttamente alla pagina di dettaglio del sistema

I template sono disponibili in **inglese** e **italiano**, selezionati tramite il campo `email_template_lang`.

## Argomenti correlati

- [Gestione Sistemi](../systems/management.md)
- [Registrazione Sistema](../systems/registration.md)
- Documentazione per sviluppatori: [Guida integrazione alerting](https://github.com/NethServer/my/blob/main/services/mimir/docs/alerting-it.md) (per integrare nuovi sistemi con l'API Alertmanager)
