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
| Create Distributors | ✅ | ❌ | ❌ | ❌ |
| Manage Distributors | ✅ | ❌ | ❌ | ❌ |
| Create Resellers | ✅ | ✅ | ❌ | ❌ |
| Manage Resellers | ✅ | ✅ (own) | ❌ | ❌ |
| Create Customers | ✅ | ✅ | ✅ | ❌ |
| Manage Customers | ✅ | ✅ (own) | ✅ (own) | ❌ |
| View All Data | ✅ | ❌ | ❌ | ❌ |

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
4. Click **Create distributor**

**Example:**
```
Name: ACME Distribution Europe
Description: Main distributor for European market
VAT: 12345678901
```

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

**Note:** If you're logged in as a Distributor, you can only create resellers under your own organization.

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

**⚠️ Warning:** Deleting an organization is permanent and will:
- Remove all users in that organization
- Delete all systems associated with it
- Remove all child organizations (cascade delete)

To delete an organization:

1. Navigate to the organization page
2. Click **Delete** (use the kebab menù)
4. Click **Delete**

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

## Organization Statistics

### Viewing Statistics

Navigate to **Dashboard** to see:

- **Distributors Overview**:
  - Total number of distributors
  - Active vs. suspended
  - Trend graph (last 30/60/90 days)

- **Resellers Overview**:
  - Total resellers per distributor
  - Active vs. suspended
  - Trend graph

- **Customers Overview**:
  - Total customers
  - Distribution by reseller/distributor
  - Growth trend

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

**Problem:** Delete button is disabled or shows error

**Solutions:**
- Remove all systems from the organization first
- Delete all child organizations first
- Check if you have permission to delete
- Ensure the organization is not the Owner organization

## Next Steps

After creating organizations:

- [Create users](03-users.md) and assign them to organizations
- [Create systems](04-systems.md) associated with customer organizations
- Set up appropriate permissions for each user

## Related Documentation

- [Users Management](03-users.md)
- [Systems Management](04-systems.md)
