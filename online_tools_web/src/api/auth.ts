// 登录请求参数接口
export interface LoginCredentials {
  username: string
  password: string
}

// 登录响应接口
export interface LoginResponse {
  token: string
  message?: string
  user?: {
    id: string | number
    username: string
    email?: string
  }
}

// 用户信息接口
export interface UserInfo {
  id: string | number
  username: string
  email?: string
  avatar?: string
  role?: string
}

// API 响应基础接口
export interface ApiResponse<T = any> {
  code?: number
  message?: string
  data?: T
}

// API 基础配置
const API_BASE_URL = 'http://localhost:8801' // 请根据实际的 online_tools 接口地址修改

/**
 * 通用请求函数
 */
async function request<T = any>(url: string, options: RequestInit = {}): Promise<T> {
  const defaultOptions: RequestInit = {
    headers: {
      'Content-Type': 'application/json',
    },
  }

  const config: RequestInit = {
    ...defaultOptions,
    ...options,
    headers: {
      ...defaultOptions.headers,
      ...options.headers,
    },
  }

  try {
    const response = await fetch(`${API_BASE_URL}${url}`, config)
    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.message || `请求失败: ${response.status}`)
    }

    return data
  } catch (error) {
    console.error('API 请求错误:', error)
    throw error
  }
}

/**
 * 登录接口
 * @param credentials - 登录凭证
 * @returns 登录响应数据
 */
export async function login(credentials: LoginCredentials): Promise<LoginResponse> {
  return request<LoginResponse>('/api/v1/login', {
    method: 'POST',
    body: JSON.stringify(credentials),
  })
}

/**
 * 登出接口
 * @returns 登出响应数据
 */
export async function logout(): Promise<ApiResponse> {
  const token = localStorage.getItem('token')
  return request<ApiResponse>('/api/logout', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  })
}

/**
 * 获取用户信息
 * @returns 用户信息
 */
export async function getUserInfo(): Promise<UserInfo> {
  const token = localStorage.getItem('token')
  return request<UserInfo>('/api/user/info', {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  })
}
