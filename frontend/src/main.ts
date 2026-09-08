//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { config as fontawesomeConfig } from '@fortawesome/fontawesome-svg-core'
import { createLogto, type LogtoConfig } from '@logto/vue'

import App from './App.vue'
import router from './router'
import i18n from './i18n'
import { LOGTO_API_RESOURCE, LOGTO_APP_ID, LOGTO_ENDPOINT } from './lib/config'
import { PiniaColada } from '@pinia/colada'
import { PiniaColadaAutoRefetch } from '@pinia/colada-plugin-auto-refetch'
import { IS_E2E } from './lib/config'

// prevent FontAwesome from automatically adding CSS (needed to fix icons style)
fontawesomeConfig.autoAddCss = false

// logto configuration
// `offline_access` makes Logto issue a refresh token to the SPA, so the SDK can
// renew the access token silently after its (~1h) TTL instead of forcing a full
// re-login. Requires the "Refresh token" grant to be enabled on the Logto app.
const logtoConfig: LogtoConfig = {
  endpoint: LOGTO_ENDPOINT,
  appId: LOGTO_APP_ID,
  // The my API resource: getAccessToken(LOGTO_API_RESOURCE) then yields a JWT
  // bound to it (audience) and to this app (client_id), which is the only kind
  // of token the backend exchanges.
  resources: [LOGTO_API_RESOURCE],
  scopes: [
    'openid',
    'profile',
    'email',
    'offline_access',
    // Organization claims in the ID token: third-party app widgets (e.g.
    // NethSpot /userinfo) resolve the user's company from these, with the
    // same identity contract used by their OIDC login
    'urn:logto:scope:organizations',
    'urn:logto:scope:organization_roles',
    // The technical role, in the same token. An app's access_control names an
    // organization list *and* a user-role list, and a widget endpoint has to
    // be able to apply both: without this claim it can only tell which
    // organization the caller belongs to, so it would serve a Reader the data
    // behind a tile My does not show them.
    'roles',
  ],
}

const app = createApp(App)

app.use(createPinia())
// Background refetching races assertions in the e2e suite, which drives a real
// backend and asserts on settled UI. Left out of e2e builds only.
app.use(PiniaColada, {
  plugins: IS_E2E ? [] : [PiniaColadaAutoRefetch({})],
})
app.use(i18n)
app.use(router)
app.use(createLogto, logtoConfig)

app.mount('#app')
