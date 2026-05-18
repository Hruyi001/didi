<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { apiGet, apiPost, hasAccessToken, setAccessToken } from '../../api'
import { connectRealtime } from '../../realtime'
import type { DriverProfile, LoginResult, RideOrder } from '../../types'

const phone = ref('13900000001')
const code = ref('123456')
const loggedIn = ref(hasAccessToken())
const driver = ref<DriverProfile | null>(null)
const orders = ref<RideOrder[]>([])
const error = ref('')
const position = ref({ lng: 116.481, lat: 39.996, speedKph: 0 })
let locationTimer: number | undefined
let pollTimer: number | undefined
let socket: WebSocket | undefined

async function login() {
  error.value = ''
  const result = await apiPost<LoginResult>('/api/auth/login', { phone: phone.value, code: code.value, role: 'DRIVER' })
  setAccessToken(result.accessToken)
  loggedIn.value = true
  await load()
  startRealtime()
}

async function load() {
  driver.value = await apiGet<DriverProfile>('/api/driver/profile')
  orders.value = await apiGet<RideOrder[]>('/api/driver/orders')
}

async function online() {
  driver.value = await apiPost<DriverProfile>('/api/driver/online')
}

async function offline() {
  driver.value = await apiPost<DriverProfile>('/api/driver/offline')
}

async function action(path: string) {
  error.value = ''
  try {
    await apiPost(path)
    await load()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '操作失败'
  }
}

async function reportLocation() {
  if (!driver.value) {
    return
  }
  error.value = ''
  try {
    await apiPost('/api/driver/location', {
      driverId: driver.value.id,
      lng: Number(position.value.lng.toFixed(6)),
      lat: Number(position.value.lat.toFixed(6)),
      speedKph: position.value.speedKph
    })
  } catch (err) {
    error.value = err instanceof Error ? err.message : '位置上报失败'
  }
}

function simulateLocationTick() {
  const active = orders.value.find((order) => ['WAITING_PICKUP', 'DRIVER_ARRIVED', 'IN_PROGRESS'].includes(order.status) && order.driverId === driver.value?.id)
  if (!active) {
    position.value.speedKph = 0
    return
  }

  if (active.status === 'WAITING_PICKUP') {
    position.value.lng -= 0.002
    position.value.lat -= 0.002
    position.value.speedKph = 28
  } else if (active.status === 'DRIVER_ARRIVED') {
    position.value.speedKph = 0
  } else {
    position.value.lng -= 0.0015
    position.value.lat -= 0.004
    position.value.speedKph = 36
  }

  reportLocation().catch(() => undefined)
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
  locationTimer = window.setInterval(simulateLocationTick, 3000)
})

onBeforeUnmount(() => {
  if (locationTimer) {
    clearInterval(locationTimer)
  }
  if (pollTimer) {
    clearInterval(pollTimer)
  }
  socket?.close()
})
</script>

<template>
  <main class="app-shell">
    <section class="hero">
      <h1>司机端</h1>
      <p>上线接单、行程操作、收入记录</p>
    </section>

    <section v-if="!loggedIn" class="card">
      <h2>司机登录</h2>
      <input v-model="phone" placeholder="司机手机号" />
      <input v-model="code" placeholder="验证码，演示固定 123456" />
      <button @click="login">登录</button>
      <p v-if="error" class="error">{{ error }}</p>
    </section>

    <template v-else>
    <section class="card" v-if="driver">
      <h2>{{ driver.name }}</h2>
      <p>{{ driver.vehicle.color }} {{ driver.vehicle.model }} · {{ driver.vehicle.plateNo }}</p>
      <div class="status">{{ driver.auditState }} / {{ driver.workStatus }}</div>
      <div class="button-row">
        <button class="success" @click="online">上线</button>
        <button class="secondary" @click="offline">下线</button>
        <button @click="reportLocation">立即上报位置</button>
      </div>
      <p class="muted">已接 {{ driver.stats.accepted }} · 拒单 {{ driver.stats.rejected }} · 超时 {{ driver.stats.timeouts }}</p>
      <p class="muted">当前位置：{{ position.lng.toFixed(3) }}, {{ position.lat.toFixed(3) }} · 速度 {{ position.speedKph }} km/h</p>
    </section>

    <section class="card">
      <h2>派单与行程</h2>
      <p v-if="error" class="error">{{ error }}</p>
      <div v-if="orders.length === 0" class="muted">暂无订单，请先在乘客端叫车。</div>
      <div v-for="order in orders" :key="order.id" class="timeline-item">
        <strong>{{ order.pickup.name }} → {{ order.dropoff.name }}</strong>
        <div class="status">{{ order.status }}</div>
        <p class="muted">派单倒计时：20 秒 · 距离上车点约 1.2 公里</p>
        <div class="button-row">
          <button v-if="order.status === 'DISPATCHING'" class="success" @click="action(`/api/driver/orders/${order.id}/accept`)">接单</button>
          <button v-if="order.status === 'DISPATCHING'" class="danger" @click="action(`/api/driver/orders/${order.id}/reject`)">拒单</button>
          <button v-if="order.status === 'WAITING_PICKUP'" @click="action(`/api/driver/orders/${order.id}/arrive`)">已到达</button>
          <button v-if="order.status === 'DRIVER_ARRIVED'" @click="action(`/api/driver/orders/${order.id}/start`)">开始行程</button>
          <button v-if="order.status === 'IN_PROGRESS'" class="success" @click="action(`/api/driver/orders/${order.id}/end`)">结束行程</button>
        </div>
      </div>
    </section>
    </template>
  </main>
</template>
