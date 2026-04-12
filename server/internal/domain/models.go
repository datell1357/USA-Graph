package domain

import "time"

// Metric 지표 데이터를 저장하기 위한 도메인 모델
type Metric struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SeriesID  string    `gorm:"uniqueIndex:idx_unique_metric_series_date" json:"series_id"` // FRED Series ID
	Value     float64   `json:"value"`
	Date      time.Time `gorm:"uniqueIndex:idx_unique_metric_series_date" json:"date"`
	CreatedAt time.Time `json:"created_at"`
}

// ScoreResult 계산된 점수 및 레짐 상태 결과
type ScoreResult struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TotalScore    float64   `json:"total_score"`
	Regime        string    `json:"regime"` // 완화, 중립, 긴축
	NetLiquidity  float64   `json:"net_liquidity_score"`
	BankSystem    float64   `json:"bank_system_score"`
	Monetary      float64   `json:"monetary_score"`
	RiskAppetite  float64   `json:"risk_appetite_score"`
	DollarGlobal  float64   `json:"dollar_global_score"`
	CalculatedAt  time.Time `gorm:"index" json:"calculated_at"`
	MetricsJSON   string    `json:"metrics_json"`
}

// FactorWeights 지표군 가중치 정보 (Score Standard.md 기준)
type FactorWeights struct {
	NetLiquidityWeights map[string]float64
	BankSystemWeights   map[string]float64
	MonetaryWeights     map[string]float64
	RiskAppetiteWeights map[string]float64
	DollarGlobalWeights map[string]float64
}
