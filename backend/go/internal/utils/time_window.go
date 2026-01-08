package utils

import (
	"sync"
	"time"
)

// TimeWindowItem represents an item in a time window with timestamp
type TimeWindowItem interface {
	GetTimestamp() int64
}

// GenericTimeWindow provides a thread-safe sliding time window implementation
type GenericTimeWindow struct {
	items    []TimeWindowItem
	duration int64 // window duration in seconds
	mutex    sync.RWMutex
}

// NewGenericTimeWindow creates a new time window with specified duration
func NewGenericTimeWindow(durationSeconds int64) *GenericTimeWindow {
	return &GenericTimeWindow{
		items:    make([]TimeWindowItem, 0),
		duration: durationSeconds,
	}
}

// Add adds an item to the time window and automatically removes expired items
func (tw *GenericTimeWindow) Add(item TimeWindowItem) {
	tw.mutex.Lock()
	defer tw.mutex.Unlock()

	currentTime := time.Now().Unix()
	cutoffTime := currentTime - tw.duration

	// Add new item
	tw.items = append(tw.items, item)

	// Remove expired items from the beginning
	startIndex := 0
	for i, existingItem := range tw.items {
		if existingItem.GetTimestamp() > cutoffTime {
			startIndex = i
			break
		}
	}
	tw.items = tw.items[startIndex:]
}

// GetItems returns all items currently in the window
func (tw *GenericTimeWindow) GetItems() []TimeWindowItem {
	tw.mutex.RLock()
	defer tw.mutex.RUnlock()

	// Return a copy to prevent external modification
	itemsCopy := make([]TimeWindowItem, len(tw.items))
	copy(itemsCopy, tw.items)
	return itemsCopy
}

// GetItemCount returns the number of items in the window
func (tw *GenericTimeWindow) GetItemCount() int {
	tw.mutex.RLock()
	defer tw.mutex.RUnlock()
	return len(tw.items)
}

// Clear removes all items from the window
func (tw *GenericTimeWindow) Clear() {
	tw.mutex.Lock()
	defer tw.mutex.Unlock()
	tw.items = tw.items[:0]
}

// GetDuration returns the window duration in seconds
func (tw *GenericTimeWindow) GetDuration() int64 {
	return tw.duration
}

// TimeWindowWithValue is a convenience wrapper for simple value-based time windows
type TimeWindowWithValue struct {
	window *GenericTimeWindow
}

// TimeWindowValueItem implements TimeWindowItem for simple values
type TimeWindowValueItem struct {
	Value     float64
	Timestamp int64
}

// GetTimestamp implements TimeWindowItem interface
func (item *TimeWindowValueItem) GetTimestamp() int64 {
	return item.Timestamp
}

// NewTimeWindowWithValue creates a time window for simple float64 values
func NewTimeWindowWithValue(durationSeconds int64) *TimeWindowWithValue {
	return &TimeWindowWithValue{
		window: NewGenericTimeWindow(durationSeconds),
	}
}

// AddValue adds a float64 value to the time window
func (twv *TimeWindowWithValue) AddValue(value float64) {
	item := &TimeWindowValueItem{
		Value:     value,
		Timestamp: time.Now().Unix(),
	}
	twv.window.Add(item)
}

// GetValues returns all values currently in the window
func (twv *TimeWindowWithValue) GetValues() []float64 {
	items := twv.window.GetItems()
	values := make([]float64, len(items))
	for i, item := range items {
		if valueItem, ok := item.(*TimeWindowValueItem); ok {
			values[i] = valueItem.Value
		}
	}
	return values
}

// GetValueCount returns the number of values in the window
func (twv *TimeWindowWithValue) GetValueCount() int {
	return twv.window.GetItemCount()
}

// Clear clears all values from the window
func (twv *TimeWindowWithValue) Clear() {
	twv.window.Clear()
}

// GetDuration returns the window duration
func (twv *TimeWindowWithValue) GetDuration() int64 {
	return twv.window.GetDuration()
}

// TimeBasedFilter provides utility functions for time-based filtering
type TimeBasedFilter struct{}

// FilterByTimeWindow filters items based on time window
func (tf *TimeBasedFilter) FilterByTimeWindow(items []TimeWindowItem, windowSeconds int64) []TimeWindowItem {
	if len(items) == 0 || windowSeconds <= 0 {
		return items
	}

	currentTime := time.Now().Unix()
	cutoffTime := currentTime - windowSeconds

	filtered := make([]TimeWindowItem, 0)
	for _, item := range items {
		if item.GetTimestamp() >= cutoffTime {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// GetRecentItems gets the most recent N items within time window
func (tf *TimeBasedFilter) GetRecentItems(items []TimeWindowItem, maxCount int, windowSeconds int64) []TimeWindowItem {
	if len(items) == 0 || maxCount <= 0 {
		return []TimeWindowItem{}
	}

	// First filter by time window
	filtered := tf.FilterByTimeWindow(items, windowSeconds)

	// Then take the most recent items (they should be sorted by timestamp)
	if len(filtered) <= maxCount {
		return filtered
	}

	// Return the most recent items (assuming items are sorted by timestamp ascending)
	startIndex := len(filtered) - maxCount
	return filtered[startIndex:]
}

// NewTimeBasedFilter creates a new time-based filter
func NewTimeBasedFilter() *TimeBasedFilter {
	return &TimeBasedFilter{}
}
