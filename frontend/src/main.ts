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
import { LOGTO_APP_ID, LOGTO_ENDPOINT } from './lib/config'
import { PiniaColada } from '@pinia/colada'
import { PiniaColadaAutoRefetch } from '@pinia/colada-plugin-auto-refetch'

// prevent FontAwesome from automatically adding CSS (needed to fix icons style)
fontawesomeConfig.autoAddCss = false

// logto configuration
// `offline_access` makes Logto issue a refresh token to the SPA, so the SDK can
// renew the access token silently after its (~1h) TTL instead of forcing a full
// re-login. Requires the "Refresh token" grant to be enabled on the Logto app.
const logtoConfig: LogtoConfig = {
  endpoint: LOGTO_ENDPOINT,
  appId: LOGTO_APP_ID,
  scopes: ['openid', 'profile', 'email', 'offline_access'],
}

const app = createApp(App)

app.use(createPinia())
app.use(PiniaColada, {
  plugins: [PiniaColadaAutoRefetch({})],
})
app.use(i18n)
app.use(router)
app.use(createLogto, logtoConfig)

app.mount('#app')
