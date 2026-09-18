<template>
  <el-container class="app-shell">
    <el-aside width="200px" class="app-aside">
      <div class="brand">QuickFeishu</div>
      <el-menu :router="true" :default-active="route.path" class="app-menu">
        <el-menu-item index="/" route="/">📊 仪表盘</el-menu-item>
        <el-menu-item index="/snapshots" route="/snapshots">📋 历史快照</el-menu-item>
        <el-menu-item index="/template" route="/template">📝 日报模板</el-menu-item>
        <el-menu-item index="/settings" route="/settings">⚙️ 设置</el-menu-item>
      </el-menu>
    </el-aside>
    <el-main class="app-main"><router-view /></el-main>
  </el-container>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
const route = useRoute()
</script>

<style scoped>
.app-shell {
  min-height: 100vh;
}

.app-aside {
  display: flex;
  flex-direction: column;
  background: #fff;
  border-right: 1px solid var(--el-border-color-lighter, #ebeef5);
}

.brand {
  flex: none;
  height: 56px;
  line-height: 56px;
  padding: 0 20px;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-color-primary, #409eff);
  border-bottom: 1px solid var(--el-border-color-lighter, #ebeef5);
}

.app-menu {
  flex: 1;
  border-right: none;
}

.app-main {
  background: var(--el-bg-color-page, #f5f7fa);
}
</style>
