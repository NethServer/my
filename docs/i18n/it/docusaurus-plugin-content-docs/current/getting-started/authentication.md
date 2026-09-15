---
sidebar_position: 1
---

# Autenticazione

Come accedere a My, gestire le proprie credenziali e cosa permettono i propri ruoli.

## Primo Accesso

### Email di Benvenuto

Quando un amministratore crea il tuo account, riceverai un'email di benvenuto contenente:

- Il tuo **indirizzo email** (usato come username)
- Una **password temporanea**
- Un **link diretto** alla piattaforma

### Accesso

1. Apri il link di accesso presente nell'email di benvenuto
2. Inserisci il tuo **indirizzo email**
3. Inserisci la **password temporanea**
4. Clicca su **Accedi**

### Cambio Password al Primo Accesso

Al primo accesso ti viene richiesto di sostituire la password temporanea:

1. Inserisci la password attuale (quella temporanea)
2. Crea una nuova password che soddisfi i requisiti qui sotto
3. Conferma la nuova password

:::warning
La password temporanea deve essere cambiata al primo accesso. Non sarà possibile procedere senza impostare una nuova password.
:::

## Requisiti Password

Ogni password impostata deve soddisfare tutti questi requisiti:

| Requisito | Dettaglio |
|-----------|-----------|
| Lunghezza minima | **12 caratteri** |
| Lunghezza massima | 128 caratteri |
| Lettera maiuscola | Almeno una (A-Z) |
| Lettera minuscola | Almeno una (a-z) |
| Numero | Almeno uno (0-9) |
| Carattere speciale | Almeno uno tra ``!@#$%^&*()_+-=[]{};':"\|,.<>/?~` `` |
| Caratteri ripetuti | Non più di 3 caratteri identici di fila |
| Pattern deboli | Rifiutati: `password`, `123456`, `qwerty`, `admin` e simili, più le sequenze tipo `123` o `abc` |

La validazione segnala **un** problema alla volta, per guidarti passo passo invece di elencare tutto insieme.

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

Puoi aggiornare:

- **Nome** e **cognome**
- **Email** (se consentito dall'amministratore: è anche il tuo username)
- **Numero di telefono**
- **Avatar** (vedi [Gestione Avatar](../features/avatar.md))

:::note
La modifica dell'email può richiedere una nuova autenticazione.
:::

## Sicurezza

### Protezione Password

- Le password non vengono mai archiviate in chiaro
- Le password temporanee scadono dopo il primo utilizzo
- I tentativi di accesso falliti vengono registrati
- Gli amministratori possono sospendere un account

### Sessioni

- Il token di accesso dura **30 minuti** e viene rinnovato in background, quindi non ti accorgi della scadenza
- Il token di refresh è valido **7 giorni**: è la durata massima di una sessione senza rifare l'accesso
- I token di refresh vengono **ruotati** a ogni uso, e riutilizzarne uno vecchio invalida l'intera catena: un token rubato non può essere riusato
- Ogni refresh rilegge anche i tuoi dati dal provider di identità
- Il logout invalida immediatamente la sessione

### Autenticazione Multi-Fattore (MFA)

My delega l'autenticazione a Logto, quindi l'MFA si configura lì, a livello di tenant, non dentro My. Quando è abilitata, dopo la password viene richiesto un secondo fattore.

- Contatta il tuo amministratore per farla abilitare
- I metodi supportati dipendono da cosa è abilitato sul tenant Logto (app authenticator, ed eventuali altri se configurati)

## Risoluzione Problemi

### Password Dimenticata

1. Nella pagina di login, clicca su **Password dimenticata?**
2. Inserisci il tuo indirizzo email
3. Controlla la tua casella di posta per il link di reset
4. Segui le istruzioni nell'email per impostare una nuova password

### Account Bloccato

Se il tuo account è stato sospeso:

- Vedrai un messaggio di errore "Account sospeso"
- Contatta il tuo amministratore per la riattivazione
- Un amministratore con i permessi appropriati può riattivarlo dalla sezione **Gestione Utenti**

### Sessione Scaduta

1. Verrai reindirizzato alla pagina di login
2. Accedi nuovamente con le tue credenziali
3. Il lavoro non salvato viene perso
4. Se il problema persiste, cancella i cookie del browser e riprova

## Ruoli

I tuoi permessi dipendono da due ruoli che agiscono insieme.

### Ruoli Organizzazione

I ruoli organizzazione determinano la posizione nella gerarchia aziendale:

- **Owner**: accesso completo alla piattaforma (Nethesis) — ogni membro dell'organizzazione Owner ha visibilità globale su tutte le aziende, i sistemi e gli utenti
- **Distributore**: può gestire rivenditori e clienti
- **Rivenditore**: può gestire i clienti
- **Cliente**: può vedere i dati della propria organizzazione

### Ruoli Utente

| Ruolo | Descrizione | Capacità Principali |
|-------|-------------|---------------------|
| **Admin** | Gestione completa dell'organizzazione | Utenti, sistemi, applicazioni, configurazione allarmi, add-on, rebranding |
| **Backoffice** | Operazioni amministrative | Utenti, applicazioni, attivazione e revoca add-on. Sui sistemi ha **sola lettura**: non può crearli né modificarli. Niente allarmi, niente rebranding |
| **Support** | Operazioni tecniche sui sistemi | Sistemi (creazione, modifica, eliminazione, rigenerazione secret), inventario, heartbeat, allarmi e silenziamenti, applicazioni. **Nessun accesso agli utenti**, nemmeno in lettura |
| **Reader** | Sola lettura | Visualizza utenti, organizzazioni, sistemi, inventario, applicazioni e add-on, ed esporta tutto ciò che può leggere. Nessuna modifica |
| **Staff** | Personale trasversale Nethesis (solo organizzazione Owner) | Gestione completa su tutte le aziende, più impersonificazione e connessione remota. Non può eliminare definitivamente sistemi o utenti |

### Permessi Combinati

I permessi effettivi sono l'**unione** del ruolo organizzazione e del ruolo utente.

**Esempio**: ruolo organizzazione **Distributore** + ruolo utente **Admin**
- Gestire rivenditori e clienti sotto la propria organizzazione (dal ruolo organizzazione)
- Creare e modificare sistemi e utenti (dal ruolo utente)

**Esempio**: ruolo organizzazione **Cliente** + ruolo utente **Reader**
- Vedere solo i dati della propria organizzazione (dal ruolo organizzazione)
- Sola lettura, nessuna modifica (dal ruolo utente)

La matrice completa è in [Gestione Utenti](../platform/users.md#riferimento-permessi).

## Prossimi Passi

Una volta effettuato l'accesso, in base ai tuoi permessi puoi:

- [Gestire le organizzazioni](../platform/organizations.md)
- [Gestire gli utenti](../platform/users.md) — serve `manage:users` (Admin, Backoffice o Staff)
- [Gestire i sistemi](../systems/management.md) — serve `manage:systems` (Admin, Support o Staff)
- Consultare la tua [Dashboard](../features/dashboard.md)
