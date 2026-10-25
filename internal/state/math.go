package state

import (
	"math"
	"sort"
)

// CalculateHolisticScore computes the locked 360° Holistic Report score:
// 35% CBT + 25% Daily Quiz + 25% Attitude (MAD filtered) + 15% Attendance.
func CalculateHolisticScore(cbt, quiz, attitude, attendance float64) float64 {
	raw := 0.35*cbt + 0.25*quiz + 0.25*attitude + 0.15*attendance
	return math.Round(raw*10) / 10
}

// FilterOutlierAttitudeRatings implements the Median Absolute Deviation (MAD)
// Outlier Consensus Filter so a single unfair low rating without a violation
// record gets down-weighted to 0.15.
func FilterOutlierAttitudeRatings(scores []float64, hasViolation []bool) (filtered, median, mad, threshold float64, outlierCount int) {
	if len(scores) == 0 {
		return 100, 100, 0, 100, 0
	}
	median = calcMedian(scores)
	devs := make([]float64, len(scores))
	for i, s := range scores {
		devs[i] = math.Abs(s - median)
	}
	mad = calcMedian(devs)
	if mad < 2.0 {
		mad = 2.0
	}
	threshold = math.Round((median-2.5*mad)*10) / 10

	var totalWeighted, totalWeight float64
	for i, s := range scores {
		hasViol := i < len(hasViolation) && hasViolation[i]
		isOutlier := s < threshold && !hasViol
		weight := 1.0
		if isOutlier {
			weight = 0.15
			outlierCount++
		}
		totalWeighted += s * weight
		totalWeight += weight
	}
	if totalWeight > 0 {
		filtered = math.Round((totalWeighted/totalWeight)*10) / 10
	} else {
		filtered = median
	}
	return filtered, median, mad, threshold, outlierCount
}

func calcMedian(nums []float64) float64 {
	if len(nums) == 0 {
		return 0
	}
	cp := append([]float64(nil), nums...)
	sort.Float64s(cp)
	mid := len(cp) / 2
	if len(cp)%2 == 1 {
		return cp[mid]
	}
	return (cp[mid-1] + cp[mid]) / 2
}

// CalculateTeacherImpactScore computes the objective teacher rating.
func CalculateTeacherImpactScore(growth, invConfusion, leaderVal, badgeNorm, bonus float64) float64 {
	base := 0.45*growth + 0.25*invConfusion + 0.15*leaderVal + 0.15*badgeNorm + bonus
	if base > 100 {
		base = 100
	}
	return math.Round(base*10) / 10
}

// CalculateHaversineMeters returns the distance in meters between two GPS points.
func CalculateHaversineMeters(lat1, lon1, lat2, lon2 float64) int {
	const R = 6371000.0
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	dPhi := (lat2 - lat1) * math.Pi / 180
	dLam := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dPhi/2)*math.Sin(dPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLam/2)*math.Sin(dLam/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return int(math.Round(R * c))
}

// CalculateTransformationIndex computes the discipline recovery score and trend.
func CalculateTransformationIndex(activeCases, recovered, relapsed int) (float64, string) {
	total := recovered + relapsed
	if total <= 0 {
		return 75.0, "STABIL"
	}
	recRatio := float64(recovered) / float64(total)
	penalty := (float64(relapsed) / float64(total)) * 30.0
	score := math.Round((50.0+recRatio*50.0-penalty)*10) / 10
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	trend := "STABIL"
	if score >= 75 {
		trend = "NAIK"
	} else if score < 60 {
		trend = "TURUN"
	}
	return score, trend
}
