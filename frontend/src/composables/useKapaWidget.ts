//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { KAPA_WEBSITE_ID } from '@/lib/config'

// Kapa AI assistant (https://docs.kapa.ai/integrations/website-widget), the
// same widget the legacy my shows to logged-in users. Enabled only in builds
// that set VITE_KAPA_WEBSITE_ID (production); the proxy CSP must allow the
// kapa and reCAPTCHA origins (THIRD_PARTY_SCRIPT_SRC / _FRAME_SRC / _CONNECT_SRC).

export const KAPA_SCRIPT_ID = 'kapa-widget-script'
const KAPA_SCRIPT_SRC = 'https://widget.kapa.ai/kapa-widget.bundle.js'

type KapaFn = ((...args: unknown[]) => void) & { q?: unknown[][] }

declare global {
  interface Window {
    Kapa?: KapaFn
  }
}

// Queue the calls made before the bundle has loaded; the bundle replays them.
function ensureKapaQueue() {
  if (window.Kapa) {
    return
  }
  const queue: unknown[][] = []
  const kapa: KapaFn = (...args: unknown[]) => {
    queue.push(args)
  }
  kapa.q = queue
  window.Kapa = kapa
}

function loadScript(language: string) {
  const existing = document.getElementById(KAPA_SCRIPT_ID)
  if (existing) {
    existing.setAttribute('data-language', language)
    return
  }

  const attributes: Record<string, string> = {
    'data-website-id': KAPA_WEBSITE_ID,
    'data-project-name': 'Nethesis',
    // primary button colour of the app (sky-700 light / sky-500 dark)
    'data-project-color': '#0069a8',
    'data-project-color-dark': '#00a6f4',
    'data-project-logo': `${window.location.origin}/favicon.svg`,
    'data-language': language,
    // follow the `dark` class the theme store sets on <html>
    'data-color-scheme-selector': '.dark',
    'data-search-mode-enabled': 'true',
    'data-modal-z-index': '10000000',
    // no floating launcher, it covers the row actions of the tables: the
    // chat is opened from the help menu of the top bar (openKapaWidget)
    'data-button-hide': 'true',
    // rendered by the composable, so it can be unmounted at logout
    'data-render-on-load': 'false',
  }

  const script = document.createElement('script')
  script.id = KAPA_SCRIPT_ID
  script.async = true
  script.src = KAPA_SCRIPT_SRC
  for (const [name, value] of Object.entries(attributes)) {
    script.setAttribute(name, value)
  }
  document.body.appendChild(script)
}

export const isKapaWidgetEnabled = () => !!KAPA_WEBSITE_ID

export function openKapaWidget() {
  window.Kapa?.('open')
}

export function useKapaWidget() {
  const { locale } = useI18n()

  if (!KAPA_WEBSITE_ID) {
    return
  }

  onMounted(() => {
    ensureKapaQueue()
    loadScript(locale.value)
    window.Kapa?.('render')
  })

  // data-language is read at render time: re-render on a language switch
  watch(locale, (newLocale) => {
    loadScript(newLocale)
    window.Kapa?.('unmount')
    window.Kapa?.('render')
  })

  onUnmounted(() => {
    window.Kapa?.('unmount')
  })
}
