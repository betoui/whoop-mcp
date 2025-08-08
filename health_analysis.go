package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// HealthAnalyzer provides health data analysis for therapeutic insights
type HealthAnalyzer struct {
	// In-memory cache for analysis results
	cache map[string]interface{}
}

// NewHealthAnalyzer creates a new health analyzer instance
func NewHealthAnalyzer() *HealthAnalyzer {
	return &HealthAnalyzer{
		cache: make(map[string]interface{}),
	}
}

// AnalyzeHealthSummary creates a comprehensive health summary for therapy sessions
func (h *HealthAnalyzer) AnalyzeHealthSummary(recoveries []WhoopRecovery, sleepData []WhoopSleep, workouts []WhoopWorkout, cycles []WhoopCycle, startDate, endDate time.Time, userID int) (*HealthSummary, error) {

	// Analyze recovery trends
	recoveryTrend := h.analyzeRecoveryTrend(recoveries)

	// Analyze sleep patterns
	sleepAnalysis := h.analyzeSleepPatterns(sleepData)

	// Analyze stress indicators
	stressIndicators := h.analyzeStressIndicators(recoveries, sleepData)

	// Analyze activity patterns
	activityPatterns := h.analyzeActivityPatterns(workouts, cycles)

	// Generate therapy insights
	therapyInsights := h.generateTherapyInsights(recoveryTrend, sleepAnalysis, stressIndicators, activityPatterns)

	// Detect red flags
	redFlags := h.detectRedFlags(recoveries, sleepData, workouts, stressIndicators)

	summary := &HealthSummary{
		UserID: userID,
		DateRange: DateRange{
			Start: startDate,
			End:   endDate,
		},
		RecoveryTrend:    recoveryTrend,
		SleepAnalysis:    sleepAnalysis,
		StressIndicators: stressIndicators,
		ActivityPatterns: activityPatterns,
		TherapyInsights:  therapyInsights,
		RedFlags:         redFlags,
	}

	return summary, nil
}

// analyzeRecoveryTrend analyzes recovery score trends and patterns
func (h *HealthAnalyzer) analyzeRecoveryTrend(recoveries []WhoopRecovery) RecoveryTrend {
	if len(recoveries) == 0 {
		return RecoveryTrend{
			Trend: "no_data",
		}
	}

	// Sort by creation date
	sort.Slice(recoveries, func(i, j int) bool {
		return recoveries[i].CreatedAt.Before(recoveries[j].CreatedAt)
	})

	var scores []float64
	var lastSevenDays []float64

	for i, recovery := range recoveries {
		score := recovery.Score.RecoveryScore
		scores = append(scores, score)

		// Last 7 days for trend analysis
		if i >= len(recoveries)-7 {
			lastSevenDays = append(lastSevenDays, score)
		}
	}

	// Calculate average
	average := h.calculateMean(scores)

	// Calculate consistency (standard deviation)
	consistency := 1.0 - (h.calculateStdDev(scores) / 100.0)
	if consistency < 0 {
		consistency = 0
	}

	// Determine trend
	trend := "stable"
	weeklyChange := 0.0

	if len(scores) >= 7 {
		firstHalf := scores[:len(scores)/2]
		secondHalf := scores[len(scores)/2:]

		firstAvg := h.calculateMean(firstHalf)
		secondAvg := h.calculateMean(secondHalf)

		weeklyChange = secondAvg - firstAvg

		if weeklyChange > 5 {
			trend = "improving"
		} else if weeklyChange < -5 {
			trend = "declining"
		}
	}

	return RecoveryTrend{
		AverageScore:     average,
		Trend:            trend,
		WeeklyChange:     weeklyChange,
		ConsistencyScore: consistency,
		LastSevenDays:    lastSevenDays,
	}
}

// analyzeSleepPatterns analyzes sleep quality and patterns for mental health indicators
func (h *HealthAnalyzer) analyzeSleepPatterns(sleepData []WhoopSleep) SleepAnalysis {
	if len(sleepData) == 0 {
		return SleepAnalysis{
			SleepQualityTrend: "no_data",
		}
	}

	var totalSleepHours []float64
	var efficiencies []float64
	var debts []float64
	var disturbances []int

	for _, sleep := range sleepData {
		// Calculate sleep duration in hours
		sleepDuration := float64(sleep.Score.StageSummary.TotalInBedTimeMilli-sleep.Score.StageSummary.TotalAwakeTimeMilli) / (1000 * 60 * 60)
		totalSleepHours = append(totalSleepHours, sleepDuration)

		// Sleep efficiency (changed in V2)
		efficiency := sleep.Score.SleepEfficiencyPercentage / 100.0 // Convert percentage to decimal
		efficiencies = append(efficiencies, efficiency)

		// Sleep debt calculation
		needed := float64(sleep.Score.SleepNeeded.BaselineMilli+sleep.Score.SleepNeeded.NeedFromSleepDebtMilli) / (1000 * 60 * 60)
		debt := needed - sleepDuration
		debts = append(debts, debt)

		// Disturbances
		disturbances = append(disturbances, sleep.Score.StageSummary.DisturbanceCount)
	}

	avgHours := h.calculateMean(totalSleepHours)
	avgEfficiency := h.calculateMean(efficiencies)
	avgDebt := h.calculateMean(debts)
	avgDisturbances := float64(h.calculateMeanInt(disturbances))

	// Calculate consistency based on variance in sleep times
	consistency := 1.0 - (h.calculateStdDev(totalSleepHours) / 8.0) // Normalize to 8-hour baseline
	if consistency < 0 {
		consistency = 0
	}

	// Determine optimal bedtime (simplified analysis)
	optimalBedtime := "22:00" // Default recommendation

	// Determine sleep quality trend
	qualityTrend := "stable"
	if len(efficiencies) >= 7 {
		firstHalf := efficiencies[:len(efficiencies)/2]
		secondHalf := efficiencies[len(efficiencies)/2:]

		if h.calculateMean(secondHalf) > h.calculateMean(firstHalf)+0.05 {
			qualityTrend = "improving"
		} else if h.calculateMean(secondHalf) < h.calculateMean(firstHalf)-0.05 {
			qualityTrend = "declining"
		}
	}

	return SleepAnalysis{
		AverageHours:         avgHours,
		AverageEfficiency:    avgEfficiency,
		AverageDebt:          avgDebt,
		ConsistencyScore:     consistency,
		DisturbanceFrequency: avgDisturbances,
		OptimalBedtime:       optimalBedtime,
		SleepQualityTrend:    qualityTrend,
	}
}

// analyzeStressIndicators identifies physiological stress markers
func (h *HealthAnalyzer) analyzeStressIndicators(recoveries []WhoopRecovery, sleepData []WhoopSleep) StressIndicators {
	if len(recoveries) == 0 {
		return StressIndicators{
			StressLevel: "unknown",
		}
	}

	var hrvValues []float64
	var restingHRValues []float64
	var recoveryScores []float64

	elevatedHRVDays := 0
	highRestingHRDays := 0
	poorRecoveryStreak := 0
	currentPoorStreak := 0

	for _, recovery := range recoveries {
		hrv := recovery.Score.HRVRmssd
		rhr := recovery.Score.RestingHeartRate
		score := recovery.Score.RecoveryScore

		hrvValues = append(hrvValues, hrv)
		restingHRValues = append(restingHRValues, rhr)
		recoveryScores = append(recoveryScores, score)

		// Check for elevated HRV (indicating potential stress)
		if len(hrvValues) > 1 {
			avgHRV := h.calculateMean(hrvValues[:len(hrvValues)-1])
			if hrv > avgHRV*1.2 { // 20% above baseline
				elevatedHRVDays++
			}
		}

		// Check for elevated resting heart rate
		if len(restingHRValues) > 1 {
			avgRHR := h.calculateMean(restingHRValues[:len(restingHRValues)-1])
			if rhr > avgRHR+10 { // 10 bpm above baseline
				highRestingHRDays++
			}
		}

		// Track poor recovery streaks
		if score < 33 { // Poor recovery threshold
			currentPoorStreak++
			if currentPoorStreak > poorRecoveryStreak {
				poorRecoveryStreak = currentPoorStreak
			}
		} else {
			currentPoorStreak = 0
		}
	}

	// Calculate physiological stress score (0-100)
	stressFactors := 0.0
	if len(recoveries) > 0 {
		stressFactors += float64(elevatedHRVDays) / float64(len(recoveries)) * 30   // 30% weight
		stressFactors += float64(highRestingHRDays) / float64(len(recoveries)) * 25 // 25% weight
		stressFactors += float64(poorRecoveryStreak) / 7.0 * 25                     // 25% weight

		avgRecovery := h.calculateMean(recoveryScores)
		if avgRecovery < 50 {
			stressFactors += (50 - avgRecovery) / 50 * 20 // 20% weight
		}
	}

	// Determine stress level
	stressLevel := "low"
	if stressFactors > 70 {
		stressLevel = "critical"
	} else if stressFactors > 50 {
		stressLevel = "high"
	} else if stressFactors > 30 {
		stressLevel = "moderate"
	}

	return StressIndicators{
		ElevatedHRVDays:     elevatedHRVDays,
		HighRestingHRDays:   highRestingHRDays,
		PoorRecoveryStreak:  poorRecoveryStreak,
		StressLevel:         stressLevel,
		PhysiologicalStress: stressFactors,
	}
}

// analyzeActivityPatterns analyzes workout patterns and exercise habits
func (h *HealthAnalyzer) analyzeActivityPatterns(workouts []WhoopWorkout, cycles []WhoopCycle) ActivityPatterns {
	if len(workouts) == 0 && len(cycles) == 0 {
		return ActivityPatterns{
			IntensityBalance: "unknown",
			OvertrainingRisk: "unknown",
		}
	}

	weeklyWorkouts := len(workouts)
	if len(workouts) > 0 {
		// Estimate weekly count based on data duration
		firstWorkout := workouts[0].Start
		lastWorkout := workouts[len(workouts)-1].Start
		daysDiff := lastWorkout.Sub(firstWorkout).Hours() / 24
		if daysDiff > 0 {
			weeklyWorkouts = int(float64(len(workouts)) * 7.0 / daysDiff)
		}
	}

	var strainValues []float64
	for _, workout := range workouts {
		strainValues = append(strainValues, workout.Score.Strain)
	}

	// Add cycle strain data
	for _, cycle := range cycles {
		strainValues = append(strainValues, cycle.Score.Strain)
	}

	avgStrain := 0.0
	if len(strainValues) > 0 {
		avgStrain = h.calculateMean(strainValues)
	}

	// Calculate workout consistency
	consistency := 0.0
	if len(workouts) > 1 {
		var daysBetweenWorkouts []float64
		for i := 1; i < len(workouts); i++ {
			days := workouts[i].Start.Sub(workouts[i-1].Start).Hours() / 24
			daysBetweenWorkouts = append(daysBetweenWorkouts, days)
		}
		consistency = 1.0 - (h.calculateStdDev(daysBetweenWorkouts) / 7.0) // Normalize to weekly
		if consistency < 0 {
			consistency = 0
		}
	}

	// Determine overtraining risk
	overtrainingRisk := "low"
	if avgStrain > 18 && weeklyWorkouts > 6 {
		overtrainingRisk = "high"
	} else if avgStrain > 15 && weeklyWorkouts > 5 {
		overtrainingRisk = "moderate"
	}

	// Count active recovery days (low strain days)
	activeRecoveryDays := 0
	for _, strain := range strainValues {
		if strain > 0 && strain < 10 {
			activeRecoveryDays++
		}
	}

	// Determine intensity balance
	intensityBalance := "balanced"
	highIntensityDays := 0
	for _, strain := range strainValues {
		if strain > 15 {
			highIntensityDays++
		}
	}

	if len(strainValues) > 0 {
		highIntensityRatio := float64(highIntensityDays) / float64(len(strainValues))
		if highIntensityRatio > 0.5 {
			intensityBalance = "high_intensity_focused"
		} else if highIntensityRatio < 0.2 {
			intensityBalance = "low_intensity_focused"
		}
	}

	return ActivityPatterns{
		WeeklyWorkouts:     weeklyWorkouts,
		AverageStrain:      avgStrain,
		WorkoutConsistency: consistency,
		OvertrainingRisk:   overtrainingRisk,
		ActiveRecoveryDays: activeRecoveryDays,
		IntensityBalance:   intensityBalance,
	}
}

// generateTherapyInsights creates actionable insights for therapy sessions
func (h *HealthAnalyzer) generateTherapyInsights(recovery RecoveryTrend, sleep SleepAnalysis, stress StressIndicators, activity ActivityPatterns) []TherapyInsight {
	var insights []TherapyInsight

	// Recovery insights
	if recovery.Trend == "declining" {
		insights = append(insights, TherapyInsight{
			Category:   "recovery",
			Insight:    fmt.Sprintf("Recovery scores have declined by %.1f%% recently, which may indicate increased stress or inadequate rest", math.Abs(recovery.WeeklyChange)),
			Severity:   "concern",
			Actionable: true,
			Suggestion: "Consider discussing stress management techniques and sleep hygiene improvements",
		})
	}

	if recovery.ConsistencyScore < 0.6 {
		insights = append(insights, TherapyInsight{
			Category:   "recovery",
			Insight:    "Recovery scores show high variability, suggesting inconsistent stress levels or sleep patterns",
			Severity:   "info",
			Actionable: true,
			Suggestion: "Explore daily routine consistency and identify potential stressors causing fluctuations",
		})
	}

	// Sleep insights
	if sleep.AverageHours < 7 {
		severity := "concern"
		if sleep.AverageHours < 6 {
			severity = "alert"
		}
		insights = append(insights, TherapyInsight{
			Category:   "sleep",
			Insight:    fmt.Sprintf("Average sleep duration of %.1f hours is below recommended 7-9 hours", sleep.AverageHours),
			Severity:   severity,
			Actionable: true,
			Suggestion: "Discuss sleep barriers and develop a personalized sleep improvement plan",
		})
	}

	if sleep.AverageEfficiency < 0.85 {
		insights = append(insights, TherapyInsight{
			Category:   "sleep",
			Insight:    fmt.Sprintf("Sleep efficiency of %.1f%% indicates difficulty staying asleep", sleep.AverageEfficiency*100),
			Severity:   "concern",
			Actionable: true,
			Suggestion: "Explore factors affecting sleep quality such as anxiety, environment, or habits",
		})
	}

	if sleep.SleepQualityTrend == "declining" {
		insights = append(insights, TherapyInsight{
			Category:   "sleep",
			Insight:    "Sleep quality has been declining, which may impact mood and cognitive function",
			Severity:   "concern",
			Actionable: true,
			Suggestion: "Investigate recent life changes or stressors that might be affecting sleep",
		})
	}

	// Stress insights
	if stress.StressLevel == "critical" || stress.StressLevel == "high" {
		insights = append(insights, TherapyInsight{
			Category:   "stress",
			Insight:    "Physiological markers indicate elevated stress levels that may be impacting overall well-being",
			Severity:   "alert",
			Actionable: true,
			Suggestion: "Prioritize stress reduction techniques and consider addressing underlying stressors",
		})
	}

	if stress.PoorRecoveryStreak >= 3 {
		insights = append(insights, TherapyInsight{
			Category:   "stress",
			Insight:    fmt.Sprintf("Extended period of poor recovery (%d days) suggests chronic stress or burnout", stress.PoorRecoveryStreak),
			Severity:   "alert",
			Actionable: true,
			Suggestion: "Evaluate workload, relationships, and coping mechanisms for signs of overwhelm",
		})
	}

	// Activity insights
	if activity.OvertrainingRisk == "high" {
		insights = append(insights, TherapyInsight{
			Category:   "activity",
			Insight:    "High training load may be contributing to physical and mental stress",
			Severity:   "concern",
			Actionable: true,
			Suggestion: "Discuss the role of exercise in stress management and potential need for recovery time",
		})
	}

	if activity.WeeklyWorkouts == 0 {
		insights = append(insights, TherapyInsight{
			Category:   "activity",
			Insight:    "Lack of recorded physical activity may indicate low energy or motivation",
			Severity:   "info",
			Actionable: true,
			Suggestion: "Explore barriers to physical activity and discuss gentle movement as mood support",
		})
	}

	return insights
}

// detectRedFlags identifies critical health patterns requiring immediate attention
func (h *HealthAnalyzer) detectRedFlags(recoveries []WhoopRecovery, sleepData []WhoopSleep, workouts []WhoopWorkout, stress StressIndicators) []RedFlag {
	var redFlags []RedFlag

	// Critical stress indicators
	if stress.StressLevel == "critical" {
		redFlags = append(redFlags, RedFlag{
			Type:           "chronic_stress",
			Description:    "Multiple physiological stress markers indicate potential burnout or chronic stress condition",
			Severity:       "critical",
			DetectedAt:     time.Now(),
			Recommendation: "Consider immediate stress intervention and possible medical evaluation",
		})
	}

	// Extended poor recovery
	if stress.PoorRecoveryStreak >= 7 {
		redFlags = append(redFlags, RedFlag{
			Type:           "extended_poor_recovery",
			Description:    fmt.Sprintf("Recovery scores have been poor for %d consecutive days", stress.PoorRecoveryStreak),
			Severity:       "high",
			DetectedAt:     time.Now(),
			Recommendation: "Evaluate for signs of depression, anxiety, or physical health issues",
		})
	}

	// Severe sleep deprivation
	if len(sleepData) > 0 {
		var recentSleep []float64
		recentDays := 3
		if len(sleepData) < recentDays {
			recentDays = len(sleepData)
		}

		for i := len(sleepData) - recentDays; i < len(sleepData); i++ {
			sleepHours := float64(sleepData[i].Score.StageSummary.TotalInBedTimeMilli-sleepData[i].Score.StageSummary.TotalAwakeTimeMilli) / (1000 * 60 * 60)
			recentSleep = append(recentSleep, sleepHours)
		}

		avgRecentSleep := h.calculateMean(recentSleep)
		if avgRecentSleep < 5 {
			redFlags = append(redFlags, RedFlag{
				Type:           "severe_sleep_deprivation",
				Description:    fmt.Sprintf("Average sleep in recent %d days is critically low (%.1f hours)", recentDays, avgRecentSleep),
				Severity:       "critical",
				DetectedAt:     time.Now(),
				Recommendation: "Immediate sleep assessment and intervention required",
			})
		}
	}

	// Sudden dramatic changes in recovery
	if len(recoveries) >= 7 {
		recentScores := make([]float64, 0, 3)
		baselineScores := make([]float64, 0, 7)

		for i := len(recoveries) - 3; i < len(recoveries); i++ {
			recentScores = append(recentScores, recoveries[i].Score.RecoveryScore)
		}

		for i := len(recoveries) - 10; i < len(recoveries)-3 && i >= 0; i++ {
			baselineScores = append(baselineScores, recoveries[i].Score.RecoveryScore)
		}

		if len(baselineScores) > 0 {
			recentAvg := h.calculateMean(recentScores)
			baselineAvg := h.calculateMean(baselineScores)

			if recentAvg < baselineAvg-30 { // 30 point drop
				redFlags = append(redFlags, RedFlag{
					Type:           "dramatic_recovery_decline",
					Description:    fmt.Sprintf("Recovery scores dropped dramatically from %.1f to %.1f", baselineAvg, recentAvg),
					Severity:       "high",
					DetectedAt:     time.Now(),
					Recommendation: "Investigate sudden life changes, illness, or acute stressors",
				})
			}
		}
	}

	return redFlags
}

// Helper functions for statistical calculations
func (h *HealthAnalyzer) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (h *HealthAnalyzer) calculateMeanInt(values []int) int {
	if len(values) == 0 {
		return 0
	}

	sum := 0
	for _, v := range values {
		sum += v
	}
	return sum / len(values)
}

func (h *HealthAnalyzer) calculateStdDev(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	mean := h.calculateMean(values)
	sumSquaredDiff := 0.0

	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}

	variance := sumSquaredDiff / float64(len(values)-1)
	return math.Sqrt(variance)
}

// FormatInsightsForTherapy formats insights into a readable text for therapy sessions
func (h *HealthAnalyzer) FormatInsightsForTherapy(summary *HealthSummary) string {
	var builder strings.Builder

	builder.WriteString("# Health Summary for Therapy Session\n\n")
	builder.WriteString(fmt.Sprintf("**Analysis Period:** %s to %s\n\n",
		summary.DateRange.Start.Format("2006-01-02"),
		summary.DateRange.End.Format("2006-01-02")))

	// Recovery Section
	builder.WriteString("## Recovery Trends\n")
	builder.WriteString(fmt.Sprintf("- **Average Score:** %.1f%% (%s trend)\n",
		summary.RecoveryTrend.AverageScore, summary.RecoveryTrend.Trend))
	builder.WriteString(fmt.Sprintf("- **Consistency:** %.1f%% (higher is better)\n",
		summary.RecoveryTrend.ConsistencyScore*100))
	if summary.RecoveryTrend.WeeklyChange != 0 {
		builder.WriteString(fmt.Sprintf("- **Recent Change:** %.1f points\n",
			summary.RecoveryTrend.WeeklyChange))
	}
	builder.WriteString("\n")

	// Sleep Section
	builder.WriteString("## Sleep Analysis\n")
	builder.WriteString(fmt.Sprintf("- **Average Duration:** %.1f hours\n", summary.SleepAnalysis.AverageHours))
	builder.WriteString(fmt.Sprintf("- **Sleep Efficiency:** %.1f%%\n", summary.SleepAnalysis.AverageEfficiency*100))
	builder.WriteString(fmt.Sprintf("- **Sleep Debt:** %.1f hours\n", summary.SleepAnalysis.AverageDebt))
	builder.WriteString(fmt.Sprintf("- **Quality Trend:** %s\n", summary.SleepAnalysis.SleepQualityTrend))
	builder.WriteString("\n")

	// Stress Section
	builder.WriteString("## Stress Indicators\n")
	builder.WriteString(fmt.Sprintf("- **Stress Level:** %s\n", summary.StressIndicators.StressLevel))
	if summary.StressIndicators.PoorRecoveryStreak > 0 {
		builder.WriteString(fmt.Sprintf("- **Poor Recovery Streak:** %d days\n", summary.StressIndicators.PoorRecoveryStreak))
	}
	builder.WriteString("\n")

	// Activity Section
	builder.WriteString("## Activity Patterns\n")
	builder.WriteString(fmt.Sprintf("- **Weekly Workouts:** %d\n", summary.ActivityPatterns.WeeklyWorkouts))
	builder.WriteString(fmt.Sprintf("- **Average Strain:** %.1f\n", summary.ActivityPatterns.AverageStrain))
	builder.WriteString(fmt.Sprintf("- **Overtraining Risk:** %s\n", summary.ActivityPatterns.OvertrainingRisk))
	builder.WriteString("\n")

	// Red Flags Section
	if len(summary.RedFlags) > 0 {
		builder.WriteString("## ⚠️ Red Flags Requiring Attention\n")
		for _, flag := range summary.RedFlags {
			builder.WriteString(fmt.Sprintf("- **%s** (%s): %s\n",
				strings.Title(strings.ReplaceAll(flag.Type, "_", " ")),
				flag.Severity, flag.Description))
			builder.WriteString(fmt.Sprintf("  *Recommendation:* %s\n", flag.Recommendation))
		}
		builder.WriteString("\n")
	}

	// Therapy Insights Section
	if len(summary.TherapyInsights) > 0 {
		builder.WriteString("## 💡 Therapy Discussion Points\n")
		for _, insight := range summary.TherapyInsights {
			severity := ""
			switch insight.Severity {
			case "alert":
				severity = "⚠️ "
			case "concern":
				severity = "⚡ "
			case "info":
				severity = "ℹ️ "
			}

			builder.WriteString(fmt.Sprintf("- %s**%s**: %s\n",
				severity, strings.Title(insight.Category), insight.Insight))
			if insight.Suggestion != "" {
				builder.WriteString(fmt.Sprintf("  *Suggestion:* %s\n", insight.Suggestion))
			}
		}
	}

	return builder.String()
}
