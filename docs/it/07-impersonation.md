# Impersonificazione Utente

Scopri come usare la funzionalità di impersonificazione per risolvere problemi degli utenti rispettando privacy e consenso.

## Cos'è l'Impersonificazione Utente?

L'impersonificazione utente consente agli amministratori autorizzati di accedere temporaneamente alla piattaforma My come un altro utente. Questa funzionalità è utile per:

- **Risoluzione Problemi**: Riprodurre problemi che gli utenti stanno sperimentando
- **Supporto**: Aiutare gli utenti con operazioni complesse
- **Formazione**: Dimostrare funzionalità agli utenti
- **Testing**: Verificare permessi e controlli di accesso

## Caratteristiche Principali

### 🔒 Design Focalizzato sulla Privacy

- **Consenso Utente Richiesto**: Gli utenti devono abilitare esplicitamente l'impersonificazione
- **Tempo Limitato**: Gli utenti controllano per quanto tempo è consentita l'impersonificazione (1-168 ore)
- **Trasparenza Completa**: Tutte le azioni loggate e visibili all'utente
- **Revoca Facile**: Gli utenti possono disabilitare il consenso in qualsiasi momento

### 🛡️ Controlli di Sicurezza

- **Basato su Permessi**: Solo utenti Super Admin o dell'organizzazione Owner possono impersonificare
- **No Auto-Impersonificazione**: Non puoi impersonificare il tuo stesso account
- **No Concatenazione**: Non puoi impersonificare mentre stai già impersonificando un altro utente
- **Scadenza Automatica**: Il consenso scade automaticamente dopo la durata definita dall'utente
- **Tracciamento Sessioni**: Ogni sessione di impersonificazione ha un ID univoco per l'audit

### 📊 Traccia Audit Completa

- Ogni chiamata API durante l'impersonificazione viene loggata
- Gli utenti possono vedere tutte le azioni eseguite durante l'impersonificazione
- Dati sensibili automaticamente oscurati dai log
- Organizzazione basata su sessioni per revisione facile

## Chi Può Impersonificare?

### Permessi Richiesti

**Ruolo Super Admin:**
- Gli utenti con ruolo Super Admin hanno il permesso `impersonate:users`
- Possono impersonificare qualsiasi utente (con il loro consenso)
- Assegnato solo dagli utenti dell'organizzazione Owner

**Utenti Organizzazione Owner:**
- Hanno automaticamente capacità di impersonificazione
- Possono impersonificare utenti nella loro gerarchia organizzativa
- Nessun assegnazione ruolo aggiuntivo necessaria

**Tutti gli Altri:**
- Non possono vedere le funzionalità di impersonificazione
- Non possono impersonificare nessun utente

## Flusso di Lavoro Impersonificazione

### Passo 1: Utente Abilita Consenso

Prima che l'impersonificazione possa avvenire, l'utente target deve abilitare il consenso.

**Per gli Utenti:**

1. Accedi al tuo account
2. Naviga su **Impostazioni Account** > **Impersonificazione**
3. Trova la sezione **Consenso all'Impersonificazione**
4. Clicca **Abilita Impersonificazione**
5. Imposta durata (1-168 ore)
6. Clicca **Salva**

**Cosa accade:**
- Il consenso viene registrato con timestamp
- L'amministratore viene notificato che il consenso è disponibile
- Scade automaticamente dopo la durata
- Può essere revocato in qualsiasi momento

**Opzioni Durata:**
- **1-24 ore**: Risoluzione problemi a breve termine
- **24-72 ore**: Supporto multi-giorno
- **72-168 ore**: Accesso esteso (max 1 settimana)

### Passo 2: Amministratore Impersonifica Utente

**Per Amministratori (Super Admin o Owner):**

1. Naviga su **Utenti**
2. Trova l'utente target
3. Controlla se **Impersonifica utente** è disponibile (usando il menu kebab)
4. Clicca **Impersonifica Utente**
5. Conferma l'azione
6. Ora stai agendo come quell'utente

**Durante l'Impersonificazione:**

Vedrai:
- **Banner in alto**: "Stai impersonificando [Nome Utente]"
- **Pulsante Esci**: Clicca per tornare al tuo account
- **Tutte le funzionalità**: Esattamente come le vede l'utente
- **Permessi dell'utente**: Filtrati dai loro permessi effettivi

### Passo 3: Esegui Azioni di Supporto

Durante l'impersonificazione:

- Naviga la piattaforma come farebbe l'utente
- Riproduci problemi segnalati
- Esegui azioni per conto dell'utente
- Testa funzionalità e permessi
- Documenta le scoperte

**Ricorda:**
- Tutte le azioni sono loggate
- L'utente può vedere tutto ciò che fai
- Tratta i dati dell'utente con rispetto
- Esci dall'impersonificazione quando hai finito

### Passo 4: Esci dall'Impersonificazione

**Per uscire dall'impersonificazione:**

1. Clicca pulsante **Esci Impersonificazione** nel banner
3. Torni al tuo account originale
4. La sessione di impersonificazione viene chiusa

**Uscita Automatica:**
- La sessione scade dopo la durata del consenso dell'utente
- Se l'utente revoca il consenso durante l'impersonificazione
- Se il token scade (segue la durata del consenso)

## Per gli Utenti: Gestione Consenso

### Abilitazione Consenso Impersonificazione

**Quando abilitare:**
- Quando hai un problema e hai bisogno di supporto
- Quando richiedi aiuto dall'amministratore
- Prima di una sessione di formazione
- Quando l'amministratore chiede il consenso

**Come abilitare:**

1. Vai su **Impostazioni Account** > **Impersonificazione**
2. Clicca **Consenso all'Impersonificazione**
3. Scegli durata:
   ```
   ○ 1 ora   - Supporto rapido
   ○ 24 ore  - Supporto stesso giorno
   ○ 72 ore  - Problema multi-giorno
   ○ Personalizzato - Specifica ore (max 168)
   ```
4. Clicca **Abilita**

**Conferma:**
```
✓ Consenso impersonificazione abilitato
  Scade: [Data e ora]
  Durata: [X] ore
```

### Controllo Stato Consenso

**Per controllare se il consenso è attivo:**

1. Vai su **Impostazioni Account** > **Impersonificazione**
2. Visualizza sezione **Consenso Impersonificazione**:
   ```
   Stato: Attivo
   Scade: 2025-11-07 10:30:00 UTC
   ```

### Revoca Consenso

**Per disabilitare il consenso:**

1. Vai su **Impostazioni Account** > **Impersonificazione**
2. Clicca **Revoca Consenso**
3. Conferma l'azione

**Effetti:**
- Consenso immediatamente disabilitato
- Sessioni di impersonificazione attive terminate
- L'amministratore non può più impersonificare
- Può essere ri-abilitato in qualsiasi momento

### Visualizzazione Audit Impersonificazione

**Per vedere chi ti ha impersonificato:**

1. Vai su **Impostazioni Account** > **Impersonificazione**
2. Sotto **Sessioni**
3. Vedi storico completo:
   ```
   Iniziata: 2025-11-06 10:00:00 UTC
   Terminata: 2025-11-06 11:30:00 UTC
   Durata: 1.5 ore
   Impersonificatore: John Admin (john@example.com)
   Stato: In corso
   ```

4. Clicca **Mostra log audit** per vedere tutte le azioni

**Informazioni Audit:**
- Data e ora di ogni azione
- Endpoint API chiamato
- Dati sensibili automaticamente oscurati
- Risultato (successo/fallimento)

## Per Amministratori: Uso Impersonificazione

### Controllo Disponibilità Impersonificazione

**Nell'elenco utenti:**

Gli utenti con consenso attivo mostrano:
- Stato **Impersonifica utente** abilitato
- Tempo di scadenza consenso
- Clicca per impersonificare

**Utenti senza consenso:**
- Stato **Impersonifica utente** disabilitato

### Avvio Impersonificazione

**Requisiti:**
- L'utente ha consenso attivo
- Hai ruolo Super Admin o ruolo organizzazione Owner
- L'utente non è eliminato o sospeso
- Non stai già impersonificando qualcuno

**Passi:**

1. **Trova Utente**:
   - Naviga su **Utenti**
   - Cerca utente target

2. **Verifica Consenso**:
   - Controlla stato **Impersonifica utente** abilitato
   - Controlla tempo scadenza consenso
   - Assicurati tempo sufficiente per le tue esigenze

3. **Inizia Impersonificazione**:
   - Clicca **Impersonifica Utente** (usando menu kebab)
   - Conferma dialogo:
     ```
     Agirai temporaneamente come utente Edoardo Spadoni e avrai i suoi permessi.

      Per tornare al tuo account, clicca l'icona di chiusura sul badge di impersonificazione nella barra superiore.

     [Annulla] [Impersonifica utente]
     ```

4. **Conferma**:
   - Ora stai impersonificando l'utente
   - Il banner appare in alto
   - La sessione inizia

### Durante la Sessione di Impersonificazione

**Indicatori Visivi:**

Banner in alto di ogni pagina:

**Cosa Vedi:**
- Esatta stessa interfaccia dell'utente
- Permessi dell'utente (potrebbero essere più restrittivi dei tuoi)
- Organizzazione e dati dell'utente
- Personalizzazioni e preferenze dell'utente

**Cosa Puoi Fare:**
- Navigare tutte le pagine accessibili all'utente
- Eseguire qualsiasi azione eseguibile dall'utente
- Creare/modificare/eliminare in base ai permessi dell'utente
- Testare funzionalità e riprodurre problemi

**Cosa Non Puoi Fare:**
- Accedere a funzionalità non accessibili all'utente
- Bypassare le restrizioni dei permessi dell'utente
- Impersonificare un altro utente mentre stai impersonificando
- Modificare il tuo stesso account

**Best Practice:**
- Documenta le tue azioni
- Minimizza il tempo in impersonificazione
- Esegui solo le azioni necessarie
- Informa l'utente di cosa hai fatto
- Esci quando hai finito

### Uscita dall'Impersonificazione

**Uscita Normale:**

- Clicca pulsante **X** nel banner
- Torna al tuo account

**Uscita Automatica:**

L'impersonificazione termina automaticamente quando:
- La durata del consenso scade
- L'utente revoca il consenso
- Il token della sessione scade
- Effettui il logout
- L'utente viene sospeso/eliminato

## Sicurezza e Privacy

### Cosa viene Loggato

**Informazioni Loggate:**
- Timestamp di ogni azione
- Endpoint API e metodo (GET, POST, ecc.)
- Codice stato HTTP (200, 404, ecc.)
- Parametri richiesta (dati sensibili oscurati)
- Stato risposta (dati sensibili oscurati)

**Automaticamente Oscurati:**
- Password
- Token di autenticazione
- Secret di sistema
- Qualsiasi campo contenente "password", "secret", "token"

**Esempio Entry Log:**
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
    "password": "[OSCURATO]"
  }
}
```

### Protezione Dati

**Controllo Utente:**
- Gli utenti scelgono quando abilitare il consenso
- Gli utenti controllano la durata
- Gli utenti possono revocare in qualsiasi momento
- Gli utenti vedono la traccia audit completa

**Protezione Piattaforma:**
- Nessun accesso senza consenso
- Scadenza automatica
- Logging completo
- Oscuramento dati sensibili

**Conformità:**
- Traccia audit per requisiti normativi
- Modello di accesso basato su consenso
- Visibilità e controllo utente
- Privacy dati rispettata

## Casi d'Uso Comuni

### Risoluzione Problemi Utente

**Scenario:** L'utente segnala che non riesce a vedere una funzionalità

**Flusso di Lavoro:**
1. L'utente abilita consenso impersonificazione (1 ora)
2. L'amministratore impersonifica l'utente
3. L'amministratore naviga all'area segnalata
4. Riproduce il problema
5. Identifica problema permessi/configurazione
6. Esce dall'impersonificazione
7. Risolve il problema nelle impostazioni dell'utente
8. L'utente conferma la risoluzione

### Formazione Nuovi Utenti

**Scenario:** Formazione utente su flusso di lavoro complesso

**Flusso di Lavoro:**
1. L'utente abilita impersonificazione (24 ore)
2. L'amministratore impersonifica
3. Esegue i passaggi del flusso di lavoro
4. Documenta ogni azione
5. Esce dall'impersonificazione
6. Condivide l'audit con l'utente
7. L'utente rivede le azioni eseguite
8. L'utente pratica indipendentemente

### Verifica Permessi

**Scenario:** Verificare che l'utente abbia i permessi corretti

**Flusso di Lavoro:**
1. L'utente abilita impersonificazione (1 ora)
2. L'amministratore impersonifica
3. Testa accesso a varie funzionalità
4. Documenta cosa è visibile/accessibile
5. Esce dall'impersonificazione
6. Aggiusta permessi se necessario

## Risoluzione Problemi

### Impossibile Impersonificare Utente

**Problema:** Il pulsante Impersonifica è disabilitato

**Soluzioni:**
1. Controlla che l'utente abbia abilitato il consenso:
   - Chiedi all'utente di abilitare in Profilo > Sicurezza
   - Verifica che il consenso non sia scaduto
2. Verifica di avere i permessi:
   - Ruolo Super Admin OPPURE
   - Ruolo organizzazione Owner
3. Controlla stato utente:
   - L'utente non è sospeso
   - L'utente non è eliminato
4. Verifica di non stare già impersonificando:
   - Esci dall'impersonificazione corrente prima

### Consenso Non Mostrato

**Problema:** L'utente ha abilitato il consenso ma l'amministratore non lo vede

**Soluzioni:**
1. Aggiorna la pagina (Ctrl+F5)
2. Attendi 30 secondi (propagazione cache)
3. Controlla tempo scadenza consenso
4. Verifica che l'utente abbia salvato il consenso
5. Controlla che l'utente non l'abbia accidentalmente revocato

### Sessione Impersonificazione Termina Inaspettatamente

**Problema:** Espulso dalla sessione di impersonificazione

**Possibili Cause:**
- L'utente ha revocato il consenso
- La durata del consenso è scaduta
- Il token è scaduto
- L'utente è stato sospeso
- Interruzione di rete

**Soluzioni:**
1. Controlla se il consenso è ancora attivo
2. Chiedi all'utente di ri-abilitare il consenso
3. Controlla tempo scadenza consenso
4. Verifica la tua connessione di rete

### Impossibile Vedere Dati Utente

**Problema:** Durante l'impersonificazione, impossibile vedere i dati attesi

**Spiegazione:**
- Vedi esattamente ciò che vede l'utente
- L'utente potrebbe avere permessi ristretti
- L'accesso all'organizzazione potrebbe essere limitato
- Questo è il comportamento previsto

**Soluzioni:**
1. Verifica i ruoli assegnati all'utente
2. Controlla l'appartenenza organizzazione dell'utente
3. Rivedi i permessi gerarchici
4. Aggiusta i permessi dell'utente se necessario

## Best Practice

### Per gli Utenti

**Abilitazione Consenso:**
- Abilita solo quando richiesto o necessario
- Imposta la durata minima necessaria
- Revoca quando il supporto è completato
- Rivedi la traccia audit dopo l'impersonificazione

**Privacy:**
- Fidati dei tuoi amministratori
- Il consenso è interamente volontario
- Tu controlli quando e per quanto tempo
- Puoi vedere tutto ciò che hanno fatto

### Per gli Amministratori

**Prima di Impersonificare:**
- Hai uno scopo chiaro per l'impersonificazione
- Richiedi all'utente di abilitare il consenso
- Pianifica cosa devi fare
- Stima il tempo necessario

**Durante l'Impersonificazione:**
- Lavora efficientemente
- Documenta le tue azioni
- Esegui solo le operazioni necessarie
- Rispetta la privacy dell'utente
- Esci prontamente quando hai finito

**Dopo l'Impersonificazione:**
- Informa l'utente di cosa è stato fatto
- Documenta le scoperte
- Condividi l'audit se richiesto
- Fai follow-up sui problemi trovati

### Per le Organizzazioni

**Policy:**
- Definisci quando l'impersonificazione è appropriata
- Documenta il processo di approvazione
- Forma gli amministratori
- Rivedi regolarmente le tracce audit

**Sicurezza:**
- Limita l'assegnazione ruolo Super Admin
- Monitora l'uso dell'impersonificazione
- Rivedi i log audit
- Investiga pattern inusuali

## Documentazione Correlata

- [Gestione Utenti](03-users.md)
- [Guida Autenticazione](01-authentication.md)
- [Documentazione API Backend](https://github.com/NethServer/my/blob/main/backend/README.md)
