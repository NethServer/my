---
sidebar_position: 5
---

# Esportazione Dati

La funzionalità di esportazione consente di scaricare i dati della piattaforma My in formati strutturati per analisi, reportistica e archiviazione.

## Panoramica

L'esportazione è disponibile per tutte le sezioni principali della piattaforma e supporta diversi formati di output.

## Export Supportati

| Risorsa | Formati | Descrizione |
|---------|---------|-------------|
| **Distributori** | CSV, PDF | Elenco distributori con dettagli |
| **Rivenditori** | CSV, PDF | Elenco rivenditori con dettagli |
| **Clienti** | CSV, PDF | Elenco clienti con dettagli |
| **Utenti** | CSV, PDF | Elenco utenti con ruoli e stato |
| **Sistemi** | CSV, PDF | Elenco sistemi con stato e ultimo heartbeat |

## Come Esportare

### Passo 1: Naviga alla Sezione

Vai alla pagina di elenco della risorsa che vuoi esportare (es. Sistemi, Utenti, ecc.).

### Passo 2: Applica i Filtri

Applica eventuali filtri per selezionare i dati da esportare. L'esportazione includerà solo i dati che corrispondono ai filtri attivi.

:::tip
Verifica i filtri attivi prima di esportare. L'esportazione include esattamente i dati visualizzati nell'elenco.
:::

### Passo 3: Seleziona il Formato

Clicca sul pulsante **Esporta** e seleziona il formato desiderato:

- **CSV** - Formato tabulare, ideale per fogli di calcolo (Excel, Google Sheets)
- **PDF** - Formato documento, ideale per stampa e archiviazione

### Passo 4: Scarica il File

Il file viene generato e scaricato automaticamente nel browser.

### Passo 5: Verifica

Apri il file scaricato per verificare che i dati siano corretti e completi.

## Formato CSV

Il file CSV utilizza:

- **Separatore**: virgola (`,`)
- **Codifica**: UTF-8
- **Intestazioni**: Prima riga con i nomi delle colonne
- **Escape**: Virgolette doppie per campi contenenti virgole

## Formato PDF

Il file PDF include:

- **Intestazione** con data e ora di generazione
- **Tabella** con i dati formattati
- **Piè di pagina** con numero di pagina

## Limiti

| Parametro | Valore |
|-----------|--------|
| Record massimi per export | 10.000 |
| Timeout generazione | 60 secondi |

:::warning
Se i dati superano il limite di 10.000 record, applica filtri aggiuntivi per ridurre il set di dati prima dell'esportazione. I record oltre il limite non vengono inclusi nel file esportato.
:::

## Permessi

L'esportazione è disponibile per tutti gli utenti autenticati. I dati esportati sono filtrati in base alla visibilità gerarchica dell'utente:

- **Owner**: Esporta tutti i dati della piattaforma
- **Distributore**: Esporta i dati delle proprie organizzazioni subordinate
- **Rivenditore**: Esporta i dati dei propri clienti
- **Cliente**: Esporta solo i dati della propria organizzazione

:::note
I dati esportati rispettano sempre la visibilità gerarchica. Non è possibile esportare dati a cui non si ha accesso nell'interfaccia.
:::
