---
sidebar_position: 2
---

# Applications

Manage software applications detected through system inventory and assigned to organizations.

## Overview

Applications in My represent software instances (such as NethVoice, NethSecurity, WebTop) that are detected through system inventory data. Each application is associated with a system and can be assigned to an organization for management purposes.

## Viewing Applications

### Application List

Navigate to **Applications** to see the list of all applications visible to you. The list displays:

- Application type (e.g., NethVoice, NethSecurity, WebTop)
- Version
- Associated system
- Organization

### Filtering

Use the available filters to narrow down the application list:

- **Type**: Filter by application type (NethVoice, NethSecurity, WebTop, etc.)
- **Version**: Filter by specific version
- **System**: Filter by the system the application belongs to
- **Organization**: Filter by organization. An unassigned application belongs to the organization of the system hosting it, so a company's applications show up before any assignment; the same rule drives the application counters on the organization pages

## Application Details

Click on an application to view its detailed information:

- **Type**: The kind of application (e.g., NethVoice, NethSecurity)
- **Version**: The installed version
- **Associated System**: The system where the application is running
- **Organization**: The organization the application belongs to

## Assigning to Organizations

Assignment controls which organization has visibility and management access to the application.

:::note
An application is assigned to **one** organization at a time, not to several. Assigning it elsewhere replaces the previous assignment.
:::

### Assign an Application

1. Navigate to the application details page
2. Use the **Assign** action
3. Select the target organization
4. Confirm the assignment

The organizations on offer are filtered by your position in the hierarchy: you can only assign applications to organizations you manage.

### Unassign an Application

1. Navigate to the application details page
2. Use the **Unassign** action
3. Confirm the removal

An unassigned application does not disappear: it goes back to being counted against the organization of the system hosting it.

## Application Notes

You can add notes to applications to record additional context or operational information:

1. Navigate to the application details page
2. Find the **Notes** section
3. Add or edit your notes
4. Save changes

## Totals and Trends

The [Dashboard](./dashboard.md) displays the total count of applications visible to your account, with a badge linking to the unassigned ones. Growth over time is available from the API through `/backend/api/applications/trend`.

## Permissions

| Operation | Permission | Staff | Admin | Backoffice | Support | Reader |
|-----------|------------|:-----:|:-----:|:----------:|:-------:|:------:|
| View applications | `read:applications` | Yes | Yes | Yes | Yes | Yes |
| Edit an application | `manage:applications` | Yes | Yes | Yes | Yes | No |
| Assign / unassign to an organization | `manage:applications` | Yes | Yes | Yes | Yes | No |
| Edit notes | `manage:applications` | Yes | Yes | Yes | Yes | No |

Every role except Reader holds `manage:applications`, so assigning an
application is not a backoffice-only operation.
