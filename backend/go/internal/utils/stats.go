package utils

// Useage
// AnalyzeSpreadZScore: Now uses utils.CalculateZScore for Z-score computation
// DetectDepthAnomaly: Now uses utils.CalculateMean and utils.CalculateStdDev for statistical calculations
// CalculatePercentile: Now delegates to utils.CalculatePercentile instead of implementing its own logic
// PerformLinearRegression: Now uses utils.PerformLinearRegression instead of implementing its own regression logic
import (
	"math"
)

// CalculateMean calculates the mean of a slice of float64 values
func CalculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// CalculateStdDev calculates the standard deviation of a slice of float64 values
func CalculateStdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := CalculateMean(values)

	var sumSquares float64
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}

	variance := sumSquares / float64(len(values)-1) // Using sample standard deviation
	if variance < 0 {
		variance = 0 // Prevent negative variance due to floating point errors
	}

	return math.Sqrt(variance)
}

// CalculateZScore calculates the Z-score of a value relative to a dataset
func CalculateZScore(value float64, values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := CalculateMean(values)
	stdDev := CalculateStdDev(values)

	if stdDev == 0 {
		return 0 // All values are the same, so Z-score is 0
	}

	return (value - mean) / stdDev
}

// CalculatePercentile calculates the percentile of a sorted slice of float64 values
// 计算一个已排序的 float64 类型切片的百分位数
func CalculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Note: This assumes values are already sorted
	if len(values) == 1 {
		return values[0]
	}

	// Calculate the position
	n := len(values)
	pos := (percentile / 100.0) * float64(n-1)

	// If position is an integer, return the value at that position
	if pos == float64(int(pos)) {
		return values[int(pos)]
	}

	// Otherwise, interpolate between the two nearest values
	lowerIndex := int(math.Floor(pos))
	upperIndex := int(math.Ceil(pos))

	if lowerIndex < 0 {
		lowerIndex = 0
	}
	if upperIndex >= n {
		upperIndex = n - 1
	}

	// Linear interpolation
	weight := pos - float64(lowerIndex)
	return values[lowerIndex] + weight*(values[upperIndex]-values[lowerIndex])
}

// PerformLinearRegression
// performs linear regression on x, y data points
// Returns slope and intercept
// 对给定的 x 和 y 数据点进行线性回归分析，并返回拟合直线的斜率与截距
// 设y=ax+b；a是斜率，b是截距
// 实现原理：
// 代码使用的是最小二乘法的解析解（正规方程），这种方式对于小型、条件良好的数据集非常快速和准确。而 numpy.polyfit或 scikit-learn的底层可能会使用奇异值分解（SVD）​ 等数值计算方法。SVD 的优势在于数值稳定性更高，即使在可能因矩阵“病态”而失败的情况下，它也能给出一个解
//
// 根据美联储的研究文献：
// 流动性应该是"弹性"的 - 能够适应市场条件变化
// 过度集中或过度分散都不健康
// 适度波动是市场健康的标志
//
// 健康市场的流动性特征：
// ✅ 稳定在合理区间内波动
// ✅ 具有反周期调节能力
// ✅ 能够承受适度冲击
// ❌ 持续下降是危险信号
// ❌ 过度波动也是问题
// 所以我们的预警系统检测流动性下降趋势是完全正确和必要的！
//
// 线性回归仍有价值，但不应单独使用：
// ✅ 适合场景：
// 初步趋势筛选
// 与其他方法结合验证
// 长期趋势分析
// ❌ 不适合场景：
// 短期高频预警
// 极端市场条件下
// 需要高精度的情况
func PerformLinearRegression(x, y []float64) (slope, intercept float64) {
	n := len(x)
	if n != len(y) || n < 2 {
		return 0, 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i := 0; i < n; i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	denominator := float64(n)*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0, 0
	}

	slope = (float64(n)*sumXY - sumX*sumY) / denominator
	intercept = (sumY - slope*sumX) / float64(n)

	return slope, intercept
}
