package rabbitmq

const (
	QueueJobs             = "jobs"
	QueueSMS              = "sms"
	QueuePushNotification = "pushNotification"
	QueueReportGeneration = "reportGeneration"
)

// AppQueues is the authoritative list of queues this service owns.
// Add new queues here; bootstrap initializes all of them on startup.
var AppQueues = []QueueConfig{
	DefaultQueueConfig(QueueSMS),
	DefaultQueueConfig(QueuePushNotification),
	DefaultQueueConfig(QueueReportGeneration),
	DefaultQueueConfig(QueueJobs),
}
