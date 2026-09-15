---
sidebar_position: 2
---

# Applicazioni

Gestione delle applicazioni rilevate dall'inventario dei sistemi e assegnate alle organizzazioni.

## Panoramica

Le applicazioni su My rappresentano istanze software (come NethVoice, NethSecurity, WebTop) rilevate attraverso i dati di inventario dei sistemi. Ogni applicazione è legata a un sistema e può essere assegnata a un'organizzazione per finalità di gestione.

## Visualizzazione Applicazioni

### Elenco

Vai su **Applicazioni** per vedere l'elenco di tutte le applicazioni visibili. L'elenco mostra:

- Tipo di applicazione (es. NethVoice, NethSecurity, WebTop)
- Versione
- Sistema associato
- Organizzazione

### Filtri

Usa i filtri disponibili per restringere l'elenco:

- **Tipo**: filtra per tipo di applicazione (NethVoice, NethSecurity, WebTop, ecc.)
- **Versione**: filtra per versione specifica
- **Sistema**: filtra per il sistema a cui appartiene l'applicazione
- **Organizzazione**: filtra per organizzazione. Un'applicazione non assegnata appartiene all'organizzazione del sistema che la ospita, quindi le applicazioni di un'azienda compaiono già prima di qualsiasi assegnazione; la stessa regola guida i contatori delle applicazioni nelle pagine delle organizzazioni

## Dettagli Applicazione

Cliccando su un'applicazione si accede alle sue informazioni di dettaglio:

- **Tipo**: il genere di applicazione (es. NethVoice, NethSecurity)
- **Versione**: la versione installata
- **Sistema associato**: il sistema su cui l'applicazione è in esecuzione
- **Organizzazione**: l'organizzazione a cui l'applicazione appartiene

## Assegnazione alle Organizzazioni

L'assegnazione controlla quale organizzazione ha visibilità e gestione su quell'applicazione.

:::note
Un'applicazione è assegnata a **una sola** organizzazione alla volta, non a più organizzazioni. Assegnarla a un'altra sostituisce l'assegnazione precedente.
:::

### Assegnare un'Applicazione

1. Vai al dettaglio dell'applicazione
2. Usa l'azione **Assegna**
3. Seleziona l'organizzazione di destinazione
4. Conferma l'assegnazione

Le organizzazioni proposte sono filtrate in base alla tua posizione nella gerarchia: puoi assegnare applicazioni solo alle organizzazioni che gestisci.

### Rimuovere l'Assegnazione

1. Vai al dettaglio dell'applicazione
2. Usa l'azione **Rimuovi assegnazione**
3. Conferma la rimozione

Un'applicazione senza assegnazione non sparisce: torna a essere conteggiata sull'organizzazione del sistema che la ospita.

## Note Applicazione

È possibile aggiungere note a un'applicazione per registrare contesto o informazioni operative:

1. Vai al dettaglio dell'applicazione
2. Individua la sezione **Note**
3. Aggiungi o modifica le note
4. Salva le modifiche

## Totali e Trend

La [Dashboard](./dashboard.md) mostra il totale delle applicazioni visibili al tuo account, con un badge che porta a quelle non assegnate. I dati di crescita nel tempo sono disponibili da API tramite `/backend/api/applications/trend`.

## Permessi

| Operazione | Permesso | Staff | Admin | Backoffice | Support | Reader |
|------------|----------|:-----:|:-----:|:----------:|:-------:|:------:|
| Visualizza applicazioni | `read:applications` | Sì | Sì | Sì | Sì | Sì |
| Modifica un'applicazione | `manage:applications` | Sì | Sì | Sì | Sì | No |
| Assegna / rimuovi da un'organizzazione | `manage:applications` | Sì | Sì | Sì | Sì | No |
| Modifica note | `manage:applications` | Sì | Sì | Sì | Sì | No |

Tutti i ruoli tranne Reader hanno `manage:applications`, quindi assegnare
un'applicazione non è un'operazione riservata al Backoffice.
