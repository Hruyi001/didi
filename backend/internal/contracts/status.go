package contracts

type OrderStatus string

type DispatchStatus string

type PaymentStatus string

type ReviewStatus string

type DriverWorkStatus string

type AccountRole string

const (
	RolePassenger AccountRole = "PASSENGER"
	RoleDriver    AccountRole = "DRIVER"
	RoleAdmin     AccountRole = "ADMIN"
)

const (
	OrderCreated        OrderStatus = "CREATED"
	OrderDispatching    OrderStatus = "DISPATCHING"
	OrderWaitingPickup  OrderStatus = "WAITING_PICKUP"
	OrderDriverArrived  OrderStatus = "DRIVER_ARRIVED"
	OrderInProgress     OrderStatus = "IN_PROGRESS"
	OrderWaitingPayment OrderStatus = "WAITING_PAYMENT"
	OrderCompleted      OrderStatus = "COMPLETED"
	OrderCanceled       OrderStatus = "CANCELED"
	OrderDispatchFailed OrderStatus = "DISPATCH_FAILED"
)

const (
	DispatchPending  DispatchStatus = "PENDING"
	DispatchOffered  DispatchStatus = "OFFERED"
	DispatchAccepted DispatchStatus = "ACCEPTED"
	DispatchRejected DispatchStatus = "REJECTED"
	DispatchTimeout  DispatchStatus = "TIMEOUT"
	DispatchCanceled DispatchStatus = "CANCELED"
	DispatchFailed   DispatchStatus = "FAILED"
)

const (
	PaymentUnpaid    PaymentStatus = "UNPAID"
	PaymentPaying    PaymentStatus = "PAYING"
	PaymentPaid      PaymentStatus = "PAID"
	PaymentFailed    PaymentStatus = "FAILED"
	PaymentRefunding PaymentStatus = "REFUNDING"
	PaymentRefunded  PaymentStatus = "REFUNDED"
	PaymentClosed    PaymentStatus = "CLOSED"
)

const (
	ReviewNotReviewed ReviewStatus = "NOT_REVIEWED"
	ReviewReviewed    ReviewStatus = "REVIEWED"
	ReviewExpired     ReviewStatus = "EXPIRED"
)

const (
	DriverOffline    DriverWorkStatus = "OFFLINE"
	DriverOnlineIdle DriverWorkStatus = "ONLINE_IDLE"
	DriverDispatched DriverWorkStatus = "DISPATCHED"
	DriverServing    DriverWorkStatus = "SERVING"
	DriverSuspended  DriverWorkStatus = "SUSPENDED"
)
