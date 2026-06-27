package rabbitmq

const (
	QueueJobs             = "jobs"
	QueueSMS              = "sms"
	QueuePushNotification = "pushNotification"
	QueueReportGeneration = "reportGeneration"

	// dead-letter queue for jobs that ran out of retries — kept for replay, not dropped
	QueueJobsDLQ = QueueJobs + ".dlq"

	// delayed-message exchange for jobs; publish with an x-delay header (ms), 0 = now
	ExchangeJobsDelayed = QueueJobs + ".delayed"
)

// every queue this service owns; bootstrap declares all of them on startup
var AppQueues = []QueueConfig{
	DefaultQueueConfig(QueueSMS),
	DefaultQueueConfig(QueuePushNotification),
	DefaultQueueConfig(QueueReportGeneration),
	// Declare the DLQ before the jobs queue that dead-letters into it.
	DefaultQueueConfig(QueueJobsDLQ),
	{
		Name:               QueueJobs,
		Durable:            true,
		Type:               "classic", // delayed message plugin requires classic queue type
		UseDelayedExchange: true,
		// dead-letter via the default exchange, which routes by key to jobs.dlq
		DeadLetterRoutingKey: QueueJobsDLQ,
	},
}
