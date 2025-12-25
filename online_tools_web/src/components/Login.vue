<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'
import { login } from '../api/auth'

const router = useRouter()
const loading = ref<boolean>(false)

const loginForm = reactive({
  username: '',
  password: ''
})

const handleLogin = async (): Promise<void> => {
  if (!loginForm.username || !loginForm.password) {
    ElMessage.error('请输入用户名和密码')
    return
  }

  loading.value = true

  try {
    const response = await login({
      username: loginForm.username,
      password: loginForm.password
    })
    
    // 登录成功，保存 token
    if (response.token) {
      localStorage.setItem('token', response.token)
      ElMessage.success('登录成功')
      // 跳转到主页面
      router.push('/home')
    }
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '登录失败，请检查用户名和密码')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-container">
    <el-card class="login-box">
      <template #header>
        <div class="card-header">
          <h1>在线工具平台</h1>
          <p>欢迎登录</p>
        </div>
      </template>
      
      <el-form @submit.prevent="handleLogin" :model="loginForm" label-width="0">
        <el-form-item>
          <el-input
            v-model="loginForm.username"
            placeholder="请输入用户名"
            :prefix-icon="User"
            size="large"
            :disabled="loading"
          />
        </el-form-item>

        <el-form-item>
          <el-input
            v-model="loginForm.password"
            type="password"
            placeholder="请输入密码"
            :prefix-icon="Lock"
            size="large"
            :disabled="loading"
            show-password
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            @click="handleLogin"
            style="width: 100%"
          >
            {{ loading ? '登录中...' : '登录' }}
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.login-box {
  width: 100%;
  max-width: 400px;
}

.card-header {
  text-align: center;
}

.card-header h1 {
  font-size: 28px;
  font-weight: 700;
  color: #333;
  margin: 0 0 8px 0;
}

.card-header p {
  font-size: 16px;
  color: #666;
  margin: 0;
}
</style>
