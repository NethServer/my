//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { PciDevice, Distro, Memory, Product, Processors } from './inventory'

export type { PciDevice }

export interface UserDomain {
  name: string
  location: string
  protocol: string
  schema: string
  providers_count: number
  active_users: number
  total_users: number
  total_groups: number
}

export interface Ns8ClusterFacts {
  leader_node_id: string
  user_domains: UserDomain[]
  subscription: string
  admins: {
    count: number
    '2fa': number
  }
  repositories: string[]
  smarthost: {
    enabled: boolean
    manual_configuration: boolean
  }
  backup: {
    backup_count: number
    destination_count: number
    destination_providers: string[]
  }
  update_disabled: boolean
  update_disabled_reason: string
  update_schedule_active: boolean
  ui_name: string
}

export interface Ns8NodeFacts {
  cluster_leader: boolean
  fqdn: string
  default_ipv4: string
  default_ipv6: string
  kernel_version: string
  uptime_seconds: number
  timezone: string
  volumesconf_mountpoint_count: number
  volumesconf_application_types: string[]
  version: string
  distro: Distro
  processors: Processors
  product: Product
  virtual: string
  memory: Memory
  pci: PciDevice[]
  ui_name: string
  update_available: boolean
}

export interface Ns8ModuleFacts {
  id: string
  version: string
  module: string
  name: string
  source: string
  node: string
  certification_level: number
  update_available: boolean
  user_domains: string[]
  fqdns: string[]
  ui_name: string
  // samba
  server_role?: string
  provision_type?: string
  has_nbalias?: boolean
  has_file_server_flag?: boolean
  shared_folders_count?: number
  description_count?: number
  audit_enabled_count?: number
  audit_failed_events_count?: number
  recycle_enabled_count?: number
  recycle_retention_count?: number
  recycle_versions_count?: number
  browseable_count?: number
  acls_length_max?: number
  acls_length_min?: number
  // traefik
  custom_path_routes?: number
  custom_host_routes?: number
  custom_certificates?: number
  acme_manual_certificates?: number
  acme_auto_certificates?: number
  acme_failed_certificates?: number
  name_module_map?: Record<string, string>
  // loki
  retention_days?: number
  cloud_log_manager?: boolean
  syslog?: boolean
  // imapsync
  tasks_total_count?: number
  tasks_delete_count?: number
  tasks_delete_older_count?: number
  tasks_inbox_count?: number
  tasks_inbox_and_delete_count?: number
  tasks_cron_enabled_count?: number
  tasks_sieve_enabled_count?: number
  // mattermost, nethvoice, webtop, piler
  active_users?: number
  total_users?: number
  // mail
  addresses_total_count?: number
  adduser_domains_count?: number
  addgroup_domains_count?: number
  addresses_wildcard_count?: number
  addresses_adduser_count?: number
  destinations_public_count?: number
  filter?: {
    antivirus: {
      enabled: boolean
      clamav_official_sigs: boolean
      third_party_sigs_rating: string
      memory_info: {
        installed: number
        available: number
        recommended: number
      }
    }
    antispam: {
      enabled: boolean
      spam_flag_threshold: number
      deny_message_threshold: number
      greylist: {
        enabled: boolean
        threshold: number
      }
    }
  }
  bypass_rules_count?: number
  mailboxes_total_count?: number
  mailboxes_custom_spam_retention_count?: number
  mailboxes_custom_quota_count?: number
  mailboxes_forward_total_count?: number
  mailboxes_forward_keepcopy_count?: number
  mailboxes_disabled_count?: number
  public_mailboxes_total_count?: number
  domains_total_count?: number
  domains_description_count?: number
  relay_rules_total_count?: number
  relay_rules_tls_count?: number
  relay_rules_has_password_count?: number
  relay_rules_enabled_count?: number
  relay_settings_postfix_restricted_sender_enabled?: boolean
  relay_settings_networks_count?: number
  always_bcc_enabled?: boolean
  configuration_user_domain_schema?: string
  mailbox_settings_spam_prefix_enabled?: boolean
  mailbox_settings_sharedseen_enabled?: boolean
  mailbox_settings_spam_folder_enabled?: boolean
  mailbox_settings_spam_retention_value?: number
  mailbox_settings_quota_enabled?: boolean
  master_users_count?: number
  queue_settings_maximal_queue_lifetime?: string
}

export interface Ns8Facts {
  cluster: Ns8ClusterFacts
  nodes: Record<string, Ns8NodeFacts>
  modules: Ns8ModuleFacts[]
}
