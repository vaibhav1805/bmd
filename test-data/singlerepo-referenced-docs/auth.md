# User Authentication

## Overview

This module handles all authentication concerns. It integrates with the [Notification System](alerts.md) to send login alerts.

## Implementation

We use JWT tokens issued after successful password verification. Sessions are managed independently of the [Payment Processing](payments.md) system.

## Token Format

```
header.payload.signature
```

## Integration Points

- See [overview.md](overview.md) for system context
- Login events are logged to [Analytics](metrics.md)
- Security alerts sent via [Notifications](alerts.md)
