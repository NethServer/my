import type {ReactNode} from 'react';
import Heading from '@theme/Heading';
import Translate, {translate} from '@docusaurus/Translate';
import styles from './styles.module.css';

type FeatureItem = {
  icon: string;
  title: ReactNode;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    icon: '🔐',
    title: <Translate id="homepage.auth.title">Centralized Authentication</Translate>,
    description: (
      <Translate id="homepage.auth.description">
        Built on Logto as Identity Provider with JWT-based authentication, token
        exchange, and multi-factor authentication support across all services.
      </Translate>
    ),
  },
  {
    icon: '🏢',
    title: <Translate id="homepage.hierarchy.title">Business Hierarchy</Translate>,
    description: (
      <Translate id="homepage.hierarchy.description">
        Multi-tenant organization model with four levels: Owner, Distributor,
        Reseller, and Customer. Each level manages the entities below it.
      </Translate>
    ),
  },
  {
    icon: '🛡️',
    title: <Translate id="homepage.rbac.title">Role-Based Access</Translate>,
    description: (
      <Translate id="homepage.rbac.description">
        Dual-role RBAC combining organization roles for business hierarchy with
        user roles for technical capabilities like Admin and Support.
      </Translate>
    ),
  },
  {
    icon: '📊',
    title: <Translate id="homepage.monitoring.title">System Monitoring</Translate>,
    description: (
      <Translate id="homepage.monitoring.description">
        Heartbeat tracking classifies systems as active, inactive or never seen.
        Inventory collection captures system state with worker-based processing.
      </Translate>
    ),
  },
  {
    icon: '🔍',
    title: <Translate id="homepage.changes.title">Change Detection</Translate>,
    description: (
      <Translate id="homepage.changes.description">
        Automatic diff analysis between inventory snapshots, with a severity per
        change (info, warning, critical) and change notifications.
      </Translate>
    ),
  },
  {
    icon: '👤',
    title: <Translate id="homepage.selfservice.title">Self-Service</Translate>,
    description: (
      <Translate id="homepage.selfservice.description">
        Users manage their own profile, avatar, and password. Operators access
        systems directly via browser-based support sessions or native SSH.
      </Translate>
    ),
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
    <section
      className={styles.features}
      aria-label={translate({
        id: 'homepage.features.ariaLabel',
        message: 'Main features',
      })}>
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
