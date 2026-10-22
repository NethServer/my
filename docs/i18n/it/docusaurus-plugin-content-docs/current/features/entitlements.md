---
sidebar_position: 8
---

# Add-on

Licenze granulari per i sistemi: servizi firewall e moduli delle applicazioni NethServer 8, acquistati su NethShop e verificati in tempo reale. Dietro ogni add-on c'è un **entitlement**, la licenza che abilita quell'add-on su quel sistema.

:::info ALPHA
L'interfaccia degli add-on è attualmente in alpha. Il flusso di acquisto su NethShop è in fase di rilascio e alcune schermate potranno cambiare.
:::

## Panoramica

Un **entitlement** è una licenza che abilita un add-on su un sistema. Esistono due tipi:

- **Service** — un add-on del firewall NethSecurity, valido per l'intero sistema (es. *Advanced Threat Shield*, *High Availability*, *Sandbox*)
- **Module** — un add-on per un'applicazione di un cluster NethServer 8 (es. *Chat* per NethVoice), acquistato per l'intero cluster in uno di due modi:
  - **Intero cluster** — copre tutte le istanze dell'applicazione, su ogni nodo (es. *Advanced Threat Shield* per CrowdSec)
  - **A licenze** — si acquista con un numero di **licenze** (la fascia dello shop, es. *1 - 10 istanze*), e ogni licenza si assegna a un'istanza dell'applicazione, su qualsiasi nodo. Solo le istanze con una licenza usano l'add-on

Ogni add-on si acquista per l'intero sistema, identificato dalla sua system key: un add-on non si acquista mai per una singola istanza applicazione.

Gli entitlements si acquistano su **NethShop** e compaiono sul sistema automaticamente all'attivazione della subscription. Il rinnovo estende la scadenza; l'annullamento della subscription revoca la licenza. Le funzionalità dell'appliance validano la licenza in tempo reale su My Nethesis (`/auth`): senza un entitlement attivo la funzionalità non viene erogata.

## La tab Add-on

Ogni sistema di tipo noto (NethSecurity o NethServer 8) mostra una tab **Add-on**:

- Su un **firewall** la tabella elenca i servizi disponibili: quelli acquistati mostrano riferimento di pagamento, validità e prossimo rinnovo; gli altri offrono **Buy on NethShop**.
- Su un **cluster** le card elencano i moduli delle applicazioni presenti sul cluster, con gli stessi dettagli. Un modulo venduto a licenze mostra anche quante licenze sono assegnate, e **Gestisci licenze** apre l'elenco delle istanze dell'applicazione (dall'inventory), ognuna con il suo nodo, dove si assegna o si libera una licenza. Quando le licenze sono tutte assegnate, **Acquista altre licenze** apre lo shop per una fascia superiore: si possono acquistare solo le fasce con più licenze di quelle del sistema, perché le licenze si aumentano soltanto. Ogni istanza mostra il nodo, l'FQDN e l'azienda a cui è assegnata, così è facile riconoscerla. L'organizzazione Owner può anche cambiare il numero di licenze di una licenza dal menu della card.

Un'istanza che esce dall'inventory — rimossa, o spostata su un altro nodo, che le dà un id nuovo — libera da sola la sua licenza, pronta per essere assegnata di nuovo. Se l'add-on scade o viene revocato le istanze conservano le licenze, che tornano valide con il rinnovo.

Il pulsante **Buy on NethShop** apre lo shop con il sistema già preselezionato: l'acquisto è legato al bersaglio giusto senza input manuale.

## Trovare i sistemi che hanno un add-on

L'elenco dei sistemi ha un filtro **Add-on**: scegli uno o più add-on per tenere
solo i sistemi che li possiedono. Il menu propone gli add-on presenti sui tuoi
sistemi, e la corrispondenza conta solo le licenze valide in quel momento,
quindi un add-on scaduto o annullato esclude il sistema. Il filtro vale anche
per l'esportazione CSV e PDF, che porta quindi le stesse righe dell'elenco a
schermo. Vedi [Gestione Sistemi](../systems/management.md#filtri-e-ricerca).

## Ruoli e permessi

| Capacità | Chi |
|---|---|
| Vedere entitlements e scadenze (`read:entitlements`) | Tutti i ruoli utente, nella propria gerarchia |
| Acquistare su NethShop / annullare una subscription (`manage:entitlements`) | Admin, Backoffice — e il personale Nethesis, su tutta la flotta |
| Assegnare e liberare le licenze di un add-on (`manage:entitlements`) | Chi può acquistare, sui sistemi della propria gerarchia — e l'organizzazione Owner |
| Gestire il catalogo, grant manuali, vista sull'intera flotta | Organizzazione Owner (Nethesis) |

Distributori e reseller non possono auto-attivarsi add-on: tutto passa dallo shop.

## Catalogo add-on (Nethesis)

La voce **Add-on** nel menu laterale è disponibile a chiunque abbia il permesso `read:entitlements`, ma la tab **Configurazione** con il catalogo dei tipi di add-on resta riservata all'organizzazione Owner. La creazione richiede il kind (Service o Module), l'applicazione di destinazione per i moduli (l'id si compone automaticamente, es. `nethvoice` + `chat` → `nethvoice-chat`), nome e descrizione. Un modulo si vende per l'**intero cluster** o **a licenze**: la scelta non si può cambiare dopo la creazione, perché licenze e assegnazioni dipendono da essa. Un tipo appena creato è **immediatamente acquistabile da tutti**; regole di disponibilità opzionali possono riservarlo a ruoli o organizzazioni specifiche.

La cancellazione di un tipo è rifiutata finché esistono licenze che lo referenziano — comprese quelle revocate o scadute, conservate per audit. L'elenco del catalogo marca questi tipi con `in_use`, così l'azione di eliminazione è disabilitata invece di fallire.

## Reportistica

`GET /backend/api/entitlements/grants` (con filtri per entitlement, organizzazione, origine, stato e finestra di scadenza) e `GET /backend/api/entitlements/stats` forniscono il report licenze: chi acquista vede la propria gerarchia — ogni modulo con scadenza e rinnovo — mentre l'organizzazione Owner vede l'intera flotta.

La pagina **Add-on** presenta gli stessi dati in forma di dashboard, nella tab **Report**: contatori di stato, add-on in scadenza, dettaglio per add-on, distribuzione dei rinnovi e trend di attivazione a 12 mesi (`GET /backend/api/entitlements/report`, più le slice paginate `/report/organizations` e `/report/tiers`). Ogni aggregato segue la stessa visibilità dell'elenco licenze: un distributore o un reseller vede le proprie organizzazioni, un customer solo i propri sistemi, l'organizzazione Owner l'intera flotta. La tabella per organizzazione non viene mostrata ai customer, che non hanno nulla sotto di sé. Il pannello **Licenze** riassume gli add-on venduti a licenze in vigore: licenze acquistate, licenze assegnate e quanti add-on le hanno esaurite.

## Per gli sviluppatori

- Le licenze vivono in `system_entitlements` (una riga per sistema + entitlement; i rinnovi aggiornano `valid_until` in place, le revoche conservano la riga per audit). Accanto sono conservati lo snapshot dell'acquisto (`purchased_by`) e il tier (`variant`), entrambi di sola visualizzazione. Un tipo di catalogo con `seated = true` si vende a licenze: la licenza porta il numero acquistato in `seats`, e `entitlement_seats` registra quale istanza usa ogni licenza (le righe liberate restano per audit, con il motivo: `manual` o `instance_removed`).
- L'enforcement è servito da collect: `GET /auth/service/<id>[?scope=<istanza>]` con le credenziali Basic del sistema risponde `200` con licenza attiva, `403` senza. Per un add-on venduto a licenze l'istanza va indicata in `scope` e deve avere una delle licenze acquistate (prima le assegnazioni più vecchie): senza `scope` la risposta è `403`. Collect libera le licenze delle istanze assenti da ogni inventory. Gli id legacy (`ng-*`) sono risolti tramite `legacy_alias` del catalogo, quindi i feed dell'appliance continuano a chiamare i path storici senza modifiche. `/auth-hash` risponde alle stesse domande con `Authorization: Bearer <sha256("<system_key>:<token>")>`, hash che collect impara alla prima richiesta del sistema.
- Lo shop attiva e rinnova le licenze con `POST /backend/api/entitlements/activate` (idempotente, indirizzato per `system_key`) e le revoca con `POST /backend/api/entitlements/deactivate`. Per un add-on venduto a licenze l'attivazione porta `seats`, le licenze della fascia acquistata: una fascia superiore lo aumenta sulla stessa licenza e le assegnazioni restano dove sono, mentre una inferiore non toglie mai licenze. Un ordine il cui acquirente è un utente My Nethesis che non vede il sistema nella propria gerarchia viene rifiutato (`403`), così non può mai finire sul sistema di qualcun altro.
- Le licenze si assegnano con `POST /backend/api/systems/<id>/entitlements/<addon>/seats` (`{"module_id": "nethvoice5"}`) e si liberano con `DELETE /backend/api/systems/<id>/entitlements/<addon>/seats/<module_id>`; `GET /backend/api/systems/<id>/entitlements` restituisce `seats` e `seat_assignments` per questi add-on.
- `GET /backend/api/systems` accetta un filtro ripetibile `addon=<id catalogo>` (più id corrispondono a uno qualsiasi di essi) e, con `include_addons=true`, aggiunge a ogni sistema la lista `addons` degli id di catalogo che possiede in quel momento. La lista è opt-in perché costa una query in più per pagina, inutile in una lettura massiva. Le scelte del filtro arrivano da `GET /backend/api/filters/systems`, che restituisce gli add-on presenti nella gerarchia del chiamante come coppie `{id, display_name}`; `GET /backend/api/entitlements/catalog` risolve qualsiasi id nel suo nome.
