//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { useLoginStore } from '@/stores/login'

const READ_DISTRIBUTORS = 'read:distributors'
const MANAGE_DISTRIBUTORS = 'manage:distributors'
const READ_RESELLERS = 'read:resellers'
const MANAGE_RESELLERS = 'manage:resellers'
const READ_CUSTOMERS = 'read:customers'
const MANAGE_CUSTOMERS = 'manage:customers'
const READ_USERS = 'read:users'
const MANAGE_USERS = 'manage:users'

export const canReadDistributors = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(READ_DISTRIBUTORS)
}

export const canManageDistributors = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(MANAGE_DISTRIBUTORS)
}

export const canReadResellers = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(READ_RESELLERS)
}

export const canManageResellers = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(MANAGE_RESELLERS)
}

export const canReadCustomers = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(READ_CUSTOMERS)
}

export const canManageCustomers = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(MANAGE_CUSTOMERS)
}

export const canReadUsers = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(READ_USERS)
}

export const canManageUsers = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(MANAGE_USERS)
}

export const canImpersonateUsers = () => {
  const loginStore = useLoginStore()
  // Only owner organization users can impersonate
  return loginStore.userInfo?.org_role === 'Owner'
}
