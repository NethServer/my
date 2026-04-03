---
sidebar_position: 6
---

# Data Import

Import data in bulk from CSV files to create organizations and users.

## Overview

My allows you to import distributors, resellers, customers, and users from CSV files. The import process uses a two-step flow: **validate** first, then **confirm**. This ensures you can review and fix any issues before data is created.

## Supported Imports

The following resources can be imported:

- **Distributors** -- Name, VAT, contact details
- **Resellers** -- Name, VAT, contact details
- **Customers** -- Name, VAT, contact details
- **Users** -- Email, name, organization, roles

## How to Import

### Step 1: Download the Template

Click the **Import** button and select **Download Template**. The CSV template contains the correct headers and example rows to guide you.

### Step 2: Fill in the CSV

Open the template in a spreadsheet application and fill in your data. Each row represents one entity to create.

:::tip
Keep the header row exactly as provided. Do not rename, reorder, or remove columns.
:::

### Step 3: Upload and Validate

Upload your CSV file. The system validates every row and returns a detailed report:

- **Valid** rows are ready to be imported
- **Error** rows have field-level issues (e.g., missing required fields, invalid email format)
- **Duplicate** rows match existing records in the system

Review the validation report carefully before proceeding.

### Step 4: Confirm the Import

Once you are satisfied with the validation results, confirm the import. Only valid rows are created -- error and duplicate rows are automatically skipped. You can also manually exclude specific valid rows before confirming.

### Step 5: Review Results

After confirmation, a summary shows how many records were created, skipped, or failed. If any rows failed during creation, the error details are provided for each one.

## CSV Format

### Organization Columns (Distributors, Resellers, Customers)

| Column | Required | Description |
|--------|----------|-------------|
| `name` | Yes | Organization name (max 255 characters) |
| `description` | No | Organization description |
| `vat` | Yes | VAT number |
| `address` | No | Street address |
| `city` | No | City |
| `main_contact` | No | Primary contact person |
| `email` | No | Contact email (must be valid if provided) |
| `phone` | No | Phone number (international format if provided) |
| `language` | No | Language code: `it` or `en` (defaults to `it`) |
| `notes` | No | Additional notes |

### User Columns

| Column | Required | Description |
|--------|----------|-------------|
| `email` | Yes | User email address (must be unique) |
| `name` | Yes | Full name (max 255 characters) |
| `phone` | No | Phone number (international format if provided) |
| `organization` | Yes | Organization name (must exist and be in your hierarchy) |
| `roles` | Yes | Role names separated by `;` (e.g., `Admin;Support`) |

:::note
When importing users, the organization is matched **by name** within your visible hierarchy. If the organization name does not exist or is outside your hierarchy, the row is marked as an error.
:::

## Validation Rules

The following checks are performed during validation:

- **Required fields** -- Marked fields cannot be empty
- **Format validation** -- Email addresses, phone numbers, and language codes are checked
- **Duplicate detection (within CSV)** -- Duplicate names (organizations) or emails (users) within the same file are flagged
- **Duplicate detection (database)** -- Names or emails that already exist in the system are flagged
- **Organization resolution** -- For user imports, organization names are resolved to existing organizations within your hierarchy
- **Role resolution** -- For user imports, role names are matched against available roles
- **Permission checks** -- For user imports, each row is checked against your RBAC permissions

## Import Limits

- Maximum **1,000 rows** per CSV file
- Maximum **10 MB** file size
- Supported encodings: UTF-8, UTF-8 with BOM, Latin-1

## Permissions

Import availability depends on your organization role and permissions:

### Organization Import

| Resource | Who Can Import |
|----------|---------------|
| Distributors | Owner |
| Resellers | Owner, Distributor |
| Customers | Owner, Distributor, Reseller |

### User Import

User import requires the `manage:users` permission. The organization field in each CSV row is validated against your hierarchy -- you can only import users into organizations you manage.

:::warning
The import session expires after **30 minutes**. If you do not confirm within this time, you need to re-upload and validate the CSV file.
:::

## User Welcome Emails

When importing users, the system automatically sends a welcome email to each created user with their temporary password and login instructions. This uses the same email flow as single user creation.
