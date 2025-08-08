package main

import (
	"encoding/json"
	"time"
)

// MCP Protocol Types
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type MCPTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema MCPInputSchema `json:"inputSchema"`
}

type MCPInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// Whoop API Response Types
type WhoopUser struct {
	UserID    int    `json:"user_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type WhoopRecovery struct {
	CycleID    int64     `json:"cycle_id"`
	SleepID    string    `json:"sleep_id"` // UUID in V2
	UserID     int64     `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ScoreState string    `json:"score_state"`
	Score      struct {
		UserCalibrating  bool    `json:"user_calibrating"`
		RecoveryScore    float64 `json:"recovery_score"`
		RestingHeartRate int     `json:"resting_heart_rate"`
		HRVRmssd         float64 `json:"hrv_rmssd_milli"`
		SkinTempCelsius  float64 `json:"skin_temp_celsius"`
		SpO2Percentage   float64 `json:"spo2_percentage"`
	} `json:"score"`
}

type WhoopSleep struct {
	ID             string    `json:"id"`              // UUID in V2
	V1ID           *int64    `json:"v1_id,omitempty"` // Legacy ID for migration
	UserID         int64     `json:"user_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Start          time.Time `json:"start"`
	End            time.Time `json:"end"`
	TimezoneOffset string    `json:"timezone_offset"`
	Nap            bool      `json:"nap"`
	ScoreState     string    `json:"score_state"`
	Score          struct {
		StageSummary struct {
			TotalInBedTimeMilli         int `json:"total_in_bed_time_milli"`
			TotalAwakeTimeMilli         int `json:"total_awake_time_milli"`
			TotalNoDataTimeMilli        int `json:"total_no_data_time_milli"`
			TotalLightSleepTimeMilli    int `json:"total_light_sleep_time_milli"`
			TotalSlowWaveSleepTimeMilli int `json:"total_slow_wave_sleep_time_milli"`
			TotalRemSleepTimeMilli      int `json:"total_rem_sleep_time_milli"`
			SleepCycleCount             int `json:"sleep_cycle_count"`
			DisturbanceCount            int `json:"disturbance_count"`
		} `json:"stage_summary"`
		SleepNeeded struct {
			BaselineMilli             int `json:"baseline_milli"`
			NeedFromSleepDebtMilli    int `json:"need_from_sleep_debt_milli"`
			NeedFromRecentStrainMilli int `json:"need_from_recent_strain_milli"`
			NeedFromRecentNapMilli    int `json:"need_from_recent_nap_milli"`
		} `json:"sleep_needed"`
		RespiratoryRate            float64 `json:"respiratory_rate"`
		SleepPerformancePercentage int     `json:"sleep_performance_percentage"`
		SleepConsistencyPercentage int     `json:"sleep_consistency_percentage"`
		SleepEfficiencyPercentage  float64 `json:"sleep_efficiency_percentage"`
	} `json:"score"`
}

type WhoopWorkout struct {
	ID             string    `json:"id"`              // UUID in V2
	V1ID           *int64    `json:"v1_id,omitempty"` // Legacy ID for migration
	UserID         int64     `json:"user_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Start          time.Time `json:"start"`
	End            time.Time `json:"end"`
	TimezoneOffset string    `json:"timezone_offset"`
	SportName      string    `json:"sport_name"`
	SportID        *int      `json:"sport_id,omitempty"` // Legacy field
	ScoreState     string    `json:"score_state"`
	Score          struct {
		Strain              float64 `json:"strain"`
		AverageHeartRate    int     `json:"average_heart_rate"`
		MaxHeartRate        int     `json:"max_heart_rate"`
		Kilojoule           float64 `json:"kilojoule"`
		PercentRecorded     float64 `json:"percent_recorded"`
		DistanceMeter       float64 `json:"distance_meter"`
		AltitudeGainMeter   float64 `json:"altitude_gain_meter"`
		AltitudeChangeMeter float64 `json:"altitude_change_meter"`
		ZoneDurations       struct {
			ZoneZeroMilli  int `json:"zone_zero_milli"`
			ZoneOneMilli   int `json:"zone_one_milli"`
			ZoneTwoMilli   int `json:"zone_two_milli"`
			ZoneThreeMilli int `json:"zone_three_milli"`
			ZoneFourMilli  int `json:"zone_four_milli"`
			ZoneFiveMilli  int `json:"zone_five_milli"`
		} `json:"zone_durations"`
	} `json:"score"`
}

type WhoopCycle struct {
	ID             int64     `json:"id"` // Still integer ID in V2
	UserID         int64     `json:"user_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Start          time.Time `json:"start"`
	End            time.Time `json:"end"`
	TimezoneOffset string    `json:"timezone_offset"`
	ScoreState     string    `json:"score_state"`
	Score          struct {
		Strain           float64 `json:"strain"`
		Kilojoule        float64 `json:"kilojoule"`
		AverageHeartRate int     `json:"average_heart_rate"`
		MaxHeartRate     int     `json:"max_heart_rate"`
	} `json:"score"`
}

// Health Analysis Types
type HealthSummary struct {
	UserID           int              `json:"user_id"`
	DateRange        DateRange        `json:"date_range"`
	RecoveryTrend    RecoveryTrend    `json:"recovery_trend"`
	SleepAnalysis    SleepAnalysis    `json:"sleep_analysis"`
	StressIndicators StressIndicators `json:"stress_indicators"`
	ActivityPatterns ActivityPatterns `json:"activity_patterns"`
	TherapyInsights  []TherapyInsight `json:"therapy_insights"`
	RedFlags         []RedFlag        `json:"red_flags"`
}

type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type RecoveryTrend struct {
	AverageScore     float64   `json:"average_score"`
	Trend            string    `json:"trend"` // "improving", "declining", "stable"
	WeeklyChange     float64   `json:"weekly_change"`
	ConsistencyScore float64   `json:"consistency_score"`
	LastSevenDays    []float64 `json:"last_seven_days"`
}

type SleepAnalysis struct {
	AverageHours         float64 `json:"average_hours"`
	AverageEfficiency    float64 `json:"average_efficiency"`
	AverageDebt          float64 `json:"average_debt"`
	ConsistencyScore     float64 `json:"consistency_score"`
	DisturbanceFrequency float64 `json:"disturbance_frequency"`
	OptimalBedtime       string  `json:"optimal_bedtime"`
	SleepQualityTrend    string  `json:"sleep_quality_trend"`
}

type StressIndicators struct {
	ElevatedHRVDays     int     `json:"elevated_hrv_days"`
	HighRestingHRDays   int     `json:"high_resting_hr_days"`
	PoorRecoveryStreak  int     `json:"poor_recovery_streak"`
	StressLevel         string  `json:"stress_level"` // "low", "moderate", "high", "critical"
	PhysiologicalStress float64 `json:"physiological_stress"`
}

type ActivityPatterns struct {
	WeeklyWorkouts     int     `json:"weekly_workouts"`
	AverageStrain      float64 `json:"average_strain"`
	WorkoutConsistency float64 `json:"workout_consistency"`
	OvertrainingRisk   string  `json:"overtraining_risk"` // "low", "moderate", "high"
	ActiveRecoveryDays int     `json:"active_recovery_days"`
	IntensityBalance   string  `json:"intensity_balance"`
}

type TherapyInsight struct {
	Category   string `json:"category"` // "sleep", "recovery", "stress", "activity"
	Insight    string `json:"insight"`
	Severity   string `json:"severity"` // "info", "concern", "alert"
	Actionable bool   `json:"actionable"`
	Suggestion string `json:"suggestion,omitempty"`
}

type RedFlag struct {
	Type           string    `json:"type"`
	Description    string    `json:"description"`
	Severity       string    `json:"severity"` // "moderate", "high", "critical"
	DetectedAt     time.Time `json:"detected_at"`
	Recommendation string    `json:"recommendation"`
}

// Tool Input Types
type HealthSummaryInput struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	UserID    *int   `json:"user_id,omitempty"`
}

type StressAnalysisInput struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	UserID    *int   `json:"user_id,omitempty"`
}

type SleepAnalysisInput struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	UserID    *int   `json:"user_id,omitempty"`
}

type TrendAnalysisInput struct {
	Metric string `json:"metric"` // "recovery", "sleep", "strain"
	Days   int    `json:"days"`   // number of days to analyze
	UserID *int   `json:"user_id,omitempty"`
}

// API Response Wrappers
type WhoopAPIResponse struct {
	Data      interface{} `json:"data"`
	NextToken *string     `json:"next_token,omitempty"`
}

type WhoopRecoveryResponse struct {
	Data      []WhoopRecovery `json:"data"`
	NextToken *string         `json:"next_token,omitempty"`
}

type WhoopSleepResponse struct {
	Data      []WhoopSleep `json:"data"`
	NextToken *string      `json:"next_token,omitempty"`
}

type WhoopWorkoutResponse struct {
	Data      []WhoopWorkout `json:"data"`
	NextToken *string        `json:"next_token,omitempty"`
}

type WhoopCycleResponse struct {
	Data      []WhoopCycle `json:"data"`
	NextToken *string      `json:"next_token,omitempty"`
}
