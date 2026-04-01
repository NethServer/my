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
      ],
    },
    {
      type: 'category',
      label: 'Features',
      items: [
        'features/dashboard',
        'features/applications',
        'features/avatar',
        'features/rebranding',
        'features/export',
      ],
    },
    'contributing',
  ],
};

export default sidebars;
