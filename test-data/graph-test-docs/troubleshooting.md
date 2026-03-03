# Troubleshooting Guide

Common issues and their solutions.

## Database Connection Errors

**Problem**: Cannot connect to PostgreSQL

**Solutions**:
- Verify DATABASE_URL is correct
- Check PostgreSQL is running: `pg_isready`
- Ensure user has correct permissions
- Check firewall rules allow connections

## Authentication Token Expired

**Problem**: Requests return 401 Unauthorized

**Solutions**:
- Token may have expired (24-hour expiration)
- Refresh token with POST /auth/refresh
- Re-login and obtain new token
- Check JWT_SECRET matches on server

## Service Timeouts

**Problem**: Requests timeout or hang

**Solutions**:
- Check service logs for errors
- Verify database performance
- Check Redis is accessible
- Look for network connectivity issues
- Increase timeout values if needed

## Memory Leaks

**Problem**: Application memory usage increases over time

**Solutions**:
- Check for unreleased database connections
- Review goroutine count in logs
- Profile with pprof
- Check for circular references in code

## Payment Processing Failures

**Problem**: Payment requests fail intermittently

**Solutions**:
- Verify payment gateway credentials
- Check network connectivity to gateway
- Review payment logs for error codes
- Ensure sufficient funds available
- Contact payment provider if issues persist
