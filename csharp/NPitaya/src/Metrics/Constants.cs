using System;

namespace NPitaya.Metrics
{
    public class Constants
    {
        internal static string PitayaKey = "pitaya";
        internal static string ResponseTimeMetricKey = "response_time_ns";
        internal static string ConnectedClientsMetricKey = "connected_clients";
        internal static string CountServersMetricKey = "count_servers";
        internal static string ChannelCapacityMetricKey = "channel_capacity";
        internal static string DroppedMessagesMetricKey = "dropped_messages";
        internal static string ProccessDelayMetricKey = "handler_delay_ns";
        internal static string GoroutinesMetricKey = "goroutines";
        internal static string HeapSizeMetricKey = "heapsize";
        internal static string HeapObjectsMetricKey = "heapobjects";
        internal static string WorkerJobsTotalMetricKey = "worker_jobs_total";
        internal static string WorkerJobsRetryMetricKey = "worker_jobs_retry_total";
        internal static string WorkerQueueSizeMetricKey = "worker_queue_size";
        internal static string ExceededRateLimitingMetricKey = "exceeded_rate_limiting";

        public enum Status
        {
            fail,
            success
        }

        public class MetricNotFoundException : Exception
        {
            public MetricNotFoundException()
            {
            }

            public MetricNotFoundException(string message) : base(message)
            {
            }

            public MetricNotFoundException(string message, Exception inner) : base(message, inner)
            {
            }
        }
    }
}