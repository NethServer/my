import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  guideSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Getting Started',
      collapsed: false,
      items: [
        'getting-started/authentication',
        'getting-started/account',
        'getting-started/api-keys',
      ],
    },
    {
      type: 'category',
      label: 'Platform Management',
      items: [
        'platform/organizations',
        'platform/users',
        'platform/impersonation',
      ],
    },
    {
      type: 'category',
      label: 'Systems',
      items: [
        'systems/management',
        'systems/registration',
        'systems/inventory-heartbeat',
        'systems/backups',
        'systems/org-reassignment',
      ],
    },
    {
      type: 'category',
      label: 'Features',
      items: [
        'features/dashboard',
        'features/applications',
        'features/entitlements',
        'features/avatar',
        'features/rebranding',
        'features/import',
        'features/export',
        'features/alerting',
      ],
    },
    'contributing',
  ],
};

export default sidebars;
