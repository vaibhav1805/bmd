# Monitoring & Alerts

## Overview

The e-commerce platform uses Prometheus for metrics collection, Grafana for dashboards, and PagerDuty for alerting.

## Metrics Collection

### Prometheus

Services emit metrics on `/metrics` endpoint (Prometheus format):

```
http://order-service:9001/metrics
```

Example metrics:

```
# HELP http_requests_total Total HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="POST",path="/api/v1/orders",status="200"} 1234
http_requests_total{method="POST",path="/api/v1/orders",status="400"} 12

# HELP http_request_duration_seconds HTTP request latency
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="POST",le="0.1"} 1200
http_request_duration_seconds_bucket{method="POST",le="0.5"} 1230
```

### Prometheus Configuration

**prometheus.yml:**

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'order-service'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - ecommerce
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: order-service
      - source_labels: [__address__]
        target_label: __param_target
      - target_label: __address__
        replacement: 127.0.0.1:9090
```

## Key Metrics

### Application Metrics

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `request_success_rate` | % of requests succeeding | < 99.5% |
| `request_latency_p95` | 95th percentile latency | > 500ms |
| `request_latency_p99` | 99th percentile latency | > 1s |
| `orders_created_total` | Total orders created | Baseline alert if drops |
| `payment_success_rate` | % of payments succeeding | < 98% |
| `inventory_reservations_failed` | Failed inventory reserves | > 5/min |

### Infrastructure Metrics

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `container_memory_usage_bytes` | Pod memory usage | > 80% of limit |
| `container_cpu_usage_seconds_total` | Pod CPU usage | > 80% of limit |
| `node_disk_bytes_available` | Disk space available | < 10% |
| `node_memory_MemAvailable_bytes` | Node memory available | < 20% |

### Database Metrics

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `pg_connections_active` | Active connections | > 80% of max |
| `pg_query_duration_seconds` | Query latency | p95 > 1s |
| `pg_replication_lag_seconds` | Replication lag | > 10s |
| `pg_transactions_rolled_back_total` | Rollback rate | > 1% of commits |

## Grafana Dashboards

### Service Overview Dashboard

Shows health of individual service:

- Request rate (req/s)
- Success rate (%)
- Latency (p50, p95, p99)
- Error rate by code
- Resource utilization (CPU, memory)
- Database connection pool

### Business Metrics Dashboard

High-level business KPIs:

- Orders per minute
- Revenue (total, by hour)
- Payment success rate
- Average order value
- Customer acquisition cost
- Conversion funnel

### Infrastructure Dashboard

System-level metrics:

- Cluster CPU/memory utilization
- Node health
- Pod restart counts
- Persistent volume usage
- Network I/O
- Disk I/O

## Alerting Rules

Alert rules defined in **prometheus-alerts.yml:**

```yaml
groups:
  - name: ecommerce
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        annotations:
          summary: "High error rate for {{ $labels.service }}"
          description: "Error rate is {{ $value | humanizePercentage }}"

      - alert: HighLatency
        expr: histogram_quantile(0.95, http_request_duration_seconds) > 0.5
        for: 5m
        annotations:
          summary: "High latency for {{ $labels.service }}"
          description: "p95 latency is {{ $value }}s"

      - alert: PaymentServiceDown
        expr: up{job="payment-processor"} == 0
        for: 1m
        annotations:
          summary: "Payment service is down"
          description: "Payment processor has been unreachable for 1 minute"

      - alert: InsufficientInventory
        expr: rate(inventory_reservations_failed_total[5m]) > 5
        for: 2m
        annotations:
          summary: "High inventory reservation failure rate"
          description: "{{ $value }} failures per minute"

      - alert: DatabaseDown
        expr: pg_up == 0
        for: 1m
        annotations:
          summary: "Database connection failed"
          description: "Cannot connect to database for {{ $labels.instance }}"
```

## Alert Severity Levels

### Critical (Page immediately)

- **Payment service down** — Blocks all orders
- **Database unreachable** — Complete service outage
- **API Gateway errors > 10%** — Severe customer impact
- **Memory/CPU at 95%+** — Risk of OOM kill/throttling

**Action:** PagerDuty page on-call engineer.

### High (Investigate within 1 hour)

- **Service error rate > 5%** — Degraded experience
- **Latency p95 > 1s** — Performance issue
- **Inventory reservation failures > 10/min**
- **Disk space < 20%** — Risk of running out

**Action:** Slack alert to team, create incident.

### Medium (Investigate next business day)

- **Single pod restart** — Possible flaky test
- **Disk space < 30%** — Plan cleanup
- **Memory usage growing** — Possible memory leak

**Action:** Slack alert, add to backlog.

### Info (For trending)

- **API latency increasing over time** — Performance trending
- **Request volume patterns** — Baseline metrics

## Alert Integration

### PagerDuty

```yaml
alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']
```

**alertmanager.yml:**

```yaml
global:
  resolve_timeout: 5m

route:
  receiver: pagerduty
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h

receivers:
  - name: pagerduty
    pagerduty_configs:
      - service_key: ${PAGERDUTY_SERVICE_KEY}
        severity: '{{ .GroupLabels.severity }}'
        client: Prometheus
        client_url: '{{ .ExternalURL }}'
```

### Slack

For non-critical alerts:

```yaml
receivers:
  - name: slack
    slack_configs:
      - api_url: ${SLACK_WEBHOOK_URL}
        channel: '#alerts'
        title: 'Alert: {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

## Custom Metrics

Services should emit domain-specific metrics:

```go
// Example: Order Service
ordersCreated := prometheus.NewCounterVec(
  prometheus.CounterOpts{
    Name: "orders_created_total",
    Help: "Total orders created",
  },
  []string{"status", "payment_method"},
)

// Record metric
ordersCreated.WithLabelValues("success", "credit_card").Inc()
```

## Performance Benchmarks

Target metrics for normal operation:

| Metric | Target |
|--------|--------|
| Request success rate | > 99.9% |
| API latency (p95) | < 200ms |
| API latency (p99) | < 500ms |
| Database latency (p95) | < 100ms |
| Payment processing | < 2s |
| Page load time | < 2s |

## Log Aggregation

Logs collected via ELK Stack (Elasticsearch, Logstash, Kibana):

**Log format (JSON):**

```json
{
  "timestamp": "2024-02-28T10:30:00Z",
  "level": "ERROR",
  "service": "order-service",
  "requestId": "req_123",
  "message": "Payment processing failed",
  "error": "CARD_DECLINED",
  "statusCode": 402,
  "duration_ms": 1234,
  "userId": "usr_123",
  "orderId": "ord_789"
}
```

Query examples:

```
# Find errors for user
service:order-service AND level:ERROR AND userId:usr_123

# Find slow requests
service:order-service AND duration_ms:[1000 TO *]

# Find payment failures
service:payment-processor AND error:CARD_DECLINED
```

## SLA Targets

| Component | SLA | Budget |
|-----------|-----|--------|
| API Availability | 99.9% | 8.6 hours/month |
| Payment Processing | 99.5% | 3.6 hours/month |
| Database | 99.95% | 2.2 hours/month |
| Overall Service | 99.0% | 7.2 hours/month |

## See Also

- [Deployment Guide](deployment.md)
- [Troubleshooting](troubleshooting.md)
- [Architecture](../architecture.md)
