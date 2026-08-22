import { createRouter, createWebHistory } from 'vue-router'
import { useAdminStore } from '@/stores/admin'

// 前台路由组（M3）：懒加载 + View Transitions（不支持的浏览器自动降级为普通切换）
const siteRoutes = {
  path: '/',
  component: () => import('@/layouts/SiteLayout.vue'),
  children: [
    { path: '', name: 'home', component: () => import('@/views/site/HomeView.vue'), viewTransition: 'default' },
    { path: 'post/:slug', name: 'post', component: () => import('@/views/site/PostDetailView.vue'), viewTransition: 'default' },
    { path: 'category/:name', name: 'category', component: () => import('@/views/site/ArticleListView.vue'), props: { mode: 'category' }, viewTransition: 'default' },
    { path: 'tag/:name', name: 'tag', component: () => import('@/views/site/ArticleListView.vue'), props: { mode: 'tag' }, viewTransition: 'default' },
    { path: 'archive', name: 'archive', component: () => import('@/views/site/ArchiveView.vue'), viewTransition: 'default' },
    { path: 'about', name: 'about', component: () => import('@/views/site/AboutView.vue'), viewTransition: 'default' },
    { path: 'search', name: 'search', component: () => import('@/views/site/SearchView.vue'), viewTransition: 'default' },
  ],
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    siteRoutes,
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
        { path: 'profile', component: () => import('@/views/admin/Profile.vue') },
        { path: 'settings', component: () => import('@/views/admin/SiteSettings.vue') },
      ],
    },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('@/views/site/NotFoundView.vue') },
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
