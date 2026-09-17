---
sidebar_position: 1
---

# Dashboard

La dashboard è la pagina su cui si arriva dopo l'accesso a My. Dà una visione immediata di quello che gestisci e porta direttamente agli elenchi già filtrati.

## Panoramica

La dashboard è composta da due file:

1. **Card contatore** -- una per ogni risorsa che hai il permesso di leggere, ognuna con un totale e dei badge che aprono un elenco già filtrato
2. **Applicazioni di terze parti** -- i servizi esterni collegati alla piattaforma, come NethShop

Ogni card viene mostrata solo se possiedi il permesso di lettura di quella risorsa, quindi la dashboard non mostra mai un contatore che non potresti aprire.

## Card Contatore

| Card | Mostrata se hai | Contatore | Badge |
|------|-----------------|-----------|-------|
| **Allarmi** | `read:systems` | Totale allarmi aperti | Critici, warning, silenziati -- ognuno apre gli Allarmi filtrati per quella severità o stato |
| **Sistemi** | `read:systems` | Totale sistemi | Attivi, inattivi, in attesa -- ognuno apre i Sistemi filtrati per quello stato |
| **Applicazioni** | `read:applications` | Totale applicazioni | Non assegnate -- apre le Applicazioni filtrate su quelle non assegnate |
| **Distributori** | `read:distributors` | Totale distributori | -- |
| **Rivenditori** | `read:resellers` | Totale rivenditori | -- |
| **Clienti** | `read:customers` | Totale clienti | -- |
| **Utenti** | `read:users` | Totale utenti | -- |

Un badge compare solo quando il suo conteggio è maggiore di zero, quindi una flotta in salute mostra una card pulita.

:::note
Le card seguono i **permessi**, non solo la posizione in gerarchia. Un utente Support, ad esempio, non ha `read:users`, quindi la card Utenti non viene mostrata anche se la sua organizzazione ha utenti.
:::

## Applicazioni di Terze Parti

Sotto i contatori, la dashboard elenca le applicazioni di terze parti registrate sulla piattaforma. Ogni riquadro mostra il nome dell'applicazione, la sua descrizione e un pulsante che la apre con la tua identità My già autenticata.

Quali riquadri vedi dipende da due cose:

- **I portali del tuo distributore.** L'organizzazione Owner decide, distributore per distributore, quali portali possono usare i suoi rivenditori e clienti (vedi [Creare un Distributore](../platform/organizations.md#portali)). Gli utenti di un rivenditore o di un cliente vedono solo quelli; un rivenditore il cui distributore non ha portali abilitati non vede alcun riquadro.
- **I tuoi ruoli.** Ogni portale dichiara quali ruoli organizzazione e ruoli utente possono aprirlo: un portale per utenti Admin e Support non viene offerto a un Reader, anche se il distributore lo ha.

Distributori e organizzazione Owner non sono mai limitati dalla lista di un distributore: i loro utenti vedono tutti i portali ammessi dai loro ruoli.

Per i portali che Nethesis contrassegna come protetti sull'identity provider, la stessa regola sull'organizzazione vale anche all'accesso: un utente di un rivenditore o cliente fuori da una gerarchia abilitata viene rifiutato dall'identity provider anche avendo a portata di mano la pagina di accesso del portale. Il filtro per ruolo non viene applicato lì e resta una regola della Dashboard.

Un'applicazione non abilitata per la tua organizzazione viene mostrata con il pulsante disabilitato.

Alcune applicazioni pubblicano anche un piccolo widget di riepilogo letto in tempo reale dall'applicazione stessa -- ad esempio il riepilogo dell'account NethShop. Il widget è nascosto per l'organizzazione Owner, perché i dati dell'account dello shop interessano ai partner che acquistano, non agli amministratori della piattaforma.

## Regole di Visibilità

I valori dei contatori sono sempre limitati al tuo ramo della gerarchia: non vedi mai dati esterni a esso.

- Organizzazione **Owner**: ogni risorsa, su tutta la piattaforma
- **Distributore**: i propri rivenditori e clienti, con i loro utenti, sistemi e applicazioni
- **Rivenditore**: i propri clienti, con i loro utenti, sistemi e applicazioni
- **Cliente**: solo la propria organizzazione

Il ruolo organizzazione decide *quali* card della gerarchia esistono -- un rivenditore non ha la card Distributori, perché non ha `read:distributors` -- mentre il ruolo utente decide il resto.

:::note Trend
Le card contatore mostrano solo i totali correnti. La crescita nel tempo è disponibile da API, tramite gli endpoint `/trend` di ogni risorsa (`/backend/api/systems/trend`, `/backend/api/users/trend`, e così via), e nella tab **Report** della pagina Add-on.
:::
