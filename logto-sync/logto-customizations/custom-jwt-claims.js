/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

/*
 * This function adds custom claims to JWT tokens issued by Logto.
 * It extracts user roles, scopes, and organization-based permissions
 * to align with the backend RBAC system.
 */
const getCustomJwtClaims = async ({ user, token, context }) => {
  try {
    // 1. Get user roles with their permissions/scopes
    const userRoles = await context.getUserRoles(user.id);

    // 2. Get organization memberships
    const orgMemberships = await context.getUserOrganizations(user.id);

    // 3. Extract user role names
    const roles = userRoles.map(role => role.name);

    // 4. Extract scopes from user roles
    const scopes = userRoles.flatMap(role =>
      role.scopes?.map(scope => scope.name) || []
    );

    // 5. Extract organization role names
    const organizationRoles = orgMemberships.flatMap(membership =>
      membership.organizationRoles?.map(role => role.name) || []
    );

    // 6. Extract organization scopes
    const organizationScopes = orgMemberships.flatMap(membership =>
      membership.organizationRoles?.flatMap(role =>
        role.scopes?.map(scope => scope.name) || []
      ) || []
    );

    return {
      roles,
      scopes,
      organization_roles: organizationRoles,
      organization_scopes: organizationScopes
    };

  } catch (error) {
    console.error('Custom JWT claims error:', error);
    return {}; // Empty fallback
  }
};

module.exports = { getCustomJwtClaims };