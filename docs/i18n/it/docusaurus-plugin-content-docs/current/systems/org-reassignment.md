---
sidebar_position: 5
---

# Riassegnare un sistema a un'altra organizzazione

A volte un sistema deve passare da un cliente a un altro — un MSP
restituisce un server al cliente finale, l'organizzazione viene
riorganizzata, oppure un sistema era stato registrato sotto il tenant
sbagliato. MY supporta questa operazione nativamente: il sistema porta
con sé tutti i propri dati al nuovo proprietario, e quello precedente
perde l'accesso immediatamente.

## Quando usarla

- Un reseller o distributor prende in carico (o restituisce) un sistema
  di un cliente.
- Un sistema è stato registrato sotto la organizzazione sbagliata in
  fase di setup.
- Un'organizzazione viene fusa o divisa, e i suoi sistemi devono seguire.

## Cosa viene spostato

Quando cambi l'organizzazione di un sistema, MY sposta tutti i dati che
*appartengono al sistema* al nuovo proprietario, lasciando intatti i dati
del proprietario precedente:

| Dato | Cosa succede |
|---|---|
| **Backup di configurazione** | Copiati nell'area di storage del nuovo proprietario prima che il cambio sia confermato; le copie del precedente proprietario vengono rimosse. |
| **Storico allarmi** | Lo storico completo segue il sistema, così il nuovo proprietario vede ciò che stava accadendo prima del passaggio di consegne. |
| **Silence specifiche del sistema** | I silence a singolo scopo (che silenziano tutti gli allarmi di quel solo sistema) vengono rimossi — il nuovo proprietario può riconfigurarli se gli servono. I silence più ampi creati dal precedente proprietario (ad esempio "silenzia gli allarmi disco pieno su questo sistema durante una manutenzione") restano dove sono e scadono per conto loro. |
| **Assegnazioni delle applicazioni** | Tutte le assegnazioni app→organizzazione sul sistema vengono azzerate. Il nuovo proprietario riassegna le app che gli interessano dalla normale vista Applicazioni. |
| **Inventory, heartbeat, identità del sistema** | Intatti — `system_key`, il secret e l'intero storico inventari restano. Le appliance non devono ri-registrarsi. |

Il cambio è **atomico** sui dati che contano: o tutto è in posizione
sotto il nuovo proprietario, oppure il cambio viene rifiutato senza
effetti collaterali. La pulizia dei dati residui del proprietario
precedente avviene dopo lo switch ed è best-effort: eventuali residui
non influenzano in alcun modo la vista del nuovo proprietario.

## Come riassegnare

### Dalla UI

1. Apri la pagina di dettaglio del sistema.
2. Modifica il sistema e seleziona una **Organizzazione** diversa dal
   menu a tendina.
3. Salva. In pochi secondi il sistema mostra il nuovo proprietario e
   tutti i dati di cui sopra sono già in posizione.

### Dall'API

```bash
curl -X PUT https://my.example.com/api/systems/$SYSTEM_ID \
     -H "Authorization: Bearer $JWT" \
     -H "Content-Type: application/json" \
     -d '{"name":"unchanged","organization_id":"<new-logto-id>"}'
```

Una riassegnazione con backup nei limiti di retention si completa
tipicamente in pochi secondi. Con carichi più pesanti può richiedere
più tempo — la richiesta resta aperta finché lo spostamento non è
completamente concluso lato nuovo proprietario.

## Chi può farlo

- L'**Owner** può riassegnare qualunque sistema a qualunque
  organizzazione esistente.
- **Distributor** e **Reseller** possono riassegnare sistemi all'interno
  della propria gerarchia: non possono spostare un sistema *fuori* dal
  proprio scope verso un'organizzazione che non gestiscono.
- Il **Customer** non può riassegnare sistemi.

Una riassegnazione verso un'organizzazione che non esiste viene
rifiutata con `403 access denied`: non c'è nessun percorso che possa
lasciare un sistema sotto un'organizzazione irraggiungibile.

## Cosa vede il proprietario precedente

Non appena il cambio viene confermato:

- Il sistema scompare dalla lista "Sistemi" del proprietario precedente.
- Le chiamate API del proprietario precedente sul sistema rispondono
  `404` (il sistema non è più nel suo scope).
- Eventuali link diretti che aveva alla pagina di backup, allarmi o
  dettaglio del sistema rispondono anch'essi `404`.

Lo stesso comportamento di una cancellazione del sistema dal punto di
vista del proprietario precedente — la differenza è che i dati esistono
ancora, semplicemente sotto il nuovo proprietario.

## Modifiche concorrenti

Se due amministratori provano a riassegnare lo stesso sistema nello
stesso momento, solo il primo riesce. Il secondo riceve un chiaro
`409 system reassignment is already in progress` e può ritentare una
volta che la prima riassegnazione è terminata.

Questa protezione evita che il sistema finisca in uno stato a metà con
i dati spezzati tra due destinazioni diverse.

## Domande frequenti

**Posso riportare il sistema al proprietario originale?**
Sì. La riassegnazione è simmetrica — spostare da A a B e poi da B ad A
funziona allo stesso modo nelle due direzioni.

**Il sistema continua a funzionare durante la riassegnazione?**
Sì. L'appliance continua a inviare inventari, heartbeat e backup senza
interruzioni. Gli upload in volo nel momento del cambio atterrano
correttamente sotto il nuovo proprietario una volta completato lo
spostamento.

**E una sessione di support tunnel attiva?**
Una sessione di supporto già aperta continua finché non scade
(default 24h). Le nuove sessioni vengono avviate dal nuovo proprietario.

**Il sistema deve essere online per essere riassegnato?**
No. La riassegnazione opera solo sui dati lato piattaforma; lo stato
dell'appliance non viene toccato.

**Posso annullare una riassegnazione?**
Le riassegnazioni vengono registrate in un log di audit visibile
all'Owner della piattaforma. Non c'è un undo a un click, ma riassegnare
indietro al proprietario originale è il modo standard per farlo:
i dati seguono il sistema in entrambe le direzioni.

## Correlato

- [Gestione dei sistemi](management) — l'intero ciclo di vita di un sistema.
- [Backup di configurazione](backups) — cosa viene memorizzato, come è
  protetto, e come si calcola la quota per organizzazione.
