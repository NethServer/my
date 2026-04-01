---
sidebar_position: 1
---

# Autenticazione

Questa guida spiega come accedere a My, gestire il proprio profilo e comprendere il sistema di ruoli.

## Primo Accesso

### Email di Benvenuto

Quando un amministratore crea il tuo account, riceverai un'email di benvenuto contenente:

- Il tuo **indirizzo email** (usato come username)
- Una **password temporanea**
- Un **link diretto** alla piattaforma

### Accesso

1. Apri il browser e vai alla pagina di login di My
2. Inserisci il tuo **indirizzo email**
3. Inserisci la **password temporanea** fornita nell'email
4. Clicca su **Accedi**

### Cambio Password al Primo Accesso

Al primo accesso ti verrà richiesto di cambiare la password temporanea.

:::warning
La password temporanea deve essere cambiata al primo accesso. Non sarà possibile procedere senza impostare una nuova password.
:::

## Requisiti Password

La password deve soddisfare tutti i seguenti requisiti:

| Requisito | Dettaglio |
|-----------|-----------|
| Lunghezza minima | 8 caratteri |
| Lettera maiuscola | Almeno una (A-Z) |
| Lettera minuscola | Almeno una (a-z) |
| Numero | Almeno uno (0-9) |
| Carattere speciale | Almeno uno (!@#$%^&*) |

:::tip
Usa una passphrase lunga e unica per ogni servizio. Un password manager può aiutarti a gestire le credenziali in modo sicuro.
:::

## Gestione Profilo

### Cambio Password

1. Accedi a **Account** dal menu utente
2. Nella sezione **Cambio Password**, inserisci:
   - La password attuale
   - La nuova password
   - La conferma della nuova password
3. Clicca su **Salva**

### Aggiornamento Informazioni

Puoi aggiornare le seguenti informazioni del profilo:

- **Nome** e **Cognome**
- **Email** (se consentito dall'amministratore)
- **Numero di telefono**
- **Avatar** (vedi [Gestione Avatar](../features/avatar))

## Sicurezza

### Protezione Password

- Le password sono archiviate in modo sicuro tramite hashing
- I tentativi di accesso falliti vengono monitorati
- Gli account possono essere sospesi dagli amministratori

### Sessioni

- Le sessioni di accesso hanno una durata di **24 ore**
- I token di refresh sono validi per **7 giorni**
- Ogni refresh del token aggiorna i dati dal provider di identità
- È possibile effettuare il logout manuale dalla piattaforma

### Autenticazione Multi-Fattore (MFA)

My supporta l'autenticazione multi-fattore tramite Logto. Quando abilitata, dopo l'inserimento della password verrà richiesto un secondo fattore di autenticazione.

:::note
L'MFA è configurata a livello di tenant Logto. Contatta il tuo amministratore per abilitarla.
:::

## Risoluzione Problemi

### Password Dimenticata

1. Nella pagina di login, clicca su **Password dimenticata?**
2. Inserisci il tuo indirizzo email
3. Controlla la tua casella di posta per il link di reset
4. Segui le istruzioni nell'email per impostare una nuova password

### Account Bloccato

Se il tuo account è stato sospeso:

- Contatta il tuo amministratore per la riattivazione
- Un amministratore con i permessi appropriati può riattivare il tuo account dalla sezione **Gestione Utenti**

### Sessione Scaduta

Se la sessione è scaduta:

1. Verrai reindirizzato alla pagina di login
2. Accedi nuovamente con le tue credenziali
3. Se il problema persiste, cancella i cookie del browser e riprova

## Ruoli

### Ruoli Organizzazione

I ruoli organizzazione determinano la posizione nella gerarchia aziendale e definiscono quali entità un utente può gestire.

| Ruolo | Descrizione | Può Gestire |
|-------|-------------|-------------|
| **Owner** | Proprietario della piattaforma (Nethesis) | Tutto: distributori, rivenditori, clienti |
| **Distributore** | Partner di distribuzione | Rivenditori e clienti sotto di sé |
| **Rivenditore** | Partner di rivendita | Solo clienti sotto di sé |
| **Cliente** | Utente finale | Solo i propri dati (sola lettura) |

### Ruoli Utente

I ruoli utente determinano le capacità tecniche all'interno della piattaforma, indipendentemente dalla posizione gerarchica.

| Ruolo | Descrizione | Capacità Principali |
|-------|-------------|---------------------|
| **Super Admin** | Amministrazione completa della piattaforma | Tutte le operazioni, incluse quelle pericolose. Solo nell'organizzazione Owner |
| **Admin** | Gestione sistemi e utenti | Gestione sistemi, utenti, operazioni pericolose |
| **Backoffice** | Operazioni di backoffice | Gestione organizzazioni, applicazioni, rebranding |
| **Support** | Operazioni di supporto standard | Accesso lettura ai sistemi, operazioni di supporto |
| **Reader** | Sola lettura | Visualizzazione dati senza possibilità di modifica |

### Permessi Combinati

I permessi effettivi di un utente sono la combinazione del ruolo organizzazione e del ruolo utente.

**Esempio**: Un utente con ruolo organizzazione **Distributore** e ruolo utente **Admin** può:
- Gestire rivenditori e clienti sotto la propria organizzazione (dal ruolo organizzazione)
- Creare e modificare sistemi e utenti (dal ruolo utente)

**Esempio**: Un utente con ruolo organizzazione **Cliente** e ruolo utente **Reader** può:
- Visualizzare solo i dati della propria organizzazione (dal ruolo organizzazione)
- Solo lettura, nessuna modifica (dal ruolo utente)
