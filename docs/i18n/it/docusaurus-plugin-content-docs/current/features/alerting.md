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
- Configurare notifiche email, webhook e Telegram per la tua organizzazione
- Marcare ogni destinatario con le severità da ricevere (`critical`, `warning`, `info`, oppure tutte)
- Scegliere lingua e formato del corpo email per ogni destinatario
- Consultare lo storico degli allarmi di ciascun sistema

## Accesso

La pagina Alerting è accessibile dal menu laterale alla voce **Alerting**. I due tab hanno permessi diversi:

- Tab **Allarmi** — richiede `read:systems` (admin, support, reader). Per creare o rimuovere silences serve `manage:systems`.
- Tab **Configurazione alerting** — richiede `read:alerts` (solo admin/super); `manage:alerts` per salvare o rimuovere una configurazione.

## Selezione organizzazione

Il selettore in cima al tab **Allarmi** è usato dall'Owner per filtrare la lista degli allarmi per tenant. Il tab **Configurazione alerting** invece opera **sempre** sulla propria organizzazione — non mostra mai la configurazione di un'altra organizzazione, indipendentemente dal valore del selettore.

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

## Lavorare sugli allarmi

Ogni allarme attivo offre un set di strumenti di collaborazione, disponibili dalla riga dell'allarme:

### Presa in carico

**Assegna a me** indica che stai lavorando sull'allarme. Un solo assegnatario per allarme: assegnarsi un allarme già preso da qualcun altro esegue un **subentro** (il precedente assegnatario resta registrato nella timeline delle attività). Non esiste la rimozione manuale dell'assegnazione: viene **rilasciata automaticamente** quando l'allarme si risolve. L'assegnatario corrente è mostrato sulla riga dell'allarme, così il team sa sempre chi se ne sta occupando.

L'assegnazione è solo self-service (assegni sempre te stesso) e richiede `manage:systems`.

### Silenziamenti

Un allarme può essere **silenziato** per una durata a scelta con un commento opzionale: le notifiche si fermano mentre l'allarme resta visibile come silenziato. Modificando il silenziamento se ne aggiorna la scadenza o il commento; rimuovendolo le notifiche riprendono. I silenziamenti richiedono `manage:systems`.

### Note

È possibile aggiungere **note** libere a un allarme in qualsiasi momento, indipendentemente dai silenziamenti — utili per registrare osservazioni o passaggi di consegne. Le note compaiono nella timeline come eventi `note_added`.

### Timeline delle attività

Ogni allarme conserva una timeline con l'intera cronologia di collaborazione: silenziato / silenziamento aggiornato / riattivato, assegnato / rilasciato (incluso il rilascio automatico alla risoluzione) e note, ciascuno con autore e data.

## Configurazione alerting

Il tab **Configurazione alerting** ti permette di definire chi viene notificato quando un allarme scatta nella tua organizzazione. Quello che salvi qui è il tuo **layer** — il server lo unisce ai layer di tutte le organizzazioni sopra di te nella gerarchia (Owner → Distributor → Reseller → Customer) e pusha la YAML Alertmanager risultante a Mimir.

### Cosa vedi tu vs. cosa vede Mimir

Quello che vedi in questo tab è sempre **e solo** il tuo layer. Non vedi mai i layer delle organizzazioni sopra di te, e le organizzazioni sotto di te non vedono mai il tuo. La configurazione effettiva mergiata viene calcolata server-side al momento del render e non lascia mai il backend; non oltrepassa mai il confine del tenant.

Questo isolamento è intenzionale: confina URL dei webhook, token Telegram e indirizzi email destinatari all'organizzazione che li ha digitati.

### Contratto additivo

I layer sono additivi. Un discendente può **aggiungere** destinatari su quanto configurato dall'antenato, ma non può rimuovere o disabilitare canali che un antenato ha abilitato. Per esempio:

- L'Owner abilita `email` globalmente e aggiunge `noc@msp.example` → ogni tenant sotto eredita entrambi.
- Un Reseller può aggiungere `noc@reseller.example` sopra — ora entrambi gli indirizzi ricevono gli allarmi che li riguardano.
- Lo stesso Reseller **non può** spegnere `email` per il proprio sottoalbero. Solo l'Owner può disabilitare globalmente un canale; per i ruoli non-Owner, un `false` esplicito su un toggle viene normalizzato a `null` ("nessuna opinione") al salvataggio.

### Forma della configurazione

Il layer è un oggetto JSON flat con tre toggle di canale e tre liste di destinatari:

```json
{
  "enabled": { "email": true, "webhook": null, "telegram": null },
  "email_recipients": [
    { "address": "noc@org.example", "severities": ["critical","warning"], "language": "it", "format": "html" }
  ],
  "webhook_recipients": [
    { "name": "ops-slack", "url": "https://hooks.slack.com/services/T000/B000/XXX", "severities": ["critical"] }
  ],
  "telegram_recipients": [
    { "bot_token": "123456789:ABCDEFabcdef...", "chat_id": -1001234567890, "severities": [] }
  ]
}
```

#### Toggle dei canali (`enabled`)

Ogni canale è tri-stato:

| Valore | Significato |
|--------|-------------|
| `true` | Canale abilitato a questo layer |
| `false` | Canale disabilitato a questo layer (solo Owner; per i non-Owner `false` viene normalizzato a `null` al salvataggio) |
| `null` | Nessuna opinione a questo layer; lo stato effettivo eredita da un eventuale antenato che ha preso posizione. Se nessun layer abilita il canale, resta off. |

#### Destinatari email (`email_recipients`)

| Campo | Tipo | Descrizione |
|-------|------|-------------|
| `address` | string | Indirizzo email che riceve la notifica |
| `severities` | string[] | Sottoinsieme di `["critical","warning","info"]`. Array vuoto = "tutte le severità" |
| `language` | string | `en` o `it`. Determina lingua del subject e del body del template renderizzato |
| `format` | string | `html` (default, multipart HTML primario + text fallback) o `plain` (body solo testo) |

#### Destinatari webhook (`webhook_recipients`)

| Campo | Tipo | Descrizione |
|-------|------|-------------|
| `name` | string | Etichetta descrittiva del target (mostrata nella UI) |
| `url` | string | Endpoint HTTPS/HTTP. Validato lato server: loopback, RFC1918, RFC6598 CGNAT, link-local, multicast e metadata cloud sono rifiutati |
| `severities` | string[] | Stessa semantica delle email |

#### Destinatari Telegram (`telegram_recipients`)

| Campo | Tipo | Descrizione |
|-------|------|-------------|
| `bot_token` | string | Token ottenuto da `@BotFather` |
| `chat_id` | integer | Id numerico della chat (positivo per utenti, negativo per gruppi/canali) |
| `severities` | string[] | Stessa semantica delle email |

### Scope per severità

L'array `severities` di ciascun destinatario controlla quali severità riceve:

- **Vuoto (`[]`)** — il destinatario riceve **ogni** severità. È il default per un indirizzo "catch-all".
- **Sottoinsieme (es. `["critical"]`)** — il destinatario riceve **solo** quelle severità.

Mimir Alertmanager espande un receiver per severità (`severity-critical-receiver`, `severity-warning-receiver`, `severity-info-receiver`); un destinatario con `severities=[]` finisce in tutti e tre.

### Merge nella gerarchia

Quando il server renderizza la configurazione effettiva per un tenant, percorre la catena da Owner fino al tenant e unisce i layer con queste regole:

- **Toggle dei canali** — OR logico: se un qualunque layer della catena abilita un canale, il canale è on per il tenant.
- **Destinatari** — union con dedup stabile. Chiavi di dedup: `address` per email, `url` per webhook, `(bot_token, chat_id)` per Telegram. In caso di collisione di `language` o `format`, vince la prima occorrenza (più vicina all'Owner).
- **`severities` per destinatario** — union; se una qualunque copia che contribuisce ha `severities=[]` (tutte), la copia mergiata si allarga a `[]` (lo scope più largo vince sempre).

### Esempio: layer di livello customer che aggiunge notifiche

Supponi che l'Owner abbia già abilitato l'email con `noc@msp.example` per tutte le severità. Un Customer aggiunge questo layer per la propria organizzazione:

```json
{
  "enabled": { "email": null, "webhook": null, "telegram": null },
  "email_recipients": [
    { "address": "oncall@customer.example", "severities": ["critical"], "language": "en", "format": "plain" },
    { "address": "manager@customer.example", "severities": [], "language": "it", "format": "html" }
  ],
  "webhook_recipients": [],
  "telegram_recipients": [
    { "bot_token": "123456789:ABCDEFabcdef...", "chat_id": -1001234567890, "severities": ["critical","warning"] }
  ]
}
```

Cosa consegna Mimir per questo customer:

- `oncall@customer.example` riceve **solo** gli allarmi critical come email in inglese plain text.
- `manager@customer.example` riceve **tutti** gli allarmi come email HTML in italiano.
- `noc@msp.example` dell'Owner continua a ricevere tutti gli allarmi (ereditato).
- La chat Telegram riceve gli allarmi **critical e warning** (il toggle `telegram` dell'Owner non era on, quindi questi `telegram_recipients` non scattano finché un antenato non abilita il canale Telegram — solo l'Owner può abilitare globalmente un canale).

### Salvare e rimuovere un layer

- **Salva configurazione** persiste il tuo layer e triggera un re-render + push a Mimir per ogni tenant della tua gerarchia. La response riporta `affected_tenants` e `propagated_to`; eventuali fallimenti per tenant compaiono come warning senza rollback del salvataggio (Mimir può essere riconciliato salvando di nuovo).
- **Rimuovi questa configurazione** cancella il tuo layer per intero. Il tuo contributo sparisce dalla config mergiata; i layer degli antenati (Owner / Distributor / Reseller) restano intatti e continuano a scattare. Per silenziare completamente un tenant, ogni layer della sua catena deve eliminare il proprio contributo.

## Allarmi a livello di sistema

Nella pagina di dettaglio di ciascun sistema trovi due widget aggiuntivi:

:::note
`LinkFailed` è l'alert interno per l'heartbeat creato da Collect. Segue il timeout heartbeat configurato (10 minuti di default), separato dalla soglia di stato del sistema usata in Sistemi, e può restare attivo fino a 10 minuti dopo che il sistema torna a inviare heartbeat.
:::

### Card Allarmi attivi

Mostra gli allarmi attualmente attivi per quel sistema specifico, filtrati per la system key. Ogni voce mostra nome allarme, severità, stato, riepilogo e ora di inizio. Se il sistema non ha allarmi attivi, viene mostrato un messaggio di stato vuoto.

### Pannello Storico allarmi

Mostra una tabella paginata degli allarmi risolti per il sistema, con colonne per nome, severità, stato, riepilogo, inizio e fine. Lo storico viene recuperato dal database locale dove gli allarmi risolti vengono salvati tramite i webhook di Alertmanager.

Puoi cambiare la dimensione della pagina (5, 10, 25, 50, 100) e navigare tra le pagine usando i controlli di paginazione in fondo alla tabella.

## Notifiche email

Quando le notifiche email sono abilitate, gli allarmi vengono consegnati da Alertmanager usando template personalizzati per la piattaforma. Ogni email include:

- Il nome e la severità dell'allarme
- La system key e l'eventuale label service
- Riepilogo e descrizione localizzati (nella lingua scelta per ciascun destinatario)
- Il timestamp di firing o risoluzione
- Un pulsante **Visualizza sistema** che linka direttamente alla pagina di dettaglio del sistema

I template sono disponibili in **inglese** e **italiano**. La lingua viene scelta **per destinatario** tramite il campo `language` di ogni voce in `email_recipients[]` — destinatari diversi della stessa organizzazione possono ricevere rendering in lingue diverse. Allo stesso modo, ogni destinatario sceglie il proprio `format`: `html` per un corpo multipart con HTML primario, `plain` per un corpo solo testo (utile per sistemi di ticketing o bridge mail-to-chat).

## Notifiche Telegram

Quando le notifiche Telegram sono abilitate, gli allarmi vengono inviati come messaggi formattati a un bot Telegram. I messaggi usano la formattazione HTML e includono nome allarme, severità, system key e riepilogo localizzato.

:::note
I messaggi Telegram sono limitati a 4096 caratteri. Per descrizioni di allarme molto lunghe, il messaggio potrebbe essere troncato. Per allarmi con metadati estesi, considera di usare email o webhook.
:::

### Passaggio 1 — Crea un bot Telegram

1. Apri Telegram e avvia una conversazione con **[@BotFather](https://t.me/BotFather)**
2. Invia il comando `/newbot`
3. Segui le istruzioni: scegli un nome visualizzato e un username univoco (deve terminare con `bot`, es. `MyAlertsBot`)
4. BotFather risponde con un **bot token** nel formato `123456789:ABCDEFabcdef...` — copialo

### Passaggio 2 — Ottieni il chat ID

Il `chat_id` è l'identificatore numerico della destinazione (un utente privato, un gruppo o un canale).

**Per una chat privata con te stesso o un utente specifico:**

1. Apri Telegram e avvia una conversazione con il tuo nuovo bot (cerca il suo username)
2. Invia qualsiasi messaggio al bot (es. `/start`)
3. Apri il seguente URL nel browser, sostituendo `<BOT_TOKEN>` con il tuo token:

   ```
   https://api.telegram.org/bot<BOT_TOKEN>/getUpdates
   ```
   Eventualmente, potresti trovare il `chat_id` anche nell'URL della conversazione con il bot, nel formato `https://web.telegram.org/z/#-<CHAT_ID>` (nota il segno negativo per le chat private)
4. Trova il campo `"id"` all'interno dell'oggetto `"chat"` nella risposta JSON — quello è il tuo `chat_id` (un intero positivo, es. `123456789`)

**Per un gruppo o canale:**

1. Aggiungi il tuo bot al gruppo o canale come **amministratore**
2. Invia un messaggio nel gruppo in modo che Alertmanager abbia qualcosa da leggere
3. Chiama `getUpdates` come sopra — il `chat_id` per gruppi e canali è un numero **negativo** (es. `-1001234567890`). Eventualmente, potresti trovare il `chat_id` anche nell'URL della conversazione con il bot, nel formato `https://web.telegram.org/z/#-<CHAT_ID>` (nota il segno negativo per gruppi/canali)

### Passaggio 3 — Aggiungi il destinatario Telegram al tuo layer

Aggiungi una voce a `telegram_recipients`; abilitare il canale con `enabled.telegram = true` è necessario solo a livello Owner (il canale si propaga additivamente verso il basso).

| Campo | Tipo | Descrizione |
|-------|------|-------------|
| `bot_token` | string | Il token fornito da BotFather |
| `chat_id` | integer | L'ID numerico della chat Telegram (positivo per utenti, negativo per gruppi/canali) |
| `severities` | string[] | Sottoinsieme di `["critical","warning","info"]`. Array vuoto = tutte le severità |

Esempio (layer Owner che abilita Telegram per tutto l'albero):

```json
{
  "enabled": { "email": null, "webhook": null, "telegram": true },
  "email_recipients": [],
  "webhook_recipients": [],
  "telegram_recipients": [
    { "bot_token": "123456789:ABCDEFabcdef...", "chat_id": -1001234567890, "severities": [] }
  ]
}
```

È possibile definire più destinatari per inviare gli allarmi a più bot o chat contemporaneamente. I messaggi Telegram sono attualmente sempre renderizzati in inglese.

## Argomenti correlati

- [Gestione Sistemi](../systems/management.md)
- [Registrazione Sistema](../systems/registration.md)
- Documentazione per sviluppatori: [Guida integrazione alerting](https://github.com/NethServer/my/blob/main/services/mimir/docs/alerting-it.md) (per integrare nuovi sistemi con l'API Alertmanager)
