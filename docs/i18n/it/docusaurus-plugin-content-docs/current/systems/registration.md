---
sidebar_position: 2
---

# Registrazione Sistema

Come un sistema esterno si registra sulla piattaforma My per abilitare monitoraggio e gestione.

## Panoramica

La registrazione è il processo con cui un sistema esterno (NethServer, NethSecurity, ecc.) si autentica su My e riceve le proprie credenziali permanenti.

### Perché Serve la Registrazione

- **Sicurezza**: valida il sistema prima di accettarne i dati
- **Autenticazione**: stabilisce credenziali di lungo periodo
- **Tracciamento**: registra quando il sistema si è collegato la prima volta
- **Visibilità**: rende il system_key visibile agli amministratori

### Flusso di Registrazione

```
┌─────────────┐                                     ┌──────────────┐
│             │  1. Crea il sistema                 │              │
│    Admin    │───────────────────────────────────> │ Piattaforma  │
│             │  ← Ritorna il secret (una volta)    │      My      │
└─────────────┘                                     └──────────────┘
                                                           │
                                                           │
┌─────────────┐                                            │
│   Sistema   │  2. Configura il system_secret             │
│   Esterno   │<───────────────────────────────────────────┘
│ (NethServer)│
└─────────────┘
      │
      │  3. Chiama l'API di registrazione
      │     POST /backend/api/systems/register
      │     { "system_secret": "my_..." }
      │
      v
┌──────────────┐
│ Piattaforma  │  4. Valida il secret
│      My      │     ✓ Formato corretto
│              │     ✓ Parte pubblica esistente
│              │     ✓ Parte secret verificata (SHA256)
│              │     ✓ Non eliminato
│              │     ✓ Non gia' registrato
└──────────────┘
      │
      │  5. Restituisce il system_key
      v
┌─────────────┐
│   Sistema   │  6. Salva le credenziali:
│   Esterno   │     - system_key (username)
│             │     - system_secret (password)
└─────────────┘
      │
      │  7. Pronto per inventario e heartbeat!
      v
```

## Comprendere le Credenziali

### system_secret (Creato alla Creazione del Sistema)

**Formato:** `my_<parte_pubblica>.<parte_secret>`

**Esempio:** `my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0`

**Componenti:**
- **Prefisso**: `my_` (identifica il tipo di token)
- **Parte pubblica**: 20 caratteri esadecimali (per la ricerca su database)
- **Separatore**: `.` (punto)
- **Parte secret**: 40 caratteri esadecimali (hashata con SHA256)

**Caratteristiche:**
- Mostrato **una sola volta**, alla creazione del sistema
- Non è recuperabile in seguito (la rigenerazione ne crea uno nuovo)
- Usato per la registrazione (una volta sola)
- Usato per tutte le autenticazioni successive (inventario, heartbeat)

### system_key (Ricevuto alla Registrazione)

**Formato:** `NOC-<stringa_casuale>`

**Esempio:** `NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE`

**Caratteristiche:**
- Generato alla creazione del sistema
- Nascosto finché il sistema non si registra
- Visibile dopo la registrazione andata a buon fine
- Usato come username per l'HTTP Basic Auth
- Non cambia mai (nemmeno se il secret viene rigenerato)

## Processo di Registrazione

### Passo 1: L'Admin Crea il Sistema

Vedi [Gestione Sistemi](./management.md#creazione-sistemi) per i dettagli.

Dopo la creazione, salva il `system_secret`:
```json
{
  "system_secret": "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
}
```

### Passo 2: Configurazione del Sistema Esterno

Configura il sistema esterno con il `system_secret`. Il metodo esatto dipende dal tipo di sistema:

#### Per NethServer/NethSecurity:

1. Accedi all'interfaccia di amministrazione del sistema
2. Vai su **Impostazioni** > **Sottoscrizione**
3. Incolla il `system_secret`
4. Clicca **Registra**

#### Per Sistemi Custom (API):

Salva il secret in modo sicuro nella tua applicazione:

**Esempio di file di configurazione:**
```bash
# /etc/my/config.conf
MY_PLATFORM_URL=https://my.nethesis.it
MY_SYSTEM_SECRET=my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0
```

**Variabili d'ambiente:**
```bash
export MY_PLATFORM_URL="https://my.nethesis.it"
export MY_SYSTEM_SECRET="my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
```

### Passo 3: Chiamata all'API di Registrazione

Il sistema esterno effettua una richiesta POST per registrarsi:

**Endpoint:** `POST https://my.nethesis.it/backend/api/systems/register`

**Header:**
```
Content-Type: application/json
```

**Corpo della Richiesta:**
```json
{
  "system_secret": "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
}
```

**Esempio cURL:**
```bash
curl -X POST https://my.nethesis.it/backend/api/systems/register \
  -H "Content-Type: application/json" \
  -d '{
    "system_secret": "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
  }'
```

**Esempio Python:**
```python
import requests

url = "https://my.nethesis.it/backend/api/systems/register"
payload = {
    "system_secret": "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
}

response = requests.post(url, json=payload)
data = response.json()
system_key = data["data"]["system_key"]
print(f"Registrato! system_key: {system_key}")
```

### Passo 4: La Piattaforma Valida

La piattaforma esegue diversi controlli di sicurezza:

1. **Validazione del formato del token**:
   - Divide su `.` -- devono risultare esattamente 2 parti
   - La prima parte deve iniziare con `my_`
   - Estrae parte pubblica e parte secret

2. **Ricerca su database**:
   - Trova il sistema tramite la parte pubblica
   - Query veloce e indicizzata su `system_secret_public`

3. **Controlli di sicurezza**:
   - Il sistema non è eliminato
   - Il sistema non è già registrato
   - La parte pubblica coincide con il valore memorizzato

4. **Verifica crittografica**:
   - Verifica la parte secret contro l'hash SHA256
   - Confronto a tempo costante (previene i timing attack)

### Passo 5: Registrazione Completata

**Risposta di successo (HTTP 200):**
```json
{
  "code": 200,
  "message": "system registered successfully",
  "data": {
    "system_key": "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE",
    "registered_at": "2025-11-06T10:30:00Z",
    "message": "system registered successfully"
  }
}
```

**Cosa succede:**
- Il timestamp `registered_at` viene scritto sul database
- Il `system_key` diventa visibile agli amministratori
- Il sistema può ora autenticarsi per inventario e heartbeat

### Passo 6: Salvataggio delle Credenziali

Il sistema esterno deve salvare in modo sicuro entrambe le credenziali:

**Necessarie per le autenticazioni successive:**
- `system_key`: username per l'HTTP Basic Auth
- `system_secret`: password per l'HTTP Basic Auth

**Consigli per la conservazione:**
```bash
# File di configurazione
MY_SYSTEM_KEY=NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE
MY_SYSTEM_SECRET=my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0

# Oppure usa una conservazione sicura:
# - Keyring di sistema
# - Configurazione cifrata
# - Gestore di segreti (Vault, ecc.)
```

## Risposte di Errore

### Formato del Token Non Valido

**HTTP 400 Bad Request:**
```json
{
  "code": 400,
  "message": "invalid system secret format",
  "data": null
}
```

**Cause:**
- Il token non contiene il separatore `.`
- Il token non inizia con `my_`
- Il token è malformato

**Soluzione:**
- Verifica che il secret sia stato copiato correttamente
- Controlla che non ci siano spazi o a capo di troppo
- Assicurati di inviare il token completo

### Credenziali Non Valide

**HTTP 401 Unauthorized:**
```json
{
  "code": 401,
  "message": "invalid system secret",
  "data": null
}
```

**Cause:**
- Parte pubblica non trovata sul database
- La parte secret non corrisponde all'hash
- È stato fornito il secret sbagliato

**Soluzione:**
- Verifica che il secret sia corretto
- Controlla se il secret è stato rigenerato
- Assicurati che il sistema sia stato creato sulla piattaforma My

### Sistema Eliminato

**HTTP 403 Forbidden:**
```json
{
  "code": 403,
  "message": "system has been deleted",
  "data": null
}
```

**Cause:**
- Il sistema è stato eliminato (soft delete) da un amministratore
- Il sistema risulta eliminato sul database

**Soluzione:**
- Contatta l'amministratore per ripristinare il sistema
- Crea un nuovo sistema, se necessario

### Già Registrato

**HTTP 409 Conflict:**
```json
{
  "code": 409,
  "message": "system is already registered",
  "data": null
}
```

**Cause:**
- Il sistema ha già completato la registrazione
- Il campo `registered_at` non è nullo

**Soluzione:**
- Il sistema è già registrato, si può procedere con l'autenticazione
- Usa il `system_key` e il `system_secret` esistenti
- Non serve fare nulla, a meno che non serva una nuova registrazione

## Dopo la Registrazione

### Verificare lo Stato di Registrazione

Gli amministratori possono verificare lo stato di registrazione:

1. Vai su **Sistemi**
2. Trova il sistema e clicca **Visualizza**
3. Controlla i campi:
   - **System_key**: ora visibile (prima era nascosto)
   - **Sottoscrizione**: mostra il timestamp
   - **Stato**: può restare "sconosciuto" fino al primo inventario

### Prossimi Passi per il Sistema Esterno

Dopo una registrazione andata a buon fine, il sistema dovrebbe:

1. **Salvare le credenziali in modo sicuro**
2. **Inviare il primo inventario** (vedi [Inventario e Heartbeat](./inventory-heartbeat.md))
3. **Avviare il timer dell'heartbeat** (consigliato: ogni 5 minuti)
4. **Monitorare i fallimenti di autenticazione**

## Ri-Registrazione

### Il System Key è Usa e Getta

La registrazione non si può ripetere né annullare. Una volta che `registered_at` è valorizzato:

- `POST /backend/api/systems/register` risponde **409** per quel sistema, per sempre
- Anche **Rigenera Secret** risponde **409** -- il secret è ormai la credenziale viva con cui l'appliance si autentica su collect

Una macchina che ha bisogno di nuovo di una sottoscrizione riceve un **nuovo sistema**, con il proprio `system_secret`. La riga vecchia resta come traccia di una chiave spesa, finché un amministratore non la elimina.

### Cosa Non Influisce sulla Registrazione

La registrazione sopravvive, e non va rifatto nulla, a:

- **Riavvio del sistema**
- **Cambi di rete**
- **Aggiornamenti software**

## Sicurezza

### Protezione del Token

**Buone pratiche:**
- Conserva i token in configurazioni cifrate
- Non scrivere mai i token in chiaro nei log
- Usa canali sicuri (solo HTTPS)
- Ruota i secret periodicamente
- Revoca immediatamente i secret compromessi

### Flusso di Autenticazione

**Come funziona:**
1. Il sistema esterno divide il `system_secret` in parte pubblica e parte secret
2. La piattaforma interroga il database con la parte pubblica (ricerca indicizzata e veloce)
3. La piattaforma verifica la parte secret con uno SHA256 salato
4. La piattaforma mette in cache il risultato su Redis (TTL 24h con jitter)

**Vantaggi di sicurezza:**
- Query veloci sul database (parte pubblica indicizzata)
- Hashing SHA256 salato (salt unico per sistema)
- Consumo di memoria e CPU trascurabile
- Pattern standard del settore (GitHub, Stripe e Slack usano lo stesso schema)

### Sicurezza di Rete

**Requisiti:**
- Usa sempre HTTPS per la registrazione
- Verifica i certificati SSL/TLS
- Usa una risoluzione DNS sicura
- Evita il Wi-Fi pubblico per la registrazione iniziale

## Risoluzione Problemi

### La Registrazione Fallisce con un Errore di Rete

**Problema:** non si riesce a raggiungere l'endpoint di registrazione

**Soluzioni:**
1. Verifica la connettività di rete: `ping my.nethesis.it`
2. Verifica la risoluzione DNS: `nslookup my.nethesis.it`
3. Verifica la connettività HTTPS: `curl https://my.nethesis.it/backend/api/health`
4. Controlla le regole del firewall (consenti HTTPS in uscita)
5. Verifica le impostazioni proxy, se sei dietro un proxy aziendale

### La Registrazione Riesce ma il system_key Non Si Vede

**Problema:** la risposta indica successo ma il pannello di amministrazione non mostra il system_key

**Soluzioni:**
1. Ricarica la pagina di amministrazione (Ctrl+F5)
2. Svuota la cache del browser
3. Aspetta 30 secondi e ricarica (propagazione della cache)
4. Prova con un browser diverso
5. Verifica di stare guardando il sistema giusto

### system_secret Perso Prima della Registrazione

**Problema:** il sistema è stato creato ma il secret non è stato salvato, e il sistema non si è ancora registrato

**Soluzioni:**
1. Genera un nuovo secret: clicca **Rigenera Secret** nel pannello di amministrazione
2. Copia subito il nuovo secret
3. Configura il sistema esterno con il nuovo secret
4. Procedi con la registrazione

### system_secret Perso Dopo la Registrazione

**Problema:** il sistema è registrato ma il secret è andato perso

**Soluzioni:**
1. Se il sistema funziona: non fare nulla, le credenziali sono già salvate sulla macchina
2. Il secret non è recuperabile, e non è nemmeno rigenerabile -- **Rigenera Secret** risponde HTTP 409 su un sistema registrato
3. Se la macchina va riconfigurata da zero, crea un nuovo sistema, registralo con il suo nuovo secret, poi elimina il vecchio

### Registrazione con il Secret Sbagliato

**Problema:** registrato per errore con il secret di un altro sistema

**Soluzioni:**
1. Non è possibile: ogni secret è unico per sistema
2. La piattaforma verifica che la parte pubblica corrisponda al record del sistema
3. La registrazione fallisce se si usa il secret di un altro sistema

### Il Sistema Risulta Registrato ma Non Riesce ad Autenticarsi

**Problema:** la registrazione è andata a buon fine ma inventario/heartbeat falliscono con 401

**Soluzioni:**
1. Verifica che entrambe le credenziali siano salvate correttamente:
   - `system_key` (dalla risposta di registrazione)
   - `system_secret` (quello originale della creazione)
2. Controlla il formato dell'header HTTP Basic Auth
3. Prova l'autenticazione a mano (vedi [Inventario e Heartbeat](./inventory-heartbeat.md))
4. Verifica che non ci siano spazi di troppo nelle credenziali salvate
5. Controlla se il secret è stato rigenerato dopo la registrazione

## Argomenti Avanzati

### Registrazione Automatizzata

Per i deployment automatizzati, la registrazione si può scriptare:

**Esempio di script Bash:**
```bash
#!/bin/bash

PLATFORM_URL="https://my.nethesis.it"
SYSTEM_SECRET="my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

# Registra ed estrai il system_key
response=$(curl -s -X POST "$PLATFORM_URL/backend/api/systems/register" \
  -H "Content-Type: application/json" \
  -d "{\"system_secret\": \"$SYSTEM_SECRET\"}")

system_key=$(echo "$response" | jq -r '.data.system_key')

if [ "$system_key" != "null" ] && [ -n "$system_key" ]; then
  echo "Registrazione riuscita!"
  echo "system_key: $system_key"

  # Salva le credenziali
  echo "MY_SYSTEM_KEY=$system_key" >> /etc/my/config.conf
  echo "MY_SYSTEM_SECRET=$SYSTEM_SECRET" >> /etc/my/config.conf

  # Avvia il servizio di inventario/heartbeat
  systemctl start my-agent
else
  echo "Registrazione fallita!"
  echo "$response"
  exit 1
fi
```

### Registrazioni Multiple (Errore)

**Domanda:** cosa succede se registro lo stesso sistema più volte?

**Risposta:** dal secondo tentativo in poi la registrazione fallisce con HTTP 409 (già registrato). È voluto, per evitare ri-registrazioni accidentali.

### Annullare la Registrazione di un Sistema

**Domanda:** come annullo la registrazione di un sistema?

**Risposta:** è l'appliance stessa a poter rinunciare alle proprie credenziali, con una chiamata autenticata verso collect:

```bash
curl -X POST https://my.nethesis.it/collect/api/systems/unregister \
  -u "NOC-F64B-...:my_a1b2...c9d0"
```

L'operazione è **terminale e a senso unico**. Da quel momento la coppia di credenziali viene rifiutata ovunque -- heartbeat, inventario, backup, proxy degli allarmi e feed enterprise -- e solo la prima chiamata risponde 200, perché la revoca stessa fa fallire l'autenticazione a ogni richiesta successiva con le stesse credenziali. Il sistema non può essere registrato di nuovo: resta sulla piattaforma, marcato `unregistered`, finché un amministratore non lo elimina.

Per rimettere la stessa macchina sotto sottoscrizione, crea un nuovo sistema e registralo con il nuovo `system_secret`.

## Prossimi Passi

Dopo una registrazione andata a buon fine:

- [Configura la raccolta dell'inventario](./inventory-heartbeat.md)
- Imposta il monitoraggio heartbeat
- Verifica l'autenticazione
- Monitora lo stato del sistema dalla dashboard

## Documentazione Correlata

- [Gestione Sistemi](./management.md)
- [Inventario e Heartbeat](./inventory-heartbeat.md)
- [Documentazione API Backend](https://github.com/NethServer/my/blob/main/backend/README.md)
