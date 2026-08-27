//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { faArrowUpRightFromSquare, faBan, faCircleCheck } from '@fortawesome/free-solid-svg-icons'
import type { NeDropdownItem } from '@nethesis/vue-components'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AddonAction } from '@/components/systems/addons/AddonActionModal.vue'
import { getBuyUrl, type SystemAddonRow } from '@/lib/addons/systemAddons'
import { formatDateNoTime } from '@/lib/dateTime'
import { canBuyAddons, isAddonAdmin } from '@/lib/permissions'
import { useSystemDetail } from '@/queries/systems/systemDetail'

// The actions of one row, split the way they are rendered: buying gets a button
// of its own, the administrative actions live in a kebab.
export type AddonRowActions = {
  buy: NeDropdownItem | undefined
  menu: NeDropdownItem[]
}

// What a viewer may do with one add-on of one system, and how a grant reads.
// Shared by the NethSecurity card and the NethServer detail table so the rules
// cannot drift apart: which revocations can be undone by buying again, and what
// a suspended system takes off the table, are easy to get subtly wrong twice.
//
// The modal stays with whoever renders it — the caller passes onSelect and gets
// told which action was chosen.
export const useSystemAddonActions = () => {
  const { t, locale } = useI18n()
  const { state: systemDetail } = useSystemDetail()

  // A suspended or deleted system cannot use anything it holds — collect turns
  // its credentials away before it ever looks at the grants — so buying more is
  // pointless while it stays that way.
  const isSystemBlocked = computed(() =>
    ['suspended', 'deleted'].includes(systemDetail.value.data?.status ?? ''),
  )

  // Nothing is offered on a blocked system, and nothing can be bought that the
  // company is not allowed to have in the first place.
  const canBuy = (row: SystemAddonRow) => {
    if (!canBuyAddons() || isSystemBlocked.value) {
      return false
    }
    // an add-on that is not on sale keeps its card — a system may well hold it
    // already — but there is nowhere to send the buyer
    if (!row.addon.purchasable) {
      return false
    }
    if (!row.grant) {
      return true
    }
    // a shop-side revocation (cancelled subscription, failed payment) can be
    // undone by buying again; a deliberate one cannot
    return (
      row.grant.status === 'expired' ||
      (row.grant.status === 'revoked' && row.grant.revoked_source === 'shop')
    )
  }

  const openShop = (row: SystemAddonRow) => {
    window.open(getBuyUrl(systemDetail.value.data?.system_key ?? '', row), '_blank')
  }

  const getKebabMenuItems = (
    row: SystemAddonRow,
    onSelect: (action: AddonAction) => void,
  ): NeDropdownItem[] => {
    const items: NeDropdownItem[] = []

    if (!isAddonAdmin() || isSystemBlocked.value) {
      return items
    }

    if (!row.grant) {
      items.push({
        id: 'activate',
        label: t('addons.activate'),
        icon: faCircleCheck,
        action: () => onSelect('activate'),
      })
    } else if (row.grant.status === 'active') {
      items.push({
        id: 'revoke',
        label: t('addons.revoke'),
        icon: faBan,
        danger: true,
        action: () => onSelect('revoke'),
      })
    } else if (row.grant.status === 'revoked') {
      // restoring only clears the revocation: on an expired grant it would
      // change nothing, so it is not offered there
      items.push({
        id: 'reactivate',
        label: t('addons.reactivate'),
        icon: faCircleCheck,
        action: () => onSelect('reactivate'),
      })
    }

    return items
  }

  // Only one side is ever filled: buying needs manage:addons WITHOUT owner-level
  // authority (canBuyAddons), the administrative three need exactly that
  // authority (isAddonAdmin). Callers render the two independently all the
  // same, so an action added to the other side appears beside it instead of
  // being silently dropped.
  const getAddonActions = (
    row: SystemAddonRow,
    onSelect: (action: AddonAction) => void,
  ): AddonRowActions => ({
    buy: canBuy(row)
      ? {
          id: 'buy',
          label: t('addons.buy'),
          icon: faArrowUpRightFromSquare,
          action: () => openShop(row),
        }
      : undefined,
    menu: getKebabMenuItems(row, onSelect),
  })

  // Dates only: a licence period is counted in days, so the time of day is noise.
  const formatValidity = (row: SystemAddonRow) => {
    // A grant awaiting payment has no period yet — the backend marks the stub by
    // setting valid_until to valid_from, which would otherwise read as a licence
    // that expired the day it started.
    if (!row.grant || row.grant.status === 'pending') {
      return '-'
    }
    const from = formatDateNoTime(new Date(row.grant.valid_from), locale.value)
    const until = row.grant.valid_until
      ? formatDateNoTime(new Date(row.grant.valid_until), locale.value)
      : t('addons.never_expires')

    return `${from} - ${until}`
  }

  return {
    getAddonActions,
    formatValidity,
  }
}
