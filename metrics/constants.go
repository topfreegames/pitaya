package metrics

var (
	// ResponseTime reports the response time of handlers and rpc
	ResponseTime = "response_time_ns"
	// ConnectedClients represents the number of current connected clients in frontend servers
	ConnectedClients = "connected_clients"
	// CountServers counts the number of servers of different types
	CountServers = "count_servers"
	// ChannelCapacity represents the capacity of a channel (available slots)
	ChannelCapacity = "channel_capacity"
	// DroppedMessages reports the number of dropped messages in rpc server (messages that will not be handled)
	DroppedMessages = "dropped_messages"
	// ProcessDelay reports the message processing delay to handle the messages at the handler service
	ProcessDelay = "handler_delay_ns"
	// Goroutines reports the number of goroutines
	Goroutines = "goroutines"
	// HeapSize reports the size of heap
	HeapSize = "heapsize"
	// HeapObjects reports the number of allocated heap objects
	HeapObjects = "heapobjects"
	// WorkerJobsTotal reports the number of executed jobs
	WorkerJobsTotal = "worker_jobs_total"
	// WorkerJobsRetry reports the number of retried jobs
	WorkerJobsRetry = "worker_jobs_retry_total"
	// WorkerQueueSize reports the queue size on worker
	WorkerQueueSize = "worker_queue_size"
	// ExceededRateLimiting reports the number of requests made in a connection
	// after the rate limit was exceeded
	ExceededRateLimiting = "exceeded_rate_limiting"
)
