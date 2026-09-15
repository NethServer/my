---
sidebar_position: 3
---

# Inventario e Heartbeat

Come i sistemi esterni inviano i dati di inventario e i segnali di heartbeat alla piattaforma My.

## Panoramica

Dopo la [registrazione del sistema](./registration.md), i sistemi esterni comunicano con My attraverso due meccanismi:

1. **Inventario**: snapshot completo delle informazioni di sistema (hardware, software, configurazione)
2. **Heartbeat**: segnale periodico "sono vivo" che indica che il sistema è attivo

Entrambe le operazioni usano l'**autenticazione HTTP Basic** con le credenziali registrate.

## Autenticazione

### Credenziali

Si usano le credenziali ottenute nel ciclo di vita del sistema:

- **Username**: `system_key` (ricevuto alla registrazione)
- **Password**: `system_secret` (dalla creazione del sistema)

### Formato HTTP Basic Auth

**Formato dell'header:**
```
Authorization: Basic base64(system_key:system_secret)
```

**Esempio:**
```
system_key: NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE
system_secret: my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0

Codifica base64 di: "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE:my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

Authorization: Basic bXlfc3lzX2FiYzEyM2RlZjQ1NjpteV9hMWIyYzNkNGU1ZjZnN2g4aTlqMC5rMWwybTNuNG81cDZxN3I4czl0MHUxdjJ3M3g0eTV6NmE3YjhjOWQw
```

**La maggior parte delle librerie HTTP lo gestisce automaticamente:**
```python
import requests

requests.post(url,
    auth=('NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE', 'my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0'),
    json=data
)
```

## Heartbeat

L'heartbeat è un segnale semplice che indica che il sistema è vivo e raggiungibile.

### Scopo

- Rilevare quando un sistema diventa inattivo
- Generare allarmi per i sistemi che non rispondono
- Monitorare l'affidabilità dei sistemi

### Endpoint

```
POST https://my.nethesis.it/collect/api/systems/heartbeat
```

:::note
Il servizio collect risponde sotto **/collect** (diverso dal backend principale, che sta sotto **/backend**)
:::

### Richiesta

**Header:**
```
Authorization: Basic <credenziali>
Content-Type: application/json
```

**Corpo:**
```json
{}
```

**Oggetto JSON vuoto**: non serve alcun dato.

### Risposta

**Successo (HTTP 200):**
```json
{
  "code": 200,
  "message": "heartbeat acknowledged",
  "data": {
    "system_key": "NOC-80F8-89A4-40B0-4AE9-A670-7C5F-99B3-F3EA",
    "acknowledged": true,
    "last_heartbeat": "2025-11-07T10:37:27.360343+01:00"
  }
}
```

### Frequenza

**Consigliata:** ogni 5 minuti

**Perché 5 minuti?**
- Un sistema resta `active` finché il suo ultimo heartbeat è più recente del
  timeout della piattaforma, **20 minuti** di default (`HEARTBEAT_TIMEOUT_MINUTES`)
- Un intervallo di 5 minuti lascia spazio a tre battiti persi prima che il
  sistema cambi stato
- È un buon compromesso tra traffico di rete e reattività

### Implementazioni di Esempio

**Python:**
```python
import requests
import time

COLLECT_URL = "https://my.nethesis.it/collect"
SYSTEM_KEY = "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE"
SYSTEM_SECRET = "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

def send_heartbeat():
    """Invia l'heartbeat alla piattaforma My"""
    try:
        response = requests.post(
            f"{COLLECT_URL}/api/systems/heartbeat",
            auth=(SYSTEM_KEY, SYSTEM_SECRET),
            json={},
            timeout=10
        )
        response.raise_for_status()
        print("Heartbeat inviato correttamente")
        return True
    except Exception as e:
        print(f"Heartbeat fallito: {e}")
        return False

# Invia l'heartbeat ogni 5 minuti
while True:
    send_heartbeat()
    time.sleep(300)  # 5 minuti
```

**Bash (cron):**
```bash
#!/bin/bash
# /usr/local/bin/my-heartbeat.sh

COLLECT_URL="https://my.nethesis.it/collect"
SYSTEM_KEY="NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE"
SYSTEM_SECRET="my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

curl -s -X POST "$COLLECT_URL/api/systems/heartbeat" \
  -u "$SYSTEM_KEY:$SYSTEM_SECRET" \
  -H "Content-Type: application/json" \
  -d '{}' > /dev/null
```

**Voce di crontab (ogni 5 minuti):**
```
*/5 * * * * /usr/local/bin/my-heartbeat.sh
```

### Classificazione degli Stati

I sistemi vengono classificati in base all'heartbeat:

| Stato | Condizione | Significato |
|-------|------------|-------------|
| **Unknown** | Non ha mai inviato un heartbeat | Registrato ma mai entrato in contatto |
| **Active** | Ultimo heartbeat più recente di 20 minuti | Il sistema è in salute |
| **Inactive** | Ultimo heartbeat più vecchio di 20 minuti | Il sistema non risponde |
| **Unregistered** | Ha rinunciato alle proprie credenziali | Stato terminale, vedi [Registrazione](./registration.md#annullare-la-registrazione-di-un-sistema) |

:::note Perché lo stato è in ritardo
I 20 minuti arrivano da `HEARTBEAT_TIMEOUT_MINUTES`, e un cron rivaluta tutti i
sistemi **ogni 5 minuti** (`HEARTBEAT_CHECK_INTERVAL_SECONDS`). Un sistema
silenzioso compare quindi come `inactive` tra i 20 e i 25 minuti dopo l'ultimo
heartbeat, non esattamente a 20. Il ritorno funziona allo stesso modo: il primo
heartbeat dopo il disservizio ripristina `active` al passaggio di cron
successivo, non immediatamente.
:::

## Inventario

L'inventario è uno snapshot completo della configurazione del sistema e del software installato.

### Scopo

- Tracciare hardware e software del sistema
- Rilevare le modifiche di configurazione
- Monitorare le versioni del software
- Verificare la conformità dei sistemi
- Conservare lo storico dell'inventario

### Endpoint

```
POST https://my.nethesis.it/collect/api/systems/inventory
```

:::note
Stesso servizio collect dell'heartbeat, sotto lo stesso prefisso **/collect**
:::

### Richiesta

**Header:**
```
Authorization: Basic <credenziali>
Content-Type: application/json
```

**Struttura del corpo:**
```json
{
  "fqdn": "server.example.com",
  "ipv4_address": "192.168.1.100",
  "ipv6_address": "2001:db8::1",
  "version": "8.0.1",
  "os": {
    "name": "Rocky Linux",
    "version": "9.3",
    "kernel": "5.14.0-362.8.1.el9_3.x86_64"
  },
  "hardware": {
    "cpu_model": "Intel Xeon Gold 6248R",
    "cpu_cores": 8,
    "cpu_threads": 16,
    "memory_total_gb": 32,
    "disk_total_gb": 500
  },
  "network": {
    "hostname": "server01",
    "interfaces": {
      "eth0": {
        "ip": "192.168.1.100",
        "netmask": "255.255.255.0",
        "mac": "00:1a:2b:3c:4d:5e"
      },
      "eth1": {
        "ip": "10.0.0.10",
        "netmask": "255.255.0.0",
        "mac": "00:1a:2b:3c:4d:5f"
      }
    }
  },
  "services": {
    "nginx": {
      "version": "1.24.0",
      "status": "running"
    },
    "postgresql": {
      "version": "15.5",
      "status": "running"
    },
    "redis": {
      "version": "7.2.3",
      "status": "running"
    }
  },
  "features": {
    "docker": {
      "enabled": true,
      "version": "24.0.7"
    },
    "firewall": {
      "enabled": true,
      "type": "nftables"
    }
  },
  "custom": {
    "environment": "production",
    "datacenter": "EU-West-1",
    "backup_enabled": true
  }
}
```

### Risposta

**Successo (HTTP 200):**
```json
{
  "code": 200,
  "message": "Inventory received and queued for processing",
  "data": {
    "data_size": 16433,
    "message": "Your inventory data has been received and will be processed shortly",
    "queue_status": "queued",
    "system_id": "0a98637c-077b-428a-8e57-c2fbb892051a",
    "timestamp": "2025-11-07T10:39:05.897352+01:00"
  }
}
```

### Frequenza

**Consigliata:** ogni 6 ore (4 volte al giorno)

**Perché 6 ore?**
- È un compromesso tra freschezza del dato e carico di rete e storage
- Intercetta le modifiche quotidiane
- Contiene la crescita del database
- È sufficiente per la maggior parte delle esigenze di monitoraggio

**Casi particolari:**
- **Dopo una modifica al sistema**: invia subito
- **Durante un aggiornamento**: invia prima e dopo
- **Su richiesta**: l'amministratore può sollecitarlo via API

### Schema dell'Inventario

#### Campi Obbligatori

**Dati minimi richiesti:**
```json
{
  "fqdn": "server.example.com",
  "ipv4_address": "192.168.1.100",
  "os": {
    "name": "NethSec",
    "type": "nethsecurity",
    "family": "OpenWRT",
    "release": {
        "full": "8.6.0-dev+43d54cd33.20251020175318",
        "major": 7
    }
  }
}
```

#### Sezioni Facoltative

Tutte le altre sezioni sono facoltative ma consigliate:

- `os`: informazioni sul sistema operativo
- `hardware`: caratteristiche hardware fisiche o virtuali
- `network`: configurazione di rete
- `services`: servizi installati e relative versioni
- `features`: funzionalità abilitate
- `custom`: dati personalizzati (formato libero)

#### Rilevamento Automatico

Alcuni campi vengono ricavati automaticamente dalla piattaforma:

- **Tipo di sistema**: rilevato dai dati di inventario (ns8, nsec, ecc.)
- **Stato**: aggiornato automaticamente in base all'heartbeat
- **Ultimo aggiornamento**: data e ora di ricezione dell'inventario

### Implementazioni di Esempio

**Python:**
```python
import requests
import platform
import psutil
import json

COLLECT_URL = "https://my.nethesis.it/collect"
SYSTEM_KEY = "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE"
SYSTEM_SECRET = "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

def collect_inventory():
    """Raccoglie l'inventario del sistema"""
    return {
        "fqdn": platform.node(),
        "ipv4_address": get_primary_ip(),  # Implementazione tua
        "version": "8.0.1",
        "os": {
            "name": platform.system(),
            "version": platform.release(),
            "kernel": platform.version()
        },
        "hardware": {
            "cpu_cores": psutil.cpu_count(logical=False),
            "cpu_threads": psutil.cpu_count(logical=True),
            "memory_total_gb": round(psutil.virtual_memory().total / (1024**3), 2)
        },
        "services": collect_services(),  # Implementazione tua
        "features": collect_features(),  # Implementazione tua
    }

def send_inventory():
    """Invia l'inventario alla piattaforma My"""
    try:
        inventory = collect_inventory()

        response = requests.post(
            f"{COLLECT_URL}/api/systems/inventory",
            auth=(SYSTEM_KEY, SYSTEM_SECRET),
            json=inventory,
            timeout=30
        )
        response.raise_for_status()

        data = response.json()
        print("Inventario inviato correttamente")
        print(f"Stato in coda: {data['data']['queue_status']}")
        return True

    except Exception as e:
        print(f"Invio dell'inventario fallito: {e}")
        return False

# Invia l'inventario ogni 6 ore
import time
while True:
    send_inventory()
    time.sleep(21600)  # 6 ore
```

**Bash:**
```bash
#!/bin/bash
# /usr/local/bin/my-inventory.sh

COLLECT_URL="https://my.nethesis.it/collect"
SYSTEM_KEY="NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE"
SYSTEM_SECRET="my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

# Raccolta dell'inventario (esempio: adattalo al tuo sistema)
INVENTORY=$(cat <<EOF
{
  "fqdn": "$(hostname -f)",
  "ipv4_address": "$(hostname -I | awk '{print $1}')",
  "version": "8.0.1",
  "os": {
    "name": "$(uname -s)",
    "version": "$(uname -r)"
  },
  "hardware": {
    "cpu_cores": $(nproc),
    "memory_total_gb": $(free -g | awk '/^Mem:/{print $2}')
  }
}
EOF
)

# Invio alla piattaforma
curl -X POST "$COLLECT_URL/api/systems/inventory" \
  -u "$SYSTEM_KEY:$SYSTEM_SECRET" \
  -H "Content-Type: application/json" \
  -d "$INVENTORY"
```

## Storico e Timeline dell'Inventario

My conserva lo storico completo delle modifiche di inventario di ogni sistema, così da poter vedere come si è evoluto nel tempo, dalla prima registrazione a oggi.

### Cosa Viene Conservato

- **Tutte le modifiche (diff)**: ogni variazione rilevata viene conservata in modo permanente e non viene mai cancellata. Ogni diff registra il campo cambiato, il valore precedente e quello nuovo.
- **Snapshot di inventario**: gli snapshot JSON completi vengono conservati con densità esponenziale, più fitti vicino al presente e via via più radi per le date lontane:

| Età | Frequenza degli snapshot |
|-----|--------------------------|
| Ultimi 7 giorni | Tutti gli snapshot |
| Da 7 giorni a 1 mese | 1 al giorno |
| Da 1 mese a 3 mesi | 1 alla settimana |
| Da 3 mesi a 1 anno | 1 al mese |
| Oltre 1 anno | 1 al trimestre |

:::tip
Il **primo snapshot in assoluto** ricevuto per un sistema (la baseline) e il **più recente** (lo stato corrente) vengono sempre conservati, indipendentemente dall'età.
:::

### La Timeline

La timeline mostra l'evoluzione completa di un sistema raggruppata per data. Per ogni invio di inventario che conteneva modifiche puoi vedere:

- Quando è avvenuta la modifica
- Quali campi sono cambiati (percorso, valore precedente, valore nuovo)
- Severità e categoria di ogni modifica

Poiché tutti i diff vengono conservati in modo permanente, la timeline resta interamente navigabile anche per i sistemi registrati da anni. Puoi filtrare per intervallo di date, severità, categoria o tipo di modifica.

### Consultare lo Storico

**Nel pannello di amministrazione:**
1. Vai su **Sistemi** > **Dettagli sistema**
2. Apri la tab **Inventario**
3. Usa la vista **Timeline** per lo storico raggruppato per data
4. Usa la vista **Storico** per gli snapshot grezzi, paginati

## Rilevamento Modifiche

My rileva automaticamente le differenze tra uno snapshot di inventario e il successivo.

### Cosa Viene Tracciato

- Modifiche hardware (CPU, memoria, disco)
- Cambi di versione del software
- Aggiunta o rimozione di servizi
- Modifiche alla configurazione di rete
- Attivazione o disattivazione di funzionalità
- Modifiche ai campi personalizzati

### Categorie di Modifiche

Le modifiche vengono classificate per tipo:

- **OS**: sistema operativo e kernel
- **Hardware**: hardware fisico o virtuale
- **Network**: interfacce e configurazione di rete
- **Features**: funzionalità abilitate
- **Services**: servizi installati
- **System**: impostazioni generali di sistema

### Livelli di Severità

Ogni modifica ha un livello di severità:

- **Critical**: richiede attenzione immediata (es. guasto hardware)
- **High**: modifiche importanti (es. aggiornamento del sistema operativo)
- **Medium**: modifiche rilevanti (es. aggiornamento di un servizio)
- **Low**: modifiche minori (es. aggiornamento di una metrica)

### Consultare le Modifiche

**Nel pannello di amministrazione:**
1. Vai su **Sistemi** > **Dettagli sistema**
2. Apri la tab **Inventario**
3. Guarda la sezione **Modifiche**
4. Consulta il diff dettagliato tra le versioni

**Tipi di modifica:**
- **Create**: nuovo campo aggiunto
- **Update**: valore del campo cambiato
- **Delete**: campo rimosso

**Esempio di log delle modifiche:**
```
[2025-11-06 10:30] Versione OS aggiornata
  - Prima: Rocky Linux 9.2
  - Dopo: Rocky Linux 9.3
  - Severità: High
  - Categoria: OS

[2025-11-06 10:30] Versione Nginx aggiornata
  - Prima: 1.23.0
  - Dopo: 1.24.0
  - Severità: Medium
  - Categoria: Services

[2025-11-06 10:30] Nuova funzionalità abilitata: Docker
  - Valore: 24.0.7
  - Severità: Medium
  - Categoria: Features
```

## Monitoraggio dal Pannello di Amministrazione

### Stato in Tempo Reale

**Vista dashboard** (vedi [Dashboard](../features/dashboard.md)):
- Totale dei sistemi, con badge per attivi, inattivi e in attesa che aprono l'elenco già filtrato
- Allarmi aperti, con badge per severità

**Elenco dei sistemi:**
- Indicatore dello stato heartbeat
- Data dell'ultimo heartbeat
- Data dell'ultimo inventario
- Segnalazione delle modifiche

### Avvisi (se configurati)

Avvisi automatici per:
- Sistema che diventa inattivo (nessun heartbeat per oltre 20 minuti)
- Modifiche critiche rilevate nell'inventario
- Nuovo sistema registrato
- Discordanza di versione del sistema
- Vulnerabilità di sicurezza rilevate

:::note
L'alert interno `LinkFailed` viene generato da Collect per ogni sistema già in
stato `inactive`, quindi segue lo stesso `HEARTBEAT_TIMEOUT_MINUTES` (20 di
default). Collect lo aggiorna ogni 5 minuti finché il sistema resta inattivo, e
l'alert porta un TTL pari al doppio di quell'intervallo: può quindi restare
visibile fino a 10 minuti dopo la ripresa dell'heartbeat. È un ritardo voluto,
che evita il flapping quando gli heartbeat arrivano a ridosso del timeout.
:::

### Salute del Sistema

Ogni inventario elaborato porta con sé un **punteggio di salute** ricavato dalle modifiche rilevate. Parte da 100 e perde punti per ogni diff, in base alla severità:

| Severità | Punti sottratti |
|----------|-----------------|
| Critical | 10 |
| High | 5 |
| Medium | 2 |
| Low | 1 |

Il punteggio non scende sotto 0, e un inventario senza modifiche vale 100. Misura quanto è stato dirompente l'ultimo insieme di modifiche, non l'affidabilità dell'heartbeat né la freschezza dell'inventario.

## Risoluzione Problemi

### L'Autenticazione Fallisce (HTTP 401)

**Problema:** "Invalid system credentials" oppure "Unauthorized"

**Soluzioni:**
1. Verifica che le credenziali siano corrette:
   ```bash
   echo -n "system_key:system_secret" | base64
   ```
2. Controlla che non ci siano spazi di troppo nelle credenziali
3. Assicurati che il sistema sia registrato
4. Verifica che il secret non sia stato rigenerato
5. Prova con curl:
   ```bash
   curl -v -u "system_key:system_secret" \
     https://my.nethesis.it/collect/api/systems/heartbeat \
     -H "Content-Type: application/json" \
     -d '{}'
   ```

### Timeout di Connessione

**Problema:** la richiesta va in timeout, nessuna risposta

**Soluzioni:**
1. Verifica la connettività di rete:
   ```bash
   ping my.nethesis.it
   ```
2. Verifica che l'HTTPS sia raggiungibile:
   ```bash
   curl -sI https://my.nethesis.it/collect/api/systems/heartbeat
   ```
3. Controlla le regole del firewall (consenti HTTPS in uscita verso my.nethesis.it)
4. Verifica la risoluzione DNS
5. Prova da un'altra rete

### L'Inventario Non Viene Aggiornato

**Problema:** l'inventario risulta inviato ma non compare nel pannello di amministrazione

**Soluzioni:**
1. Aspetta 60 secondi e ricarica (propagazione della cache)
2. Verifica di stare guardando il sistema giusto
3. Controlla che l'inventario sia stato inviato all'endpoint corretto (`/collect/api/systems/inventory`)
4. Verifica che il sistema non sia eliminato
5. Controlla i log del sistema per eventuali errori

### Il Sistema Risulta Inattivo

**Problema:** il sistema risulta inattivo nonostante invii l'heartbeat

**Soluzioni:**
1. Controlla la frequenza dell'heartbeat (deve stare ben sotto il timeout di 20 minuti)
2. Verifica che l'heartbeat arrivi davvero alla piattaforma:
   ```bash
   curl -v https://my.nethesis.it/collect/api/systems/heartbeat \
     -u "key:secret" -H "Content-Type: application/json" -d '{}'
   ```
3. Controlla che l'orologio di sistema sia sincronizzato (NTP)
4. Verifica che non ci sia deriva dell'orologio
5. Consulta i log del servizio collect (solo amministratori)

### Modifiche Non Rilevate

**Problema:** l'inventario è stato inviato ma non compare alcuna modifica

**Soluzioni:**
1. Verifica che i dati siano effettivamente cambiati tra un inventario e l'altro
2. Controlla che i campi cambiati siano tra quelli supportati
3. Piccole variazioni numeriche possono non far scattare il rilevamento
4. Anche i campi personalizzati vengono confrontati
5. Attendi l'inventario successivo e confronta

## Best Practice

### Heartbeat

- Invia con regolarità ogni 5 minuti
- Usa un'attività pianificata (cron, timer systemd)
- Registra i fallimenti dell'heartbeat per la diagnosi
- Implementa una logica di retry (backoff esponenziale)
- Monitora il tasso di successo degli heartbeat

### Inventario

- Invia sempre l'inventario completo
- Non inviare aggiornamenti parziali
- Includi tutti i dati rilevanti
- Usa nomi di campo coerenti
- Valida il JSON prima di inviarlo
- Invia subito dopo una modifica significativa

### Gestione degli Errori

- Implementa una logica di retry per i problemi di rete
- Registra tutti gli errori con il loro contesto
- Non ritentare sugli errori di autenticazione (401)
- Usa il backoff esponenziale per i tentativi
- Genera un avviso in caso di fallimenti ripetuti

### Sicurezza

- Conserva le credenziali in modo sicuro
- Non scrivere mai le credenziali nei log
- Usa esclusivamente HTTPS
- Verifica i certificati SSL
- Monitora i fallimenti di autenticazione

:::warning
Per un sistema registrato non esiste rotazione delle credenziali: `Rigenera Secret` viene rifiutato una volta valorizzato `registered_at`. Tratta il `system_secret` come permanente per tutta la vita del sistema, e sostituisci il sistema stesso se viene compromesso.
:::

### Prestazioni

- Comprimi gli inventari di grandi dimensioni
- Raggruppa la raccolta dei dati
- Evita invii di inventario non necessari
- Usa strutture dati efficienti
- Monitora la banda di rete consumata

## Documentazione Correlata

- [Registrazione Sistema](./registration.md)
- [Gestione Sistemi](./management.md)
- [Documentazione API Backend](https://github.com/NethServer/my/blob/main/backend/README.md)
- [Documentazione Servizio Collect](https://github.com/NethServer/my/blob/main/collect/README.md)
