---
sidebar_position: 2
---

# Gestione Utenti

La gestione utenti consente di creare, modificare e amministrare gli account utente all'interno della piattaforma My.

## Ruoli

### Ruoli Organizzazione

I ruoli organizzazione determinano la posizione nella gerarchia aziendale:

| Ruolo | Descrizione | Visibilità |
|-------|-------------|------------|
| **Owner** | Proprietario della piattaforma | Tutte le organizzazioni |
| **Distributore** | Partner di distribuzione | Rivenditori e clienti propri |
| **Rivenditore** | Partner di rivendita | Clienti propri |
| **Cliente** | Utente finale | Solo la propria organizzazione |

### Ruoli Utente

I ruoli utente determinano le capacità tecniche:

| Ruolo | Descrizione | Capacità Principali |
|-------|-------------|---------------------|
| **Super Admin** | Amministrazione completa | Tutte le operazioni, incluse quelle critiche. Solo nell'organizzazione Owner |
| **Admin** | Gestione avanzata | Gestione sistemi, utenti, operazioni pericolose |
| **Backoffice** | Operazioni di backoffice | Gestione organizzazioni, applicazioni, rebranding |
| **Support** | Supporto tecnico | Accesso lettura sistemi, operazioni di supporto |
| **Reader** | Sola lettura | Visualizzazione dati senza possibilità di modifica |

### Permessi Combinati

I permessi effettivi di un utente sono la combinazione del ruolo organizzazione e del ruolo utente:

```
Permessi Effettivi = Permessi Ruolo Organizzazione + Permessi Ruolo Utente
```

**Esempio 1**: Distributore + Admin
- Può gestire rivenditori e clienti sotto la propria organizzazione
- Può creare e modificare sistemi e utenti
- Può eseguire operazioni pericolose (reset password, sospensione)

**Esempio 2**: Cliente + Reader
- Può visualizzare solo i dati della propria organizzazione
- Nessuna possibilità di modifica

**Esempio 3**: Owner + Super Admin
- Accesso completo a tutta la piattaforma
- Tutte le operazioni disponibili, inclusa l'impersonificazione

## Creazione Utenti

### Procedura

1. Vai a **Utenti**
2. Clicca su **Nuovo Utente**
3. Compila i campi richiesti:
   - **Email** - Indirizzo email (usato come username)
   - **Nome** e **Cognome**
   - **Organizzazione** - L'organizzazione di appartenenza
   - **Ruolo Utente** - Il ruolo tecnico da assegnare
   - **Numero di Telefono** (opzionale) - Telefono di contatto
4. Clicca su **Crea**

**Esempio:**
```
Nome Completo: Mario Rossi
Email: mario.rossi@techsolutions.it
Organizzazione: Tech Solutions Italia (Rivenditore)
Ruoli Utente: Admin, Support
Telefono: +39 02 1234567
```

### Cosa Succede Dopo la Creazione

Quando un utente viene creato:

1. L'account viene creato nel provider di identità (Logto)
2. Viene generata una **password temporanea**
3. L'utente riceve un'**email di benvenuto** con le credenziali
4. Al primo accesso, l'utente deve cambiare la password

:::note
L'email di benvenuto contiene la password temporanea e il link diretto alla piattaforma. Assicurati che l'indirizzo email sia corretto.
:::

## Gestione Utenti

### Visualizzazione Elenco

La pagina elenco utenti mostra:

- **Nome** e **Cognome**
- **Email**
- **Organizzazione** di appartenenza
- **Ruolo Organizzazione**
- **Ruolo Utente**
- **Stato** (attivo, sospeso)
- **Ultimo accesso**

### Filtri e Ricerca

È possibile filtrare gli utenti per:

- **Ricerca testuale** - Cerca per nome, cognome o email
- **Organizzazione** - Filtra per organizzazione di appartenenza
- **Ruolo Utente** - Filtra per ruolo tecnico
- **Stato** - Attivo o sospeso

### Dettagli Utente

Cliccando su un utente si accede alla pagina di dettaglio con:

- **Informazioni personali** - Nome, cognome, email, telefono
- **Organizzazione** - Dettagli dell'organizzazione di appartenenza
- **Ruoli** - Ruolo organizzazione e ruolo utente
- **Stato** - Attivo o sospeso
- **Avatar** - Immagine profilo o iniziali
- **Date** - Creazione, ultimo accesso

### Modifica Utente

Per modificare un utente:

1. Vai all'elenco utenti
2. Clicca sull'utente da modificare
3. Clicca su **Modifica**
4. Aggiorna i campi desiderati:
   - Nome, cognome, email, telefono
   - Organizzazione
   - Ruolo utente
5. Clicca su **Salva**

## Reset Password

Gli amministratori possono forzare il reset della password di un utente:

1. Vai al dettaglio dell'utente
2. Clicca su **Reset Password**
3. Conferma l'operazione
4. La nuova password temporanea viene inviata all'email dell'utente

:::warning
Il reset della password invalida immediatamente la sessione corrente dell'utente. L'utente dovrà accedere di nuovo con la nuova password temporanea.
:::

:::note
Non è possibile resettare la propria password tramite questa funzione. Per cambiare la propria password, usa la sezione **Account**.
:::

## Sospensione e Riattivazione

### Sospensione

Per sospendere un utente:

1. Vai al dettaglio dell'utente
2. Clicca su **Sospendi**
3. Conferma l'operazione

Un utente sospeso:
- Non può accedere alla piattaforma
- Le sessioni attive vengono invalidate
- L'account resta nel sistema e può essere riattivato

### Riattivazione

Per riattivare un utente sospeso:

1. Vai al dettaglio dell'utente sospeso
2. Clicca su **Riattiva**
3. Conferma l'operazione

:::note
Non è possibile sospendere o riattivare il proprio account.
:::

## Eliminazione

Per eliminare un utente:

1. Vai al dettaglio dell'utente
2. Clicca su **Elimina**
3. Conferma l'operazione

:::warning
L'utente deve essere prima sospeso (misura di sicurezza). Non è possibile eliminare un utente attivo.
:::

:::danger
L'eliminazione di un utente è permanente. L'account viene rimosso dal provider di identità e non può essere recuperato.
:::

## Self-Service

Gli utenti possono gestire autonomamente:

- **Password** - Cambio dalla pagina Account
- **Informazioni profilo** - Modifica nome, cognome, email, telefono
- **Avatar** - Upload e gestione dell'immagine profilo
- **Consenso impersonificazione** - Attivazione/disattivazione

Per maggiori dettagli, consulta la pagina [Impostazioni Account](../getting-started/account).

## Riferimento Permessi

### Permessi per Ruolo Utente

| Operazione | Super Admin | Admin | Backoffice | Support | Reader |
|------------|:-----------:|:-----:|:----------:|:-------:|:------:|
| Visualizza utenti | Si | Si | Si | Si | Si |
| Crea utenti | Si | Si | No | No | No |
| Modifica utenti | Si | Si | No | No | No |
| Elimina utenti | Si | Si | No | No | No |
| Reset password | Si | Si | No | No | No |
| Sospendi/Riattiva | Si | Si | No | No | No |
| Gestione sistemi | Si | Si | No | Si | No |
| Gestione organizzazioni | Si | Si | Si | No | No |
| Gestione applicazioni | Si | Si | Si | No | No |
| Gestione rebranding | Si | Si | Si | No | No |
| Impersonificazione | Si | No | No | No | No |
| Esportazione dati | Si | Si | Si | Si | Si |

### Restrizioni Gerarchiche

Le operazioni sui dati sono limitate dalla posizione nella gerarchia:

- **Owner**: Può gestire tutti gli utenti di tutte le organizzazioni
- **Distributore**: Può gestire gli utenti delle proprie organizzazioni subordinate (rivenditori e clienti)
- **Rivenditore**: Può gestire gli utenti delle proprie organizzazioni subordinate (clienti)
- **Cliente**: Può visualizzare solo gli utenti della propria organizzazione

:::warning
Non è possibile:
- Sospendere o eliminare il proprio account
- Resettare la propria password dalla gestione utenti (usare la pagina Account)
- Creare utenti con un ruolo superiore al proprio
:::

## Statistiche e Report

### Totali

La pagina utenti mostra i totali:

- **Totale utenti** nella piattaforma (filtrato per visibilità gerarchica)
- **Utenti attivi** e **sospesi**
- **Distribuzione per ruolo**

### Esportazione

È possibile esportare l'elenco utenti in formato CSV o PDF. L'esportazione include tutti gli utenti visibili in base ai filtri applicati.

Per maggiori dettagli, consulta la pagina [Esportazione Dati](../features/export).

## Best Practice

- **Assegna il ruolo minimo necessario** - Seguire il principio del privilegio minimo
- **Usa email aziendali** - Evita indirizzi email personali per gli account della piattaforma
- **Controlla regolarmente** gli account inattivi e sospendi quelli non più necessari
- **Documenta i ruoli** assegnati e le motivazioni per le eccezioni
- **Sospendi** invece di eliminare se la rimozione potrebbe essere temporanea

## Risoluzione Problemi

### Email di Benvenuto Non Ricevuta

Se il nuovo utente non ha ricevuto l'email di benvenuto:

1. Controlla la cartella spam dell'utente
2. Verifica che l'indirizzo email sia corretto
3. Controlla la configurazione SMTP (solo admin)
4. Condividi manualmente la password temporanea in modo sicuro
5. Reimposta la password per inviare una nuova email

### L'Utente Ha Permessi Errati

Se l'utente non può accedere alle funzionalità previste:

1. Verifica che i ruoli utente siano assegnati correttamente
2. Controlla che l'appartenenza all'organizzazione sia corretta
3. Conferma che la gerarchia organizzativa sia corretta
4. Rivedi i permessi combinati (ruolo org + ruolo utente)
5. Controlla se le modifiche ai ruoli recenti si sono propagate

### Impossibile Creare un Utente

- Verifica di avere i permessi necessari (ruolo Admin o superiore)
- Controlla che l'email non sia già in uso
- Assicurati di aver selezionato un'organizzazione valida

### Utente Non Riesce ad Accedere

- Verifica che l'account non sia sospeso
- Controlla che l'email sia corretta
- Verifica che la password temporanea non sia scaduta
- Prova a eseguire un reset della password

### Utente Non Visibile nell'Elenco

- Verifica il tuo ruolo nella gerarchia: puoi vedere solo gli utenti delle organizzazioni al tuo livello o inferiore
- Controlla i filtri attivi
- Verifica che l'utente appartenga a un'organizzazione nella tua gerarchia
