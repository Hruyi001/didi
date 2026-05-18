import { getAccessToken } from './api'

export function connectRealtime(onEvent: (payload: any) => void) {
  const protocol = location.protocol === 'https:' ? 'wss' : 'ws'
  const token = getAccessToken()
  const query = token ? `?token=${encodeURIComponent(token)}` : ''
  const socket = new WebSocket(`${protocol}://${location.host}/ws${query}`)
  socket.onmessage = (event) => {
    try {
      const payload = JSON.parse(event.data)
      if (payload.type === 'order.updated' || payload.type === 'driver.location') {
        onEvent(payload)
      }
    } catch {
      // ignore malformed events
    }
  }
  return socket
}
