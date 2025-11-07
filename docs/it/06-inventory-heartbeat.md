# Inventario e Heartbeat

Scopri come i sistemi esterni inviano dati di inventario e segnali di heartbeat alla piattaforma My.

## Panoramica

Dopo la [registrazione del sistema](05-system-registration.md), i sistemi esterni comunicano con My attraverso due meccanismi:

1. **Inventario**: Snapshot completo delle informazioni del sistema (hardware, software, configurazione)
2. **Heartbeat**: Segnale periodico "Sono vivo" per indicare che il sistema è online

Entrambe le operazioni usano **HTTP Basic Authentication** con le credenziali registrate.

## Autenticazione

### Credenziali

Usa le credenziali ottenute durante il ciclo di vita del sistema:

- **Username**: `system_key` (ricevuto alla registrazione)
- **Password**: `system_secret` (dalla creazione del sistema)

### HTTP Basic Auth

**Formato header:**
```
Authorization: Basic base64(system_key:system_secret)
```

**Esempio:**
```
system_key: NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE
system_secret: my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0

Base64 encode: "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE:my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

Authorization: Basic bXlfc3lzX2FiYzEyM2RlZjQ1NjpteV9hMWIyYzNkNGU1ZjZnN2g4aTlqMC5rMWwybTNuNG81cDZxN3I4czl0MHUxdjJ3M3g0eTV6NmE3YjhjOWQw
```

**La maggior parte delle librerie HTTP gestisce questo automaticamente:**
```python
import requests

requests.post(url,
    auth=('NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE', 'my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0'),
    json=data
)
```

## Heartbeat

L'heartbeat è un semplice segnale per indicare che il sistema è vivo e raggiungibile.

### Scopo

- Rilevare quando i sistemi vanno offline
- Attivare avvisi per sistemi morti
- Monitorare l'affidabilità del sistema

### Endpoint

```
POST https://my.nethesis.it/collect/api/systems/heartbeat
```

**Nota:** Il servizio Collect è su **/collect** (diverso dal backend principale su **/backend**)

### Richiesta

**Headers:**
```
Authorization: Basic <credentials>
Content-Type: application/json
```

**Body:**
```json
{}
```

**Oggetto JSON vuoto** - nessun dato necessario!

### Risposta

**Successo (HTTP 200):**
```json
{
  "code": 200,
  "message": "heartbeat riconosciuto",
  "data": {
    "system_key": "NOC-80F8-89A4-40B0-4AE9-A670-7C5F-99B3-F3EA",
    "acknowledged": true,
    "last_heartbeat": "2025-11-07T10:37:27.360343+01:00"
  }
}
```

### Frequenza

**Raccomandato:** Ogni 5 minuti

**Perché 5 minuti?**
- La piattaforma considera il sistema "active" se heartbeat < 15 minuti
- Intervallo di 5 minuti fornisce 3 battiti mancati prima di contrassegnare come morto
- Equilibrio tra traffico di rete e reattività

### Esempio di Implementazione

**Python:**
```python
import requests
import time

COLLECT_URL = "https://my.nethesis.it/collect"
SYSTEM_KEY = "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE"
SYSTEM_SECRET = "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

def send_heartbeat():
    """Invia heartbeat alla piattaforma My"""
    try:
        response = requests.post(
            f"{COLLECT_URL}/api/systems/heartbeat",
            auth=(SYSTEM_KEY, SYSTEM_SECRET),
            json={},
            timeout=10
        )
        response.raise_for_status()
        print("Heartbeat inviato con successo")
        return True
    except Exception as e:
        print(f"Heartbeat fallito: {e}")
        return False

# Invia heartbeat ogni 5 minuti
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

**Entry Crontab (ogni 5 minuti):**
```
*/5 * * * * /usr/local/bin/my-heartbeat.sh
```

### Stato Heartbeat

I sistemi sono classificati in base all'heartbeat:

| Stato | Condizione | Colore | Significato |
|--------|-----------|-------|---------|
| **Active** | < 15 minuti | 🟢 Verde | Il sistema è sano |
| **Inactive** | ≥ 15 minuti | 🟡 Giallo | Il sistema è offline |
| **Unknown** | Mai inviato | ⚪ Grigio | Mai comunicato |

## Inventario

L'inventario è uno snapshot completo della configurazione del sistema e del software installato.

### Scopo

- Tracciare hardware e software del sistema
- Rilevare modifiche alla configurazione
- Monitorare versioni software
- Audit della conformità del sistema
- Tracciare lo storico dell'inventario

### Endpoint

```
POST https://my.nethesis.it/collect/api/systems/inventory
```

**Nota:** Stesso servizio collect, porta **8081**

### Richiesta

**Headers:**
```
Authorization: Basic <credentials>
Content-Type: application/json
```

**Struttura Body:**
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
  "message": "Inventario ricevuto e messo in coda per l'elaborazione",
  "data": {
    "data_size": 16433,
    "message": "I dati del tuo inventario sono stati ricevuti e saranno elaborati a breve",
    "queue_status": "queued",
    "system_id": "0a98637c-077b-428a-8e57-c2fbb892051a",
    "timestamp": "2025-11-07T10:39:05.897352+01:00"
  }
}
```

### Frequenza

**Raccomandato:** Ogni 6 ore (4 volte al giorno)

**Perché 6 ore?**
- Equilibrio tra freschezza e carico rete/storage
- Cattura modifiche giornaliere
- Riduce la crescita del database
- Sufficiente per la maggior parte delle esigenze di monitoraggio

**Casi speciali:**
- **Dopo modifiche al sistema**: Invia immediatamente
- **Durante aggiornamenti**: Invia prima e dopo
- **On-demand**: L'admin può attivare via API

### Schema Inventario

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

#### Sezioni Opzionali

Tutte le altre sezioni sono opzionali ma raccomandate:

- `os`: Informazioni sistema operativo
- `hardware`: Specifiche hardware fisico/virtuale
- `network`: Configurazione di rete
- `services`: Servizi installati e versioni
- `features`: Funzionalità abilitate e capacità
- `custom`: Qualsiasi dato personalizzato (forma libera)

#### Auto-Rilevamento

Alcuni campi sono auto-rilevati dalla piattaforma:

- **Tipo Sistema**: Rilevato dai dati dell'inventario (ns8, nsec, ecc.)
- **Stato**: Auto-aggiornato in base all'heartbeat
- **Ultimo Aggiornamento**: Timestamp ricezione inventario

### Esempio di Implementazione

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
    """Raccoglie inventario del sistema"""
    return {
        "fqdn": platform.node(),
        "ipv4_address": get_primary_ip(),  # Tua implementazione
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
        "services": collect_services(),  # Tua implementazione
        "features": collect_features(),  # Tua implementazione
    }

def send_inventory():
    """Invia inventario alla piattaforma My"""
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
        print(f"Inventario inviato con successo")
        print(f"Modifiche rilevate: {data['data']['changes_detected']}")
        return True

    except Exception as e:
        print(f"Invio inventario fallito: {e}")
        return False

# Invia inventario ogni 6 ore
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

# Raccoglie inventario (esempio - personalizza per il tuo sistema)
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

# Invia alla piattaforma
curl -X POST "$COLLECT_URL/api/systems/inventory" \
  -u "$SYSTEM_KEY:$SYSTEM_SECRET" \
  -H "Content-Type: application/json" \
  -d "$INVENTORY"
```

## Rilevamento Modifiche

My rileva automaticamente le modifiche tra gli snapshot dell'inventario.

### Cosa viene Tracciato

- Modifiche hardware (CPU, memoria, disco)
- Modifiche versione software
- Aggiunte/rimozioni servizi
- Modifiche configurazione di rete
- Toggle funzionalità
- Modifiche campi personalizzati

### Categorie di Modifiche

Le modifiche sono categorizzate per tipo:

- **OS**: Sistema operativo e kernel
- **Hardware**: Hardware fisico/virtuale
- **Network**: Interfacce di rete e configurazione
- **Features**: Funzionalità abilitate e capacità
- **Services**: Servizi installati
- **System**: Impostazioni generali di sistema

### Severità Modifiche

Ogni modifica ha un livello di severità:

- **Critical**: Richiede attenzione immediata (es. guasto hardware)
- **High**: Modifiche importanti (es. aggiornamento OS)
- **Medium**: Modifiche notevoli (es. aggiornamento servizio)
- **Low**: Modifiche minori (es. aggiornamento metriche)

### Visualizzazione Modifiche

**Nel Pannello Admin:**
1. Naviga su **Sistemi** > **Dettagli Sistema**
2. Clicca tab **Inventario**
3. Visualizza sezione **Modifiche**
4. Vedi diff dettagliato tra versioni

**Tipi di modifica:**
- 🟢 **Create**: Nuovo campo aggiunto
- 🟡 **Update**: Valore campo modificato
- 🔴 **Delete**: Campo rimosso

**Esempio log modifiche:**
```
[2025-11-06 10:30] Versione OS aggiornata
  - Vecchio: Rocky Linux 9.2
  - Nuovo: Rocky Linux 9.3
  - Severità: High
  - Categoria: OS

[2025-11-06 10:30] Versione Nginx aggiornata
  - Vecchio: 1.23.0
  - Nuovo: 1.24.0
  - Severità: Medium
  - Categoria: Services

[2025-11-06 10:30] Nuova funzionalità abilitata: Docker
  - Valore: 24.0.7
  - Severità: Medium
  - Categoria: Features
```

## Monitoraggio nel Pannello Admin

### Stato Real-time

**Vista dashboard:**
- Conteggio sistemi totali
- Ripartizione Active / Inactive / Unknown
- Modifiche inventario recenti
- Sistemi che richiedono attenzione

**Elenco sistemi:**
- Indicatore stato heartbeat (🟢🟡⚪)
- Ultimo tempo heartbeat
- Ultimo tempo inventario
- Notifiche modifiche

### Avvisi (se configurati)

Avvisi automatici per:
- Sistema va offline (nessun heartbeat per 15+ minuti)
- Modifiche critiche rilevate nell'inventario
- Nuovo sistema registrato
- Discordanza versione sistema
- Vulnerabilità di sicurezza rilevate

### Salute Sistema

**Punteggio salute basato su:**
- Affidabilità heartbeat (% uptime)
- Freschezza inventario
- Numero di modifiche
- Conteggio problemi critici

## Risoluzione Problemi

### Autenticazione Fallisce (HTTP 401)

**Problema:** "Credenziali sistema non valide" o "Non autorizzato"

**Soluzioni:**
1. Verifica che le credenziali siano corrette:
   ```bash
   echo -n "system_key:system_secret" | base64
   ```
2. Controlla spazi extra nelle credenziali
3. Assicurati che il sistema sia registrato
4. Verifica che il secret non sia stato rigenerato
5. Testa con curl:
   ```bash
   curl -v -u "system_key:system_secret" \
     https://my.nethesis.it/collect/api/systems/heartbeat \
     -H "Content-Type: application/json" \
     -d '{}'
   ```

### Timeout Connessione

**Problema:** Timeout richiesta, nessuna risposta

**Soluzioni:**
1. Controlla connettività di rete:
   ```bash
   ping my.nethesis.it
   ```
2. Verifica che la porta 8081 sia accessibile:
   ```bash
   telnet my.nethesis.it 8081
   ```
3. Controlla regole firewall (consenti uscita verso porta 8081)
4. Verifica risoluzione DNS
5. Testa da rete diversa

### Inventario Non si Aggiorna

**Problema:** Inventario inviato con successo ma non visibile nel pannello admin

**Soluzioni:**
1. Attendi 60 secondi e aggiorna (propagazione cache)
2. Verifica che stai visualizzando il sistema corretto
3. Controlla che l'inventario sia stato inviato all'endpoint corretto (porta 8081)
4. Verifica che il sistema non sia eliminato
5. Controlla log sistema per errori

### Heartbeat Mostra come "Dead"

**Problema:** Il sistema mostra stato rosso/morto nonostante invii heartbeat

**Soluzioni:**
1. Controlla frequenza heartbeat (deve essere < 15 minuti)
2. Verifica che l'heartbeat raggiunga la piattaforma:
   ```bash
   curl -v https://my.nethesis.it/collect/api/systems/heartbeat \
     -u "key:secret" -H "Content-Type: application/json" -d '{}'
   ```
3. Controlla che l'ora di sistema sia sincronizzata (NTP)
4. Verifica nessuna deriva del clock
5. Rivedi log servizio collect (solo admin)

### Modifiche Non Rilevate

**Problema:** Inventario inviato ma nessuna modifica mostrata

**Soluzioni:**
1. Verifica che i dati siano effettivamente cambiati tra inventari
2. Controlla che i campi modificati siano supportati
3. Le piccole modifiche numeriche potrebbero non attivare il rilevamento
4. I campi personalizzati sono tracciati per modifiche
5. Attendi il prossimo inventario e confronta

## Best Practice

### Heartbeat

- Invia ogni 5 minuti in modo consistente
- Usa task schedulato (cron, systemd timer)
- Logga i fallimenti heartbeat per debug
- Implementa logica di retry (exponential backoff)
- Monitora tasso di successo heartbeat

### Inventario

- Invia inventario completo ogni volta
- Non inviare aggiornamenti parziali
- Includi tutti i dati rilevanti
- Usa nomi campo consistenti
- Valida JSON prima dell'invio
- Invia immediatamente dopo modifiche significative

### Gestione Errori

- Implementa logica di retry per fallimenti di rete
- Logga tutti gli errori con contesto
- Non riprovare errori di autenticazione (401)
- Usa exponential backoff per i retry
- Avvisa sui fallimenti ripetuti

### Sicurezza

- Memorizza credenziali in modo sicuro
- Non loggare mai le credenziali
- Usa solo HTTPS
- Verifica certificati SSL
- Ruota periodicamente le credenziali
- Monitora i fallimenti di autenticazione

### Performance

- Comprimi inventari di grandi dimensioni
- Raggruppa la raccolta dati
- Evita invii inventario non necessari
- Usa strutture dati efficienti
- Monitora la larghezza di banda di rete

## Documentazione Correlata

- [Registrazione Sistema](05-system-registration.md)
- [Gestione Sistemi](04-systems.md)
- [API Backend](https://github.com/NethServer/my/blob/main/backend/README.md)
- [Servizio Collect](https://github.com/NethServer/my/blob/main/collect/README.md)
