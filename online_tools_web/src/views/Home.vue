<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import {
  Document,
  Lock,
  Link,
  Timer,
  Key,
  Picture,
  Brush,
  Search
} from '@element-plus/icons-vue'

// 定义工具类型
interface Tool {
  id: string
  name: string
  icon: any
}

const router = useRouter()
const currentTool = ref<string>('json-formatter')

// 工具列表
const tools: Tool[] = [
  { id: 'json-formatter', name: 'JSON格式化', icon: Document },
  { id: 'base64', name: 'Base64编解码', icon: Lock },
  { id: 'url-encoder', name: 'URL编解码', icon: Link },
  { id: 'timestamp', name: '时间戳转换', icon: Timer },
  { id: 'md5', name: 'MD5加密', icon: Key },
  { id: 'qrcode', name: '二维码生成', icon: Picture },
  { id: 'color-picker', name: '颜色选择器', icon: Brush },
  { id: 'regex-tester', name: '正则测试', icon: Search }
]

// 切换工具
const selectTool = (toolId: string): void => {
  currentTool.value = toolId
}

// 退出登录
const handleLogout = (): void => {
  ElMessageBox.confirm('确定要退出登录吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    localStorage.removeItem('token')
    router.push('/login')
  }).catch(() => {
    // 取消操作
  })
}
</script>

<template>
  <el-container class="home-container">
    <!-- 左侧导航栏 -->
    <el-aside width="260px" class="sidebar">
      <div class="sidebar-header">
        🛠️ 在线工具
      </div>
      <el-menu
        :default-active="currentTool"
        background-color="#545c64"
        text-color="#fff"
        active-text-color="#ffd04b"
        @select="selectTool"
        class="sidebar-menu"
      >
        <el-menu-item
          v-for="tool in tools"
          :key="tool.id"
          :index="tool.id"
        >
          <el-icon><component :is="tool.icon" /></el-icon>
          <span>{{ tool.name }}</span>
        </el-menu-item>
      </el-menu>
      <div class="sidebar-footer">
        <el-button type="danger" plain @click="handleLogout" style="width: 100%">
          退出登录
        </el-button>
      </div>
    </el-aside>

    <!-- 右侧内容区域 -->
    <el-container class="main-container">
      <el-header height="60px" class="content-header">
        <el-page-header>
          <template #content>
            <span style="font-size: 20px; font-weight: bold;">
              {{ tools.find(t => t.id === currentTool)?.name }}
            </span>
          </template>
        </el-page-header>
      </el-header>
      
      <el-main class="content-main">
        <!-- JSON格式化工具 -->
        <el-card v-if="currentTool === 'json-formatter'" shadow="hover">
          <template #header>
            <span>JSON格式化工具</span>
          </template>
          <el-space direction="vertical" :size="16" style="width: 100%">
            <el-text type="info">在这里可以格式化和验证JSON数据</el-text>
            <el-input
              type="textarea"
              :rows="10"
              placeholder="请输入JSON数据..."
            />
            <el-button type="primary">格式化</el-button>
          </el-space>
        </el-card>

        <!-- Base64编解码 -->
        <el-card v-else-if="currentTool === 'base64'" shadow="hover">
          <template #header>
            <span>Base64编解码工具</span>
          </template>
          <el-space direction="vertical" :size="16" style="width: 100%">
            <el-text type="info">对文本进行Base64编码或解码</el-text>
            <el-input
              type="textarea"
              :rows="10"
              placeholder="请输入要编码/解码的内容..."
            />
            <el-space>
              <el-button type="primary">编码</el-button>
              <el-button type="success">解码</el-button>
            </el-space>
          </el-space>
        </el-card>

        <!-- URL编解码 -->
        <el-card v-else-if="currentTool === 'url-encoder'" shadow="hover">
          <template #header>
            <span>URL编解码工具</span>
          </template>
          <el-space direction="vertical" :size="16" style="width: 100%">
            <el-text type="info">对URL进行编码或解码</el-text>
            <el-input
              type="textarea"
              :rows="10"
              placeholder="请输入URL..."
            />
            <el-space>
              <el-button type="primary">编码</el-button>
              <el-button type="success">解码</el-button>
            </el-space>
          </el-space>
        </el-card>

        <!-- 时间戳转换 -->
        <el-card v-else-if="currentTool === 'timestamp'" shadow="hover">
          <template #header>
            <span>时间戳转换工具</span>
          </template>
          <el-space direction="vertical" :size="16" style="width: 100%">
            <el-text type="info">时间戳与日期时间相互转换</el-text>
            <el-input placeholder="请输入时间戳或日期时间..." />
            <el-button type="primary">转换</el-button>
          </el-space>
        </el-card>

        <!-- MD5加密 -->
        <el-card v-else-if="currentTool === 'md5'" shadow="hover">
          <template #header>
            <span>MD5加密工具</span>
          </template>
          <el-space direction="vertical" :size="16" style="width: 100%">
            <el-text type="info">对文本进行MD5加密</el-text>
            <el-input
              type="textarea"
              :rows="10"
              placeholder="请输入要加密的内容..."
            />
            <el-button type="primary">加密</el-button>
          </el-space>
        </el-card>

        <!-- 二维码生成 -->
        <el-card v-else-if="currentTool === 'qrcode'" shadow="hover">
          <template #header>
            <span>二维码生成工具</span>
          </template>
          <el-space direction="vertical" :size="16" style="width: 100%">
            <el-text type="info">将文本或URL转换为二维码</el-text>
            <el-input
              type="textarea"
              :rows="5"
              placeholder="请输入要生成二维码的内容..."
            />
            <el-button type="primary">生成二维码</el-button>
          </el-space>
        </el-card>

        <!-- 颜色选择器 -->
        <el-card v-else-if="currentTool === 'color-picker'" shadow="hover">
          <template #header>
            <span>颜色选择器</span>
          </template>
          <el-space direction="vertical" :size="16" style="width: 100%">
            <el-text type="info">选择颜色并获取各种格式的颜色值</el-text>
            <el-color-picker size="large" />
          </el-space>
        </el-card>

        <!-- 正则测试 -->
        <el-card v-else-if="currentTool === 'regex-tester'" shadow="hover">
          <template #header>
            <span>正则表达式测试工具</span>
          </template>
          <el-space direction="vertical" :size="16" style="width: 100%">
            <el-text type="info">测试正则表达式匹配</el-text>
            <el-input placeholder="请输入正则表达式..." />
            <el-input
              type="textarea"
              :rows="10"
              placeholder="请输入要测试的文本..."
            />
            <el-button type="primary">测试</el-button>
          </el-space>
        </el-card>
      </el-main>
    </el-container>
  </el-container>
</template>

<style scoped>
/* 使用 Element Plus 的样式变量和类 */
.home-container {
  height: 100vh;
  width: 100vw;
  overflow: hidden;
}

.sidebar {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background-color: #545c64;
}

.sidebar-header {
  padding: 20px;
  text-align: center;
  color: white;
  font-size: 18px;
  font-weight: bold;
  background-color: #545c64;
  flex-shrink: 0;
}

.sidebar-menu {
  flex: 1;
  border: none;
  overflow-y: auto;
}

.sidebar-footer {
  padding: 20px;
  background-color: #545c64;
  flex-shrink: 0;
}

.main-container {
  height: 100vh;
  overflow: hidden;
}

.content-header {
  display: flex;
  align-items: center;
  background-color: #fff;
  border-bottom: 1px solid #dcdfe6;
  flex-shrink: 0;
}

.content-main {
  overflow-y: auto;
  background-color: #f5f7fa;
}
</style>
