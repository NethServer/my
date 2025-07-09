# Background Processing

This package contains background tasks and processing systems that run independently from the main API request/response cycle.

## Statistics Cache System

The statistics cache system provides optimized performance for system-wide statistics by:

- **Background Updates**: Statistics are calculated every 5 minutes in background
- **Cached Responses**: API calls return cached data for sub-millisecond response times
- **Stale Detection**: Automatically detects and refreshes stale data (>15 minutes)
- **Large Scale Handling**: Uses pagination to efficiently process hundreds of thousands of organizations
- **Thread Safety**: Full concurrent access protection with mutex locks

### Usage

```go
// Start background updater (typically in main.go)
background.GetStatsCacheManager().StartBackgroundUpdater()

// Get cached statistics (in API handlers)
cacheManager := background.GetStatsCacheManager()
stats := cacheManager.GetStats()
```

### Configuration

- **Cache TTL**: 10 minutes
- **Update Interval**: 5 minutes
- **Stale Threshold**: 15 minutes
- **Page Size**: 100 organizations per API call
- **Rate Limiting**: 50ms delay between pages

### Future Extensions

This package is designed to accommodate additional background processing tasks:

- Cache warming systems
- Data synchronization tasks
- Batch processing jobs
- Scheduled maintenance operations
- Performance monitoring systems

Each background task should follow the same patterns:
- Singleton manager with thread safety
- Configurable intervals and timeouts
- Proper error handling and logging
- Graceful shutdown capabilities