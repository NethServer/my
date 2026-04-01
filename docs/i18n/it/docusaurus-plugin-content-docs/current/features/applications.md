---
sidebar_position: 2
---

# Applicazioni

Le applicazioni rappresentano i prodotti software di terze parti integrati nella piattaforma My, che possono essere assegnati alle organizzazioni per gestirne l'accesso.

## Panoramica

La sezione Applicazioni consente di:

- Visualizzare le applicazioni disponibili nella piattaforma
- Assegnare applicazioni alle organizzazioni
- Gestire note e descrizioni per ogni applicazione
- Monitorare l'utilizzo delle applicazioni

## Visualizzazione Applicazioni

### Elenco

La pagina elenco applicazioni mostra tutte le applicazioni disponibili con:

- **Nome** dell'applicazione
- **Descrizione**
- **Tipo** (terze parti)
- **Organizzazioni assegnate** (conteggio)
- **Stato** (attiva/inattiva)

### Filtri

È possibile filtrare le applicazioni per:

- **Ricerca testuale** - Cerca per nome o descrizione
- **Stato** - Attiva o inattiva

## Dettagli Applicazione

Cliccando su un'applicazione si accede alla pagina di dettaglio con:

- **Informazioni generali** - Nome, descrizione, tipo
- **Organizzazioni assegnate** - Elenco delle organizzazioni a cui è assegnata l'applicazione
- **Note** - Note e commenti sull'applicazione

## Assegnazione alle Organizzazioni

Per assegnare un'applicazione a un'organizzazione:

1. Vai al dettaglio dell'applicazione
2. Clicca su **Assegna Organizzazione**
3. Seleziona una o più organizzazioni dalla lista
4. Clicca su **Conferma**

Per rimuovere l'assegnazione:

1. Vai al dettaglio dell'applicazione
2. Nella lista delle organizzazioni assegnate, clicca su **Rimuovi** accanto all'organizzazione desiderata
3. Conferma l'operazione

:::note
Le organizzazioni disponibili per l'assegnazione sono filtrate in base alla tua posizione nella gerarchia. Puoi assegnare applicazioni solo alle organizzazioni che puoi gestire.
:::

## Note Applicazione

Ogni applicazione può avere note associate per documentare:

- Configurazioni specifiche
- Istruzioni di utilizzo
- Informazioni di contatto per il supporto
- Dettagli sulla licenza

Le note sono visibili a tutti gli utenti che hanno accesso all'applicazione.

## Totali e Trend

La pagina applicazioni mostra:

- **Totale applicazioni** disponibili
- **Applicazioni assegnate** per organizzazione
- **Trend** di assegnazione nel tempo

## Permessi

| Operazione | Super Admin | Admin | Backoffice | Support | Reader |
|------------|:-----------:|:-----:|:----------:|:-------:|:------:|
| Visualizza applicazioni | Si | Si | Si | Si | Si |
| Assegna a organizzazioni | Si | Si | Si | No | No |
| Modifica note | Si | Si | Si | No | No |

:::warning
Solo gli utenti con ruolo Backoffice o superiore possono assegnare applicazioni alle organizzazioni.
:::
