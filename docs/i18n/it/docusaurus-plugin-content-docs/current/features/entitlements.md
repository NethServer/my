---
sidebar_position: 8
---

# Entitlements

Licenze granulari per i sistemi: servizi firewall e moduli per applicazione, acquistati su NethShop e verificati in tempo reale.

:::info ALPHA
L'interfaccia degli entitlements è attualmente in alpha. Il flusso di acquisto su NethShop è in fase di rilascio e alcune schermate potranno cambiare.
:::

## Panoramica

Un **entitlement** è una licenza che abilita un add-on su un sistema. Esistono due tipi:

- **Service** — un add-on del firewall NethSecurity, valido per l'intero sistema (es. *Advanced Threat Shield*, *High Availability*, *Sandbox*)
- **Module** — un add-on per una **singola istanza applicazione** di un cluster NethServer 8 (es. il modulo *Chat* per `nethvoice1`, ma non per `nethvoice2`)

Gli entitlements si acquistano su **NethShop** e compaiono sul sistema automaticamente all'attivazione della subscription. Il rinnovo estende la scadenza; l'annullamento della subscription revoca la licenza. Le funzionalità dell'appliance validano la licenza in tempo reale su My Nethesis (`/auth`): senza un entitlement attivo la funzionalità non viene erogata.

## La tab Entitlements

Ogni sistema di tipo noto (NethSecurity o NethServer 8) mostra una tab **Entitlements**:

- Su un **firewall** la tabella elenca i servizi disponibili: quelli acquistati mostrano riferimento di pagamento, validità e prossimo rinnovo; gli altri offrono **Buy on NethShop**.
- Su un **cluster** la tabella è a due livelli: le istanze applicazione presenti sul sistema (dall'inventory) e, sotto ognuna, i moduli disponibili per quell'applicazione — acquistati o acquistabili per istanza.

Il pulsante **Buy on NethShop** apre lo shop con sistema (e istanza applicazione) già preselezionati: l'acquisto è legato al bersaglio giusto senza input manuale.

## Ruoli e permessi

| Capacità | Chi |
|---|---|
| Vedere entitlements e scadenze (`read:entitlements`) | Tutti i ruoli utente, nella propria gerarchia |
| Acquistare su NethShop / annullare una subscription (`manage:entitlements`) | Admin, Backoffice, Super Admin |
| Gestire il catalogo, grant manuali, vista sull'intera flotta | Organizzazione owner o Super Admin (Nethesis) |

Distributori e reseller non possono auto-attivarsi add-on: tutto passa dallo shop.

## Catalogo entitlements (Nethesis)

Gli utenti owner e Super Admin hanno la voce **Entitlements** nel menu laterale con il catalogo dei tipi di add-on. La creazione richiede il kind (Service o Module), l'applicazione di destinazione per i moduli (l'id si compone automaticamente, es. `nethvoice` + `chat` → `nethvoice-chat`), nome e descrizione. Un tipo appena creato è **immediatamente acquistabile da tutti**; regole di disponibilità opzionali possono riservarlo a ruoli o organizzazioni specifiche.

La cancellazione di un tipo è rifiutata finché esistono licenze che lo referenziano.

## Reportistica

`GET /api/entitlements/grants` (con filtri per entitlement, organizzazione, origine, stato e finestra di scadenza) e `GET /api/entitlements/stats` forniscono il report licenze: chi acquista vede la propria gerarchia — ogni modulo con scadenza e rinnovo — mentre owner e Super Admin vedono l'intera flotta.

## Per gli sviluppatori

- Le licenze vivono in `system_entitlements` (una riga per sistema + entitlement + scope; i rinnovi aggiornano `valid_until` in place, le revoche conservano la riga per audit).
- L'enforcement è servito da collect: `GET /auth/service/<id>[?scope=<istanza>]` con le credenziali Basic del sistema risponde `200` con licenza attiva, `403` senza. Gli id legacy (`ng-*`) sono risolti tramite `legacy_alias` del catalogo, quindi i feed dell'appliance continuano a chiamare i path storici senza modifiche.
- Lo shop attiva e rinnova le licenze con `POST /api/entitlements/activate` (idempotente, indirizzato per `system_key`) e le revoca con `POST /api/entitlements/deactivate`.
