package kafka

const (
	OrderCreatedTopic     = "ride.order.created"
	DispatchAcceptedTopic = "ride.dispatch.accepted"
	DispatchTimeoutTopic  = "ride.dispatch.timeout"
	DispatchFailedTopic   = "ride.dispatch.failed"
	TripEndedTopic        = "ride.trip.ended"
	PaymentPaidTopic      = "ride.payment.paid"
	DriverLocationTopic   = "ride.driver.location.updated"
	DriverApprovedTopic   = "ride.driver.approved"
)
