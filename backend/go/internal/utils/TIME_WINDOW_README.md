# Time Window Utility Package

This package provides generic, reusable time window implementations for market data analysis.

## Features

### 1. GenericTimeWindow
A thread-safe, generic sliding time window that automatically manages item expiration.

```go
// Create a 30-minute window
window := utils.NewGenericTimeWindow(1800) // 1800 seconds = 30 minutes

// Add items (must implement TimeWindowItem interface)
window.Add(myTimeWindowItem)

// Get current items
items := window.GetItems()
count := window.GetItemCount()
```

### 2. TimeWindowWithValue
Convenience wrapper for simple float64 value time windows.

```go
// Create a sentiment smoothing window (30 seconds)
sentimentWindow := utils.NewTimeWindowWithValue(30)

// Add values
sentimentWindow.AddValue(0.5)
sentimentWindow.AddValue(0.3)
sentimentWindow.AddValue(0.7)

// Get values
values := sentimentWindow.GetValues()
average := calculateAverage(values)
```

### 3. TimeBasedFilter
Utility functions for time-based filtering of existing data collections.

```go
filter := utils.NewTimeBasedFilter()

// Filter items by time window
recentItems := filter.FilterByTimeWindow(allItems, 300) // Last 5 minutes

// Get most recent N items within time window
latestItems := filter.GetRecentItems(allItems, 10, 600) // 10 most recent in last 10 minutes
```

## Benefits

### Before (Manual Management)
```go
// Complex manual window management
currentTime := time.Now().Unix()
cutoffTime := currentTime - int64(windowSeconds)
startIndex := 0
for i, item := range items {
    if item.Timestamp > cutoffTime {
        startIndex = i
        break
    }
}
items = items[startIndex:]
```

### After (Using Utilities)
```go
// Clean, automatic management
window := utils.NewGenericTimeWindow(windowSeconds)
window.Add(item)
currentItems := window.GetItems()
```

## Key Advantages

1. **Automatic Expiration**: Items automatically expire based on timestamp
2. **Thread Safety**: Built-in mutex protection for concurrent access
3. **Type Safety**: Generic interface ensures compile-time type checking
4. **Performance**: Efficient O(n) cleanup during additions
5. **Reusability**: Works with any data type that implements TimeWindowItem
6. **Memory Efficiency**: Automatically removes expired items

## Usage Examples

### Market Data Applications

1. **Liquidity Analysis**: Track liquidity metrics over time windows
2. **Price Action**: Monitor support/resistance levels with expiration
3. **Order Book Depth**: Analyze depth changes within time frames
4. **Sentiment Smoothing**: Apply time-based averaging to sentiment scores
5. **Volatility Calculation**: Compute rolling volatility metrics

### Integration Pattern

```go
type MyDataManager struct {
    liquidityWindows map[string]*utils.GenericTimeWindow
    depthWindows     map[string]*utils.TimeWindowWithValue
}

func (m *MyDataManager) AddLiquidityData(instID string, metrics *LiquidityMetrics) {
    if m.liquidityWindows[instID] == nil {
        m.liquidityWindows[instID] = utils.NewGenericTimeWindow(1800) // 30 min window
    }
    
    item := &LiquidityWindowItemWrapper{Metrics: *metrics, Timestamp: time.Now().Unix()}
    m.liquidityWindows[instID].Add(item)
}
```

## Performance Considerations

- **Memory**: Automatic cleanup prevents unbounded growth
- **CPU**: O(n) cleanup cost amortized across operations  
- **Concurrency**: Read-write mutex allows concurrent reads
- **Scalability**: Efficient for thousands of instruments with individual windows

This utility significantly reduces boilerplate code and eliminates common bugs in time window management.