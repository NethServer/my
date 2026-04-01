---
sidebar_position: 3
---

# Impersonificazione Utente

L'impersonificazione consente agli amministratori Owner di accedere alla piattaforma con l'identità di un altro utente, per finalità di supporto e risoluzione problemi.

## Cos'è l'Impersonificazione

L'impersonificazione è una funzionalità che permette a un amministratore Owner di visualizzare e operare sulla piattaforma come se fosse un altro utente, vedendo esattamente ciò che quell'utente vede.

### Caratteristiche Principali

- **Privacy** - L'utente impersonificato può controllare il consenso
- **Sicurezza** - Sessioni limitate, nessun concatenamento
- **Audit** - Tutte le azioni sono registrate con l'indicazione dell'impersonificazione
- **Trasparenza** - L'indicatore di impersonificazione è sempre visibile nell'interfaccia

## Chi Può Impersonificare

L'impersonificazione è disponibile **esclusivamente** per gli utenti con:

- **Ruolo Organizzazione**: Owner
- **Ruolo Utente**: Super Admin

:::warning
L'impersonificazione non è disponibile per nessun altro ruolo. Distributori, Rivenditori e Clienti non possono impersonificare utenti.
:::

## Flusso di Lavoro

L'impersonificazione segue un processo in 4 passi:

### Passo 1: Verifica del Consenso

Prima di impersonificare un utente, il sistema verifica che l'utente target abbia dato il consenso all'impersonificazione.

:::note
Gli utenti possono gestire il consenso all'impersonificazione dalla propria pagina [Impostazioni Account](../getting-started/account). Se il consenso non è stato dato, l'impersonificazione non è possibile. La durata del consenso è configurabile dall'utente (1-168 ore).
:::

### Passo 2: Inizio Impersonificazione

1. Vai alla pagina di dettaglio dell'utente da impersonificare
2. Clicca su **Impersonifica**
3. Il sistema genera un token JWT temporaneo con i permessi dell'utente target

### Passo 3: Utilizzo

Una volta attiva l'impersonificazione:

- L'interfaccia mostra un **banner di impersonificazione** ben visibile in ogni pagina
- Navighi la piattaforma con i permessi dell'utente impersonificato
- Puoi visualizzare esattamente ciò che l'utente vede
- La sessione segue la durata del consenso dell'utente (1-168 ore)

### Passo 4: Uscita dall'Impersonificazione

Per terminare l'impersonificazione:

1. Clicca su **Esci dall'Impersonificazione** nel banner in alto
2. Torni alla tua sessione originale con i tuoi permessi

## Per gli Utenti: Gestione del Consenso

### Attivazione del Consenso

1. Vai alla pagina **Account**
2. Nella sezione **Consenso Impersonificazione**, attiva il toggle
3. Il consenso è immediatamente attivo

### Revoca del Consenso

1. Vai alla pagina **Account**
2. Nella sezione **Consenso Impersonificazione**, disattiva il toggle
3. Il consenso viene revocato immediatamente

:::tip
Se un amministratore sta attualmente impersonificando il tuo account e revochi il consenso, l'impersonificazione attiva non viene interrotta. La revoca impedisce nuove sessioni di impersonificazione.
:::

## Per gli Amministratori: Uso dell'Impersonificazione

### Come Impersonificare un Utente

1. Vai a **Utenti**
2. Cerca e seleziona l'utente da impersonificare
3. Nella pagina di dettaglio, verifica che il consenso sia attivo
4. Clicca su **Impersonifica**
5. La sessione di impersonificazione inizia

### Limitazioni

- **Nessuna auto-impersonificazione** - Non puoi impersonificare te stesso
- **Nessun concatenamento** - Non puoi impersonificare un utente mentre stai già impersonificando un altro utente
- **Durata limitata** - La sessione scade automaticamente dopo 1 ora
- **Solo utenti attivi** - Non puoi impersonificare utenti sospesi

## Sicurezza e Privacy

### Visualizzazione Audit Impersonificazione

Per vedere chi ti ha impersonificato:

1. Vai su **Impostazioni Account** > **Impersonificazione**
2. Sotto **Sessioni** visualizza lo storico completo:
   ```
   Iniziata: 2025-11-06 10:00:00 UTC
   Terminata: 2025-11-06 11:30:00 UTC
   Durata: 1.5 ore
   Impersonificatore: John Admin (john@example.com)
   Stato: In corso
   ```
3. Clicca **Mostra log audit** per vedere tutte le azioni

### Logging

Tutte le azioni eseguite durante l'impersonificazione sono registrate con:

- **Timestamp** dell'azione
- **Identità dell'amministratore** che sta impersonificando
- **Identità dell'utente** impersonificato
- **Azione** eseguita
- Il campo `impersonated_by` nel JWT identifica l'amministratore

**Esempio entry log:**
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

### Dati Oscurati

Durante l'impersonificazione, alcuni dati sensibili dell'utente target sono oscurati per proteggere la privacy:

- Password e credenziali non sono mai visibili
- Token di autenticazione e secret di sistema
- Qualsiasi campo contenente "password", "secret", "token"

## Casi d'Uso Comuni

### Risoluzione Problemi

Quando un utente segnala un problema:

1. Impersonifica l'utente
2. Riproduci il problema dal suo punto di vista
3. Verifica permessi e visibilità
4. Esci dall'impersonificazione
5. Applica la correzione

### Verifica Permessi

Per verificare che un utente abbia i permessi corretti:

1. Impersonifica l'utente
2. Naviga nelle sezioni della piattaforma
3. Verifica cosa l'utente può vedere e fare
4. Esci dall'impersonificazione

### Formazione

Per mostrare a un utente come usare la piattaforma:

1. Impersonifica l'utente
2. Registra lo schermo o condividi la sessione
3. Mostra le funzionalità dal punto di vista dell'utente

## Risoluzione Problemi

### Impossibile Impersonificare un Utente

- Verifica di avere il ruolo Owner + Super Admin
- Verifica che l'utente target abbia dato il consenso
- Verifica che l'utente target sia attivo (non sospeso)
- Verifica di non essere già in una sessione di impersonificazione

### Consenso Non Mostrato

**Problema:** L'utente ha abilitato il consenso ma l'amministratore non lo vede

**Soluzioni:**
1. Aggiorna la pagina (Ctrl+F5)
2. Attendi 30 secondi (propagazione cache)
3. Controlla tempo scadenza consenso
4. Verifica che l'utente abbia salvato il consenso
5. Controlla che l'utente non l'abbia accidentalmente revocato

### L'Impersonificazione È Scaduta

Le sessioni di impersonificazione scadono alla scadenza del consenso dell'utente. Per continuare:

1. Esci dall'impersonificazione (se il banner è ancora visibile)
2. Ricomincia con una nuova sessione di impersonificazione

### L'Utente Ha Revocato il Consenso

Se un utente revoca il consenso:

- Le sessioni di impersonificazione attive continuano fino alla scadenza
- Non è possibile avviare nuove sessioni di impersonificazione per quell'utente
- L'utente deve riattivare il consenso dalla propria pagina Account

## Best Practice

- **Usa l'impersonificazione solo quando necessario** - Per supporto e risoluzione problemi
- **Documenta le sessioni** - Registra il motivo dell'impersonificazione
- **Esci immediatamente** dopo aver completato l'operazione
- **Informa l'utente** quando possibile, che stai per impersonificare il suo account
- **Non apportare modifiche non richieste** durante l'impersonificazione

### Per le Organizzazioni

**Policy:**
- Definisci quando l'impersonificazione è appropriata
- Documenta il processo di approvazione
- Forma gli amministratori sull'uso corretto
- Rivedi regolarmente le tracce audit

**Sicurezza:**
- Limita l'assegnazione del ruolo Super Admin
- Monitora l'uso dell'impersonificazione
- Rivedi i log audit periodicamente
- Investiga pattern inusuali di impersonificazione
