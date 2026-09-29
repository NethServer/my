---
sidebar_position: 5
---

# Data Export

Export data from list views in CSV or PDF format for reporting and analysis.

## Overview

My allows you to export data from any list view in the platform. Exports respect your current filters, so you can narrow down the data before exporting.

## Supported Exports

| Resource | Formats | Contents |
|----------|---------|----------|
| **Distributors** | CSV, PDF | Distributor list with resellers, customers and systems counters |
| **Resellers** | CSV, PDF | Reseller list with customers and systems counters |
| **Customers** | CSV, PDF | Customer list with systems counter |
| **Users** | CSV, PDF | User list with roles and status |
| **Systems** | CSV, PDF | System list with status and last heartbeat |
| **Applications** | CSV, PDF | Application list with type, version, hosting system and company |

## How to Export

1. Navigate to the list page you want to export (e.g. **Users**, **Systems**)
2. Apply any filters if needed -- the export contains exactly the rows the filters select
3. Click the **Export** button and choose the format:
   - **CSV** -- tabular, for spreadsheets and data analysis
   - **PDF** -- document, for printing and sharing
4. The file is generated and downloaded by the browser, named after the list and
   the day of the export: `users-2026-09-29.pdf`, `systems-2026-09-29.csv`

:::tip
Apply filters before exporting to get exactly the data you need. For example, filter systems by organization or status to export only a subset, or filter applications by a company and its whole hierarchy to export everything installed for that partner.
:::

## CSV Format

- **Separator**: comma (`,`)
- **Encoding**: UTF-8
- **Headers**: first row carries the column names
- **Escaping**: double quotes around fields that contain a comma

## PDF Format

- **Landscape table**, one row per record, with the same layout for every resource
- **Header** on every page with the logo, the record count, the generation timestamp,
  **who** ran the export and the **filters** that were applied
- **Status** values are colour-coded (green active, amber inactive or unassigned,
  red suspended or deleted)
- Long values are cut with an ellipsis so each record stays on one line: the CSV
  carries the full text
- **Page numbers** in the footer

## Export Limits

- Maximum **10,000 records** per export
- Beyond that the export is **refused**: My tells you how many records the
  current filters match and asks you to narrow them. No partial file is
  produced.

## Permissions

Export requires the **read permission of the resource** -- the same one that
lets you see the list:

| Resource | Required permission |
|----------|---------------------|
| Users | `read:users` |
| Systems | `read:systems` |
| Distributors | `read:distributors` |
| Resellers | `read:resellers` |
| Customers | `read:customers` |
| Applications | `read:applications` |

If you can see a list, you can export it -- and only that. A Support user, for
instance, holds no `read:users`, so it cannot export users.

Exported data always follows hierarchical visibility: Owner exports the whole
platform, a distributor its own sub-organizations, a reseller its customers, a
customer only its own organization. It is never possible to export data you
cannot reach in the interface.
