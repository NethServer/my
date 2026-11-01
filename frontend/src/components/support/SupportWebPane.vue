<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  faArrowsRotate,
  faGlobe,
  faKey,
  faTriangleExclamation,
  faUpRightFromSquare,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeButton, NeInlineNotification, formatDateTimeNoSeconds } from '@nethesis/vue-components'
import {
  buildSupportProxyUrl,
  generateSupportProxyToken,
  logSupportWorkspaceEvent,
} from '@/lib/support/support'
import {
  SUPPORT_AUTOFILL_MESSAGE,
  SUPPORT_AUTOFILL_RESULT_MESSAGE,
  SUPPORT_OPEN_LINK_MESSAGE,
  type WorkspaceLogin,
} from '@/lib/support/workspace'

// How long before expiry the tab starts warning
const EXPIRY_WARNING_MS = 5 * 60 * 1000

const props = defineProps<{
  sessionId: string
  service: string
  path: string
  systemName: string
  credentials?: WorkspaceLogin | null
  exact?: boolean
  // the tab is the one in front: only then may its page ask for a new tab
  active?: boolean
}>()

const emit = defineEmits<{
  // a link of the page points to another service of the session
  openLink: [label: string, slug: string, path: string]
}>()

// An untrusted page must not flood the workspace with tabs
const LINK_REQUEST_INTERVAL_MS = 1000
let lastLinkRequest = 0

const { t, locale } = useI18n()

const src = ref('')
const host = ref('')
const loading = ref(true)
const loadError = ref('')
const expiresAt = ref<Date | null>(null)
const now = ref(Date.now())
// bumped on reload so the iframe is recreated even with the same URL
const frameKey = ref(0)
const frameRef = ref<HTMLIFrameElement | null>(null)
const autofillNotice = ref('')
// the proxy's autofill script announces the page it runs in, and its leaving
const autofillReady = ref(false)
const autofillPath = ref('')
let clock: ReturnType<typeof setInterval> | null = null

const frameOrigin = computed(() => {
  try {
    return src.value ? new URL(src.value).origin : ''
  } catch {
    return ''
  }
})

const expiryState = computed<'ok' | 'expiring' | 'expired'>(() => {
  if (!expiresAt.value) return 'ok'
  const left = expiresAt.value.getTime() - now.value
  if (left <= 0) return 'expired'
  if (left <= EXPIRY_WARNING_MS) return 'expiring'
  return 'ok'
})

async function openUrl(): Promise<{ url: string | null; expires: Date | null }> {
  const result = await generateSupportProxyToken(props.sessionId, props.service)
  return {
    url: buildSupportProxyUrl(result.url, result.token, props.path, props.exact),
    expires: result.expires_at ? new Date(result.expires_at) : null,
  }
}

async function load() {
  loading.value = true
  loadError.value = ''
  autofillNotice.value = ''
  autofillReady.value = false
  autofillPath.value = ''
  try {
    const { url, expires } = await openUrl()
    if (!url) {
      loadError.value = t('support.workspace_load_error')
      return
    }
    host.value = new URL(url).host
    expiresAt.value = expires
    src.value = url
    frameKey.value++
  } catch {
    loadError.value = t('support.workspace_load_error')
  } finally {
    loading.value = false
  }
}

async function openInNewTab() {
  try {
    const { url } = await openUrl()
    if (url) {
      window.open(url, '_blank', 'noopener,noreferrer')
    }
  } catch {
    loadError.value = t('support.workspace_load_error')
  }
}

// The login form lives in another origin: the proxy gives the page a script
// that fills it on request, and the credentials go to that exact origin only
function fillCredentials() {
  const target = frameRef.value?.contentWindow
  if (!target || !frameOrigin.value || !props.credentials) return
  if (!autofillReady.value) {
    autofillNotice.value = t('support.workspace_autofill_unavailable')
    return
  }
  autofillNotice.value = ''
  target.postMessage(
    {
      type: SUPPORT_AUTOFILL_MESSAGE,
      username: props.credentials.username,
      password: props.credentials.password,
    },
    frameOrigin.value,
  )
}

function onMessage(event: MessageEvent) {
  if (event.source !== frameRef.value?.contentWindow || event.origin !== frameOrigin.value) return
  const data = event.data as {
    type?: string
    result?: string
    label?: unknown
    slug?: unknown
    path?: unknown
  }
  if (data?.type === SUPPORT_OPEN_LINK_MESSAGE) {
    const now = Date.now()
    if (
      props.active &&
      now - lastLinkRequest >= LINK_REQUEST_INTERVAL_MS &&
      typeof data.label === 'string' &&
      /^[a-z0-9-]{1,40}$/.test(data.label) &&
      typeof data.slug === 'string' &&
      /^[0-9a-f]{32}$/.test(data.slug) &&
      typeof data.path === 'string' &&
      data.path.startsWith('/') &&
      !data.path.startsWith('//') &&
      data.path.length <= 2048
    ) {
      lastLinkRequest = now
      emit('openLink', data.label, data.slug, data.path)
    }
    return
  }
  if (data?.type !== SUPPORT_AUTOFILL_RESULT_MESSAGE) return
  if (data.result === 'ready') {
    autofillReady.value = true
    autofillPath.value = typeof data.path === 'string' ? data.path.slice(0, 200) : ''
    return
  }
  if (data.result === 'gone') {
    autofillReady.value = false
    autofillPath.value = ''
    return
  }
  if (data.result === 'filled') {
    logSupportWorkspaceEvent(props.sessionId, 'autofill', props.service)
  }
  autofillNotice.value =
    data.result === 'not_found' || data.result === 'partial'
      ? t('support.workspace_autofill_not_found')
      : ''
}

onMounted(() => {
  window.addEventListener('message', onMessage)
  load()
  clock = setInterval(() => {
    now.value = Date.now()
  }, 30_000)
})

onBeforeUnmount(() => {
  window.removeEventListener('message', onMessage)
  if (clock) clearInterval(clock)
})

defineExpose({ reload: load })
</script>

<template>
  <div class="absolute inset-0 flex flex-col bg-white dark:bg-gray-950">
    <!-- remote content bar: the frame below is served by the customer's system -->
    <div
      class="flex h-11 shrink-0 items-center gap-3 border-b-2 border-amber-500/70 bg-gray-100 px-3 text-xs text-gray-600 dark:bg-gray-800 dark:text-gray-300"
    >
      <FontAwesomeIcon :icon="faGlobe" class="h-3.5 w-3.5 shrink-0 text-amber-600" />
      <span class="shrink-0 font-semibold text-amber-700 dark:text-amber-400">
        {{ t('support.workspace_remote_content') }}
      </span>
      <span class="min-w-0 truncate">{{ systemName }} · {{ host || service }}</span>
      <span
        v-if="expiryState !== 'ok'"
        class="flex shrink-0 items-center gap-1 text-amber-700 dark:text-amber-400"
      >
        <FontAwesomeIcon :icon="faTriangleExclamation" class="h-3 w-3" />
        {{
          expiryState === 'expired'
            ? t('support.workspace_access_expired')
            : t('support.workspace_access_expiring', {
                time: expiresAt ? formatDateTimeNoSeconds(expiresAt, locale) : '',
              })
        }}
      </span>
      <span v-if="autofillNotice" class="shrink-0 text-gray-500 dark:text-gray-400">
        {{ autofillNotice }}
      </span>
      <div class="ml-auto flex shrink-0 items-center gap-1">
        <NeButton
          v-if="credentials"
          kind="tertiary"
          size="xs"
          :disabled="loading || !!loadError"
          :title="
            autofillReady ? t('support.workspace_autofill_target', { path: autofillPath }) : ''
          "
          @click="fillCredentials"
        >
          <template #prefix>
            <FontAwesomeIcon :icon="faKey" aria-hidden="true" />
          </template>
          {{ t('support.workspace_autofill') }}
        </NeButton>
        <NeButton kind="tertiary" size="xs" :disabled="loading" @click="load">
          <template #prefix>
            <FontAwesomeIcon :icon="faArrowsRotate" aria-hidden="true" />
          </template>
          {{ t('support.workspace_reload') }}
        </NeButton>
        <NeButton kind="tertiary" size="xs" @click="openInNewTab">
          <template #prefix>
            <FontAwesomeIcon :icon="faUpRightFromSquare" aria-hidden="true" />
          </template>
          {{ t('support.workspace_open_new_tab') }}
        </NeButton>
      </div>
    </div>
    <div class="relative min-h-0 flex-1">
      <div v-if="loading" class="absolute inset-0 flex items-center justify-center text-gray-500">
        {{ t('support.workspace_loading') }}
      </div>
      <div v-else-if="loadError" class="p-6">
        <NeInlineNotification kind="error" :title="loadError" />
      </div>
      <!--
        The customer's system is untrusted content: no top-level navigation
        (it could send my to a fake login), popups inherit the sandbox, and no
        browser permission at all (not even the clipboard: a page could swap
        what the operator then pastes into a terminal).
      -->
      <iframe
        v-if="src && !loadError"
        ref="frameRef"
        :key="frameKey"
        :src="src"
        :title="`${systemName} · ${service}`"
        class="absolute inset-0 h-full w-full border-0"
        sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-downloads allow-modals"
        allow=""
        referrerpolicy="no-referrer"
      ></iframe>
    </div>
  </div>
</template>
