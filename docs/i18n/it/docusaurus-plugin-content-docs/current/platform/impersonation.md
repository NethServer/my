---
sidebar_position: 3
---

# Impersonificazione Utente

Accesso temporaneo alla piattaforma come un altro utente, riservato al personale Nethesis e sotto audit completo.

## Cos'è l'Impersonificazione

L'impersonificazione permette al personale Nethesis di accedere temporaneamente a My come un altro utente. È utile per:

- **Diagnosi**: riprodurre i problemi segnalati dagli utenti
- **Supporto**: assistere gli utenti in operazioni complesse
- **Formazione**: mostrare le funzionalità agli utenti
- **Verifica**: controllare permessi e accessi

## Caratteristiche Principali

### Trasparenza verso l'Utente

- **Trasparenza totale**: tutte le azioni sono registrate e visibili all'utente impersonificato
- **Storico delle sessioni**: l'utente vede ogni sessione in cui è stato impersonificato, con il relativo log di audit
- **A tempo**: ogni sessione ha una durata fissa e poi termina da sola

### Controlli di Sicurezza

- **Basata sui permessi**: solo il personale Nethesis (utenti dell'organizzazione Owner con ruolo Staff, o l'account `owner`) può impersonificare
- **Nessuna auto-impersonificazione**: non si può impersonificare il proprio account
- **Account owner protetto**: l'account `owner` non può mai essere impersonificato
- **Solo utenti attivi**: gli utenti sospesi o eliminati non possono essere impersonificati
- **Nessun concatenamento**: non si può impersonificare mentre si sta già impersonificando qualcun altro
- **Nessuna API key**: l'impersonificazione non è disponibile alle API key
- **Scadenza automatica**: la sessione termina da sola dopo una durata fissa
- **Tracciamento di sessione**: ogni sessione ha un identificativo univoco per l'audit

### Tracciabilità Completa

- Ogni chiamata API durante l'impersonificazione viene registrata
- L'utente può rivedere tutte le azioni compiute a suo nome
- I dati sensibili vengono oscurati automaticamente nei log
- L'organizzazione per sessione rende la revisione semplice

## Chi Può Impersonificare

### Permessi Richiesti

**Personale Nethesis (organizzazione Owner):**
- Gli utenti dell'organizzazione Owner — ruolo Staff o account `owner` — hanno il permesso `impersonate:users`
- Possono impersonificare qualsiasi utente attivo della piattaforma, in qualsiasi distributore, rivenditore o cliente
- Il consenso dell'utente bersaglio non è richiesto
- Non serve assegnare altri ruoli

**Tutti gli altri:**
- Non vedono le funzionalità di impersonificazione
- Non possono impersonificare nessun utente

## Durata della Sessione

Ogni sessione di impersonificazione ha una durata fissa, stabilita dall'amministratore della piattaforma con l'impostazione del backend `IMPERSONATION_SESSION_DURATION`. Il valore predefinito è **1 ora**, il massimo è **168 ore** (una settimana).

Durante l'impersonificazione, passa il mouse sul badge di impersonificazione nella barra superiore per vedere il tempo rimanente. Allo scadere del tempo la sessione termina e torni a usare il tuo account.

Per continuare a lavorare, clicca su **Estendi sessione** nello stesso tooltip: la sessione riparte per la sua durata completa da quel momento, senza conferma e senza un limite al numero di estensioni. Una notifica conferma il nuovo tempo rimanente, e l'estensione compare nell'audit log della sessione come **Sessione estesa**.

## Flusso di Lavoro

### Passo 1: L'Amministratore Impersonifica l'Utente

**Per gli amministratori (personale Nethesis):**

1. Vai su **Utenti**
2. Trova l'utente bersaglio
3. Clicca su **Impersona utente** (dal menu contestuale)
4. Conferma l'operazione
5. Da quel momento stai agendo come quell'utente

**Durante l'impersonificazione vedrai:**
- **Badge nella barra superiore**: "Stai impersonando [Nome Utente]"
- **Icona di chiusura**: per tornare al tuo account
- **Tutte le funzionalità**: esattamente come le vede l'utente
- **I permessi dell'utente**: filtrati sui suoi permessi reali

### Passo 2: Svolgere le Azioni di Supporto

Mentre impersonifichi:

- Naviga la piattaforma come farebbe l'utente
- Riproduci i problemi segnalati
- Compi azioni per conto dell'utente
- Verifica funzionalità e permessi
- Documenta quello che trovi

:::warning
Tutte le azioni vengono registrate e sono visibili all'utente impersonificato. Tratta i suoi dati con rispetto ed esci dall'impersonificazione appena hai finito.
:::

### Passo 3: Uscire dall'Impersonificazione

**Per uscire:**

1. Clicca sull'icona di chiusura del badge di impersonificazione nella barra superiore
2. Torni al tuo account originale
3. La sessione di impersonificazione viene chiusa e il suo token revocato

**Uscita automatica:**
- La durata della sessione scade
- L'utente impersonificato viene sospeso o eliminato

## Per gli Utenti: Le Tue Sessioni di Impersonificazione

Prima che una sessione inizi non ti viene chiesto alcun permesso. In compenso ogni sessione viene registrata, e puoi rivederla quando vuoi.

### Visualizzazione dell'Audit

**Per vedere chi ti ha impersonificato:**

1. Vai su **Account** > **Impersonificazione**
2. Apri la sezione **Sessioni**
3. Trovi lo storico completo:
   ```
   Inizio: 2025-11-06 10:00:00 UTC
   Fine: 2025-11-06 11:00:00 UTC
   Durata: 1 ora
   Impersonatore: John Admin (john@example.com)
   Stato: Completata
   ```

4. Clicca su **Mostra audit log** per vedere tutte le azioni

**Informazioni registrate:**
- Data e ora di ogni azione
- Inizio, estensioni e fine della sessione
- Endpoint API chiamato
- Dati sensibili oscurati automaticamente
- Esito (successo/errore)

## Per gli Amministratori: Uso dell'Impersonificazione

### Verifica della Disponibilità

**Nell'elenco utenti:**

La voce **Impersona utente** è abilitata per ogni utente che puoi impersonificare.

È disabilitata per:
- Il tuo account
- L'account `owner`
- Gli utenti sospesi o eliminati

### Avvio dell'Impersonificazione

**Requisiti:**
- Appartieni all'organizzazione Owner (ruolo Staff o account `owner`)
- L'utente non è eliminato né sospeso
- L'utente non sei tu e non è l'account `owner`
- Non stai già impersonificando qualcun altro
- Hai effettuato l'accesso con il tuo account, non con una API key

**Procedura:**

1. **Trova l'utente**:
   - Vai su **Utenti**
   - Cerca l'utente bersaglio

2. **Avvia l'impersonificazione**:
   - Clicca su **Impersona utente** (dal menu contestuale)
   - Conferma nel dialogo:
     ```
     Agirai temporaneamente come l'utente [Nome] e avrai i suoi permessi.

     Per tornare al tuo account, clicca sull'icona di chiusura nel badge
     di impersonificazione nella barra superiore.

     [Annulla] [Impersona utente]
     ```

3. **Conferma**:
   - Stai impersonificando l'utente
   - Compare il badge nella barra superiore
   - La sessione è iniziata, con la durata stabilita dalla piattaforma

### Durante la Sessione

**Cosa vedi:**
- Esattamente la stessa interfaccia dell'utente
- I permessi dell'utente (possono essere più restrittivi dei tuoi)
- L'organizzazione e i dati dell'utente
- Le sue personalizzazioni e preferenze

**Cosa puoi fare:**
- Navigare tutte le pagine a cui l'utente accede
- Compiere qualsiasi azione che l'utente può compiere
- Creare, modificare o eliminare in base ai permessi dell'utente
- Verificare funzionalità e riprodurre problemi

**Cosa non puoi fare:**
- Accedere a funzionalità precluse all'utente
- Aggirare le restrizioni di permesso dell'utente
- Impersonificare un altro utente mentre stai già impersonificando
- Modificare il tuo account

**Buone pratiche:**
- Documenta le tue azioni
- Riduci al minimo il tempo in impersonificazione
- Compi solo le operazioni necessarie
- Informa l'utente di cosa hai fatto
- Esci appena hai finito

### Uscita dall'Impersonificazione

**Uscita normale:**

- Clicca sull'icona di chiusura del badge di impersonificazione nella barra superiore
- Torni al tuo account

**Uscita automatica:**

L'impersonificazione termina da sola quando:
- La durata della sessione scade
- Effettui il logout
- L'utente viene sospeso o eliminato

## Sicurezza e Privacy

### Cosa Viene Registrato

**Informazioni registrate:**
- Data e ora di ogni azione
- Endpoint API e metodo (GET, POST, ecc.)
- Codice di stato HTTP (200, 404, ecc.)
- Parametri della richiesta (con i dati sensibili oscurati)
- Esito della risposta (con i dati sensibili oscurati)

**Oscurati automaticamente:**
- Password
- Token di autenticazione
- Secret di sistema
- Qualsiasi campo che contenga "password", "secret", "token", "api_key" o "key"

**Esempio di voce di log:**
```json
{
  "timestamp": "2025-11-06T10:15:23Z",
  "session_id": "imp_abc123",
  "impersonator": "admin@example.com",
  "impersonated_user": "user@example.com",
  "method": "POST",
  "endpoint": "/api/users",
  "status": 201,
  "request_body": {
    "name": "John Doe",
    "email": "john@example.com",
    "password": "[REDACTED]"
  }
}
```

:::note
Nel log l'`endpoint` è il percorso come lo vede il backend, dopo il rewrite del proxy: `/api/users`, non `/backend/api/users`.
:::

### Protezione dei Dati

**Visibilità per l'utente:**
- L'utente vede ogni sessione in cui è stato impersonificato
- L'utente vede la tracciabilità completa di ogni sessione

**Protezione della piattaforma:**
- Impersonificazione riservata al personale Nethesis
- L'account `owner` e gli utenti sospesi o eliminati non sono mai impersonificabili
- Durata fissa della sessione e scadenza automatica
- Registrazione completa
- Oscuramento dei dati sensibili

**Conformità:**
- Tracciabilità per i requisiti normativi
- Visibilità dell'utente su ogni sessione
- Privacy dei dati rispettata

## Casi d'Uso Comuni

### Risoluzione dei Problemi di un Utente

**Scenario:** un utente segnala di non vedere una funzionalità

**Flusso:**
1. L'amministratore impersonifica l'utente
2. L'amministratore va nella sezione segnalata
3. Riproduce il problema
4. Individua il problema di permessi o configurazione
5. Esce dall'impersonificazione
6. Corregge le impostazioni dell'utente
7. L'utente conferma la risoluzione

### Formazione di Nuovi Utenti

**Scenario:** formazione su un flusso di lavoro complesso

**Flusso:**
1. L'amministratore impersonifica l'utente
2. Esegue i passaggi del flusso
3. Documenta ogni azione
4. Esce dall'impersonificazione
5. Condivide l'audit con l'utente
6. L'utente rivede le azioni compiute
7. L'utente si esercita in autonomia

### Verifica dei Permessi

**Scenario:** verificare che un utente abbia i permessi corretti

**Flusso:**
1. L'amministratore impersonifica l'utente
2. Prova l'accesso alle varie funzionalità
3. Documenta cosa è visibile e accessibile
4. Esce dall'impersonificazione
5. Corregge i permessi se necessario

## Risoluzione Problemi

### Impossibile Impersonificare un Utente

**Problema:** il pulsante di impersonificazione è disabilitato

**Soluzioni:**
1. Verifica di avere i permessi:
   - Devi appartenere all'organizzazione Owner (ruolo Staff o account `owner`)
2. Controlla lo stato dell'utente:
   - L'utente non deve essere sospeso
   - L'utente non deve essere eliminato
3. Controlla l'utente bersaglio:
   - Non deve essere il tuo account
   - Non deve essere l'account `owner`
4. Verifica di non stare già impersonificando:
   - Esci prima dall'impersonificazione in corso

### La Sessione Termina Inaspettatamente

**Problema:** vieni espulso dalla sessione di impersonificazione

**Cause possibili:**
- La durata della sessione è scaduta
- L'utente è stato sospeso o eliminato
- Interruzione di rete

**Soluzioni:**
1. Avvia una nuova sessione di impersonificazione se ti serve altro tempo
2. Controlla lo stato dell'utente in **Utenti**
3. Verifica la tua connessione di rete

### Non Vedi i Dati dell'Utente

**Problema:** durante l'impersonificazione non vedi i dati che ti aspetti

**Spiegazione:**
- Vedi esattamente ciò che vede l'utente
- L'utente potrebbe avere permessi ristretti
- L'accesso alle organizzazioni potrebbe essere limitato
- È il comportamento atteso

**Soluzioni:**
1. Verifica i ruoli assegnati all'utente
2. Controlla la sua appartenenza organizzativa
3. Rivedi i permessi gerarchici
4. Correggi i permessi dell'utente se necessario

## Best Practice

### Per gli Utenti

**Privacy:**
- Rivedi le tue sessioni di impersonificazione in **Account** > **Impersonificazione**
- Puoi vedere tutto quello che è stato fatto a tuo nome
- Contatta Nethesis se trovi una sessione che non ti aspetti

### Per gli Amministratori

**Prima di impersonificare:**
- Abbi uno scopo chiaro
- Avvisa l'utente, quando possibile
- Pianifica cosa devi fare
- Stima il tempo necessario

**Durante l'impersonificazione:**
- Lavora in modo efficiente
- Documenta le tue azioni
- Compi solo le operazioni necessarie
- Rispetta la privacy dell'utente
- Esci appena hai finito

**Dopo l'impersonificazione:**
- Informa l'utente di cosa è stato fatto
- Documenta quello che hai trovato
- Condividi l'audit se richiesto
- Dai seguito ai problemi individuati

### Per le Organizzazioni

**Policy:**
- Definisci quando l'impersonificazione è appropriata
- Documenta il processo di approvazione
- Forma gli amministratori
- Rivedi periodicamente i log di audit

**Sicurezza:**
- Ricorda che l'impersonificazione è riservata al personale Nethesis (organizzazione Owner)
- Monitora l'uso dell'impersonificazione
- Rivedi i log di audit
- Indaga sugli schemi inusuali

## Documentazione Correlata

- [Gestione Utenti](./users.md)
- [Guida all'Autenticazione](../getting-started/authentication.md)
- [Documentazione API Backend](https://github.com/NethServer/my/blob/main/backend/README.md)
