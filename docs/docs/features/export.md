---
sidebar_position: 5
---

# Data Export

Export data from list views in CSV or PDF format for reporting and analysis.

## Overview

My allows you to export data from any list view in the platform. Exports respect your current filters, so you can narrow down the data before exporting.

## Supported Exports

The following lists can be exported:

- **Users list**
- **Systems list**
- **Distributors list**
- **Resellers list**
- **Customers list**

## How to Export

1. Navigate to the list page you want to export (e.g., **Users**, **Systems**)
2. Apply any filters if needed (the export respects your current filter selection)
3. Click the **Export** button
4. Choose the format:
   - **CSV** -- Comma-separated values, suitable for spreadsheets and data analysis
   - **PDF** -- Portable document format, suitable for printing and sharing
5. The file downloads automatically to your device

:::tip
Apply filters before exporting to get exactly the data you need. For example, filter systems by organization or status to export only a subset.
:::

## Export Limits

- Maximum **10,000 records** per export
- If your filtered result exceeds this limit, narrow your filters to reduce the dataset

## Permissions

Export availability is based on your read permissions for each resource:

| Resource | Required Permission |
|----------|-------------------|
| Users | `read:users` |
| Systems | `read:systems` |
| Distributors | `read:distributors` |
| Resellers | `read:resellers` |
| Customers | `read:customers` |

If you can see a list, you can export it.
