import { createRouter, createWebHistory } from 'vue-router'
import { useAdminStore } from '@/stores/admin'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/admin/login', component: () => import('@/views/admin/Login.vue') },
    {
      path: '/admin',
      component: () => import('@/layouts/AdminLayout.vue'),
      children: [
        { path: '', redirect: '/admin/dashboard' },
        { path: 'dashboard', component: () => import('@/views/admin/Dashboard.vue') },
        { path: 'posts', component: () => import('@/views/admin/PostList.vue') },
        { path: 'posts/new', component: () => import('@/views/admin/PostEditor.vue') },
        { path: 'posts/:id/edit', component: () => import('@/views/admin/PostEditor.vue') },
        { path: 'comments', component: () => import('@/views/admin/CommentManage.vue') },
        { path: 'settings', component: () => import('@/views/admin/SiteSettings.vue') },
      ],
    },
    { path: '/', redirect: '/admin' }, // M3 前台占位
    { path: '/:pathMatch(.*)*', redirect: '/admin' },
  ],
})

router.beforeEach((to) => {
  const store = useAdminStore()
  if (to.path.startsWith('/admin') && to.path !== '/admin/login' && !store.isLoggedIn) {
    return { path: '/admin/login' }
  }
  if (to.path === '/admin/login' && store.isLoggedIn) {
    return { path: '/admin/dashboard' }
  }
})

export default router
