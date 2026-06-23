---
sidebar_position: 3
---

# API Key

Crea API key personali per permettere ad applicazioni e script esterni di accedere a My per tuo conto, senza usare la tua password e senza passare dal login interattivo.

## Panoramica

Una API key è una credenziale a lunga durata legata al tuo account. Permette a integrazioni non interattive di autenticarsi alla piattaforma, ad esempio:

- Un CRM o un gestionale che legge o aggiorna i tuoi dati
- Script di monitoraggio, reportistica o automazione
- Pipeline CI/CD

Le key si gestiscono da **Impostazioni Account → API Key**.

:::warning
Tratta una API key come una password. Chi la possiede può agire al posto tuo, nei limiti del livello di accesso della key. Non inserire mai le key nel codice versionato e non condividerle via chat o email.
:::

## Livelli di accesso

Quando crei una key scegli cosa può fare:

| Accesso | Cosa può fare |
|--------|----------------|
| **Sola lettura** | Legge solo i tuoi dati |
| **Lettura e scrittura** | Legge e modifica i tuoi dati |

Una key non può mai fare più del tuo account e, indipendentemente dal livello di accesso, una key **non può**:

- Gestire le API key (creare o revocare key)
- Modificare profilo, password o altre impostazioni dell'account
- Impersonificare altri utenti
- Eseguire operazioni distruttive o modificare la configurazione degli alert

Se i permessi del tuo account cambiano, le key si adeguano automaticamente.

## Creare una API key

1. Apri **Impostazioni Account → API Key**
2. Clicca **Crea API key**
3. Compila il form:
   - **Nome** — un'etichetta per riconoscere dove viene usata la key (es. "CRM produzione")
   - **Accesso** — Sola lettura oppure Lettura e scrittura
   - **Scadenza (giorni)** — predefinito 90, massimo 365
   - **Conferma la password** — richiesta per creare la key
4. Clicca **Crea API key**

La key completa viene mostrata **una sola volta**:

```
myk_a1b2c3d4e5f607.0011223344556677889900aabbccddeeff00112233445566
```

:::danger
Copia subito la key e conservala in modo sicuro: viene mostrata una sola volta e non è più recuperabile. Se la perdi, revocala e creane una nuova.
:::

:::tip
Conserva le key in un password manager o in un gestore di segreti (come Vault), oppure passale tramite variabili d'ambiente. Usa **Sola lettura** quando l'integrazione deve solo leggere.
:::

Puoi avere al massimo **5 key attive** contemporaneamente. Revoca una key inutilizzata per liberare uno slot.

## Usare la API key

Invia la key come Bearer token nell'header `Authorization`.

**cURL:**

```bash
curl https://my.nethesis.it/api/systems \
  -H "Authorization: Bearer myk_a1b2c3d4e5f607.0011223344556677889900aabbccddeeff00112233445566"
```

**Python:**

```python
import os
import requests

headers = {"Authorization": f"Bearer {os.environ['MY_API_KEY']}"}
response = requests.get("https://my.nethesis.it/api/systems", headers=headers)
print(response.json())
```

Una key in sola lettura che chiama un'operazione di scrittura riceve `403 Forbidden`.

## Gestire le key

In **Impostazioni Account → API Key** vedi tutte le tue key con:

- **Nome** e prefisso della key (es. `myk_a1b2c3d4e5f6…`)
- **Accesso** (sola lettura oppure lettura e scrittura)
- Date di **Ultimo utilizzo** e **Scadenza**
- **Stato** (attiva, revocata o scaduta)

### Revocare una key

Per disattivare subito una key:

1. Trova la key nell'elenco
2. Clicca **Revoca**
3. Conferma

La revoca ha effetto immediato: ogni richiesta successiva con quella key riceve `401`. La revoca non è reversibile — crea una nuova key se ti serve ripristinare l'accesso.

## Ruotare una key

Per sostituire una key senza interruzioni:

1. Crea una nuova key e aggiorna l'integrazione per usarla
2. Verifica che l'integrazione funzioni con la nuova key
3. Revoca la vecchia key

## Cosa succede se il tuo account viene sospeso

Se il tuo account viene sospeso o cancellato, **tutte le tue API key smettono di funzionare** immediatamente. Se l'account viene riattivato, le key tornano a funzionare (a meno che nel frattempo siano state revocate o scadute).

## Sicurezza e limiti

- Le key vengono mostrate per intero **una sola volta**, alla creazione
- Massimo **5 key attive** per utente
- La scadenza è obbligatoria: predefinita **90 giorni**, massimo **365 giorni**
- Creare una key richiede di confermare la password
- Le richieste sono limitate per key; raffiche eccessive ricevono `429 Too Many Requests`
- L'attività delle key e gli eventi di sicurezza (creazione, revoca, autenticazioni fallite) vengono registrati per l'audit

## Risoluzione dei problemi

### 401 Unauthorized

**Problema:** le richieste ricevono `401` con messaggio "invalid api key".

**Soluzioni:**
- Verifica che la key sia copiata per intero, con il prefisso `myk_` e senza spazi o a capo aggiuntivi
- Controlla che il formato dell'header sia `Authorization: Bearer <key>`
- La key potrebbe essere revocata o scaduta — creane una nuova
- L'account potrebbe essere sospeso — contatta un amministratore

### 403 Forbidden

**Problema:** una richiesta riceve `403`.

**Soluzioni:**
- Una key in **sola lettura** non può eseguire operazioni di scrittura — crea una key in **lettura e scrittura** se necessario
- L'azione potrebbe non essere consentita alle API key (account, gestione key, impersonificazione, operazioni distruttive)

### Impossibile creare una key

- **"Password errata"** — reinserisci la password attuale dell'account
- **"Hai raggiunto il numero massimo di API key"** — revoca prima una key esistente (il limite è 5)

## Documentazione correlata

- [Impostazioni Account](account)
- [Autenticazione](authentication)
