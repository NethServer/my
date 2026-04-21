---
sidebar_position: 3
---

# Inventario e Heartbeat

Il sistema di inventario e heartbeat consente ai sistemi registrati di comunicare il proprio stato e la propria configurazione alla piattaforma My.

## Panoramica

Due meccanismi complementari mantengono la piattaforma aggiornata sullo stato dei sistemi:

- **Heartbeat** - Segnale periodico che indica che il sistema è attivo e raggiungibile
- **Inventario** - Raccolta completa dei dati di configurazione del sistema

Entrambi utilizzano **HTTP Basic Auth** per l'autenticazione, dove:
- **Username**: `system_key` (ottenuto alla registrazione)
- **Password**: `system_secret` (ottenuto alla creazione)

### Formato HTTP Basic Auth

```
Authorization: Basic base64(system_key:system_secret)
```

**Esempio:**
```
system_key: NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE
system_secret: my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0
```

La maggior parte delle librerie HTTP gestisce questo automaticamente:

```python
import requests

requests.post(url,
    auth=('NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE',
          'my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0'),
    json=data
)
```

## Heartbeat

### Endpoint

```
POST https://my.nethesis.it/api/systems/heartbeat
```

### Richiesta

```bash
curl -X POST \
  -u "SYSTEM_KEY:SYSTEM_SECRET" \
  -H "Content-Type: application/json" \
  https://my.nethesis.it/api/systems/heartbeat
```

### Risposta

```json
{
  "code": 200,
  "message": "heartbeat received successfully"
}
```

### Frequenza Consigliata

| Frequenza | Descrizione | Caso d'Uso |
|-----------|-------------|------------|
| 5 minuti | Consigliata | Monitoraggio standard |
| 1 minuto | Alta frequenza | Sistemi critici |
| 15 minuti | Bassa frequenza | Sistemi con connettività limitata |

:::tip
La frequenza consigliata è di **5 minuti**. Frequenze inferiori a 1 minuto sono sconsigliate in quanto generano carico non necessario sulla piattaforma.
:::

**Perché 5 minuti?**
- La piattaforma considera il sistema "active" se l'heartbeat è stato ricevuto entro 15 minuti
- Un intervallo di 5 minuti fornisce 3 battiti mancati prima di contrassegnare il sistema come morto
- Equilibrio ottimale tra traffico di rete e reattività del monitoraggio

### Implementazioni di Esempio

#### Python

```python
import requests
from requests.auth import HTTPBasicAuth

def send_heartbeat(system_key, system_secret, api_url):
    """Invia un heartbeat alla piattaforma My."""
    response = requests.post(
        f"{api_url}/api/systems/heartbeat",
        auth=HTTPBasicAuth(system_key, system_secret),
        headers={"Content-Type": "application/json"}
    )
    return response.status_code == 200
```

#### Bash

```bash
#!/bin/bash
# Invio heartbeat periodico

SYSTEM_KEY="your_system_key"
SYSTEM_SECRET="your_system_secret"
API_URL="https://my.nethesis.it"

curl -s -X POST \
  -u "${SYSTEM_KEY}:${SYSTEM_SECRET}" \
  -H "Content-Type: application/json" \
  "${API_URL}/api/systems/heartbeat"
```

### Classificazione degli Stati

Il sistema di heartbeat classifica automaticamente lo stato dei sistemi:

| Stato | Condizione | Descrizione |
|-------|-----------|-------------|
| **Alive** (Attivo) | Heartbeat ricevuto negli ultimi 30 minuti | Il sistema funziona normalmente |
| **Dead** (Morto) | Nessun heartbeat da oltre 30 minuti | Il sistema potrebbe essere spento o non raggiungibile |
| **Zombie** | Heartbeat sporadici | Il sistema ha comportamento instabile |

:::warning
La classificazione degli stati viene eseguita periodicamente da un cron job. Potrebbe esserci un ritardo tra l'interruzione degli heartbeat e l'aggiornamento dello stato.
:::

## Inventario

### Endpoint

```
POST https://my.nethesis.it/api/systems/inventory
```

### Schema JSON

L'inventario viene inviato come documento JSON contenente le informazioni di configurazione del sistema:

```json
{
  "inventory": {
    "os": {
      "name": "NethServer",
      "version": "8.1",
      "arch": "x86_64"
    },
    "hardware": {
      "cpu": {
        "model": "Intel Xeon E5-2680",
        "cores": 8
      },
      "memory": {
        "total_gb": 32
      },
      "disk": {
        "total_gb": 500
      }
    },
    "network": {
      "hostname": "server1.example.com",
      "interfaces": [
        {
          "name": "eth0",
          "ip": "192.168.1.100",
          "mac": "00:11:22:33:44:55"
        }
      ]
    },
    "services": [
      {
        "name": "httpd",
        "status": "running",
        "version": "2.4.57"
      }
    ]
  }
}
```

### Risposta

```json
{
  "code": 200,
  "message": "inventory received successfully",
  "data": {
    "system_key": "abc123def456"
  }
}
```

:::note
Il campo `system_key` nella risposta è presente solo alla prima richiesta (registrazione). Per le richieste successive, il campo `data` potrebbe essere vuoto.
:::

### Frequenza di Invio

| Frequenza | Descrizione | Caso d'Uso |
|-----------|-------------|------------|
| Giornaliera | Consigliata | Monitoraggio standard |
| Ogni 6 ore | Alta frequenza | Ambienti dinamici |
| Settimanale | Bassa frequenza | Sistemi stabili |

### Campi dell'Inventario

L'inventario può contenere diverse categorie di informazioni:

| Categoria | Descrizione | Esempi |
|-----------|-------------|--------|
| **os** | Sistema operativo | Nome, versione, architettura |
| **hardware** | Hardware | CPU, memoria, disco |
| **network** | Rete | Hostname, interfacce, IP |
| **services** | Servizi | Nome, stato, versione |
| **packages** | Pacchetti | Nome, versione installata |
| **users** | Utenti | Elenco utenti del sistema |
| **configuration** | Configurazione | Impostazioni specifiche |

### Implementazione di Esempio

#### Python

```python
import requests
import platform
import psutil

COLLECT_URL = "https://my.nethesis.it"
SYSTEM_KEY = "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE"
SYSTEM_SECRET = "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

def collect_inventory():
    """Raccoglie inventario del sistema"""
    return {
        "fqdn": platform.node(),
        "ipv4_address": "192.168.1.100",
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
        }
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
        print("Inventario inviato con successo")
        return True
    except Exception as e:
        print(f"Invio inventario fallito: {e}")
        return False
```

#### Bash

```bash
#!/bin/bash
# Raccolta e invio inventario

SYSTEM_KEY="your_system_key"
SYSTEM_SECRET="your_system_secret"
API_URL="https://my.nethesis.it"

# Raccolta dati
OS_NAME=$(cat /etc/os-release | grep ^NAME | cut -d= -f2 | tr -d '"')
OS_VERSION=$(cat /etc/os-release | grep ^VERSION_ID | cut -d= -f2 | tr -d '"')
CPU_CORES=$(nproc)
MEMORY_GB=$(free -g | awk '/Mem:/{print $2}')
HOSTNAME=$(hostname -f)

# Costruzione JSON
INVENTORY=$(cat <<EOF
{
  "inventory": {
    "os": {
      "name": "${OS_NAME}",
      "version": "${OS_VERSION}",
      "arch": "$(uname -m)"
    },
    "hardware": {
      "cpu": {
        "cores": ${CPU_CORES}
      },
      "memory": {
        "total_gb": ${MEMORY_GB}
      }
    },
    "network": {
      "hostname": "${HOSTNAME}"
    }
  }
}
EOF
)

# Invio inventario
curl -s -X POST \
  -u "${SYSTEM_KEY}:${SYSTEM_SECRET}" \
  -H "Content-Type: application/json" \
  -d "${INVENTORY}" \
  "${API_URL}/api/systems/inventory"
```

## Storico e Timeline

### Politica di Retention

Gli snapshot dell'inventario sono conservati con densità esponenziale:

| Eta | Frequenza Snapshot |
|-----|--------------------|
| Ultimi 7 giorni | Tutti gli snapshot |
| 7 giorni - 1 mese | 1 al giorno |
| 1 mese - 3 mesi | 1 alla settimana |
| 3 mesi - 1 anno | 1 al mese |
| Oltre 1 anno | 1 al trimestre |

Il **primo snapshot** ricevuto per un sistema (la baseline) e lo **snapshot più recente** (stato attuale) sono sempre conservati indipendentemente dall'eta.

Tutte le **modifiche (diff)** vengono conservate in modo permanente e non vengono mai eliminate.

- **Heartbeat**: Lo stato viene tracciato nel tempo

### Timeline

La pagina di dettaglio del sistema mostra una timeline cronologica con:

- **Invii di inventario** con indicazione delle modifiche rilevate
- **Variazioni di stato** (attivo, morto, zombie)
- **Heartbeat** con timestamp

## Rilevamento Modifiche

### Categorie di Modifiche

Il motore di diff analizza automaticamente le differenze tra invii successivi di inventario e le categorizza:

| Categoria | Descrizione | Esempi |
|-----------|-------------|--------|
| **Hardware** | Modifiche hardware | CPU, memoria, disco |
| **Software** | Modifiche software | Pacchetti, servizi |
| **Rete** | Modifiche di rete | IP, interfacce, DNS |
| **Configurazione** | Modifiche di configurazione | Impostazioni, parametri |

### Livelli di Severità

Ogni modifica viene classificata con un livello di severità:

| Severità | Descrizione | Esempio |
|----------|-------------|---------|
| **Info** | Modifica informativa | Aggiornamento versione pacchetto |
| **Warning** | Modifica che richiede attenzione | Modifica interfaccia di rete |
| **Critical** | Modifica critica | Riduzione memoria, rimozione disco |

### Significatività

Non tutte le modifiche sono significative. Il motore di diff utilizza una configurazione YAML per determinare quali modifiche sono significative e quali possono essere ignorate.

:::note
La configurazione del rilevamento modifiche è gestita dal servizio Collect tramite il file `differ/config.yaml`. Le modifiche non significative vengono registrate ma non generano notifiche.
:::

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

## Monitoraggio

### Dashboard

La dashboard mostra un riepilogo dello stato di tutti i sistemi:

- **Sistemi attivi** - Numero di sistemi con heartbeat recente
- **Sistemi morti** - Numero di sistemi senza heartbeat
- **Sistemi zombie** - Numero di sistemi con comportamento instabile
- **Totale sistemi** - Numero totale di sistemi registrati

### Avvisi

Avvisi automatici per:
- Sistema diventa inattivo (nessun heartbeat per 15+ minuti)
- Modifiche critiche rilevate nell'inventario
- Nuovo sistema registrato
- Discordanza versione sistema
- Vulnerabilità di sicurezza rilevate

:::note
L'alert interno `LinkFailed` viene generato da Collect dopo il timeout heartbeat configurato (10 minuti di default), separato dalla soglia di stato del sistema a 15+ minuti mostrata sopra. Collect lo aggiorna ogni 5 minuti finché il sistema resta inattivo, quindi può rimanere visibile fino a 10 minuti dopo la ripresa dell'heartbeat.
:::

### Salute Sistema

Punteggio salute basato su:
- Affidabilità heartbeat (% uptime)
- Freschezza inventario
- Numero di modifiche
- Conteggio problemi critici

### Totali

L'endpoint `/api/systems/totals` fornisce un riepilogo statistico:

```json
{
  "total": 150,
  "alive": 120,
  "dead": 25,
  "zombie": 5
}
```

## Risoluzione Problemi

### Il Sistema Non Invia Heartbeat

1. Verifica che il servizio di heartbeat sia in esecuzione
2. Controlla le credenziali (system_key e system_secret)
3. Verifica la connettività di rete verso `my.nethesis.it`
4. Controlla i log per errori HTTP (401, 403, 500)

### L'Inventario Non Viene Aggiornato

1. Verifica che lo script di raccolta inventario sia in esecuzione
2. Controlla il formato JSON dell'inventario (deve essere valido)
3. Verifica che le dimensioni del payload non superino i limiti
4. Controlla i log per errori nella risposta

### Timeout Connessione

**Problema:** Timeout richiesta, nessuna risposta

**Soluzioni:**
1. Controlla connettività di rete:
   ```bash
   ping my.nethesis.it
   ```
2. Controlla regole firewall (consenti uscita HTTPS)
3. Verifica risoluzione DNS
4. Testa da rete diversa

### Modifiche Non Rilevate

**Problema:** Inventario inviato ma nessuna modifica mostrata

**Soluzioni:**
1. Verifica che i dati siano effettivamente cambiati tra inventari
2. Controlla che i campi modificati siano supportati
3. Le piccole modifiche numeriche potrebbero non attivare il rilevamento
4. I campi personalizzati sono tracciati per modifiche
5. Attendi il prossimo inventario e confronta

### Lo Stato del Sistema è Errato

- Lo stato viene aggiornato periodicamente dal cron job
- Potrebbe esserci un ritardo tra l'invio dell'heartbeat e l'aggiornamento dello stato
- Verifica che il sistema stia effettivamente inviando heartbeat con la frequenza configurata

### Errore 401 nelle Richieste

- Le credenziali non sono valide
- Il system_secret potrebbe essere stato rigenerato
- Il formato dell'autenticazione HTTP Basic potrebbe non essere corretto
- Verifica che username e password siano nell'ordine corretto (username:password)

## Best Practice

- **Configura il heartbeat** come prima cosa dopo la registrazione
- **Invia l'inventario** regolarmente per mantenere i dati aggiornati
- **Monitora gli stati** dalla dashboard per individuare problemi rapidamente
- **Verifica le modifiche** rilevate per identificare cambiamenti non autorizzati
- **Conserva le credenziali** in modo sicuro sul sistema
- **Automatizza** l'invio di heartbeat e inventario con cron job o servizi systemd
- **Gestisci gli errori** nello script di invio per evitare perdita di dati
