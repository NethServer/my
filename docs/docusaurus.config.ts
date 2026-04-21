import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'My',
  tagline: 'Centralized management platform by Nethesis',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  url: 'https://nethserver.github.io',
  baseUrl: '/my/',

  organizationName: 'NethServer',
  projectName: 'my',

  onBrokenLinks: 'throw',
  markdown: {
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  themes: ['@docusaurus/theme-mermaid'],

  i18n: {
    defaultLocale: 'en',
    locales: ['en', 'it'],
    localeConfigs: {
      en: {
        label: 'English',
      },
      it: {
        label: 'Italiano',
      },
    },
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/NethServer/my/tree/main/docs/',
          routeBasePath: 'docs',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: 'img/logo.svg',
    colorMode: {
      defaultMode: 'light',
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: '',
      logo: {
        alt: 'My Nethesis',
        src: 'img/logo-dark.svg',
        srcDark: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'guideSidebar',
          position: 'left',
          label: 'User Guide',
        },
        {
          type: 'dropdown',
          position: 'left',
          label: 'Developer Docs',
          items: [
            {
              label: 'Project Overview',
              href: 'https://github.com/NethServer/my/blob/main/README.md',
            },
            {
              label: 'Backend API',
              href: 'https://api.my.nethesis.it/',
            },
            {
              label: 'Collect Service',
              href: 'https://github.com/NethServer/my/blob/main/collect/README.md',
            },
            {
              label: 'Alerting System',
              href: 'https://github.com/NethServer/my/blob/main/services/mimir/README.md',
            },
            {
              label: 'Sync Tool',
              href: 'https://github.com/NethServer/my/blob/main/sync/README.md',
            },
          ],
        },
        {
          type: 'localeDropdown',
          position: 'right',
        },
        {
          href: 'https://github.com/NethServer/my',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Documentation',
          items: [
            {
              label: 'User Guide',
              to: '/docs/intro',
            },
            {
              label: 'Authentication',
              to: '/docs/getting-started/authentication',
            },
            {
              label: 'Systems',
              to: '/docs/systems/management',
            },
          ],
        },
        {
          title: 'Developer',
          items: [
            {
              label: 'Backend API',
              href: 'https://api.my.nethesis.it/',
            },
            {
              label: 'Collect Service',
              href: 'https://github.com/NethServer/my/blob/main/collect/README.md',
            },
            {
              label: 'Alerting System',
              href: 'https://github.com/NethServer/my/blob/main/services/mimir/README.md',
            },
            {
              label: 'Sync Tool',
              href: 'https://github.com/NethServer/my/blob/main/sync/README.md',
            },
          ],
        },
        {
          title: 'Community',
          items: [
            {
              label: 'GitHub',
              href: 'https://github.com/NethServer/my',
            },
            {
              label: 'Nethesis',
              href: 'https://www.nethesis.it',
            },
          ],
        },
      ],
      copyright: `Copyright \u00a9 ${new Date().getFullYear()} Nethesis S.r.l.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'json', 'python', 'go'],
    },
    mermaid: {
      theme: {light: 'neutral', dark: 'dark'},
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
