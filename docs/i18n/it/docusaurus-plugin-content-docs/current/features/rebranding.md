---
sidebar_position: 4
---

# Rebranding Organizzazione

Il rebranding consente alle organizzazioni di personalizzare l'aspetto dei prodotti software con il proprio marchio, inclusi loghi, favicon e sfondi.

## Panoramica

La funzionalità di rebranding permette di:

- Personalizzare l'aspetto dei prodotti per ogni organizzazione
- Caricare loghi, favicon e sfondi personalizzati
- Gestire asset di rebranding per diversi prodotti
- Monitorare lo stato del rebranding per ogni organizzazione

## Abilitazione

Il rebranding deve essere abilitato dall'Owner per ogni organizzazione.

:::warning
Solo gli utenti con ruolo Owner possono abilitare o disabilitare il rebranding per un'organizzazione.
:::

Per abilitare il rebranding:

1. Vai alla pagina di dettaglio dell'organizzazione
2. Nella sezione **Rebranding**, attiva la funzionalità
3. Seleziona i prodotti per cui abilitare il rebranding
4. Salva le modifiche

## Prodotti Supportati

Il rebranding è disponibile per i seguenti prodotti:

- **NethServer** - Server Linux per piccole e medie imprese
- **NethSecurity** - Soluzione di sicurezza di rete
- **NethVoice** - Sistema di comunicazione unificata

Ogni prodotto ha i propri requisiti per gli asset di rebranding.

## Tipi di Asset

### Logo

Il logo principale dell'organizzazione, utilizzato nell'intestazione e nelle pagine di login.

| Parametro | Valore |
|-----------|--------|
| Formati supportati | PNG, SVG |
| Dimensione massima | 500 KB |
| Dimensioni consigliate | Variabile per prodotto |
| Sfondo | Trasparente consigliato |

### Logo Chiaro

Variante del logo per sfondi scuri.

| Parametro | Valore |
|-----------|--------|
| Formati supportati | PNG, SVG |
| Dimensione massima | 500 KB |
| Uso | Modalità scura, intestazioni con sfondo scuro |

### Favicon

L'icona visualizzata nella barra del browser.

| Parametro | Valore |
|-----------|--------|
| Formati supportati | PNG, ICO |
| Dimensione massima | 100 KB |
| Dimensioni consigliate | 32x32 pixel o 64x64 pixel |

### Sfondo Login

L'immagine di sfondo per la pagina di login.

| Parametro | Valore |
|-----------|--------|
| Formati supportati | PNG, JPEG, WebP |
| Dimensione massima | 2 MB |
| Dimensioni consigliate | 1920x1080 pixel |

## Gestione Asset

### Upload

Per caricare un asset di rebranding:

1. Vai alla pagina di dettaglio dell'organizzazione
2. Nella sezione **Rebranding**, seleziona il prodotto
3. Clicca sull'area dell'asset da caricare (logo, favicon, sfondo)
4. Seleziona il file dal dispositivo
5. L'asset viene caricato e applicato

:::tip
Per risultati ottimali:
- Usa immagini con sfondo trasparente per i loghi (formato PNG o SVG)
- Testa l'aspetto sia in modalità chiara che scura
- Verifica che il favicon sia leggibile nelle dimensioni ridotte della barra del browser
:::

### Eliminazione

Per rimuovere un asset di rebranding:

1. Vai alla pagina di dettaglio dell'organizzazione
2. Nella sezione **Rebranding**, trova l'asset da rimuovere
3. Clicca su **Elimina** accanto all'asset
4. Conferma l'operazione

Quando un asset viene rimosso, il prodotto torna a utilizzare il branding predefinito di Nethesis.

### Sostituzione

Per sostituire un asset esistente, carica semplicemente un nuovo file. Il file precedente viene automaticamente sostituito.

## Stato Rebranding

Per ogni organizzazione, è possibile verificare lo stato del rebranding:

| Stato | Descrizione |
|-------|-------------|
| **Non configurato** | Il rebranding non è stato abilitato |
| **Parziale** | Alcuni asset sono stati caricati ma non tutti |
| **Completo** | Tutti gli asset richiesti sono stati caricati |

:::note
Lo stato "Completo" viene raggiunto quando tutti gli asset obbligatori per i prodotti selezionati sono stati caricati.
:::

## Permessi

| Operazione | Super Admin | Admin | Backoffice | Support | Reader |
|------------|:-----------:|:-----:|:----------:|:-------:|:------:|
| Visualizza rebranding | Si | Si | Si | Si | Si |
| Abilita/disabilita | Si | No | No | No | No |
| Carica asset | Si | Si | Si | No | No |
| Elimina asset | Si | Si | Si | No | No |

:::warning
L'abilitazione e la disabilitazione del rebranding è riservata esclusivamente ai Super Admin dell'organizzazione Owner.
:::
