//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'

export const SYSTEM_BACKUPS_KEY = 'systemBackups'

// ── Types ─────────────────────────────────────────────────────────────────────

export interface BackupMetadata {
  id: string
  filename: string
  size: number
  sha256: string
  mimetype: string
  uploaded_at: string
}

export interface BackupListData {
  backups: BackupMetadata[]
  quota_used_bytes: number
  slots_used: number
}

export interface BackupListResponse {
  code: number
  message: string
  data: BackupListData
}

export interface BackupDownloadData {
  download_url: string
  expires_in_seconds: number
}

export interface BackupDownloadResponse {
  code: number
  message: string
  data: BackupDownloadData
}

// ── Retention policy (mirrors collect defaults; kept in sync with
//    BACKUP_MAX_PER_SYSTEM and BACKUP_MAX_SIZE_PER_SYSTEM). The backend
//    does not currently surface these values so we render them from
//    constants and update them here if the server-side defaults shift. ──

export const BACKUP_MAX_SIZE_PER_SYSTEM = 500 * 1024 * 1024

// ── API ───────────────────────────────────────────────────────────────────────

export const getSystemBackups = (systemId: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<BackupListResponse>(`${API_URL}/systems/${systemId}/backups`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getBackupDownloadUrl = (systemId: string, backupId: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<BackupDownloadResponse>(
      `${API_URL}/systems/${systemId}/backups/${encodeURIComponent(backupId)}/download`,
      {
        headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
      },
    )
    .then((res) => res.data.data)
}

export const deleteBackup = (systemId: string, backupId: string) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/systems/${systemId}/backups/${encodeURIComponent(backupId)}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}
