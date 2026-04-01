---
sidebar_position: 1
---

# Gestione Organizzazioni

Le organizzazioni rappresentano la struttura gerarchica aziendale in My. Ogni organizzazione ha un tipo specifico che ne determina i permessi e le capacità.

## Gerarchia

La gerarchia delle organizzazioni segue una struttura ad albero:

```
Owner (Nethesis)
    ↓
Distributori
    ↓
Rivenditori
    ↓
Clienti
```

Ogni livello può gestire solo le organizzazioni ai livelli inferiori. L'Owner ha visibilità e controllo su tutta la gerarchia.

## Tipi di Organizzazione

| Tipo | Creato da | Può Gestire | Descrizione |
|------|-----------|-------------|-------------|
| **Owner** | Sistema | Distributori, Rivenditori, Clienti | Organizzazione radice, unica |
| **Distributore** | Owner | Rivenditori, Clienti | Partner di distribuzione |
| **Rivenditore** | Owner, Distributore | Clienti | Partner di rivendita |
| **Cliente** | Owner, Distributore, Rivenditore | Nessuno | Utente finale |

### Permessi per Tipo di Organizzazione

| Azione | Owner | Distributore | Rivenditore | Cliente |
|--------|-------|--------------|-------------|---------|
| Crea Distributori | Si | No | No | No |
| Gestisci Distributori | Si | No | No | No |
| Crea Rivenditori | Si | Si | No | No |
| Gestisci Rivenditori | Si | Si (propri) | No | No |
| Crea Clienti | Si | Si | Si | No |
| Gestisci Clienti | Si | Si (propri) | Si (propri) | No |
| Visualizza Tutti i Dati | Si | No | No | No |

## Creazione Organizzazioni

### Creare un Distributore

Solo l'Owner può creare distributori.

1. Vai a **Organizzazioni** > **Distributori**
2. Clicca su **Nuovo Distributore**
3. Compila i campi richiesti:
   - **Nome** - Nome dell'organizzazione
   - **Descrizione** - Descrizione opzionale
   - **Partita IVA** - Identificazione IVA univoca per l'azienda
4. Clicca su **Crea**

**Esempio:**
```
Nome: ACME Distribution Europe
Descrizione: Distributore principale per il mercato europeo
Partita IVA: 12345678901
```

:::note
Il nome dell'organizzazione deve essere unico all'interno della piattaforma.
:::

### Creare un Rivenditore

L'Owner e i Distributori possono creare rivenditori.

1. Vai a **Organizzazioni** > **Rivenditori**
2. Clicca su **Nuovo Rivenditore**
3. Compila i campi richiesti:
   - **Nome** - Nome dell'organizzazione
   - **Organizzazione Padre** - Il distributore di appartenenza
   - **Descrizione** - Descrizione opzionale
   - **Partita IVA** - Identificazione IVA univoca per l'azienda
4. Clicca su **Crea**

**Esempio:**
```
Nome: Tech Solutions Italia
Descrizione: Fornitore di soluzioni IT per il mercato PMI
Partita IVA: 12345678901
```

:::note
Se sei autenticato come Distributore, puoi creare rivenditori solo nella tua organizzazione.
:::

:::tip
I distributori vedranno solo i propri rivenditori nella lista. L'Owner vede tutti i rivenditori della piattaforma.
:::

### Creare un Cliente

L'Owner, i Distributori e i Rivenditori possono creare clienti.

1. Vai a **Organizzazioni** > **Clienti**
2. Clicca su **Nuovo Cliente**
3. Compila i campi richiesti:
   - **Nome** - Nome dell'organizzazione
   - **Organizzazione Padre** - Il rivenditore (o distributore) di appartenenza
   - **Descrizione** - Descrizione opzionale
   - **Partita IVA** - Identificazione IVA univoca per l'azienda
4. Clicca su **Crea**

**Esempio:**
```
Nome: Pizza Express Milano
Descrizione: Catena di ristoranti con 5 sedi
Partita IVA: 12345678901
```

## Visualizzazione Organizzazioni

### Elenco

Le pagine di elenco delle organizzazioni mostrano tutte le organizzazioni visibili in base al proprio ruolo gerarchico, con le seguenti informazioni:

- **Nome** dell'organizzazione
- **Tipo** (Distributore, Rivenditore, Cliente)
- **Organizzazione Padre** (se applicabile)
- **Numero di utenti**
- **Numero di sistemi**
- **Data di creazione**

### Filtri e Ricerca

È possibile filtrare le organizzazioni per:

- **Ricerca testuale** - Cerca per nome
- **Tipo** - Filtra per tipo di organizzazione
- **Organizzazione Padre** - Filtra per appartenenza gerarchica
- **Stato** - Attiva o sospesa

### Dettagli Organizzazione

Cliccando su un'organizzazione si accede alla pagina di dettaglio che mostra:

- **Informazioni generali** - Nome, tipo, descrizione, data di creazione
- **Organizzazione Padre** - Dettagli sull'organizzazione superiore
- **Utenti** - Elenco degli utenti appartenenti all'organizzazione
- **Sistemi** - Elenco dei sistemi associati
- **Organizzazioni Figlie** - Elenco delle organizzazioni subordinate (se presenti)

## Gestione Organizzazioni

### Modifica

Per modificare un'organizzazione:

1. Vai all'elenco delle organizzazioni
2. Clicca sull'organizzazione da modificare
3. Clicca su **Modifica**
4. Aggiorna i campi desiderati
5. Clicca su **Salva**

:::warning
La modifica del nome dell'organizzazione si riflette su tutta la piattaforma, inclusi report e dashboard.
:::

### Eliminazione

Per eliminare un'organizzazione:

1. Vai all'elenco delle organizzazioni
2. Clicca sull'organizzazione da eliminare
3. Clicca su **Elimina**
4. Conferma l'operazione

:::danger
L'eliminazione di un'organizzazione è permanente. Assicurati che:
- Non ci siano utenti attivi nell'organizzazione
- Non ci siano sistemi associati
- Non ci siano organizzazioni figlie
:::

### Sospensione e Riattivazione

È possibile sospendere un'organizzazione senza eliminarla. Un'organizzazione sospesa:

- Non permette l'accesso ai propri utenti
- Non riceve dati dai sistemi associati
- Può essere riattivata in qualsiasi momento

Per sospendere un'organizzazione:

1. Vai alla pagina di dettaglio dell'organizzazione
2. Clicca su **Sospendi**
3. Conferma l'operazione

Per riattivare:

1. Vai alla pagina di dettaglio dell'organizzazione sospesa
2. Clicca su **Riattiva**
3. Conferma l'operazione

## Statistiche

Ogni tipo di organizzazione mostra statistiche aggregate:

- **Totale** organizzazioni per tipo
- **Trend** di crescita nel tempo (30, 60, 90 giorni)
- **Distribuzione** per organizzazione padre

Le statistiche sono visibili nella pagina di riepilogo di ciascun tipo di organizzazione.

## Esportazione

È possibile esportare l'elenco delle organizzazioni in formato:

- **CSV** - Per analisi in fogli di calcolo
- **PDF** - Per report e documentazione

L'esportazione include tutte le organizzazioni visibili in base ai filtri applicati, fino a un massimo di 10.000 record.

Per maggiori dettagli, consulta la pagina [Esportazione Dati](../features/export).

## Best Practice

- **Pianifica la gerarchia** prima di creare le organizzazioni
- **Usa nomi descrittivi** che identifichino chiaramente l'organizzazione
- **Assegna le organizzazioni padre** correttamente per mantenere la gerarchia coerente
- **Verifica le dipendenze** prima di eliminare un'organizzazione
- **Sospendi** invece di eliminare se la rimozione potrebbe essere temporanea

## Risoluzione Problemi

### Impossibile Creare un'Organizzazione

- Verifica di avere i permessi necessari per il tipo di organizzazione
- Controlla che il nome non sia già in uso
- Assicurati di aver selezionato un'organizzazione padre valida (per rivenditori e clienti)

### Organizzazione Non Visibile

- Verifica il tuo ruolo nella gerarchia: puoi vedere solo le organizzazioni al tuo livello o inferiore
- Controlla i filtri attivi che potrebbero nascondere l'organizzazione
- Se sei un distributore, puoi vedere solo i rivenditori e clienti sotto la tua organizzazione

### Impossibile Eliminare un'Organizzazione

- Verifica che non ci siano utenti attivi nell'organizzazione
- Verifica che non ci siano sistemi associati
- Verifica che non ci siano organizzazioni figlie
- Assicurati di avere i permessi necessari per l'eliminazione
