---
sidebar_position: 6
---

# Importazione Dati

Importa dati in blocco da file CSV per creare organizzazioni e utenti.

## Panoramica

My consente di importare distributori, rivenditori, clienti e utenti da file CSV. Il processo di importazione utilizza un flusso a due fasi: prima la **validazione**, poi la **conferma**. Questo garantisce la possibilita' di verificare e correggere eventuali problemi prima della creazione dei dati.

## Import Supportati

| Risorsa | Descrizione |
|---------|-------------|
| **Distributori** | Nome, P.IVA, dettagli contatto |
| **Rivenditori** | Nome, P.IVA, dettagli contatto |
| **Clienti** | Nome, P.IVA, dettagli contatto |
| **Utenti** | Email, nome, organizzazione, ruoli |

## Come Importare

### Passo 1: Scarica il Template

Clicca sul pulsante **Importa** e seleziona **Scarica Template**. Il template CSV contiene le intestazioni corrette e righe di esempio.

### Passo 2: Compila il CSV

Apri il template in un foglio di calcolo e inserisci i dati. Ogni riga rappresenta un'entita' da creare.

:::tip
Mantieni la riga di intestazione esattamente come fornita. Non rinominare, riordinare o rimuovere colonne.
:::

### Passo 3: Carica e Valida

Carica il file CSV. Il sistema valida ogni riga e restituisce un report dettagliato:

- Le righe **valide** sono pronte per l'importazione
- Le righe con **errori** hanno problemi a livello di campo (es. campi obbligatori mancanti, formato email non valido)
- Le righe **duplicate** corrispondono a record gia' esistenti nel sistema

Esamina il report di validazione prima di procedere.

### Passo 4: Conferma l'Import

Una volta soddisfatti dei risultati della validazione, conferma l'importazione. Solo le righe valide vengono create. Le righe con errori e duplicati vengono saltate automaticamente. E' possibile escludere manualmente righe valide specifiche prima della conferma.

### Passo 5: Verifica i Risultati

Dopo la conferma, un riepilogo mostra quanti record sono stati creati, saltati o falliti. Per le righe fallite durante la creazione, i dettagli dell'errore vengono forniti per ciascuna.

## Formato CSV

### Colonne Organizzazione (Distributori, Rivenditori, Clienti)

| Colonna | Obbligatorio | Descrizione |
|---------|-------------|-------------|
| `name` | Si' | Nome organizzazione (max 255 caratteri) |
| `description` | No | Descrizione |
| `vat` | Si' | Partita IVA |
| `address` | No | Indirizzo |
| `city` | No | Citta' |
| `main_contact` | No | Contatto principale |
| `email` | No | Email di contatto (formato valido se presente) |
| `phone` | No | Telefono (formato internazionale se presente) |
| `language` | No | Codice lingua: `it` o `en` (default: `it`) |
| `notes` | No | Note aggiuntive |

### Colonne Utente

| Colonna | Obbligatorio | Descrizione |
|---------|-------------|-------------|
| `email` | Si' | Email utente (deve essere univoca) |
| `name` | Si' | Nome completo (max 255 caratteri) |
| `phone` | No | Telefono (formato internazionale se presente) |
| `organization` | Si' | Nome organizzazione (deve esistere nella propria gerarchia) |
| `roles` | Si' | Nomi dei ruoli separati da `;` (es. `Admin;Support`) |

:::note
Nell'importazione utenti, l'organizzazione viene cercata **per nome** all'interno della gerarchia visibile. Se il nome dell'organizzazione non esiste o e' fuori dalla gerarchia, la riga viene segnalata come errore.
:::

## Regole di Validazione

I seguenti controlli vengono eseguiti durante la validazione:

- **Campi obbligatori** -- I campi contrassegnati non possono essere vuoti
- **Validazione formato** -- Indirizzi email, numeri di telefono e codici lingua vengono verificati
- **Rilevamento duplicati (nel CSV)** -- Nomi (organizzazioni) o email (utenti) duplicati nello stesso file vengono segnalati
- **Rilevamento duplicati (database)** -- Nomi o email gia' esistenti nel sistema vengono segnalati
- **Risoluzione organizzazione** -- Per l'import utenti, i nomi delle organizzazioni vengono risolti tra le organizzazioni esistenti nella propria gerarchia
- **Risoluzione ruoli** -- Per l'import utenti, i nomi dei ruoli vengono verificati tra i ruoli disponibili
- **Controllo permessi** -- Per l'import utenti, ogni riga viene verificata rispetto ai permessi RBAC

## Limiti

| Parametro | Valore |
|-----------|--------|
| Righe massime per file CSV | 1.000 |
| Dimensione massima file | 10 MB |
| Codifiche supportate | UTF-8, UTF-8 con BOM, Latin-1 |

## Permessi

La disponibilita' dell'importazione dipende dal ruolo organizzativo e dai permessi.

### Import Organizzazioni

| Risorsa | Chi Puo' Importare |
|---------|--------------------|
| Distributori | Owner |
| Rivenditori | Owner, Distributore |
| Clienti | Owner, Distributore, Rivenditore |

### Import Utenti

L'import utenti richiede il permesso `manage:users`. Il campo organizzazione in ogni riga del CSV viene validato rispetto alla propria gerarchia -- e' possibile importare utenti solo nelle organizzazioni gestite.

:::warning
La sessione di importazione scade dopo **30 minuti**. Se non si conferma entro questo tempo, e' necessario ricaricare e rivalidare il file CSV.
:::

## Email di Benvenuto

Quando si importano utenti, il sistema invia automaticamente un'email di benvenuto a ogni utente creato con la password temporanea e le istruzioni di accesso. Viene utilizzato lo stesso flusso email della creazione singola.
