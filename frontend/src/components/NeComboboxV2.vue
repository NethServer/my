<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<template>
  <div class="relative">
    <!-- Main input container -->
    <div
      ref="comboboxRef"
      class="min-h-[2.5rem] w-full cursor-text rounded-md border border-gray-300 bg-white px-3 py-2 shadow-sm focus-within:border-blue-500 focus-within:ring-2 focus-within:ring-blue-500"
      @click="focusInput"
    >
      <div class="flex flex-wrap items-center gap-1">
        <!-- Selected chips (only for multiple selection) -->
        <template v-if="multiple">
          <div
            v-for="(item, index) in selectedItems"
            :key="getItemKey(item, index)"
            class="flex max-w-xs items-center gap-1 rounded bg-blue-100 px-2 py-1 text-sm text-blue-800"
          >
            <span class="truncate">{{ getItemLabel(item) }}</span>
            <button
              @click.stop="removeItem(index)"
              class="flex-shrink-0 text-blue-600 hover:text-blue-800"
              :aria-label="`Remove ${getItemLabel(item)}`"
            >
              <svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        </template>

        <!-- Input field -->
        <input
          ref="inputRef"
          v-model="inputValue"
          type="text"
          class="min-w-[120px] flex-1 bg-transparent outline-none"
          :placeholder="
            !multiple && selectedItems.length > 0 && !isOpen
              ? ''
              : selectedItems.length === 0
                ? placeholder
                : ''
          "
          @input="handleInput"
          @keydown="handleKeydown"
          @focus="handleFocus"
          @blur="handleBlur"
          :aria-expanded="isOpen"
          :aria-haspopup="true"
          :aria-owns="dropdownId"
          role="combobox"
        />

        <!-- Single selection display value (shown when not focused) -->
        <div
          v-if="!multiple && selectedItems.length > 0 && !isOpen && !inputValue"
          class="pointer-events-none absolute inset-0 flex items-center px-3 text-gray-900"
        >
          <span class="truncate">{{ getItemLabel(selectedItems[0]) }}</span>
        </div>
      </div>

      <!-- Dropdown arrow -->
      <div class="pointer-events-none absolute top-1/2 right-3 -translate-y-1/2 transform">
        <svg
          class="h-4 w-4 text-gray-400 transition-transform duration-200"
          :class="{ 'rotate-180': isOpen }"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M19 9l-7 7-7-7"
          />
        </svg>
      </div>
    </div>

    <!-- Dropdown -->
    <div
      v-if="isOpen"
      :id="dropdownId"
      class="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-md border border-gray-300 bg-white shadow-lg"
      role="listbox"
      :aria-multiselectable="multiple"
    >
      <!-- No results message -->
      <div
        v-if="filteredOptions.length === 0 && !allowCustomInput"
        class="px-3 py-2 text-sm text-gray-500"
      >
        No results found
      </div>

      <!-- Custom input option -->
      <div
        v-if="allowCustomInput && inputValue.trim() && !exactMatch"
        class="cursor-pointer border-b border-gray-100 px-3 py-2 text-sm hover:bg-blue-50"
        @click="selectCustomValue"
        role="option"
      >
        <span class="text-blue-600">Add: "</span>
        <span class="font-medium">{{ inputValue.trim() }}</span>
        <span class="text-blue-600">"</span>
      </div>

      <!-- Options list -->
      <div
        v-for="(option, index) in filteredOptions"
        :key="getItemKey(option, index)"
        class="flex cursor-pointer items-center justify-between px-3 py-2 text-sm hover:bg-gray-50"
        :class="{
          'bg-blue-50': highlightedIndex === index,
          'text-gray-400': isSelected(option),
        }"
        @click="selectOption(option)"
        @mouseenter="highlightedIndex = index"
        role="option"
        :aria-selected="isSelected(option)"
      >
        <span class="truncate">{{ getItemLabel(option) }}</span>
        <svg
          v-if="isSelected(option)"
          class="h-4 w-4 flex-shrink-0 text-blue-600"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M5 13l4 4L19 7"
          />
        </svg>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted, useTemplateRef } from 'vue'

// Types
type OptionValue = string | number | Record<string, any>

interface ComboboxProps {
  modelValue?: OptionValue | OptionValue[] | null
  options?: OptionValue[]
  multiple?: boolean
  allowCustomInput?: boolean
  placeholder?: string
  valueKey?: string
  labelKey?: string
  filterFunction?: (options: OptionValue[], searchTerm: string) => OptionValue[]
}

interface ComboboxEmits {
  'update:modelValue': [value: OptionValue | OptionValue[] | null]
  change: [value: OptionValue | OptionValue[] | null]
  input: [value: string]
}

// Props with defaults
const props = withDefaults(defineProps<ComboboxProps>(), {
  modelValue: () => [],
  options: () => [],
  multiple: false,
  allowCustomInput: true,
  placeholder: 'Select or type...',
  valueKey: undefined,
  labelKey: undefined,
  filterFunction: undefined,
})

// Emits
const emit = defineEmits<ComboboxEmits>()

// Template refs using useTemplateRef
const comboboxRef = useTemplateRef<HTMLDivElement>('comboboxRef')
const inputRef = useTemplateRef<HTMLInputElement>('inputRef')

// Reactive state
const inputValue = ref<string>('')
const isOpen = ref<boolean>(false)
const highlightedIndex = ref<number>(-1)
const dropdownId = ref<string>(`combobox-${Math.random().toString(36).substr(2, 9)}`)

// Computed properties
const selectedItems = computed((): OptionValue[] => {
  if (!props.multiple) {
    return props.modelValue ? [props.modelValue as OptionValue] : []
  }
  return Array.isArray(props.modelValue) ? props.modelValue : []
})

const filteredOptions = computed((): OptionValue[] => {
  if (!inputValue.value.trim()) {
    return props.options
  }

  if (props.filterFunction) {
    return props.filterFunction(props.options, inputValue.value)
  }

  const searchTerm = inputValue.value.toLowerCase().trim()
  return props.options.filter((option: OptionValue) => {
    const label = getItemLabel(option).toLowerCase()
    return label.includes(searchTerm)
  })
})

const exactMatch = computed((): boolean => {
  if (!inputValue.value.trim()) return false
  const searchTerm = inputValue.value.trim().toLowerCase()
  return filteredOptions.value.some(
    (option: OptionValue) => getItemLabel(option).toLowerCase() === searchTerm,
  )
})

// Helper methods
const getItemLabel = (item: OptionValue): string => {
  if (typeof item === 'string' || typeof item === 'number') return String(item)
  if (props.labelKey && typeof item === 'object' && item !== null) {
    return String(item[props.labelKey] || '')
  }
  if (typeof item === 'object' && item !== null && 'label' in item) {
    return String(item.label || '')
  }
  return String(item)
}

const getItemValue = (item: OptionValue): OptionValue => {
  if (props.valueKey && typeof item === 'object' && item !== null) {
    return item[props.valueKey]
  }
  if (typeof item === 'object' && item !== null && 'value' in item) {
    return item.value
  }
  return item
}

const getItemKey = (item: OptionValue, index: number): string => {
  const value = getItemValue(item)
  return `${typeof value === 'object' ? JSON.stringify(value) : value}-${index}`
}

const isSelected = (option: OptionValue): boolean => {
  const optionValue = getItemValue(option)
  if (!props.multiple) {
    const currentValue = props.modelValue ? getItemValue(props.modelValue as OptionValue) : null
    return JSON.stringify(currentValue) === JSON.stringify(optionValue)
  }
  return selectedItems.value.some(
    (item: OptionValue) => JSON.stringify(getItemValue(item)) === JSON.stringify(optionValue),
  )
}

// Selection methods
const selectOption = (option: OptionValue): void => {
  if (!props.multiple) {
    // Single selection - store the full option object to preserve label
    emit('update:modelValue', option)
    emit('change', option)
    inputValue.value = ''
    isOpen.value = false
  } else {
    // Multiple selection - keep dropdown open
    if (isSelected(option)) {
      // Remove if already selected
      const newValue = selectedItems.value.filter(
        (item: OptionValue) =>
          JSON.stringify(getItemValue(item)) !== JSON.stringify(getItemValue(option)),
      )
      emit('update:modelValue', newValue)
      emit('change', newValue)
    } else {
      // Add to selection
      const newValue = [...selectedItems.value, option]
      emit('update:modelValue', newValue)
      emit('change', newValue)
    }
    inputValue.value = ''
    highlightedIndex.value = -1
    // Keep dropdown open for multiple selection
    nextTick(() => {
      inputRef.value?.focus()
    })
  }
}

const selectCustomValue = (): void => {
  const customValue = inputValue.value.trim()
  if (!customValue) return

  if (!props.multiple) {
    emit('update:modelValue', customValue)
    emit('change', customValue)
    inputValue.value = ''
    isOpen.value = false
  } else {
    // Multiple selection - keep dropdown open
    const newValue = [...selectedItems.value, customValue]
    emit('update:modelValue', newValue)
    emit('change', newValue)
    inputValue.value = ''
    highlightedIndex.value = -1
    // Keep dropdown open and maintain focus
    nextTick(() => {
      inputRef.value?.focus()
    })
  }
}

const removeItem = (index: number): void => {
  if (!props.multiple) {
    emit('update:modelValue', null)
    emit('change', null)
  } else {
    const newValue = selectedItems.value.filter((_: OptionValue, i: number) => i !== index)
    emit('update:modelValue', newValue)
    emit('change', newValue)
  }
  nextTick(() => {
    inputRef.value?.focus()
  })
}

// Event handlers
const handleInput = (): void => {
  emit('input', inputValue.value)
  if (!isOpen.value) {
    isOpen.value = true
  }
  highlightedIndex.value = -1
}

const handleFocus = (): void => {
  isOpen.value = true
}

const handleBlur = (): void => {
  console.log('handleBlur') ////

  // Delay closing to allow for option selection
  setTimeout(() => {
    // isOpen.value = false ////
    highlightedIndex.value = -1
    // Clear input value when blurring if not multiple selection
    if (!props.multiple) {
      inputValue.value = ''
      isOpen.value = false ////
    }
  }, 300)
}

const handleKeydown = (event: KeyboardEvent): void => {
  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      if (!isOpen.value) {
        isOpen.value = true
      } else {
        highlightedIndex.value = Math.min(
          highlightedIndex.value + 1,
          filteredOptions.value.length - 1,
        )
      }
      break

    case 'ArrowUp':
      event.preventDefault()
      if (isOpen.value) {
        highlightedIndex.value = Math.max(highlightedIndex.value - 1, -1)
      }
      break

    case 'Enter':
      event.preventDefault()
      if (isOpen.value) {
        if (highlightedIndex.value >= 0 && filteredOptions.value[highlightedIndex.value]) {
          selectOption(filteredOptions.value[highlightedIndex.value])
        } else if (props.allowCustomInput && inputValue.value.trim() && !exactMatch.value) {
          selectCustomValue()
        }
      }
      break

    case 'Escape':
      isOpen.value = false
      highlightedIndex.value = -1
      inputRef.value?.blur()
      break

    case 'Backspace':
      if (!inputValue.value && selectedItems.value.length > 0) {
        removeItem(selectedItems.value.length - 1)
      }
      break
  }
}

const focusInput = (): void => {
  inputRef.value?.focus()
}

// Click outside handler
const handleClickOutside = (event: Event): void => {
  const target = event.target as Node
  if (comboboxRef.value && !comboboxRef.value.contains(target)) {
    isOpen.value = false
    highlightedIndex.value = -1
  }
}

// Lifecycle hooks
onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

// Watchers
watch(highlightedIndex, (newIndex: number) => {
  if (newIndex >= 0 && isOpen.value) {
    nextTick(() => {
      const dropdown = document.getElementById(dropdownId.value)
      const offset = props.allowCustomInput && inputValue.value.trim() && !exactMatch.value ? 1 : 0
      const highlightedEl = dropdown?.children[newIndex + offset] as HTMLElement
      if (highlightedEl) {
        highlightedEl.scrollIntoView({ block: 'nearest' })
      }
    })
  }
})
</script>
