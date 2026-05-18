<script setup lang="ts">
import PassengerApp from './apps/passenger/PassengerApp.vue'
import DriverApp from './apps/driver/DriverApp.vue'
import AdminApp from './apps/admin/AdminApp.vue'

const app = new URLSearchParams(location.search).get('app') || 'passenger'
</script>

<template>
  <PassengerApp v-if="app === 'passenger'" />
  <DriverApp v-else-if="app === 'driver'" />
  <AdminApp v-else-if="app === 'admin'" />
  <PassengerApp v-else />
</template>
