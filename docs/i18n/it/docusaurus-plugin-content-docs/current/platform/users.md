---
sidebar_position: 2
---

# Gestione Utenti

Creazione, gestione e assegnazione dei ruoli agli utenti della piattaforma.

## Comprendere i Ruoli

My usa un sistema a doppio ruolo che combina la gerarchia commerciale con le capacità tecniche.

### Ruoli Organizzazione

Ereditati automaticamente dall'organizzazione dell'utente:

- **Owner**: accesso completo alla piattaforma (solo Nethesis). Ogni membro dell'organizzazione Owner ha visibilità globale su tutte le aziende, i sistemi e gli utenti, inclusa l'archiviazione e l'eliminazione definitiva di distributori, rivenditori e clienti
- **Distributore**: gestisce rivenditori e clienti
- **Rivenditore**: gestisce i clienti
- **Cliente**: vede i dati della propria organizzazione

### Ruoli Utente

Assegnati manualmente in base alla funzione lavorativa:

- **Admin**: gestione completa dell'organizzazione
  - Utenti: creazione, modifica, reset password, sospensione, eliminazione
  - Sistemi: creazione, modifica, sospensione, eliminazione
  - Applicazioni, configurazione allarmi, add-on, rebranding

- **Backoffice**: operazioni amministrative, nessuna gestione dei sistemi
  - Utenti: creazione, modifica, reset password, sospensione, eliminazione
  - Applicazioni: visualizzazione e assegnazione alle organizzazioni
  - Add-on: attivazione e revoca
  - Sistemi: sola lettura -- non può crearli né modificarli
  - Niente configurazione allarmi, niente rebranding

- **Support**: operazioni tecniche sui sistemi
  - Sistemi: creazione, modifica, sospensione, eliminazione, rigenerazione secret
  - Inventario, heartbeat, allarmi e silenziamenti
  - Applicazioni: visualizzazione e assegnazione
  - **Nessun accesso agli utenti**: non ne vede nemmeno l'elenco

- **Reader**: accesso in sola lettura
  - Visualizza utenti, organizzazioni, sistemi, inventario, applicazioni, add-on
  - Può esportare ogni elenco che riesce a leggere
  - Nessuna capacità di modifica

- **Staff** (solo organizzazione Owner): personale trasversale Nethesis
  - Gestione di sistemi, utenti, applicazioni, allarmi (inclusa la configurazione dei template), add-on e rebranding su tutte le aziende
  - Impersonificazione degli utenti (con il loro consenso) e connessione remota ai sistemi
  - Non può eliminare definitivamente sistemi o utenti

:::note Regole di assegnazione
Nell'organizzazione Owner l'unico ruolo assegnabile è **Staff**, che non può mai essere assegnato agli utenti delle altre aziende. L'account speciale `owner`, creato all'installazione, non compare nell'elenco dei ruoli e non è assegnabile: è l'unico con controllo completo — inclusa l'eliminazione definitiva di sistemi e utenti — ed è l'unico che può creare e gestire gli utenti dell'organizzazione Owner.
:::

### Permessi Combinati

I permessi effettivi di un utente sono la combinazione di **entrambi** i tipi di ruolo:

**Esempio 1:**
```
Organizzazione: Cliente (Pizza Express)
Ruolo utente: Admin
→ Può gestire gli utenti solo dell'organizzazione Pizza Express
→ Può gestire i sistemi solo di Pizza Express
```

**Esempio 2:**
```
Organizzazione: Distributore (ACME Distribution)
Ruolo utente: Support
→ Può vedere rivenditori e clienti sotto ACME
→ Può gestire i sistemi di tutti i clienti sotto ACME
→ Non vede affatto gli utenti (Support non ha `read:users`)
```

## Creazione Utenti

### Prerequisiti

- Serve `manage:users`, che hanno **Admin**, **Backoffice** e **Staff**. Support non può: non ha alcun accesso agli utenti
- Puoi creare utenti solo per le organizzazioni che gestisci
- Serve un indirizzo email valido per il nuovo utente

### Creare un Nuovo Utente

1. Vai su **Utenti**
2. Clicca su **Crea utente**
3. Compila il modulo:
   - **Nome**: nome visualizzato dell'utente (es. "Mario Rossi")
   - **Email**: indirizzo email dell'utente (sarà il suo username)
   - **Organizzazione**: seleziona l'organizzazione
   - **Ruoli**: seleziona uno o più ruoli (Admin, Backoffice, Support, Reader; Staff solo per l'organizzazione Owner)
   - **Numero di telefono** (facoltativo): recapito telefonico
4. Clicca su **Crea utente**

**Esempio:**
```
Nome completo: Mario Rossi
Email: mario.rossi@techsolutions.it
Organizzazione: Tech Solutions Italia (Rivenditore)
Ruoli utente: Admin, Support
Telefono: +39 02 1234567
```

### Cosa Succede Dopo la Creazione

1. L'account viene creato su Logto
2. Viene generata automaticamente una password temporanea
3. All'utente viene inviata un'email di benvenuto contenente:
   - La password temporanea
   - L'URL di accesso
   - Le istruzioni per il cambio password
4. L'utente deve cambiare la password al primo accesso

:::warning
La password temporanea viene mostrata **una sola volta**, alla creazione. Assicurati che l'utente riceva l'email di benvenuto.
:::

## Gestione Utenti

### Visualizzazione Elenco

Vai su **Utenti** per vedere:

- Nome ed email dell'utente
- Organizzazione
- Ruoli utente
- Ruolo organizzazione (derivato dall'organizzazione)
- Stato (attivo/sospeso)

### Filtri e Ricerca

Usa i filtri per trovare utenti specifici:

- **Ricerca per nome o email**: digita nella casella di ricerca
- **Ricerca per organizzazione**: seleziona una o più organizzazioni
- **Ricerca per ruolo**: Admin, Backoffice, Support, Reader, Staff
- **Ordinamento**: nome, email, organizzazione

### Dettagli Utente

Cliccando su un utente si accede alle informazioni di dettaglio:

- **Informazioni di profilo**:
  - Nome completo
  - Indirizzo email
  - Numero di telefono
  - Immagine del profilo (se configurata su Logto)

- **Appartenenza organizzativa**:
  - Organizzazione principale
  - Ruolo organizzazione (Owner/Distributore/Rivenditore/Cliente)

- **Ruoli e permessi**:
  - Ruoli utente assegnati
  - Elenco dei permessi effettivi

- **Attività**:
  - Data e ora dell'ultimo accesso
  - Data di creazione dell'account
  - Ultimo cambio password

- **Stato**:
  - Attivo o sospeso
  - Motivo della sospensione (se applicabile)

## Modifica Utenti

### Aggiornare le Informazioni

1. Vai al dettaglio dell'utente
2. Clicca su **Modifica**
3. Aggiorna i campi:
   - Nome
   - Indirizzo email
   - Organizzazione
   - Ruoli
   - Numero di telefono
4. Clicca su **Salva utente**

:::note
- Va selezionato almeno un ruolo
- Non puoi modificare il tuo account da questa interfaccia: usa la pagina Account
:::

### Reset Password Utente

Con `manage:users` puoi resettare la password di un utente:

1. Vai alla pagina dell'utente
2. Clicca su **Reset password** (dal menu contestuale)
3. Conferma l'operazione
4. Viene generata una nuova password temporanea
5. Copia la password e trasmettila all'utente

**Casi d'uso:**
- L'utente ha dimenticato la password
- Incidente di sicurezza che richiede un reset
- Recupero dell'account

## Gestione dello Stato

### Sospensione

Disabilita temporaneamente un account:

1. Vai alla pagina dell'utente
2. Clicca su **Sospendi** (dal menu contestuale)
3. Conferma

**Effetti della sospensione:**
- L'utente non può accedere
- Le sessioni attive vengono invalidate immediatamente
- I token dell'utente finiscono in blacklist
- L'utente compare come "Sospeso" negli elenchi

### Riattivazione

Riabilita un account sospeso:

1. Filtra gli utenti per stato "Sospeso"
2. Seleziona l'utente sospeso
3. Clicca su **Riattiva** (dal menu contestuale)
4. Conferma l'operazione

**Effetti della riattivazione:**
- L'utente può accedere di nuovo
- Deve usare la password che aveva già
- I permessi precedenti vengono ripristinati

### Eliminazione

**L'eliminazione archivia, non cancella.** L'utente viene eliminato in modo
soft: sparisce dagli elenchi e non può più accedere, ma il record resta e può
essere recuperato con **Ripristina**.

Per eliminare un utente:

1. Vai al dettaglio dell'utente
2. Clicca su **Elimina** (dal menu contestuale)
3. Conferma

**Effetti dell'eliminazione:**
- L'utente non può più accedere
- L'account viene archiviato, non cancellato -- **Ripristina** lo riporta indietro
- I log di audit vengono conservati
- I sistemi creati da quell'utente restano

**Prerequisiti:**
- Non è possibile eliminare il proprio account
- Serve `manage:users` (Admin, Backoffice o Staff)

:::danger Eliminazione definitiva
Cancellare un utente per sempre è un'operazione separata e richiede
`destroy:users`, che non appartiene a nessun ruolo assegnabile: solo l'account
`owner` creato all'installazione. Quella non è reversibile.
:::

## Self-Service

Gli utenti possono gestire alcuni aspetti del proprio account:

### Cambio della Propria Password

1. Icona del profilo > **Account**
2. Clicca su **Cambio password**
3. Inserisci la password attuale
4. Inserisci la nuova password (due volte)
5. Clicca su **Salva**

### Aggiornamento del Proprio Profilo

1. Icona del profilo > **Account**
2. Aggiorna:
   - Nome
   - Indirizzo email
   - Numero di telefono
3. Clicca su **Salva**

:::note
La modifica dell'email può richiedere una nuova autenticazione.
:::

## Riferimento Permessi

I permessi effettivi sono l'**unione** del ruolo organizzazione e del ruolo
utente. Tutto ciò che riguarda la gerarchia commerciale (distributori,
rivenditori, clienti) arriva dal ruolo organizzazione, quindi non cambia col
ruolo utente -- con una sola eccezione: al Reader vengono tolti i permessi
`manage:` sulla gerarchia.

| Operazione | Permesso | Staff | Admin | Backoffice | Support | Reader |
|------------|----------|:-----:|:-----:|:----------:|:-------:|:------:|
| Visualizza utenti | `read:users` | Sì | Sì | Sì | **No** | Sì |
| Crea e modifica utenti | `manage:users` | Sì | Sì | Sì | No | No |
| Reset password di un utente | `manage:users` | Sì | Sì | Sì | No | No |
| Sospendi / riattiva un utente | `manage:users` | Sì | Sì | Sì | No | No |
| Elimina un utente (archiviazione) | `manage:users` | Sì | Sì | Sì | No | No |
| Visualizza sistemi | `read:systems` | Sì | Sì | Sì | Sì | Sì |
| Crea, modifica ed elimina sistemi | `manage:systems` | Sì | Sì | No | Sì | No |
| Silenzia allarmi | `manage:systems` | Sì | Sì | No | Sì | No |
| Visualizza organizzazioni | dal ruolo organizzazione | Sì | Sì | Sì | Sì | Sì |
| Gestisci organizzazioni | dal ruolo organizzazione | Sì | Sì | Sì | Sì | No |
| Visualizza applicazioni | `read:applications` | Sì | Sì | Sì | Sì | Sì |
| Gestisci e assegna applicazioni | `manage:applications` | Sì | Sì | Sì | Sì | No |
| Visualizza configurazione allarmi | `read:alerts` | Sì | Sì | No | Sì | No |
| Modifica configurazione allarmi | `manage:alerts` | Sì | Sì | No | Sì | No |
| Leggi la configurazione allarmi effettiva | `config:alerts` | Sì | No | No | No | No |
| Visualizza add-on | `read:entitlements` | Sì | Sì | Sì | Sì | Sì |
| Attiva / revoca add-on | `manage:entitlements` | Sì | Sì | Sì | No | No |
| Catalogo add-on e grant manuali | `manage:entitlements` + org Owner | Sì | No | No | No | No |
| Visualizza rebranding | `read:rebranding` | Sì | Sì | Sì | Sì | Sì |
| Configura rebranding | `manage:rebranding` | Sì | Sì | No | No | No |
| Impersonifica utenti | `impersonate:users` | Sì | No | No | No | No |
| Connessione remota ai sistemi | `connect:systems` | Sì | No | No | No | No |

Per esportare un elenco basta il permesso `read:` di quella risorsa, quindi ogni
ruolo può esportare ciò che vede -- Support compreso, tranne gli utenti, che
non può leggere.

:::note Eliminazione definitiva
`destroy:systems` e `destroy:users` non appartengono a nessun ruolo assegnabile,
Staff incluso. Sono solo dell'account `owner` creato all'installazione, che è
anche l'unico a poter creare e gestire gli utenti dell'organizzazione Owner.
:::

### Restrizioni Gerarchiche

Gli utenti possono gestire altri utenti solo all'interno del proprio perimetro organizzativo:

**Utenti dell'organizzazione Owner:**
- Possono gestire tutti gli utenti di tutte le organizzazioni
- Gli utenti dell'organizzazione Owner stessa sono creati e gestiti solo dall'account `owner`

**Utenti di un distributore:**
- Possono gestire gli utenti dei propri rivenditori e clienti
- Non possono gestire utenti di altri distributori

**Utenti di un rivenditore:**
- Possono gestire solo gli utenti dei propri clienti
- Non possono gestire utenti del proprio distributore o di altri rivenditori

**Utenti di un cliente:**
- Possono gestire solo gli utenti della propria organizzazione

:::warning
Non è mai possibile:
- Sospendere o eliminare il proprio account
- Resettare la propria password dalla gestione utenti (usare la pagina Account)
- Creare utenti con un ruolo superiore al proprio
:::

## Statistiche

### Metriche in Dashboard

La [Dashboard](../features/dashboard.md) porta una card **Utenti** con il totale sulle organizzazioni che puoi leggere, collegata all'elenco. Viene mostrata solo se hai `read:users`.

Le ripartizioni per organizzazione, ruolo o stato si ottengono dai filtri dell'elenco, non dalla Dashboard. La crescita nel tempo è disponibile da API tramite `/backend/api/users/trend`.

### Report Utenti

Per generare un report:

1. Vai su **Utenti**
2. Scegli i filtri (organizzazione, ruolo, stato)
3. Clicca su **Azioni** > **Esporta**
4. Esporta in CSV o PDF

## Best Practice

### Gestione degli Account

- Crea utenti solo quando servono
- Usa nomi completi descrittivi
- Verifica sempre gli indirizzi email
- Documenta le responsabilità degli utenti
- Rivedi periodicamente gli account
- Rimuovi tempestivamente gli utenti inattivi

### Assegnazione dei Ruoli

- Assegna i ruoli minimi necessari (principio del privilegio minimo)
- Documenta perché un utente ha un determinato ruolo
- Rivedi le assegnazioni ogni trimestre
- Usa Admin per chi deve gestire sia utenti sia sistemi
- Usa Backoffice per chi gestisce utenti, applicazioni e add-on ma non deve toccare i sistemi
- Usa Support per il personale tecnico che lavora sui sistemi e non ha bisogno degli utenti
- Usa Reader per l'accesso in sola lettura (auditor, stakeholder)

### Sicurezza

- Forza il cambio password in caso di incidente di sicurezza
- Sospendi subito gli utenti che lasciano l'azienda
- Rivedi periodicamente le sessioni attive
- Monitora i tentativi di accesso falliti
- Tieni aggiornate le informazioni di contatto

### Assegnazione Organizzativa

- Assegna gli utenti all'organizzazione corretta
- Verifica la gerarchia organizzativa
- Aggiorna l'appartenenza quando la struttura cambia
- Non creare utenti nell'organizzazione sbagliata

## Risoluzione Problemi

### L'Utente Non Riesce ad Accedere

**Problema:** l'utente segnala di non riuscire ad accedere alla piattaforma

**Soluzioni:**
1. Verifica che l'account non sia sospeso
2. Controlla se la password temporanea è stata cambiata
3. Conferma che l'indirizzo email sia corretto
4. Resetta la password se necessario
5. Controlla lo stato del servizio Logto

### L'Utente Ha Permessi Errati

**Problema:** l'utente non riesce ad accedere alle funzionalità attese

**Soluzioni:**
1. Verifica che i ruoli utente siano assegnati correttamente
2. Controlla che l'appartenenza organizzativa sia corretta
3. Conferma che la gerarchia organizzativa sia corretta
4. Rivedi i permessi combinati (ruolo organizzazione + ruolo utente)
5. Controlla se le modifiche recenti ai ruoli si sono propagate

### Impossibile Creare un Utente

**Problema:** errore di accesso negato durante la creazione

**Soluzioni:**
1. Verifica di avere `manage:users` (Admin, Backoffice o Staff)
2. Controlla che l'organizzazione di destinazione sia nella tua gerarchia
3. Conferma che l'indirizzo email non sia già in uso
4. Assicurati che l'organizzazione non sia sospesa

### Email di Benvenuto Non Ricevuta

**Problema:** il nuovo utente non ha ricevuto l'email di benvenuto

**Soluzioni:**
1. Controlla la cartella spam dell'utente
2. Verifica che l'indirizzo email sia corretto
3. Controlla la configurazione SMTP (solo amministratori)
4. Condividi la password temporanea per un canale sicuro
5. Resetta la password per far partire una nuova email

## Prossimi Passi

Dopo aver creato gli utenti:

- [Crea i sistemi](../systems/management.md) per le organizzazioni cliente
- Configura i permessi in modo appropriato
- Forma gli utenti all'uso della piattaforma
- Imposta monitoraggio e allarmi

## Documentazione Correlata

- [Guida all'Autenticazione](../getting-started/authentication.md)
- [Gestione Organizzazioni](./organizations.md)
- [Gestione Sistemi](../systems/management.md)
