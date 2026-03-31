//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import LoginRedirectView from '../views/LoginRedirectView.vue'
import LoginView from '../views/LoginView.vue'
import { useLoginStore } from '@/stores/login'
import {
  canReadApplications,
  canReadCustomers,
  canReadDistributors,
  canReadResellers,
  canReadSystems,
  canReadUsers,
} from '@/lib/permissions'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      redirect: '/dashboard',
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: DashboardView,
    },
    {
      path: '/login',
      name: 'login',
      component: LoginView,
    },
    {
      path: '/login-redirect',
      name: 'login_redirect',
      component: LoginRedirectView,
    },
    {
      path: '/access-denied',
      name: 'access_denied',
      component: () => import('../views/AccessDeniedView.vue'),
    },
    {
      path: '/account',
      name: 'account',
      component: () => import('../views/AccountView.vue'),
    },
    {
      path: '/distributors',
      name: 'distributors',
      component: () => import('../views/DistributorsView.vue'),
    },
    {
      path: '/resellers',
      name: 'resellers',
      component: () => import('../views/ResellersView.vue'),
    },
    {
      path: '/customers',
      name: 'customers',
      component: () => import('../views/CustomersView.vue'),
    },
    {
      path: '/users',
      name: 'users',
      component: () => import('../views/UsersView.vue'),
    },
    {
      path: '/systems',
      name: 'systems',
      component: () => import('../views/SystemsView.vue'),
    },
    {
      path: '/systems/:systemId',
      name: 'system_detail',
      component: () => import('../views/SystemDetailView.vue'),
    },
    {
      path: '/applications',
      name: 'applications',
      component: () => import('../views/ApplicationsView.vue'),
    },
    {
      path: '/applications/:applicationId',
      name: 'application_detail',
      component: () => import('../views/ApplicationDetailView.vue'),
    },
    {
      path: '/distributors/:companyId',
      name: 'distributor_detail',
      component: () => import('../views/DistributorDetailView.vue'),
    },
    {
      path: '/resellers/:companyId',
      name: 'reseller_detail',
      component: () => import('../views/ResellerDetailView.vue'),
    },
    {
      path: '/customers/:companyId',
      name: 'customer_detail',
      component: () => import('../views/CustomerDetailView.vue'),
    },
  ],
})

router.beforeEach(async (to) => {
  const loginStore = useLoginStore()

  // If the user is not logged in, redirect to the login page
  if (to.name !== 'login' && to.name !== 'login_redirect' && !loginStore.isAuthenticated) {
    if (!['/', '/login', '/dashboard', '/login-redirect'].includes(to.path)) {
      // save the path requested to local storage
      localStorage.setItem('pathRequested', JSON.stringify(to))
    }
    return { name: 'login' }
  }

  // If the user is logged in, cannot access the login page
  if ((to.name === 'login' || to.name === 'login_redirect') && loginStore.isAuthenticated) {
    return { name: 'dashboard' }
  }

  // Make sure the user has the necessary permissions to access the page, otherwise redirect to access denied page
  if (false) {
    ////
    switch (
      to.name ////
    ) {
      case 'distributors':
      case 'distributor_detail':
        if (!canReadDistributors()) {
          return { name: 'access_denied' }
        }
        break
      case 'resellers':
      case 'reseller_detail':
        if (!canReadResellers()) {
          return { name: 'access_denied' }
        }
        break
      case 'customers':
      case 'customer_detail':
        if (!canReadCustomers()) {
          return { name: 'access_denied' }
        }
        break
      case 'users':
        if (!canReadUsers()) {
          return { name: 'access_denied' }
        }
        break
      case 'systems':
      case 'system_detail':
        if (!canReadSystems()) {
          return { name: 'access_denied' }
        }
        break
      case 'applications':
      case 'application_detail':
        if (!canReadApplications()) {
          return { name: 'access_denied' }
        }
        break
    }
  }
})

export default router
