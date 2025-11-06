<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<template>
  <div class="flex justify-between gap-2 py-4">
    <div class="font-medium">
      <slot name="label"></slot>
    </div>
    <span class="text-end text-gray-600 dark:text-gray-300">
      <slot name="data"></slot>
    </span>
  </div>
</template>
