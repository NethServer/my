# Gestione Sistemi

Scopri come creare e gestire i sistemi nella piattaforma My.

## Comprendere i Sistemi

Un **sistema** in My rappresenta un server o dispositivo gestito (NethServer o NethSecurity) che:

- Appartiene a un'organizzazione
- Invia dati di inventario automaticamente
- Invia segnali di heartbeat per indicare che è attivo
- Può essere monitorato e gestito da remoto

### Ciclo di Vita del Sistema

```
1. Creato da Admin/Support → riceve system_secret
2. Non ancora registrato → system_key è nascosto
3. Sistema esterno si registra → system_key diventa visibile
4. Sistema invia inventario e heartbeat → stato monitorato
```

### Stato del Sistema

- **Unknown**: Stato predefinito, nessun inventario ricevuto ancora
- **Active**: Sistema sta inviando heartbeat attivamente (< 15 minuti)
- **Inactive**: Sistema ha smesso di inviare heartbeat (> 15 minuti)
- **Deleted**: Sistema è stato eliminato in modo soft

## Creazione Sistemi

### Prerequisiti

- Devi avere il ruolo **Support** o **Admin**
- Hai bisogno di un'organizzazione cliente a cui associare il sistema
- Il sistema verrà creato nello stato "non registrato"

### Creare un Nuovo Sistema

1. Naviga su **Sistemi**
2. Clicca **Crea sistema**
3. Compila il modulo:
   - **Nome**: Nome descrittivo per il sistema (es. "Server Produzione Milano")
   - **Organizzazione**: Seleziona l'organizzazione cliente
   - **Note** (opzionale): Informazioni aggiuntive
4. Clicca **Crea sistema**

**Esempio:**
```
Nome: Server Web Produzione Milano
Organizzazione: Pizza Express Milano (Cliente)
Note: Server di produzione principale per le sedi di Milano
```

### Secret del Sistema

Dopo la creazione, vedrai:

```json
{
  "id": "sys_abc123",
  "name": "Server Web Produzione Milano",
  "system_key": "",  // ← NASCOSTO fino alla registrazione
  "system_secret": "my_a1b2c3.k1l2m3...",  // ← SALVALO! Mostrato solo una volta
  "status": "unknown",
  "registered_at": null,
  "organization": "Pizza Express Milano"
}
```

**⚠️ CRITICO:**
- Il `system_secret` viene mostrato **solo una volta** durante la creazione
- Copialo e salvalo immediatamente
- Ne avrai bisogno per registrare il sistema
- Se lo perdi, devi rigenerarlo (invalida il secret precedente)

## Visualizzazione Sistemi

### Elenco Sistemi

Naviga su **Sistemi** per vedere:

- Nome del sistema
- Tipo (ns8, nsec, ecc.)
- Versione
- FQDN e indirizzi IP
- Organizzazione
- Creato da
- Stato (unknown, online, offline)
- Stato registrazione

### Filtraggio e Ricerca

Usa i filtri per trovare sistemi specifici:

- **Cerca**: Per nome o system_key
- **Prodotto**: Filtra per tipo (NethServer o NethSecurity)
- **Versione**: Filtra per versione del sistema
- **Organizzazione**: Filtra per organizzazione cliente
- **Creato da**: Filtra per utente che ha creato il sistema
- **Stato**: unknown, online, offline, deleted
- **Ordina per**: Nome, versione, FQDN/indirizzo IP, Organizzazione, Creato da, Stato

### Dettagli Sistema

Clicca su un sistema per visualizzare informazioni complete:

#### Tab Panoramica

- **Informazioni di Base**:
  - Nome del sistema
  - Tipo del sistema (auto-rilevato)
  - Stato
  - Versione
  - Timestamp registrazione

- **Informazioni di Rete**:
  - FQDN (Fully Qualified Domain Name)
  - Indirizzo IPv4
  - Indirizzo IPv6

- **Autenticazione**:
  - System key (visibile solo dopo la registrazione)
  - Stato registrazione
  - Ultimo timestamp di autenticazione

- **Organizzazione**:
  - Nome cliente
  - Tipo organizzazione
  - Nome organizzazione

- **Stato Heartbeat**:
  - Stato corrente (active/inactive/unknown)
  - Ultimo timestamp heartbeat
  - Ultimo timestamp inventario

- **Traccia Audit**:
  - Creato da (nome utente e email)
  - Data creazione
  - Data eliminazione (se eliminato in modo soft)

#### Tab Inventario

Visualizza l'inventario dettagliato del sistema:

- **Ultimo Inventario**: Snapshot più recente dell'inventario
- **Storico Inventario**: Tutti gli inventari storici con paginazione
- **Modifiche**: Elenco delle modifiche rilevate tra gli inventari
- **Vista Diff**: Confronto dettagliato tra versioni dell'inventario

Vedi [Inventario e Heartbeat](06-inventory-heartbeat.md) per i dettagli.

## Gestione Sistemi

### Modifica Informazioni Sistema

1. Naviga alla pagina del sistema
2. Clicca **Modifica**
3. Aggiorna i campi:
   - Nome
   - Organizzazione
   - Note
4. Clicca **Salva sistema**

### Rigenerazione Secret Sistema

Se il `system_secret` è compromesso o perso:

1. Naviga alla pagina del sistema
2. Clicca **Rigenera Secret** (usando il menu kebab)
3. Conferma l'azione
4. **Copia immediatamente il nuovo secret** (mostrato solo una volta)
5. Aggiorna il secret sul sistema esterno

**⚠️ Attenzione:**
- Il vecchio secret viene invalidato immediatamente
- Il sistema non può autenticarsi fino alla configurazione del nuovo secret
- Tutti gli inventari e heartbeat falliranno fino all'aggiornamento
- Il sistema rimane registrato (registered_at invariato)

**Quando rigenerare:**
- Il secret è compromesso o divulgato
- Il secret è perso (per sistemi registrati)
- Un audit di sicurezza richiede la rotazione delle credenziali
- Migrazione del sistema a nuovo hardware

### Eliminazione Soft

L'eliminazione soft contrassegna un sistema come eliminato senza rimuovere i dati:

1. Naviga alla pagina dei dettagli del sistema
2. Clicca **Elimina** (usando il menu kebab)
3. Conferma l'azione

**Effetti:**
- Sistema contrassegnato come "deleted"
- Non può inviare inventario o heartbeat
- Nascosto dalle viste normali
- Può essere ripristinato se necessario
- Tutti i dati storici sono preservati

**Per visualizzare i sistemi eliminati:**
1. Applica filtro: Stato = "deleted"
2. Seleziona il sistema eliminato
3. Clicca **Ripristina** per annullare l'eliminazione

### Eliminazione Permanente

**⚠️ Attenzione:** Questa operazione è irreversibile!

Per eliminare permanentemente:
1. Elimina prima il sistema in modo soft
2. Naviga alla vista dei sistemi eliminati
3. Seleziona il sistema
4. Clicca **Eliminazione Permanente**
5. Digita il nome del sistema per confermare
6. Clicca **Elimina**

**Questo rimuoverà:**
- Record del sistema
- Tutto lo storico dell'inventario
- Tutti i record di heartbeat
- Tutti i dati di rilevamento modifiche

**Questo preserverà:**
- Log di audit
- Log di attività utente

## Registrazione Sistema

Dopo aver creato un sistema, il sistema esterno deve registrarsi usando il `system_secret`.

### Flusso di Registrazione

1. **Admin crea sistema** → riceve `system_secret`
2. **Admin configura sistema esterno** con il secret
3. **Sistema esterno chiama API di registrazione** con il secret
4. **Piattaforma valida e restituisce** `system_key`
5. **Sistema esterno memorizza** entrambe le credenziali per uso futuro

Vedi [Registrazione Sistema](05-system-registration.md) per istruzioni dettagliate.

### Stato Registrazione

**Prima della Registrazione:**
```json
{
  "system_key": "",  // Nascosto
  "registered_at": null,
  "status": "unknown"
}
```

**Dopo la Registrazione:**
```json
{
  "system_key": "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE",  // Ora visibile
  "registered_at": "2025-11-06T10:30:00Z",
  "status": "unknown"  // Si aggiornerà dopo il primo inventario
}
```

## Monitoraggio Sistema

### Panoramica Dashboard

Naviga su **Dashboard** per vedere:

- **Sistemi Totali**: Conteggio tra le organizzazioni accessibili
- **Stato Sistema**: Distribuzione (unknown/online/offline)
- **Stato Heartbeat**: Conteggi active/inactive/unknown
- **Modifiche Recenti**: Ultime modifiche dell'inventario
- **Avvisi**: Sistemi con problemi

### Esportazione Dati Sistema

Esporta le informazioni del sistema per reportistica:

1. Naviga su **Sistemi**
2. Applica filtri se necessario
3. Clicca **Azioni** > **Esporta**
4. Scegli il formato:
   - CSV o PDF
5. Scarica il file

## Best Practice

### Denominazione Sistemi

- Usa nomi descrittivi e coerenti
- Includi la posizione se rilevante: "Server Milano Nord"
- Includi lo scopo: "Web Produzione", "Server Backup"
- Evita caratteri speciali
- Mantieni i nomi sotto i 50 caratteri

### Organizzazione

- Raggruppa i sistemi per cliente
- Usa dati personalizzati per la categorizzazione
- Etichetta i sistemi con l'ambiente (prod/staging/dev)
- Documenta lo scopo del sistema nelle note

### Sicurezza

- Memorizza i secret in modo sicuro (password manager, vault)
- Non condividere mai i secret via email
- Revoca immediatamente i secret se compromessi
- Monitora i tentativi di autenticazione falliti

### Monitoraggio

- Controlla lo stato heartbeat quotidianamente
- Rivedi le modifiche dell'inventario settimanalmente
- Configura avvisi per i sistemi critici
- Monitora le versioni dei sistemi per gli aggiornamenti

## Risoluzione Problemi

### Sistema Non Appare nell'Elenco

**Problema:** Il sistema atteso non è visibile

**Soluzioni:**
1. Controlla se il sistema appartiene a un'organizzazione accessibile
2. Verifica che il sistema non sia eliminato in modo soft (controlla il filtro deleted)
3. Conferma di avere il ruolo Support o Admin
4. Controlla se sono applicati filtri
5. Aggiorna la pagina

### Impossibile Registrare Sistema

**Problema:** La registrazione fallisce con "invalid system secret"

**Soluzioni:**
1. Verifica che il secret sia stato copiato correttamente (senza spazi extra)
2. Controlla che il secret non sia stato rigenerato
3. Conferma che il sistema non sia eliminato
4. Assicurati che il sistema non sia già registrato
5. Vedi [Risoluzione Problemi Registrazione Sistema](05-system-registration.md#risoluzione-problemi)

### Sistema Mostra come "Inactive"

**Problema:** Lo stato heartbeat del sistema è "inactive" (giallo)

**Soluzioni:**
1. Controlla se il sistema è effettivamente in esecuzione
2. Verifica la connettività di rete
3. Controlla i log del sistema per errori
4. Conferma che le credenziali siano corrette
5. Testa manualmente l'endpoint heartbeat
6. Vedi [Inventario e Heartbeat](06-inventory-heartbeat.md)

### System_key è Nascosto

**Problema:** Impossibile vedere il campo system_key

**Spiegazione:**
- system_key è nascosto fino alla registrazione del sistema
- Questo è il comportamento previsto per sistemi non registrati
- Registra prima il sistema per rivelare system_key

**Soluzione:**
1. Usa system_secret per registrare il sistema
2. Dopo la registrazione, system_key diventa visibile
3. Vedi [Registrazione Sistema](05-system-registration.md)

### Secret Sistema Perso

**Problema:** Il secret del sistema non è stato salvato durante la creazione

**Soluzioni:**
1. Rigenera il secret del sistema
2. Configura il sistema esterno con il nuovo secret
3. Il sistema deve ri-registrarsi se già registrato
4. Il vecchio secret diventa invalido immediatamente

### Tipo Sistema Non Rilevato

**Problema:** Il tipo di sistema mostra come null o unknown

**Spiegazione:**
- Il tipo di sistema è auto-rilevato dal primo inventario
- Mostra null fino alla ricezione del primo inventario

**Soluzione:**
1. Assicurati che il sistema sia registrato
2. Invia il primo inventario dal sistema esterno
3. Il tipo verrà rilevato automaticamente
4. Vedi [Inventario e Heartbeat](06-inventory-heartbeat.md)

## Prossimi Passi

Dopo aver creato i sistemi:

- [Registra sistemi esterni](05-system-registration.md) usando system_secret
- [Configura la raccolta inventario](06-inventory-heartbeat.md)
- Configura monitoraggio e avvisi
- Rivedi regolarmente le statistiche del sistema

## Documentazione Correlata

- [Registrazione Sistema](05-system-registration.md)
- [Inventario e Heartbeat](06-inventory-heartbeat.md)
- [Gestione Organizzazioni](02-organizations.md)
