# Gestione Utenti

Impara a creare e gestire gli utenti nella piattaforma My.

## Comprendere i Ruoli Utente

My utilizza un sistema a doppio ruolo che combina gerarchia aziendale e capacità tecniche.

### Ruoli Organizzazione (Gerarchia Aziendale)

Ereditati automaticamente dall'organizzazione dell'utente:

- **Owner**: Accesso completo alla piattaforma (solo Nethesis)
- **Distributore**: Gestisce rivenditori e clienti
- **Rivenditore**: Gestisce clienti
- **Cliente**: Visualizza i dati della propria organizzazione

### Ruoli Utente (Capacità Tecniche)

Assegnati manualmente agli utenti in base alla loro funzione lavorativa:

- **Super Admin**: Amministrazione completa della piattaforma con impersonificazione
  - Tutte le capacità Admin
  - Impersonificazione utente per troubleshooting
  - Operazioni di sistema avanzate
  - Controllo completo della piattaforma

- **Admin**: Amministrazione piattaforma
  - Gestione utenti
  - Gestione organizzazioni
  - Configurazione sistema
  - Operazioni pericolose (elimina, sospendi)

- **Backoffice**: Operazioni amministrative e reportistica
  - Visualizzazione utenti e organizzazioni
  - Monitoraggio e reportistica sistemi
  - Analisi e statistiche
  - Nessuna operazione distruttiva

- **Support**: Operazioni tecniche
  - Gestione sistemi
  - Visualizzazione inventario
  - Monitoraggio heartbeat
  - Operazioni standard

- **Reader**: Accesso in sola lettura
  - Visualizza utenti e organizzazioni
  - Visualizza sistemi e stato
  - Visualizza inventario e heartbeat
  - Nessuna capacità di modifica

### Permessi Combinati

I permessi finali di un utente sono la combinazione di **entrambi** i tipi di ruolo:

**Esempio 1:**
```
Organizzazione: Cliente (Pizza Express)
Ruolo Utente: Admin
→ Può gestire utenti solo nell'organizzazione Pizza Express
→ Può gestire sistemi solo per Pizza Express
```

**Esempio 2:**
```
Organizzazione: Distributore (ACME Distribution)
Ruolo Utente: Support
→ Può visualizzare rivenditori e clienti sotto ACME
→ Può gestire sistemi per tutti i clienti sotto ACME
→ Non può gestire utenti (richiede ruolo Admin)
```

## Creazione Utenti

### Prerequisiti

- Devi avere il ruolo **Admin**
- Puoi creare utenti solo per le organizzazioni che puoi gestire
- Indirizzo email valido per il nuovo utente

### Crea un Nuovo Utente

1. Naviga su **Utenti**
2. Clicca **Crea utente**
3. Compila il modulo:
   - **Nome**: Nome visualizzato dell'utente (es. "Mario Rossi")
   - **Email**: Indirizzo email dell'utente (sarà il loro username)
   - **Organizzazione**: Seleziona l'organizzazione
   - **Ruoli**: Seleziona uno o più ruoli (Super Admin, Admin, Backoffice, Support, Reader)
   - **Numero di Telefono** (opzionale): Telefono di contatto
4. Clicca **Crea utente**

**Esempio:**
```
Nome Completo: Mario Rossi
Email: mario.rossi@techsolutions.it
Organizzazione: Tech Solutions Italia (Rivenditore)
Ruoli Utente: Admin, Support
Telefono: +39 02 1234567
```

### Cosa Succede Dopo la Creazione

1. L'account utente viene creato in Logto
2. Una password temporanea viene generata automaticamente
3. Viene inviata un'email di benvenuto all'utente contenente:
   - Password temporanea
   - URL di login
   - Istruzioni per il cambio password
4. L'utente deve cambiare la password al primo accesso

**⚠️ Importante:** La password temporanea è mostrata **una sola volta** durante la creazione. Assicurati che l'utente riceva l'email di benvenuto.

## Gestione Utenti

### Visualizzazione Elenco Utenti

Naviga su **Utenti** per vedere:

- Nome utente ed email
- Organizzazione
- Ruoli utente
- Ruolo organizzazione (derivato dall'organizzazione)
- Stato (attivo/sospeso)

### Filtri e Ricerca

Usa i filtri per trovare utenti specifici:

- **Ricerca per nome o email**: Digita nella casella di ricerca
- **Ricerca per organizzazione**: Seleziona una o più organizzazioni
- **Ricerca per ruolo**: Super Admin, Admin, Backoffice, Support, Reader
- **Ordina per**: Nome, email, organizzazione

### Dettagli Utente

Clicca su un utente per visualizzare informazioni dettagliate:

- **Informazioni Profilo**:
  - Nome completo
  - Indirizzo email
  - Numero di telefono
  - Immagine profilo (se configurata tramite Logto)

- **Appartenenza Organizzazione**:
  - Organizzazione primaria
  - Ruolo organizzazione (Owner/Distributore/Rivenditore/Cliente)

- **Ruoli e Permessi**:
  - Ruoli utente assegnati (Admin, Support)
  - Elenco permessi effettivi

- **Attività**:
  - Data e ora ultimo accesso
  - Data creazione account
  - Ultimo cambio password

- **Stato**:
  - Attivo o sospeso
  - Motivo sospensione (se applicabile)

## Modifica Utenti

### Aggiorna Informazioni Utente

1. Naviga alla pagina dei dettagli utente
2. Clicca **Modifica**
3. Aggiorna i campi:
   - Nome
   - Indirizzo email
   - Organizzazione
   - Ruoli
   - Numero di telefono
4. Clicca **Salva utente**

**Nota:**
- Almeno un ruolo deve essere selezionato
- Non puoi modificare il tuo account tramite questa interfaccia (usa Impostazioni Profilo invece)

### Reimposta Password Utente

Come Admin, puoi reimpostare la password di un utente:

1. Naviga alla pagina dell'utente
2. Clicca **Reimposta Password** (usando il menu kebab)
3. Conferma l'azione
4. Viene generata una nuova password temporanea
5. Copia la password e inviala all'utente

**Casi d'Uso:**
- Utente ha dimenticato la password
- Incidente di sicurezza che richiede reset password
- Recupero account

## Gestione Stato Utente

### Sospensione di un Utente

Disabilita temporaneamente un account utente:

1. Naviga alla pagina dell'utente
2. Clicca **Sospendi** (usando il menu kebab)
4. Clicca **Sospendi**

**Effetti della sospensione:**
- L'utente non può accedere
- Le sessioni attive vengono immediatamente invalidate
- I token utente vengono inseriti in blacklist
- L'utente appare come "Sospeso" negli elenchi

### Riattivazione di un Utente

Riabilita un account sospeso:

1. Filtra gli utenti per stato "Sospeso"
2. Seleziona l'utente sospeso
3. Clicca **Riattiva** (usando il menu kebab)
4. Conferma l'azione

**Effetti della riattivazione:**
- L'utente può accedere nuovamente
- L'utente deve usare la password esistente
- I permessi precedenti vengono ripristinati

### Eliminazione di un Utente

**⚠️ Attenzione:** L'eliminazione dell'utente è permanente e non può essere annullata.

Per eliminare un utente:

1. Naviga alla pagina dei dettagli utente
2. Clicca **Elimina** (usando il menu kebab)
4. Clicca **Elimina**

**Effetti dell'eliminazione:**
- L'account utente viene rimosso permanentemente da Logto
- Tutti i log di audit vengono preservati
- I sistemi creati da questo utente rimangono

**Prerequisiti:**
- Non puoi eliminare il tuo account
- L'utente deve essere prima sospeso (misura di sicurezza)
- Devi avere il ruolo Admin

## Funzionalità Self-Service

Gli utenti possono gestire alcuni aspetti del proprio account:

### Cambia Propria Password

1. Clicca icona profilo > **Impostazioni Profilo**
2. Clicca **Cambia Password**
3. Inserisci password corrente
4. Inserisci nuova password (due volte)
5. Clicca **Salva**

### Aggiorna Proprio Profilo

1. Clicca icona profilo > **Impostazioni Profilo**
2. Aggiorna:
   - Nome
   - Indirizzo email
   - Numero di telefono
3. Clicca **Salva**

**Nota:** Le modifiche email potrebbero richiedere riautenticazione.

## Riferimento Permessi

### Permessi Ruolo Super Admin

✅ Può eseguire:
- Tutte le capacità Admin
- Impersonificazione utente per troubleshooting
- Operazioni di sistema avanzate
- Configurazione a livello piattaforma
- Accesso completo alla traccia di audit
- Operazioni di emergenza

❌ Non può eseguire:
- Modificare lo stato del proprio account
- Eliminare il proprio account
- Bypassare il logging di audit

### Permessi Ruolo Admin

✅ Può eseguire:
- Creare utenti
- Modificare utenti
- Reimpostare password utenti
- Sospendere/riattivare utenti
- Eliminare utenti (con restrizioni)
- Gestire organizzazioni (basato sulla gerarchia)
- Visualizzare tutti i log di audit
- Configurare impostazioni piattaforma

❌ Non può eseguire:
- Impersonificazione utente
- Modificare lo stato del proprio account
- Eliminare il proprio account
- Bypassare restrizioni gerarchiche

### Permessi Ruolo Backoffice

✅ Può eseguire:
- Visualizzare utenti e organizzazioni
- Visualizzare sistemi e inventario
- Generare report e analisi
- Visualizzare statistiche e dashboard
- Esportare dati
- Visualizzare log di audit

❌ Non può eseguire:
- Creare o modificare utenti
- Gestire organizzazioni
- Creare o modificare sistemi
- Eliminare qualsiasi risorsa
- Sospendere utenti
- Reimpostare password

### Permessi Ruolo Support

✅ Può eseguire:
- Creare sistemi
- Visualizzare sistemi
- Modificare sistemi
- Rigenerare segreti sistema
- Visualizzare inventario
- Visualizzare stato heartbeat
- Visualizzare statistiche sistema

❌ Non può eseguire:
- Gestire utenti
- Gestire organizzazioni
- Eliminare sistemi
- Accedere a operazioni pericolose

### Permessi Ruolo Reader

✅ Può eseguire:
- Visualizzare utenti (informazioni di base)
- Visualizzare organizzazioni
- Visualizzare sistemi e stato
- Visualizzare dati inventario
- Visualizzare stato heartbeat
- Visualizzare statistiche di base

❌ Non può eseguire:
- Creare, modificare o eliminare qualsiasi risorsa
- Accedere a dati utente sensibili
- Visualizzare log di audit
- Generare report
- Esportare dati

### Restrizioni Gerarchiche

Gli utenti possono gestire altri utenti solo nell'ambito della propria organizzazione:

**Utenti Owner:**
- Possono gestire tutti gli utenti in tutte le organizzazioni

**Utenti Distributore:**
- Possono gestire utenti nei propri rivenditori e clienti
- Non possono gestire utenti in altri distributori

**Utenti Rivenditore:**
- Possono gestire utenti solo nei propri clienti
- Non possono gestire utenti nel proprio distributore o altri rivenditori

**Utenti Cliente:**
- Possono gestire utenti solo nella propria organizzazione

## Statistiche Utenti

### Metriche Dashboard

Naviga su **Dashboard** per visualizzare:

- **Totale Utenti**: Conteggio in tutte le organizzazioni accessibili
- **Utenti Attivi**: Utenti che hanno effettuato l'accesso di recente
- **Utenti per Organizzazione**: Grafico di distribuzione
- **Utenti per Ruolo**: Conteggio Super Admin, Admin, Backoffice, Support, Reader
- **Trend di Crescita**: Trend creazione utenti (ultimi 30/60/90 giorni)

### Report Utenti

Genera report:

1. Naviga su **Utenti**
3. Scegli i filtri (organizzazione, ruolo, stato)
4. Clicca **Azioni** > **Esporta**
5. Esporta come CSV o PDF

## Best Practice

### Gestione Account Utente

- Crea utenti solo quando necessario
- Usa nomi completi descrittivi
- Verifica sempre gli indirizzi email
- Documenta le responsabilità degli utenti
- Rivedi regolarmente gli account utente
- Rimuovi prontamente gli utenti inattivi

### Assegnazione Ruoli

- Assegna i ruoli minimi richiesti (principio del privilegio minimo)
- Documenta perché gli utenti hanno ruoli specifici
- Rivedi le assegnazioni dei ruoli trimestralmente
- Usa il ruolo Super Admin solo per amministratori di piattaforma
- Usa il ruolo Admin con parsimonia per esigenze di gestione utenti
- Usa il ruolo Backoffice per personale di reportistica e analisi
- Usa il ruolo Support per la maggior parte delle operazioni tecniche
- Usa il ruolo Reader per accesso in sola visualizzazione (revisori, stakeholder)

### Sicurezza

- Forza il cambio password per incidenti di sicurezza
- Sospendi gli utenti immediatamente alla cessazione
- Rivedi regolarmente le sessioni attive
- Monitora i tentativi di accesso falliti
- Mantieni aggiornate le informazioni di contatto

### Assegnazione Organizzazione

- Assegna gli utenti all'organizzazione corretta
- Verifica la gerarchia organizzativa
- Aggiorna l'appartenenza all'organizzazione quando la struttura cambia
- Non creare utenti nelle organizzazioni sbagliate

## Risoluzione Problemi

### L'Utente Non Può Accedere

**Problema:** L'utente riferisce di non poter accedere alla piattaforma

**Soluzioni:**
1. Verifica che l'account utente non sia sospeso
2. Controlla se la password temporanea è stata cambiata
3. Conferma che l'indirizzo email sia corretto
4. Reimposta la password se necessario
5. Controlla lo stato del servizio Logto

### L'Utente Ha Permessi Errati

**Problema:** L'utente non può accedere alle funzionalità previste

**Soluzioni:**
1. Verifica che i ruoli utente siano assegnati correttamente
2. Controlla che l'appartenenza all'organizzazione sia corretta
3. Conferma che la gerarchia organizzativa sia corretta
4. Rivedi i permessi combinati (ruolo org + ruolo utente)
5. Controlla se le modifiche ai ruoli recenti si sono propagate

### Impossibile Creare Utente

**Problema:** "Accesso negato" durante la creazione dell'utente

**Soluzioni:**
1. Verifica di avere il ruolo Admin
2. Controlla che l'organizzazione di destinazione sia nella tua gerarchia
3. Conferma che l'indirizzo email non sia già utilizzato
4. Assicurati che l'organizzazione non sia sospesa

### Email di Benvenuto Non Ricevuta

**Problema:** Il nuovo utente non ha ricevuto l'email di benvenuto

**Soluzioni:**
1. Controlla la cartella spam dell'utente
2. Verifica che l'indirizzo email sia corretto
3. Controlla la configurazione SMTP (solo admin)
4. Condividi manualmente la password temporanea in modo sicuro
5. Reimposta la password per inviare una nuova email

## Prossimi Passi

Dopo aver creato gli utenti:

- [Crea sistemi](04-systems.md) per le organizzazioni clienti
- Configura i permessi utente in modo appropriato
- Forma gli utenti sull'utilizzo della piattaforma
- Imposta monitoraggio e avvisi

## Documentazione Correlata

- [Guida Autenticazione](01-authentication.md)
- [Gestione Organizzazioni](02-organizations.md)
- [Gestione Sistemi](04-systems.md)
