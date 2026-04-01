---
sidebar_position: 3
---

# Gestione Avatar

La gestione avatar consente agli utenti di personalizzare la propria immagine profilo sulla piattaforma My.

## Panoramica

Ogni utente può:

- Caricare un'immagine personalizzata come avatar
- Rimuovere l'avatar e tornare all'immagine predefinita
- Utilizzare l'URL pubblico dell'avatar in applicazioni esterne

## Upload Avatar

### Formati Supportati

| Formato | Estensione | Tipo MIME |
|---------|------------|-----------|
| PNG | `.png` | `image/png` |
| JPEG | `.jpg`, `.jpeg` | `image/jpeg` |
| WebP | `.webp` | `image/webp` |

### Specifiche

| Parametro | Valore |
|-----------|--------|
| Dimensione massima file | 500 KB |
| Dimensione finale | 256x256 pixel |
| Ridimensionamento | Automatico |
| Ritaglio | Centrato (per immagini non quadrate) |

### Come Caricare

1. Vai alla pagina **Account**
2. Nella sezione avatar, clicca su **Carica** o sull'area dell'immagine
3. Seleziona un'immagine dal tuo dispositivo
4. L'immagine viene caricata, ridimensionata e applicata automaticamente

:::note
Se l'immagine supera i 500 KB, il caricamento viene rifiutato. Ridimensiona l'immagine prima del caricamento se necessario.
:::

:::tip
Per risultati ottimali, usa un'immagine quadrata di almeno 256x256 pixel. Le immagini non quadrate vengono ritagliate automaticamente dal centro.
:::

## Avatar Predefinito

Se non è stato caricato un avatar personalizzato, viene generato automaticamente un avatar con:

- Le **iniziali** del nome e cognome dell'utente
- Uno **sfondo colorato** generato in base al nome dell'utente
- Formato circolare

L'avatar predefinito viene aggiornato automaticamente se il nome dell'utente cambia.

## Eliminazione Avatar

Per rimuovere l'avatar personalizzato e tornare all'avatar predefinito:

1. Vai alla pagina **Account**
2. Nella sezione avatar, clicca su **Elimina**
3. L'avatar torna automaticamente all'immagine predefinita con le iniziali

:::note
L'eliminazione è immediata e non reversibile. Per ripristinare l'avatar personalizzato, sarà necessario caricarne uno nuovo.
:::

## URL Pubblico

Ogni avatar caricato è disponibile tramite un URL pubblico accessibile senza autenticazione.

### Caratteristiche dell'URL

- **Accessibile pubblicamente** - Non richiede autenticazione
- **Persistente** - L'URL rimane valido fino alla modifica dell'avatar
- **Cache-friendly** - Ottimizzato per la cache del browser

### Casi d'Uso

L'URL pubblico dell'avatar può essere utilizzato per:

- Integrazione con applicazioni esterne
- Visualizzazione in email o notifiche
- Embedding in pagine web o dashboard
