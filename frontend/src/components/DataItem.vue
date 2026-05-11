<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<template>
  <div class="flex justify-between gap-2 py-4">
    <div class="text-tertiary-neutral dark:text-tertiary-neutral font-medium">
      <slot name="label"></slot>
    </div>
    <span class="text-secondary-neutral dark:text-secondary-neutral text-end">
      <slot name="data"></slot>
    </span>
  </div>
</template>
