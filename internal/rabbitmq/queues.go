package rabbitmq

const (
	QueueJobs             = "jobs"
	QueueSMS              = "sms"
	QueuePushNotification = "pushNotification"
	QueueReportGeneration = "reportGeneration"

	// ExchangeJobsDelayed is the x-delayed-message exchange bound to QueueJobs.
	// Publish to this exchange with an x-delay header (milliseconds) to schedule
	// delivery. Use delay=0 for immediate dispatch.
	ExchangeJobsDelayed = QueueJobs + ".delayed"
)

// AppQueues is the authoritative list of queues this service owns.
// Add new queues here; bootstrap initializes all of them on startup.
var AppQueues = []QueueConfig{
	DefaultQueueConfig(QueueSMS),
	DefaultQueueConfig(QueuePushNotification),
	DefaultQueueConfig(QueueReportGeneration),
	{
		Name:               QueueJobs,
		Durable:            true,
		Type:               "classic", // delayed message plugin requires classic queue type
		UseDelayedExchange: true,
	},
}
