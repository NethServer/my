<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The logo of an add-on, from its catalog id alone: the application logo for a
  NethServer module, the product logo for a system-wide service. Views that
  already hold the Addon can reach for the two logo components directly — this
  exists for the ones holding nothing but the id, such as the report, whose
  endpoints carry no catalog fields.

  The id prefix is deliberately not read: an application name may itself contain
  a hyphen, which is why applies_to exists. So the catalog is the only source,
  and an id it does not know gets no logo rather than a guessed one.
-->

<script setup lang="ts">
import { computed } from 'vue'
import ApplicationLogo from '@/components/applications/ApplicationLogo.vue'
import SystemLogo from '@/components/systems/SystemLogo.vue'
import { getAddonApplication } from '@/lib/addons/addons'
import { useAddons } from '@/queries/addons/addons'

const { addonId } = defineProps<{ addonId: string }>()

// One shared query however many rows ask for it: the catalog arrives whole and
// is cached for the page, so this is a lookup, not a request per add-on.
const { state: catalog } = useAddons()

const addon = computed(() => catalog.value.data?.find((item) => item.id === addonId))
const application = computed(() => (addon.value ? getAddonApplication(addon.value) : ''))
</script>

<template>
  <ApplicationLogo v-if="application" :app="application" />
  <SystemLogo v-else-if="addon?.system_type" :system="addon.system_type" />
</template>
