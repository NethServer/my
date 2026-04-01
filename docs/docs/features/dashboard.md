---
sidebar_position: 1
---

# Dashboard

The Dashboard is the main landing page after logging in to My platform. It provides an at-a-glance overview of your managed entities and their status.

## Overview

The Dashboard displays summary counter cards for each entity type visible to the current user. The cards shown depend on your organization role and permissions, ensuring you only see data relevant to your scope.

## Counter Cards

Each counter card shows the total count for an entity type and links directly to its respective list page.

### Distributors

- **Visible to:** Owner only
- Shows the total number of distributor organizations
- Click to navigate to the Distributors list

### Resellers

- **Visible to:** Owner, Distributor
- Shows the total number of reseller organizations within your hierarchy
- Click to navigate to the Resellers list

### Customers

- **Visible to:** Owner, Distributor, Reseller
- Shows the total number of customer organizations within your hierarchy
- Click to navigate to the Customers list

### Users

- **Visible to:** All roles
- Shows the total number of users across your accessible organizations
- Click to navigate to the Users list

### Systems

- **Visible to:** All roles with `read:systems` permission
- Shows the total number of systems across your accessible organizations
- Click to navigate to the Systems list

### Applications

- **Visible to:** All roles with `read:applications` permission
- Shows the total number of applications across your accessible organizations
- Click to navigate to the Applications list

## Trend Analysis

Entity counters include trend data showing growth over configurable time periods:

- **30 days**: Short-term growth
- **60 days**: Medium-term growth
- **90 days**: Quarterly growth

Trend information helps you understand how your managed entities are growing over time.

## Visibility Rules

The Dashboard respects the full authorization model:

- **Owner** users see all entity types (distributors, resellers, customers, users, systems, applications)
- **Distributor** users see resellers, customers, users, systems, and applications within their hierarchy
- **Reseller** users see customers, users, systems, and applications within their hierarchy
- **Customer** users see users, systems, and applications within their own organization

Counter values reflect only the entities within your organizational scope -- you never see data from outside your hierarchy branch.
