---
sidebar_position: 5
---

# Esportazione Dati

La funzionalità di esportazione consente di scaricare i dati della piattaforma My in CSV o PDF, per analisi e reportistica.

## Panoramica

È possibile esportare i dati da qualsiasi elenco della piattaforma. L'esportazione rispetta i filtri attivi, quindi si può restringere il set di dati prima di esportarlo.

## Export Supportati

| Risorsa | Formati | Contenuto |
|---------|---------|-----------|
| **Distributori** | CSV, PDF | Elenco distributori con contatori di rivenditori, clienti e sistemi |
| **Rivenditori** | CSV, PDF | Elenco rivenditori con contatori di clienti e sistemi |
| **Clienti** | CSV, PDF | Elenco clienti con contatore dei sistemi |
| **Utenti** | CSV, PDF | Elenco utenti con ruoli e stato |
| **Sistemi** | CSV, PDF | Elenco sistemi con stato e ultimo heartbeat |
| **Applicazioni** | CSV, PDF | Elenco applicazioni con tipo, versione, sistema che le ospita e azienda |

## Come Esportare

1. Vai alla pagina di elenco della risorsa da esportare (es. **Utenti**, **Sistemi**)
2. Applica eventuali filtri -- l'esportazione contiene esattamente le righe selezionate dai filtri
3. Clicca **Esporta** e scegli il formato:
   - **CSV** -- tabulare, per fogli di calcolo e analisi dati
   - **PDF** -- documento, per stampa e condivisione
4. Il file viene generato e scaricato dal browser, con il nome dell'elenco e la data
   dell'esportazione: `utenti-2026-09-29.pdf`, `sistemi-2026-09-29.csv`

:::tip
Applica i filtri prima di esportare per ottenere esattamente i dati che ti servono. Ad esempio, filtra i sistemi per organizzazione o stato per esportarne solo un sottoinsieme, oppure filtra le applicazioni per un'azienda e tutta la sua gerarchia per esportare tutto ciò che è installato per quel partner.
:::

## Formato CSV

- **Separatore**: virgola (`,`)
- **Codifica**: UTF-8
- **Intestazioni**: la prima riga contiene i nomi delle colonne
- **Escape**: virgolette doppie sui campi che contengono una virgola

## Formato PDF

- **Tabella orizzontale**, una riga per record, con lo stesso impaginato per ogni risorsa
- **Intestazione** su ogni pagina con logo, numero di record, data e ora di generazione,
  **chi** ha eseguito l'esportazione e i **filtri** applicati
- Gli **stati** sono colorati (verde attivo, ambra inattivo o non assegnato,
  rosso sospeso o cancellato)
- I valori lunghi vengono troncati con i puntini così ogni record resta su una riga:
  il testo completo è nel CSV
- **Numero di pagina** a piè di pagina

## Limiti

- Massimo **10.000 record** per esportazione
- Oltre quella soglia l'esportazione viene **troncata silenziosamente**: il file
  viene comunque prodotto, ma le righe eccedenti non ci sono. Restringi i filtri
  per essere sicuro di avere tutto.

## Permessi

Per esportare serve il **permesso di lettura della risorsa**, lo stesso che
permette di vederne l'elenco:

| Risorsa | Permesso richiesto |
|---------|--------------------|
| Utenti | `read:users` |
| Sistemi | `read:systems` |
| Distributori | `read:distributors` |
| Rivenditori | `read:resellers` |
| Clienti | `read:customers` |
| Applicazioni | `read:applications` |

Se vedi un elenco puoi esportarlo -- e solo quello. Un utente Support, ad
esempio, non ha `read:users`, quindi non può esportare gli utenti.

I dati esportati rispettano sempre la visibilità gerarchica: l'Owner esporta
l'intera piattaforma, un distributore le proprie organizzazioni subordinate, un
rivenditore i propri clienti, un cliente solo la propria organizzazione. Non è
mai possibile esportare dati a cui non si ha accesso nell'interfaccia.
