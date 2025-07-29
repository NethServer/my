//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'

export const ProfileInfoSchema = v.object({
  name: v.pipe(v.string(), v.nonEmpty('users.name_required')),
  email: v.pipe(v.string(), v.nonEmpty('users.email_required'), v.email('users.email_invalid')),
  phone: v.optional(
    v.pipe(v.string(), v.regex(/^\+?[\d\s\-\(\)]{7,20}$/, 'users.phone_invalid_format')),
  ),
})

export const ChangePasswordSchema = v.pipe(
  v.object({
    current_password: v.pipe(v.string(), v.nonEmpty('account.current_password_required')),
    new_password: v.pipe(v.string(), v.nonEmpty('account.new_password_required')),
    confirm_password: v.pipe(v.string(), v.nonEmpty('account.confirm_password_required')),
  }),
  v.forward(
    v.partialCheck(
      [['new_password'], ['confirm_password']],
      (input) => input.new_password === input.confirm_password,
      'account.passwords_do_not_match',
    ),
    ['confirm_password'],
  ),
  v.forward(
    v.partialCheck(
      [['current_password'], ['new_password']],
      (input) => input.current_password !== input.new_password,
      'account.new_password_must_be_different_from_current',
    ),
    ['new_password'],
  ),
)

export type ProfileInfo = v.InferOutput<typeof ProfileInfoSchema>
export type ChangePassword = v.InferOutput<typeof ChangePasswordSchema>

export const postChangeInfo = (profileInfo: ProfileInfo) => {
  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/me/change-info`, profileInfo, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const postChangePassword = (changePasswordData: ChangePassword) => {
  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/me/change-password`, changePasswordData, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}
