---
sidebar_position: 1
---

# Gestione Organizzazioni

Creazione e gestione di distributori, rivenditori e clienti nella gerarchia commerciale.

## Comprendere la Gerarchia

My usa una struttura gerarchica che rispecchia i rapporti commerciali:

```
Owner (Nethesis)
    ↓
Distributori
    ↓
Rivenditori
    ↓
Clienti
```

### Tipi di Organizzazione

**Owner (Nethesis)**
- Organizzazione di primo livello
- Controllo completo della piattaforma
- Può gestire tutti i distributori, rivenditori e clienti
- Esiste una sola organizzazione Owner

**Distributori**
- Creati dall'Owner
- Possono gestire i propri rivenditori e clienti
- Non vedono i dati degli altri distributori
- Controllo completo sul proprio ramo della gerarchia

**Rivenditori**
- Creati dall'Owner o dai distributori
- Possono gestire i propri clienti
- Non vedono i dati degli altri rivenditori
- Operano all'interno del distributore a cui sono assegnati

**Clienti**
- Creati da Owner, distributori o rivenditori
- Organizzazioni utente finale
- Vedono solo i propri dati
- Non possono creare sotto-organizzazioni

### Permessi per Tipo di Organizzazione

| Operazione | Owner | Distributore | Rivenditore | Cliente |
|------------|-------|--------------|-------------|---------|
| Crea distributori | &#10003; | &#10007; | &#10007; | &#10007; |
| Gestisci distributori | &#10003; | &#10007; | &#10007; | &#10007; |
| Crea rivenditori | &#10003; | &#10003; | &#10007; | &#10007; |
| Gestisci rivenditori | &#10003; | &#10003; (propri) | &#10007; | &#10007; |
| Crea clienti | &#10003; | &#10003; | &#10003; | &#10007; |
| Gestisci clienti | &#10003; | &#10003; (propri) | &#10003; (propri) | &#10007; |
| Vedi tutti i dati | &#10003; | &#10007; | &#10007; | &#10007; |

## Creazione Organizzazioni

### Prerequisiti

- Devi aver effettuato l'accesso con i permessi adeguati
- Gli utenti Owner possono creare qualsiasi tipo di organizzazione
- Gli utenti di un distributore possono creare rivenditori e clienti
- Gli utenti di un rivenditore possono creare solo clienti

### Creare un Distributore

**Ruolo richiesto:** membro dell'organizzazione Owner

1. Vai su **Organizzazioni** > **Distributori**
2. Clicca su **Crea distributore**
3. Compila il modulo:
   - **Nome azienda**: ragione sociale del distributore (es. "ACME Distribution Ltd")
   - **Descrizione** (facoltativa): informazioni aggiuntive
   - **Partita IVA**: identificazione IVA univoca per l'azienda
   - **Portali**: i portali di terze parti (NethShop, Helpdesk, NethSpot, ...) che i rivenditori e i clienti del distributore possono usare, vedi sotto
4. Clicca su **Crea distributore**

**Esempio:**
```
Nome: ACME Distribution Europe
Descrizione: Distributore principale per il mercato europeo
Partita IVA: 12345678901
Portali: NethShop, NethSpot
```

#### Portali

I portali che un partner vede nella sua [Dashboard](../features/dashboard.md#applicazioni-di-terze-parti) sono una questione commerciale: il contratto di un distributore stabilisce quali ne fanno parte per i suoi rivenditori e clienti. L'organizzazione Owner registra questa scelta sul distributore, e le organizzazioni sotto di lui la ereditano:

- Gli utenti dei rivenditori del distributore e dei loro clienti vedono solo i portali selezionati sul distributore. Rivenditori e clienti non hanno un'impostazione dei portali propria.
- Gli utenti del distributore non sono vincolati dalla lista: vedono tutti i portali ammessi dai loro ruoli, come l'organizzazione Owner.
- Un distributore senza portali selezionati nasconde ogni portale ai suoi rivenditori e clienti. Di default non è concesso nulla: un distributore appena creato parte senza portali per la sua gerarchia finché l'organizzazione Owner non ne seleziona qualcuno.
- Il filtro per ruolo dichiarato da ogni portale resta valido in aggiunta: un portale riservato agli utenti Admin e Support resta nascosto a un Reader anche se il distributore lo ha.

Solo i membri dell'organizzazione Owner vedono e modificano il campo, sia alla creazione sia alla modifica di un distributore. La scheda di dettaglio del distributore elenca i portali abilitati.

:::note
Nascondere un portale nella Dashboard rimuove la scorciatoia, non la pagina di accesso del portale. Per i portali che Nethesis contrassegna come protetti sull'identity provider, anche l'accesso viene rifiutato agli utenti di un rivenditore o cliente il cui distributore non concede il portale; l'identity provider applica la regola sull'organizzazione, mentre il filtro per ruolo resta nella Dashboard e nei portali che lo verificano da soli.
:::

### Creare un Rivenditore

**Ruolo richiesto:** membro dell'organizzazione Owner o di un distributore

1. Vai su **Organizzazioni** > **Rivenditori**
2. Clicca su **Crea rivenditore**
3. Compila il modulo:
   - **Nome azienda**: ragione sociale del rivenditore
   - **Descrizione** (facoltativa): informazioni aggiuntive
   - **Partita IVA**: identificazione IVA univoca per l'azienda
4. Clicca su **Crea rivenditore**

**Esempio:**
```
Nome: Tech Solutions Italia
Descrizione: Fornitore di soluzioni IT per il mercato PMI
Partita IVA: 12345678901
```

:::note
Se hai effettuato l'accesso come distributore, puoi creare rivenditori solo sotto la tua organizzazione.
:::

### Creare un Cliente

**Ruolo richiesto:** membro dell'organizzazione Owner, di un distributore o di un rivenditore

1. Vai su **Organizzazioni** > **Clienti**
2. Clicca su **Crea cliente**
3. Compila il modulo:
   - **Nome azienda**: ragione sociale del cliente
   - **Descrizione** (facoltativa): informazioni aggiuntive
   - **Partita IVA**: identificazione IVA univoca per l'azienda
4. Clicca su **Crea cliente**

**Esempio:**
```
Nome: Pizza Express Milano
Descrizione: Catena di ristoranti con 5 sedi
Partita IVA: 12345678901
```

## Visualizzazione Organizzazioni

### Elenco

Ogni tipo di organizzazione ha il proprio elenco:

1. Vai su **[Tipo]** (Distributori/Rivenditori/Clienti)
2. L'elenco mostra:
   - Nome dell'organizzazione
   - Descrizione
   - Numero di utenti
   - Numero di sistemi
   - Data di creazione

### Filtri e Ricerca

Usa i filtri per trovare un'organizzazione specifica:

- **Ricerca per nome**: digita nella casella di ricerca
- **Ordinamento**: per nome o descrizione

### Dettagli Organizzazione

Cliccando su un'organizzazione si accede alle informazioni di dettaglio:

- **Panoramica**: nome, descrizione, data di creazione
- **Utenti**: tutti gli utenti che appartengono a questa organizzazione
- **Sistemi**: i sistemi associati all'organizzazione (se applicabile)
- **Statistiche**: metriche di utilizzo e attività

## Gestione Organizzazioni

### Modifica

1. Vai al dettaglio dell'organizzazione
2. Clicca su **Modifica**
3. Aggiorna i campi:
   - Nome azienda
   - Descrizione
   - Partita IVA
4. Clicca su **Salva [Tipo]**

### Eliminazione

**L'eliminazione archivia, non cancella.** L'organizzazione viene eliminata in
modo soft e sparisce dagli elenchi, e la stessa operazione si propaga lungo la
gerarchia:

- Ogni **utente** dell'organizzazione viene archiviato
- Ogni **sistema** dell'organizzazione viene archiviato
- Per un distributore o un rivenditore, viene archiviata anche ogni
  **organizzazione figlia** della sua gerarchia, con i relativi utenti e sistemi

Per eliminare un'organizzazione:

1. Vai all'elenco delle organizzazioni
2. Clicca sull'organizzazione da eliminare
3. Clicca su **Elimina**
4. Conferma l'operazione

:::tip Reversibile
Un'organizzazione eliminata può essere recuperata con **Ripristina**, che
ripristina a cascata anche gli utenti e i sistemi archiviati insieme a lei.
:::

:::danger Eliminazione definitiva
**Elimina definitivamente** è l'operazione irreversibile: cancella
l'organizzazione per sempre e non si può annullare. Richiede il permesso
`destroy:` su quella risorsa, che ha solo l'organizzazione Owner.
:::

### Sospensione e Riattivazione

Invece di eliminare, si può sospendere un'organizzazione:

1. Vai alla pagina dell'organizzazione
2. Clicca su **Sospendi**
3. Conferma l'operazione

**Effetti della sospensione:**
- Gli utenti non possono accedere
- I sistemi non possono inviare dati
- L'organizzazione può essere riattivata in seguito

Per riattivare:

1. Vai alla pagina dell'organizzazione
2. Clicca su **Riattiva**
3. Conferma l'operazione

### Promozione di un Rivenditore

Un rivenditore può essere promosso a distributore. Usa l'azione **Promuovi** nel menu contestuale del rivenditore o sulla sua card di dettaglio.

La promozione:

- Sposta l'organizzazione di un livello: diventa un distributore, agganciato all'organizzazione Owner
- **Mantiene i propri clienti, utenti e sistemi**: non viene staccato nulla
- **Toglie l'accesso al distributore precedente** su quel ramo, che è il senso dell'operazione
- Lascia una traccia: l'organizzazione registra di essere stata promossa, e da chi

Requisiti:

- Autorità di livello Owner: l'azione non è concessa da `manage:resellers`
- L'organizzazione deve essere **attiva**: un rivenditore sospeso o eliminato viene rifiutato
- L'organizzazione deve essere già sincronizzata con il provider di identità

:::warning
La promozione è un cambio di gerarchia, non un ritocco estetico. Il distributore precedente perde visibilità su quel rivenditore e su tutto ciò che sta sotto di lui.
:::

## Statistiche

### Visualizzare le Statistiche

La [Dashboard](../features/dashboard.md) mostra una card contatore per ogni tipo di organizzazione che puoi leggere — distributori, rivenditori, clienti — ognuna con il totale e un collegamento all'elenco.

La pagina di dettaglio di un'organizzazione riporta i propri numeri aggregati: quanti utenti, sistemi e sotto-organizzazioni dipendono da lei.

La crescita nel tempo non è in dashboard: è disponibile da API tramite gli endpoint `/trend` (`/backend/api/distributors/trend`, `/backend/api/resellers/trend`, `/backend/api/customers/trend`).

### Esportazione Dati

Per esportare i dati delle organizzazioni:

1. Vai all'elenco delle organizzazioni
2. Applica eventuali filtri
3. Clicca su **Esporta**
4. Scegli il formato: CSV o PDF
5. Scarica il file

Per maggiori dettagli, consulta [Esportazione Dati](../features/export.md).

## Best Practice

### Convenzioni di Nomenclatura

- Usa nomi chiari e descrittivi
- Includi l'informazione geografica se rilevante (es. "ACME Europe", "Tech Solutions Italia")
- Evita caratteri speciali nei nomi
- Tieni i nomi concisi ma significativi

### Struttura della Gerarchia

- Pianifica la gerarchia prima di creare le organizzazioni
- Mantieni la struttura semplice e logica
- Evita livelli intermedi non necessari
- Documenta i rapporti commerciali

### Controllo degli Accessi

- Assegna gli utenti all'organizzazione corretta
- Rivedi periodicamente le appartenenze
- Usa nomi descrittivi per chiarezza
- Tieni aggiornate le informazioni di contatto

## Risoluzione Problemi

### Impossibile Creare un'Organizzazione

**Problema:** errore di accesso negato durante la creazione

**Soluzioni:**
- Verifica di avere il ruolo corretto (Owner/Distributore/Rivenditore)
- Controlla di stare creando il tipo di organizzazione giusto
- Assicurati che la tua appartenenza organizzativa sia corretta
- Contatta il tuo amministratore

### Organizzazione Non Visibile

**Problema:** un'organizzazione attesa non compare nell'elenco

**Soluzioni:**
- Controlla se l'organizzazione è sospesa (usa i filtri)
- Verifica di avere il permesso di vedere quel tipo di organizzazione
- Assicurati di stare guardando il livello giusto
- Controlla che l'organizzazione appartenga al tuo ramo della gerarchia

### Impossibile Eliminare un'Organizzazione

**Problema:** l'azione di eliminazione non è disponibile o restituisce errore

**Soluzioni:**
- Verifica di avere `manage:` su quel tipo di organizzazione: il Reader non ce l'ha mai
- Verifica che l'organizzazione sia dentro il tuo ramo della gerarchia
- L'organizzazione Owner non può essere eliminata
- **Non** serve svuotarla prima: l'eliminazione si propaga da sola su utenti, sistemi e organizzazioni figlie

## Prossimi Passi

Dopo aver creato le organizzazioni:

- [Crea gli utenti](./users.md) e assegnali alle organizzazioni
- [Crea i sistemi](../systems/management.md) associati alle organizzazioni cliente
- Imposta i permessi appropriati per ogni utente

## Documentazione Correlata

- [Gestione Utenti](./users.md)
- [Gestione Sistemi](../systems/management.md)
