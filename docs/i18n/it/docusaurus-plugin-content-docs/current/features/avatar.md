---
sidebar_position: 3
---

# Gestione Avatar

La gestione avatar consente agli utenti di personalizzare la propria immagine profilo sulla piattaforma My.

## Panoramica

L'avatar viene mostrato accanto al nome dell'utente in tutta la piattaforma: barra di navigazione, elenchi utenti, commenti e ovunque compaia l'identità di un utente.

Ogni utente può:

- Caricare un'immagine personalizzata come avatar
- Rimuovere l'avatar e tornare alle iniziali predefinite
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
| Dimensioni massime dell'originale | 4096x4096 pixel |
| Risultato | Contenuto entro 256x256 pixel |
| Formato finale | PNG (conversione automatica) |
| Proporzioni | Mantenute -- l'immagine viene scalata, mai ritagliata |

### Come Caricare

1. Vai alla pagina **Account** (icona del profilo in alto a destra)
2. Nella sezione avatar, clicca su **Carica** o sull'area dell'immagine
3. Seleziona un'immagine dal tuo dispositivo
4. L'immagine viene caricata, ridimensionata e applicata immediatamente

:::note
Un'immagine oltre i 500 KB, o più grande di 4096x4096 pixel, viene rifiutata. Ridimensionala prima del caricamento.
:::

:::tip
Usa un'immagine quadrata di almeno 256x256 pixel. Un'immagine non quadrata **non** viene ritagliata: viene scalata finché non entra in un riquadro di 256x256, quindi mantiene le proporzioni originali e l'interfaccia la mostra dentro un cerchio.
:::

## Avatar Predefinito

Se non è stato caricato un avatar personalizzato, l'interfaccia mostra un segnaposto generato con:

- Le **iniziali** ricavate dal nome visualizzato (es. "Mario Rossi" mostra "MR")
- Uno **sfondo colorato**
- Forma circolare

Il segnaposto segue il nome visualizzato, quindi si aggiorna da solo quando il nome cambia.

## Eliminazione Avatar

Per rimuovere l'avatar personalizzato e tornare alle iniziali:

1. Vai alla pagina **Account**
2. Nella sezione avatar, clicca su **Elimina**
3. Viene ripristinato il segnaposto con le iniziali

:::note
L'eliminazione è immediata e non reversibile. Per avere di nuovo un avatar personalizzato, sarà necessario caricarne uno nuovo.
:::

## URL Pubblico

Ogni avatar è disponibile a un URL pubblico, utilizzabile per l'integrazione con altri servizi:

```
/backend/api/public/users/{user_id}/avatar
```

Caratteristiche:

- **Accessibile pubblicamente** -- non richiede autenticazione
- **Con rate limit** -- 10 richieste al secondo per IP, burst 30; oltre quella soglia risponde `429`
- **In cache per un'ora** nel browser che lo richiede (`Cache-Control: private, max-age=3600`), non nelle cache condivise
- **Persistente** -- l'URL resta valido anche quando l'avatar cambia: indirizza l'utente, non l'immagine
- Risponde **`204 No Content`** quando l'utente non ha un avatar. È il caso normale, non un errore, e non viene mai messo in cache, così un avatar appena caricato compare al caricamento successivo della pagina

Casi d'uso:

- Integrazione con applicazioni esterne
- Visualizzazione in email o notifiche
- Embedding in pagine web o dashboard
