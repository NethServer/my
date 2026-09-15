---
sidebar_position: 3
---

# Impersonificazione Utente

Accesso temporaneo alla piattaforma come un altro utente, con il suo consenso e sotto audit completo.

## Cos'è l'Impersonificazione

L'impersonificazione permette agli amministratori autorizzati di accedere temporaneamente a My come un altro utente. È utile per:

- **Diagnosi**: riprodurre i problemi segnalati dagli utenti
- **Supporto**: assistere gli utenti in operazioni complesse
- **Formazione**: mostrare le funzionalità agli utenti
- **Verifica**: controllare permessi e accessi

## Caratteristiche Principali

### Progettata attorno alla Privacy

- **Consenso obbligatorio**: l'utente deve abilitare esplicitamente l'impersonificazione
- **A tempo**: è l'utente a decidere per quanto resta permessa (1-168 ore)
- **Trasparenza totale**: tutte le azioni sono registrate e visibili all'utente
- **Revoca immediata**: l'utente può disattivare il consenso in qualsiasi momento

### Controlli di Sicurezza

- **Basata sui permessi**: solo il personale Nethesis (utenti dell'organizzazione Owner) può impersonificare
- **Nessuna auto-impersonificazione**: non si può impersonificare il proprio account
- **Nessun concatenamento**: non si può impersonificare mentre si sta già impersonificando qualcun altro
- **Scadenza automatica**: il consenso scade da solo alla durata scelta dall'utente
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
- Possono impersonificare qualsiasi utente dell'intera gerarchia (con il suo consenso)
- Non serve assegnare altri ruoli

**Tutti gli altri:**
- Non vedono le funzionalità di impersonificazione
- Non possono impersonificare nessun utente

## Flusso di Lavoro

### Passo 1: L'Utente Abilita il Consenso

Prima che l'impersonificazione possa avvenire, l'utente bersaglio deve abilitare il consenso.

**Per gli utenti:**

1. Accedi al tuo account
2. Vai su **Account** > **Impersonificazione**
3. Individua la sezione **Consenso all'impersonificazione**
4. Clicca su **Abilita impersonificazione**
5. Imposta la durata (1-168 ore)
6. Clicca su **Salva**

**Cosa succede:**
- Il consenso viene registrato con data e ora
- L'amministratore vede che il consenso è disponibile
- Scade automaticamente alla fine della durata
- Può essere revocato in qualsiasi momento

**Durate tipiche:**
- **1-24 ore**: diagnosi rapida
- **24-72 ore**: supporto su più giorni
- **72-168 ore**: accesso prolungato (massimo una settimana)

### Passo 2: L'Amministratore Impersonifica l'Utente

**Per gli amministratori (personale Nethesis):**

1. Vai su **Utenti**
2. Trova l'utente bersaglio
3. Controlla che **Impersonifica utente** sia disponibile (dal menu contestuale)
4. Clicca su **Impersonifica utente**
5. Conferma l'operazione
6. Da quel momento stai agendo come quell'utente

**Durante l'impersonificazione vedrai:**
- **Banner in alto**: "Stai impersonificando [Nome Utente]"
- **Pulsante di uscita**: per tornare al tuo account
- **Tutte le funzionalità**: esattamente come le vede l'utente
- **I permessi dell'utente**: filtrati sui suoi permessi reali

### Passo 3: Svolgere le Azioni di Supporto

Mentre impersonifichi:

- Naviga la piattaforma come farebbe l'utente
- Riproduci i problemi segnalati
- Compi azioni per conto dell'utente
- Verifica funzionalità e permessi
- Documenta quello che trovi

:::warning
Tutte le azioni vengono registrate e sono visibili all'utente impersonificato. Tratta i suoi dati con rispetto ed esci dall'impersonificazione appena hai finito.
:::

### Passo 4: Uscire dall'Impersonificazione

**Per uscire:**

1. Clicca sul pulsante **Esci dall'impersonificazione** nel banner
2. Torni al tuo account originale
3. La sessione di impersonificazione viene chiusa

**Uscita automatica:**
- La sessione scade alla fine della durata del consenso
- L'utente revoca il consenso durante la sessione
- Il token scade (segue la durata del consenso)

## Per gli Utenti: Gestione del Consenso

### Attivazione del Consenso

**Quando attivarlo:**
- Quando hai un problema e ti serve assistenza
- Quando chiedi aiuto a un amministratore
- Prima di una sessione di formazione
- Quando l'amministratore te lo chiede

**Come attivarlo:**

1. Vai su **Account** > **Impersonificazione**
2. Clicca su **Consenso all'impersonificazione**
3. Scegli la durata:
   ```
   1 ora    - Supporto rapido
   24 ore   - Supporto in giornata
   72 ore   - Problema su più giorni
   Custom   - Specifica le ore (massimo 168)
   ```
4. Clicca su **Abilita**

**Conferma:**
```
Consenso all'impersonificazione abilitato
  Scade: [data e ora]
  Durata: [X] ore
```

### Verifica dello Stato del Consenso

**Per controllare se il consenso è attivo:**

1. Vai su **Account** > **Impersonificazione**
2. Guarda la sezione **Consenso all'impersonificazione**:
   ```
   Stato: Attivo
   Scade: 2025-11-07 10:30:00 UTC
   ```

### Revoca del Consenso

**Per disattivare il consenso:**

1. Vai su **Account** > **Impersonificazione**
2. Clicca su **Revoca consenso**
3. Conferma l'operazione

**Effetti:**
- Il consenso viene disabilitato immediatamente
- Le sessioni di impersonificazione attive vengono terminate
- L'amministratore non può più impersonificarti
- Puoi riabilitarlo quando vuoi

### Visualizzazione dell'Audit

**Per vedere chi ti ha impersonificato:**

1. Vai su **Account** > **Impersonificazione**
2. Apri la sezione **Sessioni**
3. Trovi lo storico completo:
   ```
   Inizio: 2025-11-06 10:00:00 UTC
   Fine: 2025-11-06 11:30:00 UTC
   Durata: 1,5 ore
   Impersonificatore: John Admin (john@example.com)
   Stato: In corso
   ```

4. Clicca su **Mostra log di audit** per vedere tutte le azioni

**Informazioni registrate:**
- Data e ora di ogni azione
- Endpoint API chiamato
- Dati sensibili oscurati automaticamente
- Esito (successo/errore)

## Per gli Amministratori: Uso dell'Impersonificazione

### Verifica della Disponibilità

**Nell'elenco utenti:**

Gli utenti con consenso attivo mostrano:
- La voce **Impersonifica utente** abilitata
- La data di scadenza del consenso
- Il click avvia l'impersonificazione

**Gli utenti senza consenso:**
- Hanno la voce **Impersonifica utente** disabilitata

### Avvio dell'Impersonificazione

**Requisiti:**
- L'utente ha un consenso attivo
- Appartieni all'organizzazione Owner (ruolo Staff o account `owner`)
- L'utente non è eliminato né sospeso
- Non stai già impersonificando qualcun altro

**Procedura:**

1. **Trova l'utente**:
   - Vai su **Utenti**
   - Cerca l'utente bersaglio

2. **Verifica il consenso**:
   - Controlla che **Impersonifica utente** sia abilitato
   - Controlla la scadenza del consenso
   - Assicurati che il tempo residuo ti basti

3. **Avvia l'impersonificazione**:
   - Clicca su **Impersonifica utente** (dal menu contestuale)
   - Conferma nel dialogo:
     ```
     Agirai temporaneamente come l'utente [Nome] e avrai i suoi permessi.

     Per tornare al tuo account, clicca sull'icona di chiusura sul badge
     di impersonificazione nella barra in alto.

     [Annulla] [Impersonifica utente]
     ```

4. **Conferma**:
   - Stai impersonificando l'utente
   - Compare il banner in alto
   - La sessione è iniziata

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

- Clicca sulla **X** nel banner
- Torni al tuo account

**Uscita automatica:**

L'impersonificazione termina da sola quando:
- La durata del consenso scade
- L'utente revoca il consenso
- Il token di sessione scade
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

**Controllo dell'utente:**
- È l'utente a scegliere quando abilitare il consenso
- È l'utente a decidere la durata
- L'utente può revocare in qualsiasi momento
- L'utente vede la tracciabilità completa

**Protezione della piattaforma:**
- Nessun accesso senza consenso
- Scadenza automatica
- Registrazione completa
- Oscuramento dei dati sensibili

**Conformità:**
- Tracciabilità per i requisiti normativi
- Modello di accesso basato sul consenso
- Visibilità e controllo in capo all'utente
- Privacy dei dati rispettata

## Casi d'Uso Comuni

### Risoluzione dei Problemi di un Utente

**Scenario:** un utente segnala di non vedere una funzionalità

**Flusso:**
1. L'utente abilita il consenso (1 ora)
2. L'amministratore lo impersonifica
3. L'amministratore va nella sezione segnalata
4. Riproduce il problema
5. Individua il problema di permessi o configurazione
6. Esce dall'impersonificazione
7. Corregge le impostazioni dell'utente
8. L'utente conferma la risoluzione

### Formazione di Nuovi Utenti

**Scenario:** formazione su un flusso di lavoro complesso

**Flusso:**
1. L'utente abilita il consenso (24 ore)
2. L'amministratore lo impersonifica
3. Esegue i passaggi del flusso
4. Documenta ogni azione
5. Esce dall'impersonificazione
6. Condivide l'audit con l'utente
7. L'utente rivede le azioni compiute
8. L'utente si esercita in autonomia

### Verifica dei Permessi

**Scenario:** verificare che un utente abbia i permessi corretti

**Flusso:**
1. L'utente abilita il consenso (1 ora)
2. L'amministratore lo impersonifica
3. Prova l'accesso alle varie funzionalità
4. Documenta cosa è visibile e accessibile
5. Esce dall'impersonificazione
6. Corregge i permessi se necessario

## Risoluzione Problemi

### Impossibile Impersonificare un Utente

**Problema:** il pulsante di impersonificazione è disabilitato

**Soluzioni:**
1. Controlla che l'utente abbia abilitato il consenso:
   - Chiedigli di abilitarlo in **Account** > **Impersonificazione**
   - Verifica che il consenso non sia scaduto
2. Verifica di avere i permessi:
   - Devi appartenere all'organizzazione Owner (ruolo Staff o account `owner`)
3. Controlla lo stato dell'utente:
   - L'utente non deve essere sospeso
   - L'utente non deve essere eliminato
4. Verifica di non stare già impersonificando:
   - Esci prima dall'impersonificazione in corso

### Consenso Non Mostrato

**Problema:** l'utente ha abilitato il consenso ma l'amministratore non lo vede

**Soluzioni:**
1. Ricarica la pagina (Ctrl+F5)
2. Aspetta 30 secondi (propagazione della cache)
3. Controlla la scadenza del consenso
4. Verifica che l'utente abbia salvato il consenso
5. Controlla che non l'abbia revocato per errore

### La Sessione Termina Inaspettatamente

**Problema:** vieni espulso dalla sessione di impersonificazione

**Cause possibili:**
- L'utente ha revocato il consenso
- La durata del consenso è scaduta
- Il token è scaduto
- L'utente è stato sospeso
- Interruzione di rete

**Soluzioni:**
1. Controlla se il consenso è ancora attivo
2. Chiedi all'utente di riabilitarlo
3. Controlla la scadenza del consenso
4. Verifica la tua connessione di rete

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

**Abilitare il consenso:**
- Abilitalo solo quando serve o ti viene chiesto
- Imposta la durata minima necessaria
- Revocalo quando il supporto è concluso
- Rivedi l'audit dopo l'impersonificazione

**Privacy:**
- Il consenso è del tutto volontario
- Sei tu a decidere quando e per quanto
- Puoi vedere tutto quello che è stato fatto

### Per gli Amministratori

**Prima di impersonificare:**
- Abbi uno scopo chiaro
- Chiedi all'utente di abilitare il consenso
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
