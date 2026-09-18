<template>
  <el-container style="min-height: 100vh">
    <el-aside width="200px">
      <el-menu :router="true" default-active="/">
        <el-menu-item index="/" route="/">📊 仪表盘</el-menu-item>
        <el-menu-item index="/snapshots" route="/snapshots">📋 历史快照</el-menu-item>
        <el-menu-item index="/template" route="/template">📝 日报模板</el-menu-item>
        <el-menu-item index="/settings" route="/settings">⚙️ 设置</el-menu-item>
      </el-menu>
    </el-aside>
    <el-main><router-view /></el-main>
  </el-container>
</template>
