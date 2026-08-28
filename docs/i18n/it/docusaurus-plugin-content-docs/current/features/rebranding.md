---
sidebar_position: 4
---

# Rebranding Organizzazione

Il rebranding consente a un'organizzazione di personalizzare l'aspetto dei prodotti con il proprio marchio: loghi, favicon, sfondo e nome.

## Panoramica

Il rebranding ha due lati. L'Owner decide **quali organizzazioni possono personalizzare il marchio**, e ognuna di quelle organizzazioni configura **il proprio branding**. L'identità di un'organizzazione scende lungo la gerarchia: i rivenditori e i clienti sotto di essa mostrano lo stesso marchio, a meno che non abbiano un branding proprio.

La configurazione è per prodotto: un'organizzazione può usare gli stessi loghi per NethVoice e NethSecurity, oppure darne di diversi a ciascuno.

## Scelta delle Organizzazioni

:::warning
Solo gli utenti dell'organizzazione Owner decidono chi può personalizzare il marchio.
:::

Da **Impostazioni → Rebranding** l'Owner vede tutte le organizzazioni abilitate, i prodotti personalizzati da ciascuna e la data dell'ultima modifica. **Aggiungi aziende** apre un elenco con distributori, rivenditori e clienti non ancora abilitati: un'intera selezione viene aggiunta in un'unica operazione.

**Rimuovi dal rebranding** toglie un'organizzazione dall'elenco e ne **elimina anche gli asset**. Riaggiungendola si riparte da una configurazione vuota.

## Configurazione del Branding

Un amministratore di un'organizzazione abilitata configura il branding della propria organizzazione da **Impostazioni → Rebranding**:

1. Seleziona i **prodotti da personalizzare**
2. Inserisci il **nome del marchio** mostrato accanto al logo
3. Carica solo gli asset che vuoi personalizzare
4. **Salva**

Il salvataggio scrive tutto in un'unica operazione: gli asset vengono applicati a ogni prodotto selezionato e un prodotto tolto dalla selezione perde la propria configurazione. La conferma indica quante organizzazioni a valle mostrano ora quel marchio.

## Prodotti Supportati

Il rebranding è disponibile per i seguenti prodotti:

| Prodotto | Id a catalogo |
|----------|---------------|
| NethVoice | `nethvoice` |
| NethSecurity | `nsec` |
| NethService | `webtop` |
| NS8 | `ns8` |

Ogni prodotto ha il proprio insieme di asset. `GET /api/rebranding/products` restituisce la stessa lista, ed è da lì che nasce il selettore dei prodotti.

## Tipi di Asset

Gli asset configurabili per ogni prodotto e organizzazione:

| Asset | Descrizione | Dimensione massima | Formati |
|-------|-------------|--------------------|---------|
| `logo_light_rect` | Logo rettangolare per sfondi chiari | 2 MB | PNG, SVG, WebP |
| `logo_dark_rect` | Logo rettangolare per sfondi scuri | 2 MB | PNG, SVG, WebP |
| `logo_light_square` | Logo quadrato per sfondi chiari | 2 MB | PNG, SVG, WebP |
| `logo_dark_square` | Logo quadrato per sfondi scuri | 2 MB | PNG, SVG, WebP |
| `favicon` | Icona della scheda del browser | 512 KB | PNG, ICO, SVG |
| `background_image` | Immagine di sfondo | 5 MB | PNG, JPEG, WebP, SVG |
| `product_name` | Nome del prodotto personalizzato | 100 caratteri | Testo (opzionale) |

Un asset lasciato vuoto usa il valore predefinito del prodotto. Rimuovere un singolo asset ripristina quel solo valore predefinito: il resto del branding resta.

:::tip
Per risultati ottimali:
- Usa loghi con sfondo trasparente (SVG è il formato consigliato)
- Verifica l'aspetto sia in tema chiaro sia in tema scuro
- Controlla che la favicon resti leggibile alle dimensioni della scheda del browser
:::

## Dove Vengono Serviti gli Asset

Gli asset sono serviti sia agli utenti autenticati sia, per le pagine che hanno bisogno di un semplice `<img>` come la schermata di login, da un endpoint pubblico con rate limit. Entrambi rispondono con un `ETag`, quindi un'anteprima che ricarica gli asset dopo ogni salvataggio non riscarica nulla.

## Permessi

| Operazione | Super Admin | Admin | Backoffice | Support | Reader |
|------------|:-----------:|:-----:|:----------:|:-------:|:------:|
| Visualizza branding e asset (propri, delle organizzazioni sotto e di quella da cui si eredita) | Sì | Sì | Sì | Sì | Sì |
| Configura il branding della propria organizzazione | Sì | Sì | No | No | No |
| Aggiungi/rimuovi organizzazioni dal rebranding | Sì | Sì | No | No | No |

:::warning
Aggiungere o rimuovere organizzazioni dal rebranding è riservato all'organizzazione Owner. Un distributore o un rivenditore configura **solo** il proprio branding: le organizzazioni sotto di lui lo ereditano e non vengono mai sovrascritte, quindi chi ha un branding proprio lo mantiene.
:::
