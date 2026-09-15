---
sidebar_position: 1
---

# Gestione Sistemi

Creazione, monitoraggio e gestione dei sistemi collegati alla piattaforma My.

## Comprendere i Sistemi

Un sistema rappresenta un'installazione NethServer o NethSecurity registrata su My. Ogni sistema appartiene a un'organizzazione cliente e comunica con la piattaforma tramite credenziali proprie.

### Ciclo di Vita del Sistema

```
1. Creato da Admin/Support → riceve il system_secret
2. Non ancora registrato → il system_key è nascosto
3. Il sistema esterno si registra → il system_key diventa visibile
4. Il sistema invia inventario e heartbeat → stato monitorato
```

### Stati del Sistema

| Stato | Significato |
|-------|-------------|
| **Unknown** | Creato, ma non ha mai inviato un heartbeat |
| **Active** | Ultimo heartbeat più recente di 20 minuti |
| **Inactive** | Ultimo heartbeat più vecchio di 20 minuti |
| **Suspended** | Sospeso da un amministratore; non può inviare dati |
| **Unregistered** | L'appliance ha rinunciato alle proprie credenziali -- stato terminale, vedi [Registrazione](./registration.md#annullare-la-registrazione-di-un-sistema) |
| **Deleted** | Eliminato in modo soft; ripristinabile |

La finestra di 20 minuti arriva da `HEARTBEAT_TIMEOUT_MINUTES`, e un cron rivaluta tutti i sistemi ogni 5 minuti: il passaggio a `inactive` si vede quindi tra i 20 e i 25 minuti dopo l'ultimo heartbeat. Vedi [Inventario e Heartbeat](./inventory-heartbeat.md#classificazione-degli-stati).

## Creazione Sistemi

### Prerequisiti

- Serve il ruolo **Support** o **Admin**
- Serve un'organizzazione cliente a cui associare il sistema
- Il sistema viene creato nello stato "non registrato"

### Creare un Nuovo Sistema

1. Vai su **Sistemi**
2. Clicca su **Crea sistema**
3. Compila il modulo:
   - **Nome**: nome descrittivo del sistema (es. "Server di produzione Milano")
   - **Organizzazione**: seleziona l'organizzazione cliente
   - **Note** (facoltative): informazioni aggiuntive
4. Clicca su **Crea sistema**

**Esempio:**
```
Nome: Server Web Produzione Milano
Organizzazione: Pizza Express Milano (Cliente)
Note: Server di produzione principale per le sedi di Milano
```

### Secret del Sistema

Dopo la creazione vedrai:

```json
{
  "id": "sys_abc123",
  "name": "Server Web Produzione Milano",
  "system_key": "",
  "system_secret": "my_a1b2c3.k1l2m3...",
  "status": "unknown",
  "registered_at": null,
  "organization": "Pizza Express Milano"
}
```

:::danger
Il `system_secret` viene mostrato **una sola volta**, alla creazione. Copialo e salvalo subito: ti serve per registrare il sistema. Se lo perdi *prima* della registrazione puoi rigenerarlo, ma una volta che il sistema si è registrato la rigenerazione viene rifiutata e l'unica via è creare un nuovo sistema.
:::

## Visualizzazione Sistemi

### Elenco

La pagina **Sistemi** mostra tutti i sistemi visibili con:

- Nome del sistema
- Tipo (NethServer o NethSecurity)
- Versione
- Organizzazione
- Stato
- Ultimo heartbeat

### Filtri e Ricerca

Usa i filtri per trovare sistemi specifici:

- **Ricerca**: per nome o system_key
- **Prodotto**: filtra per tipo (NethServer o NethSecurity)
- **Versione**: filtra per versione del sistema
- **Organizzazione**: filtra per organizzazione cliente
- **Creato da**: filtra per utente che ha creato il sistema
- **Add-on**: filtra per add-on acquistato (vedi [Add-on](../features/entitlements.md))
- **Stato**: unknown, active, inactive, suspended, deleted
- **Ordinamento**: nome, versione, FQDN/indirizzo IP, organizzazione, creato da, stato

Il menu **Add-on** elenca gli add-on posseduti da almeno uno dei tuoi sistemi,
quindi un'opzione non restituisce mai un elenco vuoto. Selezionandone più di uno
la ricerca si allarga: un sistema corrisponde se ne possiede almeno uno. Contano
solo gli add-on validi in quel momento, quindi uno scaduto o annullato esclude
il sistema.

### Dettagli Sistema

Cliccando su un sistema si accede alle informazioni complete:

#### Tab Panoramica

- **Informazioni di base**:
  - Nome del sistema
  - Tipo (rilevato automaticamente)
  - Stato
  - Versione
  - Data di registrazione

- **Informazioni di rete**:
  - FQDN (nome di dominio completo)
  - Indirizzo IPv4
  - Indirizzo IPv6

- **Autenticazione**:
  - System key (visibile solo dopo la registrazione)
  - Stato della registrazione
  - Ultima autenticazione

- **Organizzazione**:
  - Nome del cliente
  - Tipo di organizzazione
  - Nome dell'organizzazione

- **Stato heartbeat**:
  - Stato corrente (active/inactive/unknown)
  - Data dell'ultimo heartbeat
  - Data dell'ultimo inventario

- **Tracciabilità**:
  - Creato da (nome ed email dell'utente)
  - Data di creazione
  - Data di eliminazione (se eliminato in modo soft)

#### Tab Inventario

Mostra l'inventario dettagliato del sistema:

- **Ultimo inventario**: lo snapshot più recente
- **Storico inventari**: tutti gli inventari passati, con paginazione
- **Modifiche**: elenco delle variazioni rilevate tra un inventario e l'altro
- **Vista diff**: confronto dettagliato tra due versioni dell'inventario

Vedi [Inventario e Heartbeat](./inventory-heartbeat.md) per i dettagli.

## Gestione Sistemi

### Modifica

1. Vai al dettaglio del sistema
2. Clicca su **Modifica**
3. Aggiorna i campi modificabili:
   - Nome
   - Organizzazione
   - Note
4. Clicca su **Salva**

:::note
Il tipo, la versione e i dati di rete non si modificano a mano: arrivano dall'inventario inviato dal sistema.
:::

### Rigenerazione Secret

:::danger Solo prima della registrazione
Il secret può essere rigenerato **solo finché il sistema non si è registrato**.
Una volta valorizzato `registered_at`, **Rigenera Secret** risponde HTTP 409:
l'appliance si autentica con il secret con cui si è registrata, e da qui non
esiste modo di installargliene uno nuovo.
:::

Finché il sistema non è registrato:

1. Vai al dettaglio del sistema
2. Clicca su **Rigenera Secret** (dal menu contestuale)
3. Conferma l'operazione
4. **Copia subito il nuovo secret**: viene mostrato una sola volta
5. Configura il nuovo secret sul sistema esterno

Il secret precedente viene invalidato immediatamente.

**Quando rigenerare:**
- Il secret è andato perso prima che il sistema riuscisse a registrarsi
- Il secret è trapelato prima di essere usato
- Il sistema era stato preparato ma mai messo in esercizio, e vuoi credenziali nuove

**Se il sistema è già registrato** e le sue credenziali sono compromesse o
perse, non c'è un percorso di rotazione: crea un **nuovo sistema**, registra la
macchina con il nuovo secret, poi elimina la riga vecchia. L'appliance può anche
rinunciare da sola alle proprie credenziali -- vedi
[Registrazione](./registration.md#annullare-la-registrazione-di-un-sistema).

### Eliminazione Soft

L'eliminazione soft marca il sistema come eliminato senza rimuovere i dati:

1. Vai al dettaglio del sistema
2. Clicca su **Elimina** (dal menu contestuale)
3. Conferma l'operazione

**Effetti:**
- Il sistema viene marcato come "eliminato"
- Non può più inviare inventario o heartbeat
- Sparisce dalle viste normali
- Le sue applicazioni spariscono da elenchi, totali e contatori delle organizzazioni finché il sistema non viene ripristinato (vengono conservate, non cancellate)
- Può essere ripristinato in qualsiasi momento
- Tutti i dati storici vengono conservati

**Per vedere i sistemi eliminati:**
1. Applica il filtro Stato = "deleted"
2. Seleziona il sistema eliminato
3. Clicca su **Ripristina**

### Eliminazione Permanente

:::danger
Questa operazione è irreversibile!
:::

Per eliminare definitivamente:
1. Elimina prima il sistema in modo soft
2. Vai alla vista dei sistemi eliminati
3. Seleziona il sistema
4. Clicca su **Elimina definitivamente**
5. Digita il nome del sistema per confermare
6. Clicca su **Elimina**

**Viene rimosso:**
- Il record del sistema
- Tutto lo storico dell'inventario
- Tutti i record di heartbeat
- Tutti i dati di rilevamento modifiche

**Viene conservato:**
- I log di audit
- I log di attività degli utenti

## Registrazione

Dopo la creazione, il sistema esterno deve registrarsi usando il `system_secret`.

### Flusso di Registrazione

1. **L'admin crea il sistema** → riceve il `system_secret`
2. **L'admin configura il sistema esterno** con il secret
3. **Il sistema esterno chiama l'API di registrazione** con il secret
4. **La piattaforma valida e restituisce** il `system_key`
5. **Il sistema esterno salva** entrambe le credenziali per gli usi futuri

Vedi [Registrazione Sistema](./registration.md) per le istruzioni dettagliate.

### Stato Registrazione

**Prima della registrazione:**
```json
{
  "system_key": "",
  "registered_at": null,
  "status": "unknown"
}
```

**Dopo la registrazione:**
```json
{
  "system_key": "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE",
  "registered_at": "2025-11-06T10:30:00Z",
  "status": "unknown"
}
```

## Monitoraggio

### Panoramica in Dashboard

La [Dashboard](../features/dashboard.md) porta due card rilevanti:

- **Sistemi**: il totale sulle organizzazioni che puoi leggere, con badge per attivi, inattivi e in attesa che aprono l'elenco già filtrato
- **Allarmi**: gli allarmi aperti sullo stesso perimetro, con badge per severità

### Esportazione Dati

Per esportare le informazioni sui sistemi:

1. Vai su **Sistemi**
2. Applica eventuali filtri
3. Clicca su **Azioni** > **Esporta**
4. Scegli il formato: CSV o PDF
5. Scarica il file

## Best Practice

### Nomenclatura dei Sistemi

- Usa nomi descrittivi che identifichino ruolo e sede
- Mantieni una convenzione coerente in tutta la flotta
- Tieni i nomi sotto i 50 caratteri
- Evita caratteri speciali

### Organizzazione

- Associa ogni sistema all'organizzazione cliente corretta
- Rivedi periodicamente le associazioni
- Usa le note per il contesto operativo

### Sicurezza

- Conserva il `system_secret` in modo sicuro e non versionarlo mai
- Copia il secret subito: viene mostrato una sola volta
- Se un secret è compromesso su un sistema registrato, sostituisci il sistema

### Monitoraggio

- Verifica regolarmente i sistemi in stato `inactive`
- Controlla che l'heartbeat arrivi con una cadenza ben sotto i 20 minuti
- Usa gli allarmi per accorgerti dei disservizi senza guardare l'elenco

## Risoluzione Problemi

### Sistema Non Presente nell'Elenco

**Problema:** un sistema atteso non è visibile

**Soluzioni:**
1. Controlla che il sistema appartenga a un'organizzazione accessibile
2. Verifica che non sia stato eliminato in modo soft (usa il filtro sui cancellati)
3. Conferma di avere il ruolo Support o Admin
4. Controlla i filtri attivi
5. Ricarica la pagina

### Impossibile Registrare il Sistema

**Problema:** la registrazione fallisce con "invalid system secret"

**Soluzioni:**
1. Verifica che il secret sia stato copiato correttamente (senza spazi di troppo)
2. Controlla che il secret non sia stato rigenerato
3. Conferma che il sistema non sia eliminato
4. Assicurati che il sistema non sia già registrato
5. Vedi [Risoluzione problemi della registrazione](./registration.md#risoluzione-problemi)

### Sistema Sempre in Stato "Inactive"

**Problema:** lo stato heartbeat del sistema resta "inactive"

**Soluzioni:**
1. Controlla che il sistema sia effettivamente acceso
2. Verifica la connettività di rete
3. Controlla i log del sistema per eventuali errori
4. Conferma che le credenziali siano corrette
5. Prova l'endpoint heartbeat a mano
6. Vedi [Inventario e Heartbeat](./inventory-heartbeat.md)

### Il system_key è Nascosto

**Problema:** il campo system_key non è visibile

**Spiegazione:**
- Il system_key resta nascosto finché il sistema non si registra
- È il comportamento atteso per i sistemi non registrati
- Registra il sistema per renderlo visibile

**Soluzione:**
1. Usa il system_secret per registrare il sistema
2. Dopo la registrazione il system_key diventa visibile
3. Vedi [Registrazione Sistema](./registration.md)

### Secret Perso

**Problema:** il secret del sistema non è stato salvato alla creazione

**Se il sistema non si è ancora registrato:**
1. Rigenera il secret del sistema
2. Copia subito quello nuovo
3. Configuralo sul sistema esterno: il secret vecchio è invalido immediatamente

**Se il sistema è già registrato:**
La rigenerazione viene rifiutata con HTTP 409 e non esiste un percorso di
rotazione. Crea un nuovo sistema, registra la macchina con il nuovo secret, poi
elimina la riga vecchia.

### Tipo Sistema Non Rilevato

**Problema:** il tipo di sistema risulta nullo o sconosciuto

**Spiegazione:**
- Il tipo viene rilevato automaticamente dal primo inventario
- Resta nullo finché non arriva il primo inventario

**Soluzione:**
1. Assicurati che il sistema sia registrato
2. Invia il primo inventario dal sistema esterno
3. Il tipo verrà rilevato automaticamente
4. Vedi [Inventario e Heartbeat](./inventory-heartbeat.md)

## Prossimi Passi

Dopo aver creato i sistemi:

- [Registra i sistemi esterni](./registration.md) usando il system_secret
- [Configura la raccolta dell'inventario](./inventory-heartbeat.md)
- Imposta monitoraggio e allarmi
- Consulta regolarmente le statistiche dei sistemi

## Documentazione Correlata

- [Registrazione Sistema](./registration.md)
- [Inventario e Heartbeat](./inventory-heartbeat.md)
- [Gestione Organizzazioni](../platform/organizations.md)
