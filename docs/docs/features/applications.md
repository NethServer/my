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
- **Organization**: Filter by organization

## Application Details

Click on an application to view its detailed information:

- **Type**: The kind of application (e.g., NethVoice, NethSecurity)
- **Version**: The installed version
- **Associated System**: The system where the application is running
- **Organization**: The organization the application belongs to

## Assigning to Organizations

Admin users can assign or unassign applications to organizations. This controls which organization has visibility and management access to the application.

### Assign an Application

1. Navigate to the application details page
2. Use the **Assign** action
3. Select the target organization
4. Confirm the assignment

### Unassign an Application

1. Navigate to the application details page
2. Use the **Unassign** action
3. Confirm the removal

## Application Notes

You can add notes to applications to record additional context or operational information:

1. Navigate to the application details page
2. Find the **Notes** section
3. Add or edit your notes
4. Save changes

## Totals and Trends

The [Dashboard](dashboard) displays the total count of applications visible to your account. Trend data shows application growth over 30, 60, and 90 day periods.

## Permissions

| Action | Required Permission |
|--------|-------------------|
| View applications | `read:applications` |
| Edit / Assign / Unassign | `manage:applications` |
