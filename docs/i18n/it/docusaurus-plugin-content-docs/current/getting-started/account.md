---
sidebar_position: 2
---

# Impostazioni Account

Gestione di lingua, profilo, avatar, password, API key e consenso all'impersonificazione.

## Panoramica

La pagina Account è accessibile dal menu utente in alto a destra e si compone delle seguenti sezioni:

- **Impostazioni generali** -- lingua dell'interfaccia
- **Profilo** -- informazioni personali
- **Avatar** -- immagine del profilo
- **Cambio password** -- aggiornamento credenziali
- **API key** -- credenziali per l'accesso programmatico
- **Consenso impersonificazione** -- se gli amministratori possono agire al posto tuo

## Impostazioni Generali

### Selezione Lingua

L'interfaccia è disponibile in:

- **Inglese** (predefinito)
- **Italiano**

Al primo accesso la lingua segue quella del browser; in seguito segue la preferenza salvata. Per cambiarla:

1. Vai alla pagina **Account**
2. Individua l'impostazione **Lingua**
3. Scegli la lingua

La modifica viene applicata immediatamente a tutta l'interfaccia e viene ricordata per gli accessi successivi.

## Gestione Profilo

### Modifica Informazioni

1. Vai alla pagina **Account**
2. Nella sezione **Profilo** aggiorna:
   - **Nome**: il nome visualizzato su tutta la piattaforma
   - **Email**: il tuo indirizzo email, che è anche il tuo username
   - **Numero di telefono**: facoltativo
3. Clicca su **Salva profilo**

:::note
Un nuovo indirizzo email non viene applicato subito: riceverai un codice di verifica a quell'indirizzo e la modifica diventa effettiva solo quando lo inserisci. Fino ad allora resta attivo l'indirizzo attuale. Il codice scade dopo 10 minuti e puoi richiederne uno nuovo dalla stessa finestra.
:::

## Gestione Avatar

L'avatar compare su tutta la piattaforma accanto al tuo nome, nei commenti e negli elenchi utenti.

### Upload Avatar

1. Vai alla pagina **Account**
2. Clicca sull'area dell'avatar o sul pulsante **Carica**
3. Scegli un'immagine: PNG, JPEG o WebP, al massimo **500 KB** e **4096x4096** pixel
4. L'immagine viene scalata per rientrare in 256x256, convertita in PNG e applicata immediatamente

### Avatar Predefinito

Senza un avatar caricato, l'interfaccia mostra le tue iniziali su un cerchio colorato, ricavate dal nome visualizzato.

### Eliminazione Avatar

1. Vai alla pagina **Account**
2. Clicca su **Elimina** sull'avatar attuale
3. Viene ripristinato il segnaposto con le iniziali

### URL Pubblico

Il tuo avatar è raggiungibile senza autenticazione all'indirizzo `/backend/api/public/users/{user_id}/avatar`.

Per i dettagli completi, vedi [Gestione Avatar](../features/avatar.md).

## Cambio Password

1. Vai alla pagina **Account**
2. Nella sezione **Cambio password** inserisci:
   - **Password attuale**
   - **Nuova password**
   - **Conferma password**
3. Clicca su **Salva**

:::warning
La nuova password deve rispettare la policy della piattaforma: almeno **12 caratteri**, con una maiuscola, una minuscola, un numero e un carattere speciale. Vedi [Requisiti Password](./authentication.md#requisiti-password).
:::

## API Key

La pagina Account include una sezione **API key** dove puoi creare e revocare key personali per l'accesso programmatico da applicazioni e script esterni. Per i dettagli completi consulta [API Key](./api-keys.md).

## Consenso Impersonificazione

La sezione **Impersonificazione** controlla se gli amministratori dell'organizzazione Owner possono usare temporaneamente la piattaforma al posto tuo. Per capire come funziona, vedi [Impersonificazione](../platform/impersonation.md).

:::note
Il consenso è facoltativo e sempre revocabile da te. Con il consenso revocato, nessuno può aprire una sessione a tuo nome.
:::

## Risoluzione Problemi

### Le Modifiche al Profilo Non Vengono Salvate

**Problema:** dopo aver cliccato Salva le modifiche non risultano applicate

**Soluzioni:**
- Verifica che tutti i campi obbligatori siano compilati
- Controlla che l'indirizzo email sia valido
- Ricarica la pagina e ripeti la modifica
- Se stai cambiando l'email, inserisci il codice di verifica ricevuto al nuovo indirizzo; se è scaduto, richiedine uno nuovo

### L'Upload dell'Avatar Fallisce

**Problema:** l'avatar non viene caricato o mostra un errore

**Soluzioni:**
- Verifica che il file sia in un formato supportato (PNG, JPEG o WebP)
- Controlla che il file sia sotto i 500 KB
- Assicurati che le dimensioni non superino 4096x4096 pixel
- Prova con un'altra immagine

### Il Cambio Password Fallisce

**Problema:** non riesci a cambiare la password

**Soluzioni:**
- Verifica che la password attuale sia corretta
- Assicurati che la nuova password rispetti tutti i requisiti: almeno 12 caratteri, maiuscola, minuscola, numero, carattere speciale, non più di 3 caratteri identici di fila e nessun pattern debole comune
- Controlla che nuova password e conferma coincidano
- Se hai dimenticato la password attuale, usa invece il link "Password dimenticata?" nella pagina di login
