---
sidebar_position: 1
---

# Organizations Management

Learn how to manage the business hierarchy in My platform.

## Understanding Organization Hierarchy

My uses a hierarchical organization structure that reflects the business relationships:

```
Owner (Nethesis)
    ↓
Distributors
    ↓
Resellers
    ↓
Customers
```

### Organization Types

**Owner (Nethesis)**
- Top-level organization
- Complete platform control
- Can manage all distributors, resellers, and customers
- Only one Owner organization exists

**Distributors**
- Created by Owner
- Can manage their resellers and customers
- Cannot see other distributors' data
- Full control over their branch of the hierarchy

**Resellers**
- Created by Owner or Distributors
- Can manage their customers
- Cannot see other resellers' data
- Work within their assigned distributor

**Customers**
- Created by Owner, Distributors, or Resellers
- End-user organizations
- Can only view their own data
- Cannot create sub-organizations

### Permissions by Organization Type

| Action | Owner | Distributor | Reseller | Customer |
|--------|-------|-------------|----------|----------|
| Create Distributors | &#10003; | &#10007; | &#10007; | &#10007; |
| Manage Distributors | &#10003; | &#10007; | &#10007; | &#10007; |
| Create Resellers | &#10003; | &#10003; | &#10007; | &#10007; |
| Manage Resellers | &#10003; | &#10003; (own) | &#10007; | &#10007; |
| Create Customers | &#10003; | &#10003; | &#10003; | &#10007; |
| Manage Customers | &#10003; | &#10003; (own) | &#10003; (own) | &#10007; |
| View All Data | &#10003; | &#10007; | &#10007; | &#10007; |

## Creating Organizations

### Prerequisites

- You must be logged in with appropriate permissions
- Owner users can create all organization types
- Distributor users can create resellers and customers
- Reseller users can create customers only

### Creating a Distributor

**Required Role:** Owner organization member

1. Navigate to **Organizations** > **Distributors**
2. Click **Create distributor**
3. Fill in the form:
   - **Company name**: Distributor company name (e.g., "ACME Distribution Ltd")
   - **Description** (optional): Additional information
   - **VAT number**: unique VAT identification for a company
   - **Portals**: the third-party portals (NethShop, Helpdesk, NethSpot, ...) the distributor's resellers and customers may use, see below
4. Click **Create distributor**

**Example:**
```
Name: ACME Distribution Europe
Description: Main distributor for European market
VAT: 12345678901
Portals: NethShop, NethSpot
```

#### Portals

The portals a partner sees on its [Dashboard](../features/dashboard.md#third-party-applications) are a commercial matter: a distributor's contract says which of them its resellers and customers get. The Owner organization records that choice on the distributor, and the organizations below inherit it:

- The users of the distributor's resellers and of their customers see only the portals ticked on the distributor. Resellers and customers have no portal setting of their own.
- The distributor's own users are not bound by the list: they see every portal their roles admit, like the Owner organization.
- A distributor with no portal ticked hides every portal from its resellers and customers. Nothing is granted by default: a newly created distributor starts with no portal for its subtree until the Owner organization ticks some.
- The role filter each portal declares still applies on top: a portal reserved to Admin and Support users stays hidden from a Reader even when the distributor has it.

Only members of the Owner organization see and edit the field, both when creating and when editing a distributor. The distributor detail card lists the enabled portals.

:::note
Hiding a portal on the Dashboard removes the shortcut, not the portal's own login page. Which portals also refuse the sign-in of a user outside an enabled hierarchy is decided on the identity provider, per application.
:::

### Creating a Reseller

**Required Role:** Owner or Distributor organization member

1. Navigate to **Organizations** > **Resellers**
2. Click **Create reseller**
3. Fill in the form:
   - **Company name**: Reseller company name (e.g., "ACME Distribution Ltd")
   - **Description** (optional): Additional information
   - **VAT number**: unique VAT identification for a company
4. Click **Create reseller**

**Example:**
```
Name: Tech Solutions Italia
Description: IT solutions provider for SMB market
VAT: 12345678901
```

:::note
If you are logged in as a Distributor, you can only create resellers under your own organization.
:::

### Creating a Customer

**Required Role:** Owner, Distributor, or Reseller organization member

1. Navigate to **Organizations** > **Customers**
2. Click **Create customer**
3. Fill in the form:
   - **Company name**: Customer company name (e.g., "ACME Distribution Ltd")
   - **Description** (optional): Additional information
   - **VAT number**: unique VAT identification for a company
4. Click **Create customer**

**Example:**
```
Name: Pizza Express Milano
Description: Restaurant chain with 5 locations
VAT: 12345678901
```

## Viewing Organizations

### Organization List

Each organization type has its own list view:

1. Navigate to **[Type]** (Distributors/Resellers/Customers)
2. View the list with the following information:
   - Organization name
   - Description
   - Number of users
   - Number of systems
   - Creation date

### Filtering and Search

Use the filter options to find specific organizations:

- **Search by name**: Type in the search box
- **Sort by**: Name, description

### Organization Details

Click on an organization to view detailed information:

- **Overview**: Name, description, creation date
- **Users**: All users belonging to this organization
- **Systems**: Systems associated with this organization (if applicable)
- **Statistics**: Usage metrics and activity

## Managing Organizations

### Editing Organization Information

1. Navigate to the organization details page
2. Click **Edit**
3. Update the fields:
   - Company name
   - Description
   - VAT
4. Click **Save [Type]**

### Deleting Organizations

**Delete archives, it does not erase.** The organization is soft-deleted and
disappears from the lists, and the same operation cascades down the hierarchy:

- Every **user** of the organization is archived
- Every **system** of the organization is archived
- For a distributor or a reseller, every **child organization** in its
  hierarchy is archived too, together with their users and systems

To delete an organization:

1. Navigate to the organization page
2. Click **Delete** (use the kebab menu)
3. Confirm

:::tip Reversible
A deleted organization can be brought back with **Restore**, which
cascade-restores the users and systems archived along with it.
:::

:::danger Permanent deletion
**Destroy** is the irreversible one: it erases the organization for good and
cannot be undone. It requires the `destroy:` permission on that resource, which
only the Owner organization holds.
:::

### Suspending Organizations

Instead of deleting, you can suspend an organization:

1. Navigate to the organization page
2. Click **Suspend**
3. Confirm the action

**Effects of suspension:**
- Users cannot log in
- Systems cannot send data
- Can be reactivated later

To reactivate:
1. Filter by "Suspended" status
2. Select the organization
3. Click **Reactivate**

### Promoting a Reseller

A reseller can be promoted to distributor. Use the **Promote** action in the reseller's kebab menu or on its detail card.

The promotion:

- Moves the organization up one tier: it becomes a distributor, attached to the Owner organization
- **Keeps its own customers, users and systems** -- nothing is detached
- **Removes the former distributor's access** to that branch, which is the point of the operation
- Leaves a trace: the organization records that it was promoted, and by whom

Requirements:

- Owner-level authority -- the action is not granted by `manage:resellers`
- The organization must be **active**: a suspended or deleted reseller is rejected
- The organization must already be synchronised with the identity provider

:::warning
Promotion is a hierarchy change, not a cosmetic one. The previous distributor loses visibility on that reseller and on everything beneath it.
:::

## Organization Statistics

### Viewing Statistics

The [Dashboard](../features/dashboard.md) shows one counter card per organization type you can read -- distributors, resellers, customers -- each with its total and a link into the list.

An organization's own detail page carries its aggregate numbers: how many users, systems and sub-organizations hang off it.

Growth over time is not on the Dashboard; it is available from the API through the `/trend` endpoints (`/backend/api/distributors/trend`, `/backend/api/resellers/trend`, `/backend/api/customers/trend`).

### Exporting Data

Export organization data for reporting:

1. Navigate to the organization list
2. Apply filters if needed
3. Click **Export**
4. Choose format: CSV or PDF
5. Download the file

## Best Practices

### Naming Conventions

- Use clear, descriptive names
- Include geographical information if relevant (e.g., "ACME Europe", "Tech Solutions Italia")
- Avoid special characters in names
- Keep names concise but meaningful

### Organization Structure

- Plan your hierarchy before creating organizations
- Keep the structure simple and logical
- Avoid creating unnecessary intermediate levels
- Document the business relationships

### Access Control

- Assign users to the correct organization
- Review organization membership regularly
- Use descriptive organization names for clarity
- Keep contact information up to date

## Troubleshooting

### Cannot Create Organization

**Problem:** "Access denied" error when creating an organization

**Solutions:**
- Verify you have the correct role (Owner/Distributor/Reseller)
- Check you're trying to create the correct organization type
- Ensure your organization membership is correct
- Contact your administrator

### Cannot See Organization

**Problem:** Expected organization not visible in the list

**Solutions:**
- Check if organization is suspended (use filters)
- Verify you have permission to view that organization type
- Ensure you're viewing the correct organization level
- Check if the organization belongs to your hierarchy branch

### Cannot Delete Organization

**Problem:** Delete action is unavailable or returns an error

**Solutions:**
- Check you hold `manage:` on that organization type -- Reader never does
- Check the organization is inside your branch of the hierarchy
- The Owner organization cannot be deleted
- You do **not** need to empty it first: deleting cascades over users, systems and child organizations by itself

## Next Steps

After creating organizations:

- [Create users](./users.md) and assign them to organizations
- [Create systems](../systems/management.md) associated with customer organizations
- Set up appropriate permissions for each user

## Related Documentation

- [Users Management](./users.md)
- [Systems Management](../systems/management.md)
