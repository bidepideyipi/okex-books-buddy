# 时间窗口工具包

该包为市场数据分析提供了通用、可重用的时间窗口实现。

## 功能特性

### 1. GenericTimeWindow（通用时间窗口）
一个线程安全的通用滑动时间窗口，自动管理项目过期。

```go
// 创建一个30分钟的窗口
window := utils.NewGenericTimeWindow(1800) // 1800秒 = 30分钟

// 添加项目（必须实现TimeWindowItem接口）
window.Add(myTimeWindowItem)

// 获取当前项目
items := window.GetItems()
count := window.GetItemCount()
```

### 2. TimeWindowWithValue（带值时间窗口）
为简单float64值时间窗口提供的便利包装器。

```go
// 创建情绪平滑窗口（30秒）
sentimentWindow := utils.NewTimeWindowWithValue(30)

// 添加值
sentimentWindow.AddValue(0.5)
sentimentWindow.AddValue(0.3)
sentimentWindow.AddValue(0.7)

// 获取值
values := sentimentWindow.GetValues()
average := calculateAverage(values)
```

### 3. TimeBasedFilter（基于时间的过滤器）
用于对现有数据集合进行基于时间过滤的工具函数。

```go
filter := utils.NewTimeBasedFilter()

// 按时间窗口过滤项目
recentItems := filter.FilterByTimeWindow(allItems, 300) // 最近5分钟

// 获取时间窗口内最近的N个项目
latestItems := filter.GetRecentItems(allItems, 10, 600) // 最近10分钟内的10个最新项目
```

## 优势对比

### 重构前（手工管理）
```go
// 复杂的手工窗口管理
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

### 重构后（使用工具类）
```go
// 简洁的自动管理
window := utils.NewGenericTimeWindow(windowSeconds)
window.Add(item)
currentItems := window.GetItems()
```

## 自动清理机制

时间窗口采用惰性清理策略，在每次添加新项目时自动移除过期数据：

1. **触发时机**：每次调用`Add()`方法时
2. **计算逻辑**：`cutoffTime = currentTime - windowDuration`
3. **清理过程**：
   - 从列表开头遍历项目
   - 找到第一个时间戳 > cutoffTime 的项目位置
   - 删除该位置之前的所有过期项目
   - 保留有效期内的项目
4. **性能特点**：O(n)时间复杂度，摊销到每次添加操作中

## 核心优势

1. **自动过期**：项目根据时间戳自动过期
2. **线程安全**：内置互斥锁保护并发访问
3. **类型安全**：泛型接口确保编译时类型检查
4. **性能优化**：添加过程中的O(n)清理成本
5. **高度复用**：适用于任何实现TimeWindowItem接口的数据类型
6. **内存高效**：自动移除过期项目

## 使用示例

### 市场数据应用场景

1. **流动性分析**：跟踪时间窗口内的流动性指标
2. **价格行为**：监控有过期机制的支撑/阻力位
3. **订单簿深度**：分析时间框架内的深度变化
4. **情绪平滑**：对情绪评分应用基于时间的平均
5. **波动率计算**：计算滚动波动率指标

### 集成模式

```go
type MyDataManager struct {
    liquidityWindows map[string]*utils.GenericTimeWindow
    depthWindows     map[string]*utils.TimeWindowWithValue
}

func (m *MyDataManager) AddLiquidityData(instID string, metrics *LiquidityMetrics) {
    if m.liquidityWindows[instID] == nil {
        m.liquidityWindows[instID] = utils.NewGenericTimeWindow(1800) // 30分钟窗口
    }
    
    item := &LiquidityWindowItemWrapper{Metrics: *metrics, Timestamp: time.Now().Unix()}
    m.liquidityWindows[instID].Add(item)
}
```

## 性能考量

- **内存**：自动清理防止无限制增长
  - 清理机制：每次调用`Add()`方法时，会计算当前时间减去窗口持续时间得到截止时间(cutoffTime)
  - 遍历现有项目列表，找到第一个时间戳大于截止时间的项目索引
  - 删除该索引之前的所有过期项目，只保留有效期内的项目
  - 这种惰性清理策略确保了O(n)的时间复杂度，同时避免频繁的内存分配
- **CPU**：O(n)清理成本在操作间摊销
- **并发**：读写互斥锁允许并发读取
- **可扩展性**：对于具有独立窗口的数千种交易对都很高效

这个工具显著减少了样板代码，消除了时间窗口管理中的常见错误。

## 技术细节

### 内存管理
- 使用切片重切片技术(`items[startIndex:]`)避免创建新数组
- 读操作返回副本防止外部修改影响内部状态
- 内建互斥锁确保并发安全

### 时间复杂度
- 添加操作：O(n) - n为需要清理的过期项目数量
- 查询操作：O(1) - 直接返回切片长度或复制切片
- 清理成本摊销到每次添加操作中，整体效率很高