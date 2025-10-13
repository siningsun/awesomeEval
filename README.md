          ┌──────────────────┐
          │  Producer / API  │
          │   (提交任务)     │
          └────────┬─────────┘
                   │
                   ▼
          ┌──────────────────┐
          │   RabbitMQ       │
          │  Queue: eval_jobs│
          │  Dead Letter Q   │
          └────────┬─────────┘
                   │
           (消息投递)
                   │
                   ▼
          ┌──────────────────┐
          │   EvalWorker     │
          │ ┌──────────────┐ │
          │ │ Goroutine Pool│ │
          │ └──────┬───────┘ │
          │        │
          │  ┌─────▼─────┐
          │  │ HandleTask│
          │  │  - Fetch task state
          │  │  - Check status
          │  │  - Set context.WithTimeout
          │  │  - RunEvalTask
          │  │  - Update DB (atomic/transaction)
          │  └─────┬─────┘
          │        │
          │  Success│Failure
          │        │
          ▼        ▼
    ┌────────────┐ ┌──────────────┐
    │ Ack message│ │ Retry Queue  │
    │ (update DB)│ │ Increment x-retry
    └─────┬──────┘ │ If > max, go DLQ
    │        └──────┬───────┘
    ▼               ▼
    Metrics & Logs   Dead Letter Q / EvalDLQWorker
    (Prometheus,     (标记失败, update DB)
    TraceID)
