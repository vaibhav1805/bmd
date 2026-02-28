# Troubleshooting Guide

## General Diagnosis

### Check Service Status

```bash
# Check if service is running
kubectl get pods -l app=order-service

# Get service logs
kubectl logs -f deployment/order-service

# Describe pod for events
kubectl describe pod <pod-name>

# Check resource limits
kubectl top pods -l app=order-service
```

## Common Issues

### Service Won't Start

**Symptoms:**
- Pod in `CrashLoopBackOff` state
- Repeated restart attempts

**Diagnosis:**

```bash
# View startup logs
kubectl logs -f <pod-name> --previous

# Check pod events
kubectl describe pod <pod-name>

# Common causes: config missing, database unreachable, port in use
```

**Solutions:**

1. **Missing environment variable**
   ```bash
   # Check ConfigMap
   kubectl get configmap order-service-config -o yaml

   # Check Secrets
   kubectl get secrets -o yaml
   ```

2. **Database connection failed**
   ```bash
   # Test database connectivity from pod
   kubectl exec -it <pod-name> -- psql -h postgres -U user -d orders_db
   ```

3. **Health check failing**
   ```bash
   # Test health endpoint manually
   kubectl port-forward <pod-name> 9001:9001
   curl http://localhost:9001/health
   ```

### High Error Rate

**Symptoms:**
- Error rate exceeds 1%
- Alert: `HighErrorRate`

**Diagnosis:**

```bash
# View recent errors in logs
kubectl logs -f deployment/order-service | grep ERROR

# Check for specific error codes
kubectl logs deployment/order-service | grep PAYMENT_DECLINED | wc -l

# Check metrics
curl http://order-service:9001/metrics | grep http_requests_total
```

**Common causes:**

1. **Upstream service down**
   ```bash
   # Check payment-processor status
   kubectl get pods -l app=payment-processor
   kubectl logs deployment/payment-processor
   ```

2. **Database connection pool exhausted**
   ```bash
   # Check active connections
   kubectl exec <postgres-pod> -- psql -c "SELECT count(*) FROM pg_stat_activity;"

   # Scale service down to reduce connections
   kubectl scale deployment order-service --replicas=1
   ```

3. **Rate limiting**
   ```bash
   # Check rate limit cache
   # Look for RATE_LIMIT_EXCEEDED errors
   kubectl logs deployment/api-gateway | grep RATE_LIMIT
   ```

### High Latency

**Symptoms:**
- Alert: `HighLatency`
- p95 latency > 500ms

**Diagnosis:**

```bash
# Check slow queries
kubectl exec <postgres-pod> -- psql -d orders_db -c \
  "SELECT query, calls, mean_time FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"

# Check Redis performance (if used)
redis-cli --latency

# Check network latency between services
kubectl run -it --rm debug --image=nicolaka/netshoot --restart=Never -- \
  ping order-service.ecommerce.svc.cluster.local
```

**Solutions:**

1. **Add database indexes**
   ```sql
   -- Example: index on frequently queried columns
   CREATE INDEX idx_orders_customer ON orders(customer_id, created_at);
   ANALYZE orders;
   ```

2. **Enable caching**
   ```bash
   # Scale up Redis
   kubectl scale deployment redis --replicas=3
   ```

3. **Scale service replicas**
   ```bash
   kubectl scale deployment order-service --replicas=5
   ```

### Memory Leak

**Symptoms:**
- Memory usage increases over time
- Pod crashes with OOMKilled
- Alert: memory usage > 80%

**Diagnosis:**

```bash
# Monitor memory trend
kubectl top pods -l app=order-service --containers

# Check memory usage over time (if using Prometheus)
# Query: rate(container_memory_usage_bytes[5m])

# Heap dump (for Node.js)
kubectl exec <pod> -- kill -USR2 $(pgrep node)

# Analyze heap
npm install -g clinic
clinic doctor node --collect-only ./app.js
```

**Solutions:**

1. **Restart pods** (temporary fix)
   ```bash
   kubectl rollout restart deployment/order-service
   ```

2. **Fix the leak** (permanent fix)
   - Check for event listener leaks
   - Check for circular references
   - Review code for uncleared caches
   - Add memory profiling

3. **Increase memory limit** (temporary workaround)
   ```bash
   kubectl set resources deployment order-service \
     --limits memory=1Gi \
     --requests memory=512Mi
   ```

### Database Connection Issues

**Symptoms:**
- Errors: "too many connections"
- Service timeout errors

**Diagnosis:**

```bash
# Check active connections
kubectl exec <postgres-pod> -- psql -c \
  "SELECT datname, usename, count(*) FROM pg_stat_activity GROUP BY datname, usename;"

# Check connection pool status
# Query the service's connection pool metrics
curl http://order-service:9001/metrics | grep db_connection_pool
```

**Solutions:**

1. **Increase connection pool**
   ```yaml
   # In deployment config
   env:
     - name: DB_POOL_SIZE
       value: "50"
   ```

2. **Reduce active replicas**
   ```bash
   kubectl scale deployment order-service --replicas=2
   ```

3. **Kill idle connections**
   ```sql
   SELECT pg_terminate_backend(pid)
   FROM pg_stat_activity
   WHERE state = 'idle' AND query_start < now() - interval '1 hour';
   ```

### Payment Processing Failures

**Symptoms:**
- Payment failures spike
- Alert: `PaymentDeclined` or `PaymentGatewayError`

**Diagnosis:**

```bash
# Check payment service logs
kubectl logs deployment/payment-processor | grep -i error

# Check payment gateway status
# Via payment processor dashboard or status page

# Test payment gateway connectivity
kubectl exec <payment-pod> -- curl -I https://api.stripe.com/health
```

**Solutions:**

1. **Verify API keys**
   ```bash
   # Check secrets
   kubectl get secret payment-keys -o yaml | grep STRIPE_SECRET_KEY
   ```

2. **Check rate limits**
   ```bash
   # Payment gateway may be rate limiting our requests
   kubectl logs deployment/payment-processor | grep "rate limit"
   ```

3. **Retry failed payments**
   ```bash
   # Manually retry from Order Service CLI
   kubectl exec -it <order-pod> -- npm run retry-failed-payments
   ```

## Debugging Techniques

### Port Forward to Service

```bash
# Forward local port to service
kubectl port-forward svc/order-service 9001:9001

# Now access service locally
curl http://localhost:9001/health
```

### Execute Commands in Pod

```bash
# Open shell
kubectl exec -it <pod-name> -- /bin/bash

# Run one-off command
kubectl exec <pod-name> -- npm run migrate:status

# Run with env vars
kubectl exec <pod-name> -- env | grep DB_
```

### Tail Logs from Multiple Pods

```bash
# Tail all order-service pods
kubectl logs -f -l app=order-service

# Tail with previous pod logs
kubectl logs -f <pod-name> --previous

# Tail with timestamps
kubectl logs -f <pod-name> --timestamps
```

### Check Events

```bash
# Get recent events in namespace
kubectl get events --sort-by='.lastTimestamp'

# Describe resource for events
kubectl describe pod <pod-name>
```

## Performance Tuning

### Database Query Optimization

```sql
-- Find slow queries (PostgreSQL)
SELECT query, mean_time, calls
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 20;

-- Explain slow query
EXPLAIN ANALYZE SELECT * FROM orders WHERE customer_id = 'usr_123' ORDER BY created_at DESC;

-- Add index for frequent query
CREATE INDEX idx_orders_customer_date ON orders(customer_id, created_at DESC);
```

### Caching Strategy

```bash
# Check Redis usage
redis-cli info stats

# Check cache hit rate
redis-cli info stats | grep keyspace_hits

# Clear cache if needed
redis-cli FLUSHDB
```

### Network Optimization

```bash
# Check service DNS resolution
kubectl exec <pod> -- nslookup order-service

# Measure network latency
kubectl exec <pod> -- ping -c 3 payment-processor

# Check network policies that might be blocking traffic
kubectl get networkpolicy -A
```

## Monitoring and Metrics

### Real-time Metrics

```bash
# Watch resource usage
watch -n 1 'kubectl top pods -l app=order-service'

# Monitor pod events
watch -n 1 'kubectl get pods -l app=order-service'

# Check queue depth (Kafka)
kafka-consumer-groups --bootstrap-server kafka:9092 --group order-service --describe
```

### Historical Analysis

Use Prometheus queries:

```
# Memory trend
rate(container_memory_usage_bytes{pod=~"order-service-.*"}[1h])

# CPU trend
rate(container_cpu_usage_seconds_total{pod=~"order-service-.*"}[1h])

# Error rate trend
rate(http_requests_total{status=~"5.."}[5m])
```

## Escalation Paths

### Level 1: Automated Response

- Pod restart
- Service scale-up
- Cache clearing
- Connection pool reset

### Level 2: On-Call Engineer

- Database analysis
- Code review for hotspots
- Dependency investigation
- Infrastructure changes

### Level 3: Architecture Review

- System design changes
- Database schema optimization
- Service redesign
- Infrastructure migration

## See Also

- [Monitoring & Alerts](monitoring.md)
- [Deployment Guide](deployment.md)
- [Architecture](../architecture.md)
- [REST API Standards](../protocols/rest-api.md)
