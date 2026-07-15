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
const IMPERSONATE_USERS = 'impersonate:users'
const READ_SYSTEMS = 'read:systems'
const MANAGE_SYSTEMS = 'manage:systems'
const READ_APPLICATIONS = 'read:applications'
const MANAGE_APPLICATIONS = 'manage:applications'
const DESTROY_DISTRIBUTORS = 'destroy:distributors'
const DESTROY_RESELLERS = 'destroy:resellers'
const DESTROY_CUSTOMERS = 'destroy:customers'
const DESTROY_USERS = 'destroy:users'
const DESTROY_SYSTEMS = 'destroy:systems'
const READ_ALERTS = 'read:alerts'
const MANAGE_ALERTS = 'manage:alerts'
const READ_ENTITLEMENTS = 'read:entitlements'
const MANAGE_ENTITLEMENTS = 'manage:entitlements'

export const canReadEntitlements = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(READ_ENTITLEMENTS)
}

export const canManageEntitlements = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(MANAGE_ENTITLEMENTS)
}

// Administrative surface (catalog, manual grants, fleet view): owner org or
// Super Admin only — matches the backend isEntitlementAdmin gate.
export const isEntitlementAdmin = () => {
  const loginStore = useLoginStore()
  return (
    loginStore.userInfo?.org_role === 'Owner' ||
    (loginStore.userInfo?.user_roles ?? []).includes('Super Admin')
  )
}

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
  return loginStore.permissions.includes(IMPERSONATE_USERS)
}

export const canReadSystems = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(READ_SYSTEMS)
}

export const canManageSystems = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(MANAGE_SYSTEMS)
}

export const canReadApplications = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(READ_APPLICATIONS)
}

export const canManageApplications = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(MANAGE_APPLICATIONS)
}

export const canDestroyDistributors = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(DESTROY_DISTRIBUTORS)
}

export const canDestroyResellers = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(DESTROY_RESELLERS)
}

export const canDestroyCustomers = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(DESTROY_CUSTOMERS)
}

export const canDestroyUsers = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(DESTROY_USERS)
}

export const canDestroySystems = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(DESTROY_SYSTEMS)
}

export const canManageAlerts = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(MANAGE_ALERTS)
}

export const canReadAlerts = () => {
  const loginStore = useLoginStore()
  return loginStore.permissions.includes(READ_ALERTS)
}
