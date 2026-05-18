<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { apiGet, apiPost, hasAccessToken, setAccessToken } from '../../api'
import { connectRealtime } from '../../realtime'
import type { DriverProfile, LoginResult, RideOrder } from '../../types'

const phone = ref('13700000001')
const code = ref('123456')
const loggedIn = ref(hasAccessToken())
const error = ref('')
const drivers = ref<DriverProfile[]>([])
const orders = ref<RideOrder[]>([])
const statusFilter = ref('ALL')
let pollTimer: number | undefined
let socket: WebSocket | undefined

const filteredOrders = computed(() => statusFilter.value === 'ALL' ? orders.value : orders.value.filter((order) => order.status === statusFilter.value))
const abnormalOrders = computed(() => orders.value.filter((order) => ['CANCELED', 'DISPATCH_FAILED'].includes(order.status)))

async function login() {
  error.value = ''
  const result = await apiPost<LoginResult>('/api/auth/login', { phone: phone.value, code: code.value, role: 'ADMIN' })
  setAccessToken(result.accessToken)
  loggedIn.value = true
  await load()
  startRealtime()
}

async function load() {
  drivers.value = await apiGet<DriverProfile[]>('/api/admin/drivers')
  orders.value = await apiGet<RideOrder[]>('/api/admin/orders')
}

async function approve(driver: DriverProfile) {
  await apiPost(`/api/admin/drivers/${driver.id}/approve`)
  await load()
}

function startRealtime() {
  socket?.close()
  socket = connectRealtime(() => load().catch(() => undefined))
}

onMounted(() => {
  if (loggedIn.value) {
    load().catch((err) => { error.value = err instanceof Error ? err.message : '加载失败' })
    startRealtime()
  }
  pollTimer = window.setInterval(() => {
    if (loggedIn.value) {
      load().catch(() => undefined)
    }
  }, 3000)
})

onBeforeUnmount(() => {
  if (pollTimer) {
    clearInterval(pollTimer)
  }
  socket?.close()
})
</script>

<template>
  <main class="admin-shell">
    <section class="hero">
      <h1>管理后台</h1>
      <p>司机审核、订单查询、异常处理、运营统计</p>
    </section>

    <section v-if="!loggedIn" class="card">
      <h2>管理员登录</h2>
      <input v-model="phone" placeholder="管理员手机号" />
      <input v-model="code" placeholder="验证码，演示固定 123456" />
      <button @click="login">登录</button>
      <p v-if="error" class="error">{{ error }}</p>
    </section>

    <template v-else>
    <section class="table-card">
      <h2>运营概览</h2>
      <div class="button-row">
        <span class="status">订单 {{ orders.length }}</span>
        <span class="status">司机 {{ drivers.length }}</span>
        <span class="status">异常 {{ abnormalOrders.length }}</span>
      </div>
    </section>

    <section class="table-card">
      <h2>司机审核</h2>
      <table>
        <thead><tr><th>司机</th><th>手机号</th><th>车辆</th><th>审核</th><th>状态</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="driver in drivers" :key="driver.id">
            <td>{{ driver.name }}</td>
            <td>{{ driver.phone }}</td>
            <td>{{ driver.vehicle.plateNo }}</td>
            <td>{{ driver.auditState }}</td>
            <td>{{ driver.workStatus }}</td>
            <td><button class="secondary" @click="approve(driver)">审核通过</button></td>
          </tr>
        </tbody>
      </table>
    </section>

    <section class="table-card">
      <h2>订单查询</h2>
      <select v-model="statusFilter" class="input">
        <option value="ALL">全部</option>
        <option value="DISPATCHING">派单中</option>
        <option value="WAITING_PICKUP">等待接驾</option>
        <option value="IN_PROGRESS">行程中</option>
        <option value="WAITING_PAYMENT">待支付</option>
        <option value="COMPLETED">已完成</option>
        <option value="CANCELED">已取消</option>
        <option value="DISPATCH_FAILED">派单失败</option>
      </select>
      <table>
        <thead><tr><th>订单</th><th>路线</th><th>状态</th><th>金额</th><th>版本</th></tr></thead>
        <tbody>
          <tr v-for="order in filteredOrders" :key="order.id">
            <td>{{ order.id.slice(0, 8) }}</td>
            <td>{{ order.pickup.name }} → {{ order.dropoff.name }}</td>
            <td>{{ order.status }}</td>
            <td>¥{{ ((order.finalAmount || order.estimatedAmount) / 100).toFixed(2) }}</td>
            <td>{{ order.version }}</td>
          </tr>
        </tbody>
      </table>
    </section>

    <section class="table-card">
      <h2>异常订单</h2>
      <div v-if="abnormalOrders.length === 0" class="muted">暂无异常订单</div>
      <div v-for="order in abnormalOrders" :key="order.id" class="timeline-item">
        {{ order.id.slice(0, 8) }} · {{ order.status }} · {{ order.pickup.name }} → {{ order.dropoff.name }}
      </div>
    </section>
    </template>
  </main>
</template>
