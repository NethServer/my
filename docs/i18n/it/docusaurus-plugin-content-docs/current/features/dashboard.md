---
sidebar_position: 1
---

# Dashboard

La dashboard è la pagina principale di My e fornisce una panoramica immediata dello stato della piattaforma.

## Panoramica

La dashboard mostra card riassuntive per ogni tipo di entità gestita dalla piattaforma, consentendo di avere una visione d'insieme rapida e completa.

## Card Contatore

La dashboard presenta card per ciascuna delle seguenti entità:

| Entità | Descrizione | Visibile a |
|--------|-------------|------------|
| **Distributori** | Numero totale di distributori | Owner |
| **Rivenditori** | Numero totale di rivenditori | Owner, Distributori |
| **Clienti** | Numero totale di clienti | Owner, Distributori, Rivenditori |
| **Utenti** | Numero totale di utenti | Tutti i ruoli |
| **Sistemi** | Numero totale di sistemi | Tutti i ruoli |

Ogni card mostra:

- **Conteggio totale** dell'entità
- **Icona** identificativa
- **Link rapido** alla pagina di elenco corrispondente

:::note
I conteggi sono filtrati in base alla posizione gerarchica dell'utente. Un distributore vede solo i conteggi relativi alle proprie organizzazioni subordinate.
:::

## Analisi Trend

Ogni card contatore include un'indicazione del trend di crescita calcolato su diversi periodi:

- **30 giorni** - Variazione nell'ultimo mese
- **60 giorni** - Variazione negli ultimi due mesi
- **90 giorni** - Variazione nell'ultimo trimestre

Il trend mostra:

- **Freccia verso l'alto** con percentuale positiva per crescita
- **Freccia verso il basso** con percentuale negativa per diminuzione
- **Indicatore neutro** se non ci sono variazioni

## Visibilità Basata su Ruolo

La dashboard adatta automaticamente il contenuto in base al ruolo dell'utente:

### Owner

L'Owner vede tutte le card e le statistiche complete della piattaforma:
- Distributori, Rivenditori, Clienti, Utenti, Sistemi
- Statistiche globali

### Distributore

Il Distributore vede:
- Rivenditori (propri), Clienti (propri), Utenti (propri), Sistemi (propri)
- Statistiche relative alla propria gerarchia

### Rivenditore

Il Rivenditore vede:
- Clienti (propri), Utenti (propri), Sistemi (propri)
- Statistiche relative alla propria gerarchia

### Cliente

Il Cliente vede:
- Utenti (propri), Sistemi (propri)
- Statistiche relative alla propria organizzazione

:::tip
Se una card non è visibile, è perché il tuo ruolo nella gerarchia non prevede la gestione di quel tipo di entità.
:::
