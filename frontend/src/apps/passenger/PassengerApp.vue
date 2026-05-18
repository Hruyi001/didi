<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { apiGet, apiPost, hasAccessToken, setAccessToken } from '../../api'
import { connectRealtime } from '../../realtime'
import type { DriverLocation, LoginResult, RideOrder } from '../../types'

const phone = ref('13800000001')
const code = ref('123456')
const passengerId = ref('demo-passenger')
const loggedIn = ref(hasAccessToken())
const orders = ref<RideOrder[]>([])
const driverLocation = ref<DriverLocation | null>(null)
const error = ref('')
const pickup = { name: '望京 SOHO', lng: 116.481, lat: 39.996 }
const dropoff = { name: '国贸 CBD', lng: 116.457, lat: 39.914 }
let pollTimer: number | undefined
let socket: WebSocket | undefined

async function login() {
  error.value = ''
  await apiPost('/api/auth/send-code', { phone: phone.value })
  const result = await apiPost<LoginResult>('/api/auth/login', { phone: phone.value, code: code.value, role: 'PASSENGER' })
  setAccessToken(result.accessToken)
  loggedIn.value = true
  await loadOrders()
  startRealtime()
}

async function loadOrders() {
  orders.value = await apiGet<RideOrder[]>('/api/passenger/orders')
  const active = orders.value.find((order) => order.driverId)
  if (active) {
    try {
      driverLocation.value = await apiGet<DriverLocation>(`/api/passenger/orders/${active.id}/driver-location`)
    } catch {
      driverLocation.value = null
    }
  } else {
    driverLocation.value = null
  }
}

async function callRide() {
  error.value = ''
  try {
    const result = await apiPost<{ order: RideOrder }>('/api/passenger/orders', { passengerId: passengerId.value, pickup, dropoff })
    orders.value = [result.order, ...orders.value]
    await loadOrders()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '叫车失败'
  }
}

async function cancel(order: RideOrder) {
  await apiPost(`/api/passenger/orders/${order.id}/cancel`)
  await loadOrders()
}

async function pay(order: RideOrder) {
  await apiPost(`/api/passenger/orders/${order.id}/pay`)
  await loadOrders()
}

async function review(order: RideOrder) {
  await apiPost(`/api/passenger/orders/${order.id}/review`, { score: 5, content: '司机服务很好，行程顺利。' })
  await loadOrders()
}

function startRealtime() {
  socket?.close()
  socket = connectRealtime((payload) => {
    if (payload.type === 'driver.location') {
      driverLocation.value = {
        driverId: payload.driverId,
        lng: payload.lng,
        lat: payload.lat,
        speedKph: payload.speedKph,
        updatedAt: new Date().toISOString()
      }
      return
    }
    loadOrders().catch(() => undefined)
  })
}

onMounted(() => {
  if (loggedIn.value) {
    loadOrders().catch(() => undefined)
    startRealtime()
  }
  pollTimer = window.setInterval(() => {
    if (loggedIn.value) {
      loadOrders().catch(() => undefined)
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
  <main class="app-shell">
    <section class="hero">
      <h1>乘客端</h1>
      <p>地图选点、呼叫快车、支付与评价</p>
    </section>

    <section v-if="!loggedIn" class="card">
      <h2>手机号登录</h2>
      <input v-model="phone" placeholder="手机号" />
      <input v-model="code" placeholder="验证码，演示固定 123456" />
      <button @click="login">登录</button>
    </section>

    <template v-else>
      <section class="card map-card">
        <div class="pin start">上车：{{ pickup.name }}</div>
        <div class="pin end">下车：{{ dropoff.name }}</div>
      </section>

      <section class="card">
        <h2>快车预估</h2>
        <p class="muted">约 6.8 公里 · 18 分钟 · 预估 ¥28.00</p>
        <button @click="callRide">呼叫快车</button>
        <p v-if="error" class="error">{{ error }}</p>
        <p v-if="driverLocation" class="muted">司机实时位置：{{ driverLocation.lng.toFixed(3) }}, {{ driverLocation.lat.toFixed(3) }} · 速度 {{ driverLocation.speedKph }} km/h</p>
      </section>

      <section class="card">
        <h2>我的订单</h2>
        <div v-if="orders.length === 0" class="muted">暂无订单</div>
        <div v-for="order in orders" :key="order.id" class="timeline-item">
          <strong>{{ order.pickup.name }} → {{ order.dropoff.name }}</strong>
          <div class="status">{{ order.status }}</div>
          <p class="muted">预估 ¥{{ (order.estimatedAmount / 100).toFixed(2) }} · 实付 ¥{{ (order.finalAmount / 100).toFixed(2) }}</p>
          <div class="button-row">
            <button v-if="['CREATED','DISPATCHING','WAITING_PICKUP','DRIVER_ARRIVED'].includes(order.status)" class="danger" @click="cancel(order)">取消</button>
            <button v-if="order.status === 'WAITING_PAYMENT'" class="success" @click="pay(order)">模拟支付</button>
            <button v-if="order.status === 'COMPLETED' && order.reviewStatus !== 'REVIEWED'" class="secondary" @click="review(order)">评价</button>
          </div>
        </div>
      </section>
    </template>
  </main>
</template>
