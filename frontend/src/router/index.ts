//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import LoginRedirectView from '../views/LoginRedirectView.vue'
import LoginView from '../views/LoginView.vue'
import { useLoginStore } from '@/stores/login'

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
    // { ////
    //   path: '/applications/:applicationId',
    //   name: 'application_detail',
    //   component: () => import('../views/ApplicationDetailView.vue'),
    // },
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
})

export default router
