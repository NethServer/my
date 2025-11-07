# Registrazione Sistema

Scopri come i sistemi esterni si registrano con la piattaforma My per abilitare monitoraggio e gestione.

## Panoramica

La registrazione del sistema è il processo mediante il quale un sistema esterno (NethServer, NethSecurity, ecc.) si autentica con My e riceve le sue credenziali permanenti.

### Perché è Necessaria la Registrazione

- **Sicurezza**: Valida il sistema prima di consentire la trasmissione dati
- **Autenticazione**: Stabilisce credenziali a lungo termine
- **Tracciamento**: Registra quando un sistema si è connesso per la prima volta
- **Visibilità**: Rende system_key visibile agli amministratori

### Flusso di Registrazione

```
┌─────────────┐                                        ┌──────────────┐
│             │  1. Crea sistema                       │              │
│    Admin    │──────────────────────────────────────> │      My      │
│             │  ← Restituisce system_secret (1 volta) │   Platform   │
└─────────────┘                                        └──────────────┘
                                                           │
                                                           │
┌─────────────┐                                            │
│   Sistema   │  2. Configura system_secret                │
│   Esterno   │<───────────────────────────────────────────┘
│ (NethServer)│
└─────────────┘
      │
      │  3. Chiama API registrazione
      │     POST /api/systems/register
      │     { "system_secret": "my_..." }
      │
      v
┌──────────────┐
│      My      │  4. Valida secret
│   Platform   │     ✓ Formato corretto
│              │     ✓ Parte pubblica esiste
│              │     ✓ Parte secret verificata (Argon2id)
│              │     ✓ Non eliminato
│              │     ✓ Non già registrato
└──────────────┘
      │
      │  5. Restituisce system_key
      v
┌─────────────┐
│   Sistema   │  6. Memorizza credenziali:
│   Esterno   │     - system_key (username)
│             │     - system_secret (password)
└─────────────┘
      │
      │  7. Pronto per inventario e heartbeat!
      v
```

## Comprendere le Credenziali

### system_secret (Creato alla Creazione Sistema)

**Formato:** `my_<parte_pubblica>.<parte_secret>`

**Esempio:** `my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0`

**Componenti:**
- **Prefisso**: `my_` (identifica il tipo di token)
- **Parte pubblica**: 20 caratteri esadecimali (per ricerca nel database)
- **Separatore**: `.` (punto)
- **Parte secret**: 40 caratteri esadecimali (hash con Argon2id)

**Caratteristiche:**
- Mostrato **solo una volta** durante la creazione del sistema
- Non può essere recuperato successivamente (la rigenerazione ne crea uno nuovo)
- Usato per la registrazione (una sola volta)
- Usato per tutta l'autenticazione futura (inventario, heartbeat)

### system_key (Ricevuto alla Registrazione)

**Formato:** `NOC-<stringa_casuale>`

**Esempio:** `NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE`

**Caratteristiche:**
- Generato durante la creazione del sistema
- Nascosto fino alla registrazione del sistema
- Visibile dopo registrazione riuscita
- Usato come username per HTTP Basic Auth
- Non cambia mai (anche se il secret viene rigenerato)

## Processo di Registrazione

### Passo 1: Admin Crea Sistema

Vedi [Gestione Sistemi](04-systems.md#creazione-sistemi) per i dettagli.

Dopo la creazione, salva il `system_secret`:
```json
{
  "system_secret": "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
}
```

### Passo 2: Configura Sistema Esterno

Configura il sistema esterno con il `system_secret`. Il metodo esatto dipende dal tipo di sistema:

#### Per NethServer/NethSecurity:

1. Accedi all'interfaccia di amministrazione del sistema
2. Naviga su **Impostazioni** > **Sottoscrizione**
3. Incolla il `system_secret`
4. Clicca **Registra**

#### Per Sistemi Personalizzati (API):

Memorizza il secret in modo sicuro nella tua applicazione:

**Esempio file di configurazione:**
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

### Passo 3: Chiama API Registrazione

Il sistema esterno effettua una richiesta POST per registrarsi:

**Endpoint:** `POST https://my.nethesis.it/api/systems/register`

**Headers:**
```
Content-Type: application/json
```

**Request Body:**
```json
{
  "system_secret": "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
}
```

**Esempio cURL:**
```bash
curl -X POST https://my.nethesis.it/api/systems/register \
  -H "Content-Type: application/json" \
  -d '{
    "system_secret": "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
  }'
```

**Esempio Python:**
```python
import requests

url = "https://my.nethesis.it/api/systems/register"
payload = {
    "system_secret": "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
}

response = requests.post(url, json=payload)
data = response.json()
system_key = data["data"]["system_key"]
print(f"Registrato! system_key: {system_key}")
```

### Passo 4: Piattaforma Valida

La piattaforma esegue diversi controlli di sicurezza:

1. **Validazione Formato Token**:
   - Divide su `.` → deve avere esattamente 2 parti
   - La prima parte deve iniziare con `my_`
   - Estrae parti pubblica e secret

2. **Ricerca Database**:
   - Trova il sistema usando la parte pubblica
   - Query indicizzata veloce su `system_secret_public`

3. **Controlli di Sicurezza**:
   - Il sistema non è eliminato
   - Il sistema non è già registrato
   - La parte pubblica corrisponde al valore memorizzato

4. **Verifica Crittografica**:
   - Verifica la parte secret contro l'hash Argon2id
   - Confronto a tempo costante (previene attacchi timing)

### Passo 5: Registrazione Riuscita

**Risposta di Successo (HTTP 200):**
```json
{
  "code": 200,
  "message": "sistema registrato con successo",
  "data": {
    "system_key": "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE",
    "registered_at": "2025-11-06T10:30:00Z",
    "message": "sistema registrato con successo"
  }
}
```

**Cosa accade:**
- Il timestamp `registered_at` viene registrato nel database
- `system_key` diventa visibile agli amministratori
- Il sistema può ora autenticarsi per inventario e heartbeat

### Passo 6: Memorizza Credenziali

Il sistema esterno deve memorizzare in modo sicuro entrambe le credenziali:

**Necessarie per l'autenticazione futura:**
- `system_key`: Username per HTTP Basic Auth
- `system_secret`: Password per HTTP Basic Auth

**Raccomandazioni di memorizzazione:**
```bash
# File di configurazione
MY_SYSTEM_KEY=NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE
MY_SYSTEM_SECRET=my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0

# O usa memorizzazione sicura:
# - Portachiavi di sistema
# - Configurazione crittografata
# - Servizio di gestione secret (Vault, ecc.)
```

## Risposte di Errore

### Formato Token Non Valido

**HTTP 400 Bad Request:**
```json
{
  "code": 400,
  "message": "formato system secret non valido",
  "data": null
}
```

**Cause:**
- Il token non contiene il separatore `.`
- Il token non inizia con `my_`
- Il token è malformato

**Soluzione:**
- Verifica che il secret sia stato copiato correttamente
- Controlla spazi extra o interruzioni di riga
- Assicurati che sia fornito il token completo

### Credenziali Non Valide

**HTTP 401 Unauthorized:**
```json
{
  "code": 401,
  "message": "system secret non valido",
  "data": null
}
```

**Cause:**
- Parte pubblica non trovata nel database
- Parte secret non corrisponde all'hash
- Secret sbagliato fornito

**Soluzione:**
- Verifica che il secret sia corretto
- Controlla se il secret è stato rigenerato
- Assicurati che il sistema sia stato creato nella piattaforma My

### Sistema Eliminato

**HTTP 403 Forbidden:**
```json
{
  "code": 403,
  "message": "il sistema è stato eliminato",
  "data": null
}
```

**Cause:**
- Il sistema è stato eliminato in modo soft dall'amministratore
- Sistema contrassegnato come eliminato nel database

**Soluzione:**
- Contatta l'amministratore per ripristinare il sistema
- Crea un nuovo sistema se necessario

### Già Registrato

**HTTP 409 Conflict:**
```json
{
  "code": 409,
  "message": "il sistema è già registrato",
  "data": null
}
```

**Cause:**
- Il sistema ha già completato la registrazione
- Il campo `registered_at` non è null

**Soluzione:**
- Il sistema è già registrato, procedi con l'autenticazione
- Usa `system_key` e `system_secret` esistenti per l'autenticazione
- Nessuna azione necessaria a meno che non sia richiesta una ri-registrazione

## Dopo la Registrazione

### Visualizza Stato Registrazione

Gli amministratori possono visualizzare lo stato di registrazione:

1. Naviga su **Sistemi**
2. Trova il sistema e clicca **Visualizza dettagli**
3. Controlla i campi:
   - **System_key**: Ora visibile (era nascosto prima)
   - **Sottoscrizione**: Mostra il timestamp
   - **Stato**: Potrebbe ancora essere "unknown" fino al primo inventario

### Prossimi Passi per Sistema Esterno

Dopo la registrazione riuscita, il sistema dovrebbe:

1. **Memorizzare le credenziali in modo sicuro**
2. **Inviare il primo inventario** (vedi [Inventario e Heartbeat](06-inventory-heartbeat.md))
3. **Avviare il timer heartbeat** (raccomandato: ogni 5 minuti)
4. **Monitorare i fallimenti di autenticazione**

## Ri-registrazione

### Quando è Necessaria la Ri-registrazione?

La ri-registrazione **NON** è tipicamente necessaria. Un sistema rimane registrato a meno che:

- Il sistema venga eliminato e ricreato (nuovo system_secret)
- L'amministratore resetti esplicitamente la registrazione (operazione manuale del database)

### Quando NON è Necessaria la Ri-registrazione?

- **Rigenerazione secret**: Il sistema rimane registrato, usa solo il nuovo secret
- **Riavvio sistema**: La registrazione persiste
- **Modifiche di rete**: La registrazione persiste
- **Aggiornamenti software**: La registrazione persiste

## Considerazioni sulla Sicurezza

### Sicurezza Token

**Best Practice:**
- Memorizza i token in configurazione crittografata
- Non loggare mai i token in chiaro
- Usa canali sicuri (solo HTTPS)
- Ruota i secret periodicamente
- Revoca immediatamente i secret compromessi

### Flusso di Autenticazione

**Come funziona:**
1. Il sistema esterno divide `system_secret` in parti pubblica + secret
2. La piattaforma interroga il database usando la parte pubblica (ricerca indicizzata veloce)
3. La piattaforma verifica la parte secret usando Argon2id (memory-hard, resistente GPU)
4. La piattaforma memorizza in cache il risultato in Redis (TTL 5 minuti)

**Benefici di sicurezza:**
- Query database veloci (parte pubblica indicizzata)
- Crittografia forte (Argon2id: 64MB memoria, 3 iterazioni)
- Resistente al brute-force (algoritmo memory-hard)
- Pattern standard dell'industria (GitHub, Stripe, Slack usano pattern simili)

### Sicurezza di Rete

**Requisiti:**
- Usa sempre HTTPS per la registrazione
- Verifica certificati SSL/TLS
- Usa risoluzione DNS sicura
- Evita Wi-Fi pubblico per la registrazione iniziale

## Risoluzione Problemi

### Registrazione Fallisce con Errore di Rete

**Problema:** Impossibile connettersi all'endpoint di registrazione

**Soluzioni:**
1. Controlla connettività di rete: `ping my.nethesis.it`
2. Verifica risoluzione DNS: `nslookup my.nethesis.it`
3. Testa connettività HTTPS: `curl https://my.nethesis.it/api/health`
4. Controlla regole firewall (consenti HTTPS in uscita)
5. Verifica impostazioni proxy se dietro proxy aziendale

### Registrazione Riesce ma system_key Non Visibile

**Problema:** La risposta di registrazione mostra successo ma il pannello admin non mostra system_key

**Soluzioni:**
1. Aggiorna la pagina admin (Ctrl+F5)
2. Cancella cache del browser
3. Attendi 30 secondi e aggiorna (propagazione cache)
4. Controlla browser diverso
5. Verifica che stai visualizzando il sistema corretto

### system_secret Perso Prima della Registrazione

**Problema:** Il sistema è stato creato ma il secret non è stato salvato, sistema non ancora registrato

**Soluzioni:**
1. Genera nuovo secret: Clicca **Rigenera Secret** nel pannello admin
2. Copia immediatamente il nuovo secret
3. Configura il sistema esterno con il nuovo secret
4. Procedi con la registrazione

### system_secret Perso Dopo la Registrazione

**Problema:** Il sistema è registrato ma il secret è stato perso

**Soluzioni:**
1. Se il sistema funziona: Non fare nulla, le credenziali sono memorizzate sul sistema
2. Se serve riconfigurare: Rigenera secret nel pannello admin
3. Aggiorna secret sul sistema esterno
4. Il sistema rimane registrato (nessuna ri-registrazione necessaria)

### Registrazione con Secret Sbagliato

**Problema:** Registrato accidentalmente con il secret del sistema sbagliato

**Soluzioni:**
1. Questo è impossibile - ogni secret è unico per sistema
2. La piattaforma valida che la parte pubblica corrisponda al record del sistema
3. La registrazione fallirà se si usa il secret del sistema sbagliato

### Sistema Mostra come Registrato ma Non Può Autenticarsi

**Problema:** La registrazione è riuscita ma inventario/heartbeat fallisce con 401

**Soluzioni:**
1. Verifica che entrambe le credenziali siano memorizzate correttamente:
   - `system_key` (dalla risposta di registrazione)
   - `system_secret` (originale dalla creazione)
2. Controlla formato header HTTP Basic Auth
3. Testa l'autenticazione manualmente (vedi [Inventario e Heartbeat](06-inventory-heartbeat.md))
4. Verifica nessuno spazio extra nelle credenziali memorizzate
5. Controlla se il secret è stato rigenerato dopo la registrazione

## Argomenti Avanzati

### Registrazione Automatizzata

Per deployment automatizzati, la registrazione può essere scriptata:

**Esempio script Bash:**
```bash
#!/bin/bash

PLATFORM_URL="https://my.nethesis.it"
SYSTEM_SECRET="my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

# Registra ed estrai system_key
response=$(curl -s -X POST "$PLATFORM_URL/api/systems/register" \
  -H "Content-Type: application/json" \
  -d "{\"system_secret\": \"$SYSTEM_SECRET\"}")

system_key=$(echo "$response" | jq -r '.data.system_key')

if [ "$system_key" != "null" ] && [ -n "$system_key" ]; then
  echo "Registrazione riuscita!"
  echo "system_key: $system_key"

  # Memorizza credenziali
  echo "MY_SYSTEM_KEY=$system_key" >> /etc/my/config.conf
  echo "MY_SYSTEM_SECRET=$SYSTEM_SECRET" >> /etc/my/config.conf

  # Avvia servizio inventario/heartbeat
  systemctl start my-agent
else
  echo "Registrazione fallita!"
  echo "$response"
  exit 1
fi
```

### Registrazioni Multiple (Errore)

**Domanda:** Cosa succede se registro lo stesso sistema più volte?

**Risposta:** Il secondo e successivi tentativi di registrazione falliranno con HTTP 409 (già registrato). Questo è per design per prevenire ri-registrazione accidentale.

### Annullare Registrazione Sistema

**Domanda:** Come annullo la registrazione di un sistema?

**Risposta:** Non esiste un'operazione "annulla registrazione". Per resettare:
1. Elimina il sistema (eliminazione soft)
2. Ripristina il sistema
3. Il sistema rimane registrato con lo stesso `system_key`
4. Rigenera secret se necessario

Oppure:
1. Elimina sistema (eliminazione soft)
2. Elimina permanentemente sistema
3. Crea nuovo sistema (nuove credenziali, nuova registrazione)

## Prossimi Passi

Dopo la registrazione riuscita:

- [Configura la raccolta inventario](06-inventory-heartbeat.md)
- Configura monitoraggio heartbeat
- Testa l'autenticazione
- Monitora lo stato del sistema nella dashboard

## Documentazione Correlata

- [Gestione Sistemi](04-systems.md)
- [Inventario e Heartbeat](06-inventory-heartbeat.md)
- [Documentazione API Backend](https://github.com/NethServer/my/blob/main/backend/README.md)
