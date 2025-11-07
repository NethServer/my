# Contribuire alla Documentazione My

Grazie per il tuo interesse nel migliorare la documentazione My!

## Struttura Documentazione

```
docs/
├── index.md                    # Home page
├── 01-authentication.md        # Guida autenticazione
├── 02-organizations.md         # Guida organizzazioni
├── 03-users.md                 # Guida gestione utenti
├── 04-systems.md               # Guida gestione sistemi
├── 05-system-registration.md   # Flusso di lavoro registrazione
├── 06-inventory-heartbeat.md   # Guida monitoraggio
├── 07-impersonation.md         # Guida impersonificazione utente
├── stylesheets/                # CSS personalizzato
├── javascripts/                # JS personalizzato
└── images/                     # Immagini e diagrammi
```

## Linee Guida Scrittura

### Guida Stile

- **Tono**: Chiaro, professionale, utile
- **Pubblico**: Utenti finali e amministratori (non tecnici)
- **Linguaggio**: Semplice, evitando gergo quando possibile
- **Esempi**: Includere sempre esempi pratici

### Formattazione

- **Intestazioni**: Usa `##` per sezioni principali, `###` per sottosezioni
- **Blocchi codice**: Specifica sempre il linguaggio (bash, json, python, ecc.)
- **Liste**: Usa `-` per liste non ordinate, `1.` per ordinate
- **Enfasi**: Usa **grassetto** per termini importanti, *corsivo* per enfasi
- **Link**: Usa testo descrittivo, non "clicca qui"

### Esempio Struttura

```markdown
# Titolo Pagina

Breve introduzione che spiega cosa copre questa pagina.

## Sezione Principale

Spiegazione dettagliata con esempi.

### Sottosezione

Dettagli o procedure specifiche.

**Esempio:**
\`\`\`bash
comando --flag valore
\`\`\`

## Risoluzione Problemi

Problemi comuni e soluzioni.

## Documentazione Correlata

- [Link a pagina correlata](altra-pagina.md)
```

## Build Locale

### Prerequisiti

```bash
# Installa Python 3.x
python3 --version

# Installa dipendenze
pip install -r requirements.txt
```

### Sviluppo Locale

```bash
# Avvia server locale con hot reload
mkdocs serve

# Apri nel browser
open http://localhost:8000
```

La documentazione si ricaricherà automaticamente quando salvi le modifiche.

### Building

```bash
# Costruisci sito statico
mkdocs build

# L'output sarà nella directory site/
```

## Apportare Modifiche

### 1. Modifica Documentazione

Modifica il file `.md` rilevante nella directory `docs/`.

### 2. Anteprima Locale

```bash
mkdocs serve
```

Controlla le tue modifiche su http://localhost:8000

### 3. Controlla Link

Assicurati che tutti i link interni funzionino:
- Link relativi ad altri doc: `[testo](altro-file.md)`
- Link a sezioni: `[testo](altro-file.md#nome-sezione)`
- Link esterni: URL completo

### 4. Aggiungi Immagini

Se aggiungi immagini:

1. Posiziona l'immagine in `docs/images/`
2. Usa percorso relativo: `![Testo alternativo](images/nomefile.png)`
3. Ottimizza dimensione immagine (max 1MB)

### 5. Testa Build

```bash
# Testa che il build abbia successo
mkdocs build --strict

# Questo fallirà se ci sono avvisi
```

## Aggiungere Nuove Pagine

### 1. Crea File

Crea nuovo file `.md` nella directory `docs/`:

```bash
touch docs/07-nuova-funzionalita.md
```

### 2. Aggiungi alla Navigazione

Modifica `mkdocs.yml` e aggiungi alla navigazione:

```yaml
nav:
  - Guida Utente:
      - Nuova Funzionalità: 07-nuova-funzionalita.md
```

### 3. Link da Altre Pagine

Aggiungi link da pagine rilevanti:

```markdown
Vedi anche: [Guida Nuova Funzionalità](07-nuova-funzionalita.md)
```

## Funzionalità MkDocs

### Admonitions

```markdown
!!! note "Titolo"
    Contenuto qui

!!! warning
    Contenuto avviso

!!! danger
    Contenuto pericolo

!!! tip
    Contenuto suggerimento
```

### Blocchi Codice con Highlighting

````markdown
```python title="esempio.py" linenums="1"
def hello():
    print("Ciao, Mondo!")
```
````

### Tab

```markdown
=== "Tab 1"
    Contenuto per tab 1

=== "Tab 2"
    Contenuto per tab 2
```

### Task List

```markdown
- [x] Task completato
- [ ] Task incompleto
```

## Deployment

La documentazione viene deployata automaticamente quando fai push su `main`:

1. GitHub Actions viene eseguita al push
2. MkDocs costruisce il sito
3. Il sito viene deployato su GitHub Pages
4. Disponibile su: https://nethesis.github.io/my/

Puoi anche deployare manualmente:

```bash
mkdocs gh-deploy
```

## Processo di Revisione

1. Apporta le tue modifiche in un branch feature
2. Testa localmente con `mkdocs serve`
3. Assicurati che il build passi: `mkdocs build --strict`
4. Crea Pull Request
5. La documentazione verrà revisionata
6. Una volta approvata, merge su main
7. Deployment automatico su GitHub Pages

## Task Comuni

### Aggiorna Home Page

Modifica `docs/index.md`

### Aggiungi Sezione FAQ

Crea `docs/faq.md` e aggiungi alla navigazione in `mkdocs.yml`

### Correggi Link Non Funzionante

1. Cerca il link: `grep -r "link-non-funzionante" docs/`
2. Aggiorna tutte le occorrenze
3. Testa con `mkdocs serve`

### Aggiungi Risorsa Esterna

Aggiungi a `mkdocs.yml` sotto `extra`:

```yaml
extra:
  social:
    - icon: fontawesome/brands/github
      link: https://github.com/example
```

## Stile e Convenzioni

### Esempi Comandi

Mostra sempre comandi completi:

```bash
# Buono
curl -X POST https://api.example.com/endpoint \
  -H "Content-Type: application/json" \
  -d '{"key": "value"}'

# Cattivo
curl endpoint
```

### Percorsi File

Usa percorsi assoluti dalla radice del progetto:

```bash
# Buono
/Users/edospadoni/Workspace/my/backend/main.go

# Cattivo
../backend/main.go
```

### Esempi API

Mostra sia richiesta che risposta:

```bash
# Richiesta
curl -X GET https://api.example.com/resource

# Risposta (HTTP 200)
{
  "code": 200,
  "message": "successo",
  "data": {}
}
```

## Ottenere Aiuto

- Controlla la documentazione esistente per esempi
- Rivedi [documentazione MkDocs](https://www.mkdocs.org/)
- Controlla [documentazione tema Material](https://squidfunk.github.io/mkdocs-material/)
- Chiedi nelle discussioni del progetto

## Licenza

I contributi alla documentazione sono coperti dalla stessa licenza del progetto (AGPL-3.0-or-later).
