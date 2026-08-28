#!/usr/bin/env bash
# End-to-end suite for the rebranding surface: functionality and authorization.
# Runs against a local backend with the rb-* hierarchy registered in apitool.
#
#   rb-dist
#   ├── rb-res1 ── rb-cust1
#   ├── rb-res2
#   └── rb-cust2        (customer directly under the distributor)

set -uo pipefail

API=${API:-http://localhost:8080/api}
BACKEND_DIR=${BACKEND_DIR:-/Users/edospadoni/Workspace/my/backend}
SP=$(cd "$(dirname "$0")" && pwd)
ASSETS=$SP/assets
# Minted tokens are credentials: they are cached outside the repository so a
# run can never leave one behind in a commit.
CACHE=${TOKEN_CACHE:-${TMPDIR:-/tmp}/my-rebranding-e2e}
mkdir -p "$CACHE"

# Organization ids come from the apitool registry, so the suite follows whatever
# the fixture was created as. Create it once with:
#   ./apitool create-org distributor rb-dist --vat=990000000001
#   ./apitool create-user --org=rb-dist --email=<you>+rb-dist-admin@nethesis.it \
#       --name="rb dist-admin" --role=Admin --key=rb-dist-admin
#   ./apitool create-org reseller rb-res1  --vat=990000000002 --as=rb-dist-admin
#   ./apitool create-org reseller rb-res2  --vat=990000000003 --as=rb-dist-admin
#   ./apitool create-org customer rb-cust2 --vat=990000000005 --as=rb-dist-admin
#   ./apitool create-org customer rb-cust1 --vat=990000000004 --as=rb-res1-admin
# plus a user per organization (Admin), a Reader in rb-dist and a Support in rb-res1.
org_id() {
  python3 -c "
import json, sys
reg = json.load(open('$BACKEND_DIR/.api-registry.json'))
org = reg['orgs'].get('$1')
if not org:
    sys.exit('organizzazione \'$1\' non registrata in apitool — vedi le istruzioni in testa allo script')
print(org['logto_id'])
"
}

DIST=$(org_id rb-dist)   || exit 1
RES1=$(org_id rb-res1)   || exit 1
RES2=$(org_id rb-res2)   || exit 1
CUST1=$(org_id rb-cust1) || exit 1
CUST2=$(org_id rb-cust2) || exit 1
AUTHZ_D2=$(org_id authz-d2) || exit 1
OWNER_ORG=""   # read from the owner token once it is minted

PASS=0; FAIL=0; FAILED=()

# --------------------------------------------------------------------- tokens
PERSONAS="owner rb-dist-admin rb-dist-reader rb-res1-admin rb-res1-support rb-res2-admin rb-cust1-admin authz-d2-admin"
get_tokens() {
  for k in $PERSONAS; do
    cache=$CACHE/.tok-$k
    if [[ -f $cache && $(( $(date +%s) - $(stat -f %m "$cache") )) -lt 1200 ]]; then
      continue
    fi
    (cd "$BACKEND_DIR" && ./apitool token "$k" 2>/dev/null | tail -1) > "$cache"
    chmod 600 "$cache"
    [[ -s $cache ]] || { echo "no token for $k"; exit 1; }
  done
}
tok() { cat "$CACHE/.tok-$1"; }

# ---------------------------------------------------------------------- helpers
# req <persona|-> <method> <path> [curl args...]  -> STATUS, BODY
STATUS=""; BODY=""
req() {
  local persona=$1 method=$2 path=$3; shift 3
  local auth=()
  [[ $persona != "-" ]] && auth=(-H "Authorization: Bearer $(tok "$persona")")
  local out
  out=$(curl -s -m 20 -w $'\n%{http_code}' -X "$method" ${auth[@]+"${auth[@]}"} "$API$path" "$@")
  STATUS=${out##*$'\n'}
  BODY=${out%$'\n'*}
}

ok()   { PASS=$((PASS+1)); printf '  \033[32mPASS\033[0m %s\n' "$1"; }
bad()  { FAIL=$((FAIL+1)); FAILED+=("$1"); printf '  \033[31mFAIL\033[0m %s — %s\n' "$1" "$2"; }

# expect <label> <expected-status> [python-assertion-on-json]
expect() {
  local label=$1 want=$2 assertion=${3:-}
  if [[ $STATUS != "$want" ]]; then
    bad "$label" "atteso $want, ricevuto $STATUS: $(echo "$BODY" | head -c 160)"
    return
  fi
  if [[ -n $assertion ]]; then
    local res
    res=$(echo "$BODY" | ASSERTION="$assertion" python3 -c '
import sys, json, os
expr = " ".join(os.environ["ASSERTION"].split())
try:
    b = json.load(sys.stdin)
except Exception as e:
    print("body non JSON: %s" % e); sys.exit(0)
d = b.get("data", b)
try:
    if not eval(expr):
        print("assertion falsa: %s" % " ".join(expr.split()))
except Exception as e:
    print("assertion non valutabile (%s): %s" % (e, " ".join(expr.split())))
')
    if [[ -n $res ]]; then bad "$label" "$res"; return; fi
  fi
  ok "$label"
}

section() { printf '\n\033[1m%s\033[0m\n' "$1"; }

# ============================================================================
get_tokens
OWNER_ORG=$(tok owner | cut -d. -f2 | python3 -c "
import sys, base64, json
p = sys.stdin.read().strip(); p += '=' * (-len(p) % 4)
print(json.loads(base64.urlsafe_b64decode(p))['user']['organization_id'])
")
printf '\033[1mRebranding e2e — %s\033[0m\n' "$API"

# The suite asserts on counts, so it starts from a known state: every rb-*
# organization out of rebranding. Disabling deletes their assets too, which is
# exactly the clean slate wanted here. Done through the API, as the owner.
for org in "$DIST" "$RES1" "$RES2" "$CUST1" "$CUST2"; do
  req owner PATCH "/rebranding/$org/disable" >/dev/null
done

section "A. Catalogo e stato iniziale"
req owner GET /rebranding/products
expect "A1  owner legge il catalogo prodotti" 200 "len(d['products']) == 4"
req owner GET /rebranding/summary
expect "A2  owner legge il summary" 200 "'total' in d and 'distributors' in d"
BASE_TOTAL=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['total'])")
req owner GET /rebranding/organizations
expect "A3  owner legge la lista" 200 "'organizations' in d and 'pagination' in d"
req owner GET "/rebranding/organizations/available?search=rb-&limit=50"
expect "A4  il picker propone le 5 org rb-*" 200 "len([o for o in d['organizations'] if o['name'].startswith('rb-')]) == 5"

section "B. Abilitazione (solo owner)"
req rb-res1-admin POST /rebranding/organizations -H "Content-Type: application/json" -d "{\"organization_ids\":[\"$RES2\"]}"
expect "B1  un reseller Admin non abilita nessuno" 403
req rb-dist-admin PATCH "/rebranding/$DIST/enable"
expect "B2  un distributore non abilita sé stesso" 403
req owner POST /rebranding/organizations -H "Content-Type: application/json" -d "{\"organization_ids\":[\"$DIST\",\"$RES1\"]}"
expect "B3  owner abilita distributore + reseller in blocco" 200 "d['enabled'] == 2"
req owner POST /rebranding/organizations -H "Content-Type: application/json" -d "{\"organization_ids\":[\"$CUST1\",\"org-che-non-esiste\"]}"
expect "B4  un id invalido rifiuta l'intero blocco" 400 "
    d['type'] == 'validation_error'
and [(e['key'], e['message'], e['value']) for e in d['errors']] == [('organization_ids', 'unknown', 'org-che-non-esiste')]"
req owner GET "/rebranding/organizations/available?search=rb-cust1"
expect "B5  ...e non ha abilitato nemmeno l'id valido" 200 "any(o['logto_id'] == '$CUST1' for o in d['organizations'])"
req owner POST /rebranding/organizations -H "Content-Type: application/json" -d "{\"organization_ids\":[\"$OWNER_ORG\"]}"
expect "B6  l'org owner non è abilitabile" 400
req owner POST /rebranding/organizations -H "Content-Type: application/json" -d "{\"organization_ids\":[\"$DIST\"]}"
expect "B7  riabilitare un'org già abilitata è un no-op" 200 "d['enabled'] == 0"
req owner POST /rebranding/organizations -H "Content-Type: application/json" -d '{"organization_ids":[]}'
expect "B8  lista vuota rifiutata dalla validazione" 400
req owner GET /rebranding/summary
expect "B9  il summary conta le due nuove org" 200 "d['total'] == $BASE_TOTAL + 2"
req owner GET "/rebranding/organizations/available?search=rb-&limit=50"
expect "B10 il picker non ripropone le org abilitate" 200 "not any(o['logto_id'] in ('$DIST','$RES1') for o in d['organizations'])"

section "C. Configurazione del branding"
req rb-dist-admin GET "/rebranding/$DIST/status"
expect "C1  il distributore vede lo stato abilitato, senza asset" 200 "d['enabled'] is True and all(p['assets'] == [] for p in d['products'])"
req rb-res2-admin PUT "/rebranding/$RES2/config" -F "products=nethvoice"
expect "C2  un'org non abilitata non può configurare" 403
req rb-dist-admin PUT "/rebranding/$DIST/config" \
  -F "products=nethvoice,nsec" -F "brand_name=UrbanGrid" \
  -F "logo_light_rect=@$ASSETS/logo-light.svg;type=image/svg+xml" \
  -F "favicon=@$ASSETS/favicon.png;type=image/png"
expect "C3  salvataggio su due prodotti, con il conteggio a valle" 200 "d['applied_to_organizations'] == 2"
req rb-dist-admin GET "/rebranding/$DIST/status"
expect "C4  lo stato riporta nome, mime, dimensione e file name" 200 "
    [p for p in d['products'] if p['product_id']=='nethvoice'][0]['product_name'] == 'UrbanGrid'
and sorted(a['name'] for a in [p for p in d['products'] if p['product_id']=='nethvoice'][0]['assets']) == ['favicon','logo_light_rect']
and [a for a in [p for p in d['products'] if p['product_id']=='nethvoice'][0]['assets'] if a['name']=='logo_light_rect'][0]['filename'] == 'logo-light.svg'
and [a for a in [p for p in d['products'] if p['product_id']=='nethvoice'][0]['assets'] if a['name']=='logo_light_rect'][0]['mime_type'] == 'image/svg+xml'
and [a for a in [p for p in d['products'] if p['product_id']=='nethvoice'][0]['assets'] if a['name']=='logo_light_rect'][0]['size'] == 113
and len([p for p in d['products'] if p['product_id']=='nsec'][0]['assets']) == 2"

ETAG=$(curl -s -m 10 -D - -o /dev/null -H "Authorization: Bearer $(tok rb-dist-admin)" \
  "$API/rebranding/$DIST/products/nethvoice/logo_light_rect" | awk -F': ' 'tolower($1)=="etag"{print $2}' | tr -d '\r')
[[ -n $ETAG ]] && ok "C5  l'asset autenticato risponde con un ETag" || bad "C5  l'asset autenticato risponde con un ETag" "nessun ETag"
CODE=$(curl -s -m 10 -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $(tok rb-dist-admin)" \
  -H "If-None-Match: $ETAG" "$API/rebranding/$DIST/products/nethvoice/logo_light_rect")
[[ $CODE == 304 ]] && ok "C6  la richiesta condizionale risponde 304" || bad "C6  la richiesta condizionale risponde 304" "ricevuto $CODE"
CODE=$(curl -s -m 10 -o /dev/null -w '%{http_code}' "$API/public/rebranding/$DIST/products/nethvoice/logo_light_rect")
[[ $CODE == 200 ]] && ok "C7  l'endpoint pubblico serve l'immagine senza token" || bad "C7  l'endpoint pubblico serve l'immagine senza token" "ricevuto $CODE"

req rb-dist-admin PUT "/rebranding/$DIST/config" -F "products=nethvoice" -F "brand_name=UrbanGrid"
expect "C8  un prodotto deselezionato perde la configurazione" 200
req rb-dist-admin GET "/rebranding/$DIST/status"
expect "C9  ...e infatti nsec è tornato vuoto, nethvoice no" 200 "
    [p for p in d['products'] if p['product_id']=='nsec'][0]['assets'] == []
and len([p for p in d['products'] if p['product_id']=='nethvoice'][0]['assets']) == 2"
req rb-dist-admin PUT "/rebranding/$DIST/config" -F "products=nethvoice" -F "clear=favicon"
expect "C10 clear= azzera un asset lasciando gli altri" 200
req rb-dist-admin GET "/rebranding/$DIST/status"
expect "C11 ...la favicon è sparita, il logo è rimasto" 200 "
    [a['name'] for a in [p for p in d['products'] if p['product_id']=='nethvoice'][0]['assets']] == ['logo_light_rect']"
req rb-dist-admin DELETE "/rebranding/$DIST/products/nethvoice/logo_light_rect"
expect "C12 cancellazione del singolo asset" 200
req rb-dist-admin DELETE "/rebranding/$DIST/products/nethvoice/logo_light_rect"
expect "C13 ricancellarlo risponde 404, non 500" 404
req rb-dist-admin PUT "/rebranding/$DIST/config" -F "brand_name=UrbanGrid"
expect "C14 salvare senza prodotti è un errore di richiesta" 400 "
    d['type'] == 'validation_error'
and d['errors'][0]['key'] == 'products' and d['errors'][0]['message'] == 'at_least_one_required'"
req rb-dist-admin PUT "/rebranding/$DIST/config" -F "products=prodotto-inventato"
expect "C15 un prodotto inesistente è un errore di richiesta" 400 "
    d['errors'][0]['key'] == 'products' and d['errors'][0]['message'] == 'unknown'
and d['errors'][0]['value'] == 'prodotto-inventato'"
req rb-dist-admin PUT "/rebranding/$DIST/config" -F "products=nethvoice" -F "clear=logo_inventato"
expect "C16 un asset da azzerare inesistente è un errore di campo" 400 "
    d['errors'][0]['key'] == 'clear' and d['errors'][0]['message'] == 'unknown'
and d['errors'][0]['value'] == 'logo_inventato'"
req rb-dist-admin PUT "/rebranding/$DIST/config" -F "products=nethvoice" -F "brand_name=$(python3 -c 'print("x"*101)')"
expect "C17 un brand name troppo lungo è un errore di campo" 400 "
    d['errors'][0]['key'] == 'brand_name' and d['errors'][0]['message'] == 'max'"
req rb-dist-admin PUT "/rebranding/$DIST/config" -F "products=nethvoice" -F "logo_light_rect=@$ASSETS/not-an-image.txt;type=text/plain"
expect "C18 un content type non ammesso viene rifiutato" 400
req rb-res1-admin PUT "/rebranding/$RES1/config" -F "products=nethvoice" -F "brand_name=Res1Brand" \
  -F "logo_light_rect=@$ASSETS/logo-dark.svg;type=image/svg+xml"
expect "C19 il reseller configura la propria org (1 cliente a valle)" 200 "d['applied_to_organizations'] == 1"

section "D. Autorizzazione in scrittura"
req rb-res1-admin PUT "/rebranding/$RES2/config" -F "products=nethvoice"
expect "D1  un reseller non scrive su un altro reseller" 403
req rb-dist-admin PUT "/rebranding/$RES1/config" -F "products=nethvoice"
expect "D2  un distributore non scrive nemmeno su un proprio reseller" 403
req rb-res1-support PUT "/rebranding/$RES1/config" -F "products=nethvoice"
expect "D3  il ruolo Support non ha manage:rebranding" 403
req rb-dist-reader PUT "/rebranding/$DIST/config" -F "products=nethvoice"
expect "D4  il ruolo Reader non ha manage:rebranding" 403
req rb-res1-admin DELETE "/rebranding/$DIST/products/nethvoice"
expect "D5  un reseller non cancella il branding del suo distributore" 403
req rb-cust1-admin PUT "/rebranding/$RES1/config" -F "products=nethvoice"
expect "D6  un cliente non scrive sull'org da cui eredita" 403
req owner PUT "/rebranding/$RES1/config" -F "products=nethvoice" -F "brand_name=Res1Brand" \
  -F "logo_light_rect=@$ASSETS/logo-dark.svg;type=image/svg+xml"
expect "D7  l'owner scrive su qualsiasi org" 200

section "E. Autorizzazione in lettura"
req rb-res1-support GET "/rebranding/$RES1/status"
expect "E1  Support legge lo stato della propria org" 200
req rb-dist-reader GET "/rebranding/$DIST/status"
expect "E2  Reader legge lo stato della propria org" 200
req rb-res2-admin GET "/rebranding/$RES1/status"
expect "E3  un reseller non legge lo stato di un altro reseller" 403
req rb-res2-admin GET "/rebranding/$RES1/products/nethvoice/logo_light_rect"
expect "E4  ...né i suoi asset" 403
req rb-cust1-admin GET "/rebranding/$RES1/status"
expect "E5  il cliente legge l'org da cui eredita il branding" 200
req rb-cust1-admin GET "/rebranding/$DIST/status"
expect "E6  ...ma non il distributore, che il reseller gli scherma" 403
req rb-dist-admin GET "/rebranding/$RES1/status"
expect "E7  il distributore legge verso il basso" 200
req authz-d2-admin GET "/rebranding/$DIST/status"
expect "E8  un distributore di un altro ramo non legge nulla" 403
req rb-res2-admin GET /rebranding/organizations
expect "E9  la lista di un reseller non abilitato è vuota" 200 "len(d['organizations']) == 0"
req rb-dist-admin GET /rebranding/organizations
expect "E10 la lista del distributore è il suo sottoalbero" 200 "
    sorted(o['logto_id'] for o in d['organizations']) == sorted(['$DIST','$RES1'])"
req rb-dist-admin GET /rebranding/summary
expect "E11 anche il summary è limitato al sottoalbero" 200 "d['total'] == 2 and d['distributors'] == 1 and d['resellers'] == 1"
req - GET /rebranding/summary
expect "E12 senza token si prende 401" 401
CODE=$(curl -s -m 10 -o /dev/null -w '%{http_code}' "$API/public/rebranding/$RES1/products/nethvoice/logo_light_rect")
[[ $CODE == 200 ]] && ok "E13 l'immagine pubblica resta pubblica (per scelta)" || bad "E13 l'immagine pubblica resta pubblica" "ricevuto $CODE"

section "F. Ereditarietà"
req owner POST /rebranding/organizations -H "Content-Type: application/json" -d "{\"organization_ids\":[\"$CUST1\"]}"
expect "F1  owner abilita anche il cliente finale" 200 "d['enabled'] == 1"
req rb-cust1-admin GET "/rebranding/$RES1/status"
expect "F2  ora che ha il proprio branding, non legge più quello del reseller" 403
req rb-res1-admin PUT "/rebranding/$RES1/config" -F "products=nethvoice" -F "brand_name=Res1Brand"
expect "F3  ...e il reseller non ha più nessuno a valle" 200 "d['applied_to_organizations'] == 0"
req owner PATCH "/rebranding/$CUST1/disable"
expect "F4  owner riporta indietro il cliente" 200
req rb-cust1-admin GET "/rebranding/$RES1/status"
expect "F5  che torna a ereditare dal reseller" 200

section "G. Disabilitazione e cancellazione a cascata"
req rb-dist-admin PUT "/rebranding/$DIST/config" -F "products=nethvoice" -F "brand_name=UrbanGrid" \
  -F "logo_light_rect=@$ASSETS/logo-light.svg;type=image/svg+xml"
expect "G1  il distributore ricarica un logo" 200
req owner PATCH "/rebranding/$DIST/disable"
expect "G2  owner rimuove il distributore dal rebranding" 200
CODE=$(curl -s -m 10 -o /dev/null -w '%{http_code}' "$API/public/rebranding/$DIST/products/nethvoice/logo_light_rect")
[[ $CODE == 404 ]] && ok "G3  l'immagine pubblica non esiste più" || bad "G3  l'immagine pubblica non esiste più" "ricevuto $CODE"
req rb-dist-admin GET "/rebranding/$DIST/status"
expect "G4  lo stato è disabilitato e senza asset" 200 "d['enabled'] is False and all(p['assets'] == [] for p in d['products'])"
req rb-dist-admin PUT "/rebranding/$DIST/config" -F "products=nethvoice"
expect "G5  e non si può più configurare" 403
req owner GET "/rebranding/organizations/available?search=rb-dist"
expect "G6  l'org torna disponibile nel picker" 200 "any(o['logto_id'] == '$DIST' for o in d['organizations'])"
req owner POST /rebranding/organizations -H "Content-Type: application/json" -d "{\"organization_ids\":[\"$DIST\"]}"
expect "G7  owner la riabilita" 200 "d['enabled'] == 1"
req rb-dist-admin GET "/rebranding/$DIST/status"
expect "G8  riparte da zero: nessun branding resuscitato" 200 "d['enabled'] is True and all(p['assets'] == [] for p in d['products'])"

section "H. Maschera delle API key"
# Il gruppo /rebranding non aveva permessi di rotta: la maschera read delle
# chiavi filtra i permessi, ma senza un permesso da richiedere non filtrava
# nulla e una chiave in sola lettura poteva riscrivere il branding.
PW=$(python3 -c "
import json
print(json.load(open('$BACKEND_DIR/.api-registry.json'))['users']['rb-dist-admin']['password'])
")
mint_key() {  # mint_key <mode> -> token
  curl -s -m 15 -X POST "$API/me/api-keys" -H "Authorization: Bearer $(tok rb-dist-admin)" \
    -H "Content-Type: application/json" --data-binary "$(MODE="$1" PW="$PW" python3 -c "
import json, os
# dict(...) and not a literal: brace expansion mangles {'a': 1, 'b': 2} inside a
# command substitution, and the JSON never reaches curl.
print(json.dumps(dict(name='rb-e2e-' + os.environ['MODE'], mode=os.environ['MODE'],
                      expires_in_days=1, password=os.environ['PW'])))
")" | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['token'])"
}
KEY_RO=$(mint_key read)
KEY_RW=$(mint_key write)

CODE=$(curl -s -m 10 -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $KEY_RO" "$API/rebranding/$DIST/status")
[[ $CODE == 200 ]] && ok "H1  una chiave read legge lo stato" || bad "H1  una chiave read legge lo stato" "ricevuto $CODE"
CODE=$(curl -s -m 10 -o /dev/null -w '%{http_code}' -X PUT -H "Authorization: Bearer $KEY_RO" -F "products=nethvoice" "$API/rebranding/$DIST/config")
[[ $CODE == 403 ]] && ok "H2  una chiave read non configura il branding" || bad "H2  una chiave read non configura il branding" "ricevuto $CODE"
CODE=$(curl -s -m 10 -o /dev/null -w '%{http_code}' -X PATCH -H "Authorization: Bearer $KEY_RO" "$API/rebranding/$DIST/enable")
[[ $CODE == 403 ]] && ok "H3  una chiave read non abilita nessuno" || bad "H3  una chiave read non abilita nessuno" "ricevuto $CODE"
CODE=$(curl -s -m 10 -o /dev/null -w '%{http_code}' -X PUT -H "Authorization: Bearer $KEY_RW" -F "products=nethvoice" -F "brand_name=KeyProbe" "$API/rebranding/$DIST/config")
[[ $CODE == 200 ]] && ok "H4  una chiave write configura il branding" || bad "H4  una chiave write configura il branding" "ricevuto $CODE"

# Le chiavi di prova non sopravvivono alla suite.
curl -s -m 10 -H "Authorization: Bearer $(tok rb-dist-admin)" "$API/me/api-keys" | python3 -c "
import sys, json
d = json.load(sys.stdin)['data']
keys = d['api_keys'] if isinstance(d, dict) and 'api_keys' in d else d
print('\n'.join(k['id'] for k in keys if k['name'].startswith('rb-e2e-') and not k.get('revoked_at')))
" | while read -r id; do
  [[ -n $id ]] && curl -s -m 10 -o /dev/null -X DELETE -H "Authorization: Bearer $(tok rb-dist-admin)" "$API/me/api-keys/$id"
done

printf '\n\033[1mRisultato: %d passati, %d falliti\033[0m\n' "$PASS" "$FAIL"
if (( FAIL > 0 )); then printf 'Falliti:\n'; printf '  - %s\n' "${FAILED[@]}"; exit 1; fi
