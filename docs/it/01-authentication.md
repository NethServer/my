# Per Iniziare - Autenticazione

Impara come accedere alla piattaforma My e gestire il tuo account.

## Primo Accesso

Quando il tuo account viene creato da un amministratore, riceverai un'email di benvenuto contenente:

- Il tuo nome utente (indirizzo email)
- Una password temporanea
- Un link diretto alla pagina di login

### Effettuare l'Accesso

1. Apri l'URL di login fornito nell'email di benvenuto
2. Inserisci il tuo indirizzo email
3. Inserisci la password temporanea
4. Clicca **Accedi**

### Cambio Password al Primo Accesso

Al primo accesso con la password temporanea, dovrai:

1. Inserire la password corrente (temporanea)
2. Creare una nuova password sicura
3. Confermare la nuova password

**Requisiti della Password:**

- Minimo 8 caratteri
- Almeno una lettera maiuscola
- Almeno una lettera minuscola
- Almeno un numero
- Almeno un carattere speciale

## Gestione del Profilo

### Cambiare la Password

Per cambiare la password in qualsiasi momento:

1. Clicca sull'icona del profilo in alto a destra
2. Seleziona **Impostazioni Account**
3. Clicca **Cambia Password**
4. Inserisci la password corrente
5. Inserisci la nuova password
6. Conferma la nuova password
7. Clicca **Salva Modifiche**

### Aggiornare le Informazioni del Profilo

Puoi aggiornare le informazioni del profilo:

1. Clicca sull'icona del profilo in alto a destra
2. Seleziona **Impostazioni Account**
3. Aggiorna i seguenti campi:
   - **Nome Completo**: Il tuo nome visualizzato
   - **Email**: Il tuo indirizzo email (anche il tuo nome utente)
   - **Numero di Telefono**: Numero di contatto opzionale
4. Clicca **Salva profilo**

**Nota:** Le modifiche all'email potrebbero richiedere una nuova autenticazione.

## Funzionalità di Sicurezza

### Sicurezza della Password

- La tua password non viene mai memorizzata in chiaro
- Le password temporanee scadono dopo il primo utilizzo

### Gestione delle Sessioni

- Le sessioni scadono dopo 24 ore di inattività
- I token di aggiornamento sono validi per 7 giorni
- Il logout invalida immediatamente la sessione

## Risoluzione Problemi

### Password Dimenticata

Se dimentichi la password:

1. Usa il link "Hai dimenticato la password?" nella pagina di Login

### Account Bloccato

Se il tuo account è sospeso:

- Vedrai un messaggio di errore "Account sospeso"
- Contatta il tuo amministratore di sistema per riattivare l'account
- Solo gli amministratori possono sospendere/riattivare gli account

### Sessione Scaduta

Se la sessione scade:

1. Verrai automaticamente reindirizzato alla pagina di login
2. Accedi nuovamente con le tue credenziali
3. Il lavoro precedente non viene salvato durante la scadenza della sessione

## Autenticazione Multi-Fattore (MFA)

Attualmente, My utilizza Logto come provider di identità. Le impostazioni MFA sono gestite tramite Logto:

- Contatta il tuo amministratore per abilitare l'MFA
- L'MFA può essere configurata a livello di organizzazione
- Metodi supportati: App di autenticazione, SMS (se configurato)

## Prossimi Passi

Una volta effettuato l'accesso, puoi:

- [Gestire Organizzazioni](02-organizations.md) (se hai i permessi appropriati)
- [Gestire Utenti](03-users.md) (utenti Admin o Support)
- [Gestire Sistemi](04-systems.md) (utenti Support)
- Visualizzare la dashboard e le statistiche

## Ruoli Utente

I tuoi permessi dipendono dai ruoli assegnati:

### Ruoli Organizzazione (Gerarchia Aziendale)
- **Owner**: Accesso completo alla piattaforma (Nethesis)
- **Distributore**: Può gestire rivenditori e clienti
- **Rivenditore**: Può gestire clienti
- **Cliente**: Può visualizzare i dati della propria organizzazione

### Ruoli Utente (Capacità Tecniche)
- **Super Admin**: Amministrazione completa della piattaforma
- **Admin**: Amministrazione organizzazione, gestione utenti
- **Support**: Gestione sistemi, operazioni tecniche
- **Backoffice**: Gestione utenti, operazioni di backoffice
- **Reader**: Modalità lettura

I tuoi permessi effettivi sono la combinazione di entrambi i tipi di ruolo.
