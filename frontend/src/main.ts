import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import './style.css'
import App from './App.vue'
import Dashboard from './views/Dashboard.vue'
import Snapshots from './views/Snapshots.vue'
import TemplateEditor from './views/TemplateEditor.vue'
import Settings from './views/Settings.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', component: Dashboard },
    { path: '/snapshots', component: Snapshots },
    { path: '/template', component: TemplateEditor },
    { path: '/settings', component: Settings },
  ],
})

createApp(App).use(router).use(ElementPlus).mount('#app')
