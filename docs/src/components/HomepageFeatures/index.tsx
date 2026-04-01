import type {ReactNode} from 'react';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  icon: string;
  title: string;
  description: string;
};

const FeatureList: FeatureItem[] = [
  {
    icon: '\uD83D\uDD10',
    title: 'Centralized Authentication',
    description:
      'Built on Logto as Identity Provider with JWT-based authentication, token exchange, and multi-factor authentication support across all services.',
  },
  {
    icon: '\uD83C\uDFE2',
    title: 'Business Hierarchy',
    description:
      'Multi-tenant organization model with four levels: Owner, Distributor, Reseller, and Customer. Each level manages the entities below it.',
  },
  {
    icon: '\uD83D\uDEE1\uFE0F',
    title: 'Role-Based Access',
    description:
      'Dual-role RBAC combining organization roles for business hierarchy with user roles for technical capabilities like Admin and Support.',
  },
  {
    icon: '\uD83D\uDCCA',
    title: 'System Monitoring',
    description:
      'Heartbeat tracking classifies systems as alive, dead, or zombie. Inventory collection captures system state with worker-based processing.',
  },
  {
    icon: '\uD83D\uDD0D',
    title: 'Change Detection',
    description:
      'Automatic diff analysis between inventory snapshots with configurable severity levels (info, warning, critical) and change notifications.',
  },
  {
    icon: '\uD83D\uDC64',
    title: 'Self-Service',
    description:
      'Users manage their own profile, avatar, and password. Operators access systems directly via browser-based support sessions or native SSH.',
  },
];

function Feature({icon, title, description}: FeatureItem): ReactNode {
  return (
    <div className={styles.featureCard}>
      <span className={styles.featureIcon} role="img" aria-hidden="true">
        {icon}
      </span>
      <Heading as="h3" className={styles.featureTitle}>
        {title}
      </Heading>
      <p className={styles.featureDescription}>{description}</p>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className={styles.featureGrid}>
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
