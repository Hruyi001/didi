export type OrderStatus = 'CREATED' | 'DISPATCHING' | 'WAITING_PICKUP' | 'DRIVER_ARRIVED' | 'IN_PROGRESS' | 'WAITING_PAYMENT' | 'COMPLETED' | 'CANCELED' | 'DISPATCH_FAILED'
export type DriverWorkStatus = 'OFFLINE' | 'ONLINE_IDLE' | 'DISPATCHED' | 'SERVING' | 'SUSPENDED'

export interface ApiEnvelope<T> {
  data: T
  error?: string
}

export interface LoginResult {
  accessToken: string
  refreshToken: string
  role: 'PASSENGER' | 'DRIVER' | 'ADMIN'
  expiresAt: string
}

export interface Point {
  name: string
  lng: number
  lat: number
}

export interface RideOrder {
  id: string
  passengerId: string
  driverId?: string
  pickup: Point
  dropoff: Point
  status: OrderStatus
  paymentStatus: string
  reviewStatus: string
  estimatedDistance: number
  estimatedDuration: number
  estimatedAmount: number
  finalAmount: number
  version: number
  createdAt: string
}

export interface DriverProfile {
  id: string
  name: string
  phone: string
  auditState: string
  workStatus: DriverWorkStatus
  vehicle: {
    plateNo: string
    model: string
    color: string
  }
  stats: {
    accepted: number
    rejected: number
    timeouts: number
  }
}

export interface DriverLocation {
  driverId: string
  lng: number
  lat: number
  speedKph: number
  updatedAt: string
}
