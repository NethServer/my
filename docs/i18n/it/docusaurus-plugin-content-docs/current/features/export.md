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
| **Distributori** | CSV, PDF | Elenco distributori con dettagli |
| **Rivenditori** | CSV, PDF | Elenco rivenditori con dettagli |
| **Clienti** | CSV, PDF | Elenco clienti con dettagli |
| **Utenti** | CSV, PDF | Elenco utenti con ruoli e stato |
| **Sistemi** | CSV, PDF | Elenco sistemi con stato e ultimo heartbeat |

## Come Esportare

1. Vai alla pagina di elenco della risorsa da esportare (es. **Utenti**, **Sistemi**)
2. Applica eventuali filtri -- l'esportazione contiene esattamente le righe selezionate dai filtri
3. Clicca **Esporta** e scegli il formato:
   - **CSV** -- tabulare, per fogli di calcolo e analisi dati
   - **PDF** -- documento, per stampa e condivisione
4. Il file viene generato e scaricato dal browser, con nome
   `<risorsa>_export_<AAAA-MM-GG_HHMMSS>.<est>`

:::tip
Applica i filtri prima di esportare per ottenere esattamente i dati che ti servono. Ad esempio, filtra i sistemi per organizzazione o stato per esportarne solo un sottoinsieme.
:::

## Formato CSV

- **Separatore**: virgola (`,`)
- **Codifica**: UTF-8
- **Intestazioni**: la prima riga contiene i nomi delle colonne
- **Escape**: virgolette doppie sui campi che contengono una virgola

## Formato PDF

- **Intestazione** con data e ora di generazione
- I **filtri** applicati e **chi** ha eseguito l'esportazione
- **Tabella** con i dati formattati

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

Se vedi un elenco puoi esportarlo -- e solo quello. Un utente Support, ad
esempio, non ha `read:users`, quindi non può esportare gli utenti.

I dati esportati rispettano sempre la visibilità gerarchica: l'Owner esporta
l'intera piattaforma, un distributore le proprie organizzazioni subordinate, un
rivenditore i propri clienti, un cliente solo la propria organizzazione. Non è
mai possibile esportare dati a cui non si ha accesso nell'interfaccia.
