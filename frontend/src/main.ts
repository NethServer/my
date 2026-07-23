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
const logtoConfig: LogtoConfig = {
  endpoint: LOGTO_ENDPOINT,
  appId: LOGTO_APP_ID,
  scopes: [
    'openid',
    'profile',
    'email',
    // Organization claims in the ID token: third-party app widgets (e.g.
    // NethSpot /userinfo) resolve the user's company from these, with the
    // same identity contract used by their OIDC login
    'urn:logto:scope:organizations',
    'urn:logto:scope:organization_roles',
  ],
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
