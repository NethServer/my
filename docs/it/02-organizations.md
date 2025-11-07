# Gestione Organizzazioni

Impara a gestire la gerarchia aziendale nella piattaforma My.

## Comprendere la Gerarchia delle Organizzazioni

My utilizza una struttura organizzativa gerarchica che riflette le relazioni aziendali:

```
Owner (Nethesis)
    ↓
Distributori
    ↓
Rivenditori
    ↓
Clienti
```

### Tipi di Organizzazioni

**Owner (Nethesis)**
- Organizzazione di livello superiore
- Controllo completo della piattaforma
- Può gestire tutti i distributori, rivenditori e clienti
- Esiste una sola organizzazione Owner

**Distributori**
- Creati da Owner
- Possono gestire i propri rivenditori e clienti
- Non possono vedere i dati di altri distributori
- Controllo completo sul proprio ramo della gerarchia

**Rivenditori**
- Creati da Owner o Distributori
- Possono gestire i propri clienti
- Non possono vedere i dati di altri rivenditori
- Operano all'interno del distributore assegnato

**Clienti**
- Creati da Owner, Distributori o Rivenditori
- Organizzazioni utente finale
- Possono visualizzare solo i propri dati
- Non possono creare sotto-organizzazioni

### Permessi per Tipo di Organizzazione

| Azione | Owner | Distributore | Rivenditore | Cliente |
|--------|-------|--------------|-------------|---------|
| Crea Distributori | ✅ | ❌ | ❌ | ❌ |
| Gestisci Distributori | ✅ | ❌ | ❌ | ❌ |
| Crea Rivenditori | ✅ | ✅ | ❌ | ❌ |
| Gestisci Rivenditori | ✅ | ✅ (propri) | ❌ | ❌ |
| Crea Clienti | ✅ | ✅ | ✅ | ❌ |
| Gestisci Clienti | ✅ | ✅ (propri) | ✅ (propri) | ❌ |
| Visualizza Tutti i Dati | ✅ | ❌ | ❌ | ❌ |

## Creazione Organizzazioni

### Prerequisiti

- Devi essere autenticato con i permessi appropriati
- Gli utenti Owner possono creare tutti i tipi di organizzazione
- Gli utenti Distributore possono creare rivenditori e clienti
- Gli utenti Rivenditore possono creare solo clienti

### Creazione di un Distributore

**Ruolo Richiesto:** Membro organizzazione Owner

1. Naviga su **Organizzazioni** > **Distributori**
2. Clicca **Crea distributore**
3. Compila il modulo:
   - **Nome azienda**: Nome azienda del distributore (es. "ACME Distribution Ltd")
   - **Descrizione** (opzionale): Informazioni aggiuntive
   - **Partita IVA**: Identificazione IVA univoca per l'azienda
4. Clicca **Crea distributore**

**Esempio:**
```
Nome: ACME Distribution Europe
Descrizione: Distributore principale per il mercato europeo
Partita IVA: 12345678901
```

### Creazione di un Rivenditore

**Ruolo Richiesto:** Membro organizzazione Owner o Distributore

1. Naviga su **Organizzazioni** > **Rivenditori**
2. Clicca **Crea rivenditore**
3. Compila il modulo:
   - **Nome azienda**: Nome azienda del rivenditore (es. "ACME Distribution Ltd")
   - **Descrizione** (opzionale): Informazioni aggiuntive
   - **Partita IVA**: Identificazione IVA univoca per l'azienda
4. Clicca **Crea rivenditore**

**Esempio:**
```
Nome: Tech Solutions Italia
Descrizione: Fornitore di soluzioni IT per il mercato PMI
Partita IVA: 12345678901
```

**Nota:** Se sei autenticato come Distributore, puoi creare rivenditori solo nella tua organizzazione.

### Creazione di un Cliente

**Ruolo Richiesto:** Membro organizzazione Owner, Distributore o Rivenditore

1. Naviga su **Organizzazioni** > **Clienti**
2. Clicca **Crea cliente**
3. Compila il modulo:
   - **Nome azienda**: Nome azienda del cliente (es. "ACME Distribution Ltd")
   - **Descrizione** (opzionale): Informazioni aggiuntive
   - **Partita IVA**: Identificazione IVA univoca per l'azienda
4. Clicca **Crea cliente**

**Esempio:**
```
Nome: Pizza Express Milano
Descrizione: Catena di ristoranti con 5 sedi
Partita IVA: 12345678901
```

## Visualizzazione Organizzazioni

### Elenco Organizzazioni

Ogni tipo di organizzazione ha la propria vista elenco:

1. Naviga su **[Tipo]** (Distributori/Rivenditori/Clienti)
2. Visualizza l'elenco con le seguenti informazioni:
   - Nome organizzazione
   - Descrizione
   - Numero di utenti
   - Numero di sistemi
   - Data di creazione

### Filtri e Ricerca

Utilizza le opzioni di filtro per trovare organizzazioni specifiche:

- **Ricerca per nome**: Digita nella casella di ricerca
- **Ordina per**: Nome, descrizione

### Dettagli Organizzazione

Clicca su un'organizzazione per visualizzare informazioni dettagliate:

- **Panoramica**: Nome, descrizione, data di creazione
- **Utenti**: Tutti gli utenti appartenenti a questa organizzazione
- **Sistemi**: Sistemi associati a questa organizzazione (se applicabile)
- **Statistiche**: Metriche di utilizzo e attività

## Gestione Organizzazioni

### Modifica Informazioni Organizzazione

1. Naviga alla pagina dei dettagli dell'organizzazione
2. Clicca **Modifica**
3. Aggiorna i campi:
   - Nome azienda
   - Descrizione
   - Partita IVA
4. Clicca **Salva [Tipo]**


### Eliminazione Organizzazioni

**⚠️ Attenzione:** L'eliminazione di un'organizzazione è permanente e comporterà:
- Rimozione di tutti gli utenti in quell'organizzazione
- Eliminazione di tutti i sistemi associati
- Rimozione di tutte le organizzazioni figlie (eliminazione a cascata)

Per eliminare un'organizzazione:

1. Naviga alla pagina dell'organizzazione
2. Clicca **Elimina** (usa il menu kebab)
4. Clicca **Elimina**

### Sospensione Organizzazioni

Invece di eliminare, puoi sospendere un'organizzazione:

1. Naviga alla pagina dell'organizzazione
2. Clicca **Sospendi**
3. Conferma l'azione

**Effetti della sospensione:**
- Gli utenti non possono accedere
- I sistemi non possono inviare dati
- Può essere riattivata in seguito

Per riattivare:
1. Filtra per stato "Sospeso"
2. Seleziona l'organizzazione
3. Clicca **Riattiva**

## Statistiche Organizzazioni

### Visualizzazione Statistiche

Naviga su **Dashboard** per vedere:

- **Panoramica Distributori**:
  - Numero totale di distributori
  - Attivi vs. sospesi
  - Grafico andamento (ultimi 30/60/90 giorni)

- **Panoramica Rivenditori**:
  - Totale rivenditori per distributore
  - Attivi vs. sospesi
  - Grafico andamento

- **Panoramica Clienti**:
  - Totale clienti
  - Distribuzione per rivenditore/distributore
  - Trend di crescita

### Esportazione Dati

Esporta i dati delle organizzazioni per reportistica:

1. Naviga all'elenco organizzazioni
2. Applica i filtri se necessario
3. Clicca **Esporta**
4. Scegli il formato: CSV o PDF
5. Scarica il file

## Best Practice

### Convenzioni di Denominazione

- Usa nomi chiari e descrittivi
- Includi informazioni geografiche se rilevanti (es. "ACME Europe", "Tech Solutions Italia")
- Evita caratteri speciali nei nomi
- Mantieni i nomi concisi ma significativi

### Struttura Organizzativa

- Pianifica la gerarchia prima di creare organizzazioni
- Mantieni la struttura semplice e logica
- Evita di creare livelli intermedi non necessari
- Documenta le relazioni aziendali

### Controllo Accessi

- Assegna gli utenti all'organizzazione corretta
- Rivedi regolarmente l'appartenenza alle organizzazioni
- Usa nomi organizzazione descrittivi per chiarezza
- Mantieni aggiornate le informazioni di contatto

## Risoluzione Problemi

### Impossibile Creare Organizzazione

**Problema:** Errore "Accesso negato" durante la creazione di un'organizzazione

**Soluzioni:**
- Verifica di avere il ruolo corretto (Owner/Distributore/Rivenditore)
- Controlla di stare cercando di creare il tipo di organizzazione corretto
- Assicurati che la tua appartenenza all'organizzazione sia corretta
- Contatta il tuo amministratore

### Impossibile Vedere Organizzazione

**Problema:** L'organizzazione prevista non è visibile nell'elenco

**Soluzioni:**
- Controlla se l'organizzazione è sospesa (usa i filtri)
- Verifica di avere il permesso di visualizzare quel tipo di organizzazione
- Assicurati di visualizzare il livello di organizzazione corretto
- Controlla se l'organizzazione appartiene al tuo ramo gerarchico

### Impossibile Eliminare Organizzazione

**Problema:** Il pulsante Elimina è disabilitato o mostra un errore

**Soluzioni:**
- Rimuovi prima tutti i sistemi dall'organizzazione
- Elimina prima tutte le organizzazioni figlie
- Controlla se hai il permesso di eliminare
- Assicurati che l'organizzazione non sia l'organizzazione Owner

## Prossimi Passi

Dopo aver creato le organizzazioni:

- [Crea utenti](03-users.md) e assegnali alle organizzazioni
- [Crea sistemi](04-systems.md) associati alle organizzazioni clienti
- Imposta i permessi appropriati per ogni utente

## Documentazione Correlata

- [Gestione Utenti](03-users.md)
- [Gestione Sistemi](04-systems.md)
