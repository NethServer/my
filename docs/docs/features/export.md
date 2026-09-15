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
| **Distributors** | CSV, PDF | Distributor list with details |
| **Resellers** | CSV, PDF | Reseller list with details |
| **Customers** | CSV, PDF | Customer list with details |
| **Users** | CSV, PDF | User list with roles and status |
| **Systems** | CSV, PDF | System list with status and last heartbeat |

## How to Export

1. Navigate to the list page you want to export (e.g. **Users**, **Systems**)
2. Apply any filters if needed -- the export contains exactly the rows the filters select
3. Click the **Export** button and choose the format:
   - **CSV** -- tabular, for spreadsheets and data analysis
   - **PDF** -- document, for printing and sharing
4. The file is generated and downloaded by the browser, named
   `<resource>_export_<YYYY-MM-DD_HHMMSS>.<ext>`

:::tip
Apply filters before exporting to get exactly the data you need. For example, filter systems by organization or status to export only a subset.
:::

## CSV Format

- **Separator**: comma (`,`)
- **Encoding**: UTF-8
- **Headers**: first row carries the column names
- **Escaping**: double quotes around fields that contain a comma

## PDF Format

- **Header** with the generation timestamp
- The **filters** that were applied, and **who** ran the export
- **Table** of formatted data

## Export Limits

- Maximum **10,000 records** per export
- Beyond that the export is **silently truncated**: the file is produced, but the
  rows past the limit are not in it. Narrow your filters to be sure you have
  everything.

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

If you can see a list, you can export it -- and only that. A Support user, for
instance, holds no `read:users`, so it cannot export users.

Exported data always follows hierarchical visibility: Owner exports the whole
platform, a distributor its own sub-organizations, a reseller its customers, a
customer only its own organization. It is never possible to export data you
cannot reach in the interface.
