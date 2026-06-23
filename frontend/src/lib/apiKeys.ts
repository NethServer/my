//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'

export const API_KEYS_KEY = 'apiKeys'
export const API_KEYS_TABLE_ID = 'apiKeysTable'
const API_KEYS_PATH = 'me/api-keys'

export type ApiKeyMode = 'read' | 'write'

export interface ApiKey {
  id: string
  user_id: string
  organization_id: string
  name: string
  key_public: string
  mode: ApiKeyMode
  expires_at: string
  last_used_at: string | null
  last_used_ip: string | null
  revoked_at: string | null
  created_at: string
}

export interface CreatedApiKey extends ApiKey {
  // The full plaintext token, returned only once at creation.
  token: string
}

interface ListApiKeysResponse {
  code: number
  message: string
  data: {
    api_keys: ApiKey[]
  }
}

interface CreateApiKeyResponse {
  code: number
  message: string
  data: CreatedApiKey
}

export const CreateApiKeySchema = v.object({
  name: v.pipe(v.string(), v.nonEmpty('account.api_keys.name_required')),
  mode: v.pipe(v.string(), v.nonEmpty('account.api_keys.mode_required')),
  expires_in_days: v.number(),
  password: v.pipe(v.string(), v.nonEmpty('account.api_keys.password_required')),
})

export type CreateApiKey = v.InferOutput<typeof CreateApiKeySchema>

export const getApiKeys = () => {
  const loginStore = useLoginStore()

  return axios
    .get<ListApiKeysResponse>(`${API_URL}/${API_KEYS_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.api_keys)
}

export const postApiKey = (apiKey: CreateApiKey) => {
  const loginStore = useLoginStore()

  return axios
    .post<CreateApiKeyResponse>(`${API_URL}/${API_KEYS_PATH}`, apiKey, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const deleteApiKey = (id: string) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/${API_KEYS_PATH}/${id}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}
