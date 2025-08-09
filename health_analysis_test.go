package main

import (
	"testing"
	"time"
)

func TestHealthAnalyzer_CalculateMean(t *testing.T) {
	analyzer := NewHealthAnalyzer()

	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{
			name:     "empty slice",
			values:   []float64{},
			expected: 0,
		},
		{
			name:     "single value",
			values:   []float64{5.0},
			expected: 5.0,
		},
		{
			name:     "multiple values",
			values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
			expected: 3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.calculateMean(tt.values)
			if result != tt.expected {
				t.Errorf("calculateMean() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHealthAnalyzer_AnalyzeRecoveryTrend(t *testing.T) {
	analyzer := NewHealthAnalyzer()

	// Test with empty data
	t.Run("no data", func(t *testing.T) {
		trend := analyzer.analyzeRecoveryTrend([]WhoopRecovery{})
		if trend.Trend != "no_data" {
			t.Errorf("Expected trend 'no_data', got %s", trend.Trend)
		}
	})

	// Test with sample data
	t.Run("sample data", func(t *testing.T) {
		recoveries := []WhoopRecovery{
			{
				CreatedAt: time.Now().AddDate(0, 0, -7),
				Score: struct {
					UserCalibrating  bool    `json:"user_calibrating"`
					RecoveryScore    float64 `json:"recovery_score"`
					RestingHeartRate float64 `json:"resting_heart_rate"`
					HRVRmssd         float64 `json:"hrv_rmssd_milli"`
					SkinTempCelsius  float64 `json:"skin_temp_celsius"`
					SpO2Percentage   float64 `json:"spo2_percentage"`
				}{
					RecoveryScore: 75.0,
				},
			},
			{
				CreatedAt: time.Now().AddDate(0, 0, -6),
				Score: struct {
					UserCalibrating  bool    `json:"user_calibrating"`
					RecoveryScore    float64 `json:"recovery_score"`
					RestingHeartRate float64 `json:"resting_heart_rate"`
					HRVRmssd         float64 `json:"hrv_rmssd_milli"`
					SkinTempCelsius  float64 `json:"skin_temp_celsius"`
					SpO2Percentage   float64 `json:"spo2_percentage"`
				}{
					RecoveryScore: 80.0,
				},
			},
		}

		trend := analyzer.analyzeRecoveryTrend(recoveries)

		if trend.AverageScore != 77.5 {
			t.Errorf("Expected average score 77.5, got %f", trend.AverageScore)
		}

		if len(trend.LastSevenDays) != 2 {
			t.Errorf("Expected 2 scores in LastSevenDays, got %d", len(trend.LastSevenDays))
		}
	})
}

func TestHealthAnalyzer_AnalyzeSleepPatterns(t *testing.T) {
	analyzer := NewHealthAnalyzer()

	// Test with empty data
	t.Run("no data", func(t *testing.T) {
		analysis := analyzer.analyzeSleepPatterns([]WhoopSleep{})
		if analysis.SleepQualityTrend != "no_data" {
			t.Errorf("Expected trend 'no_data', got %s", analysis.SleepQualityTrend)
		}
	})
}

func TestNewHealthAnalyzer(t *testing.T) {
	analyzer := NewHealthAnalyzer()
	if analyzer == nil {
		t.Error("NewHealthAnalyzer() returned nil")
	}

	if analyzer.cache == nil {
		t.Error("HealthAnalyzer cache not initialized")
	}
}

func TestGenerateTherapyInsights(t *testing.T) {
	analyzer := NewHealthAnalyzer()

	// Create test data with concerning patterns
	recovery := RecoveryTrend{
		Trend:            "declining",
		AverageScore:     45.0,
		WeeklyChange:     -15.0,
		ConsistencyScore: 0.4,
	}

	sleep := SleepAnalysis{
		AverageHours:      5.5,  // Below recommended
		AverageEfficiency: 0.75, // Poor efficiency
		SleepQualityTrend: "declining",
	}

	stress := StressIndicators{
		StressLevel:        "high",
		PoorRecoveryStreak: 5,
	}

	activity := ActivityPatterns{
		WeeklyWorkouts:   0, // No activity
		OvertrainingRisk: "low",
	}

	insights := analyzer.generateTherapyInsights(recovery, sleep, stress, activity)

	// Should generate multiple insights for concerning patterns
	if len(insights) == 0 {
		t.Error("Expected therapy insights to be generated for concerning health patterns")
	}

	// Check that insights have required fields
	for _, insight := range insights {
		if insight.Category == "" {
			t.Error("Insight missing category")
		}
		if insight.Insight == "" {
			t.Error("Insight missing insight text")
		}
		if insight.Severity == "" {
			t.Error("Insight missing severity")
		}
	}
}
