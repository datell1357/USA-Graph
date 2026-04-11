package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"usa-graph/server/internal/app"
	"usa-graph/server/internal/domain"
	"usa-graph/server/internal/infra"
)

func main() {
	// .env 로드 (상위 디렉토리에 위치)
	if err := godotenv.Load("../.env"); err != nil {
		log.Println(".env file not found, using system environment variables")
	}

	// DB 초기화 (Render 배포 시 Disk 볼륨 경로 "/var/lib/usa-graph/usa_graph.db" 등을 위해 유연화)
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "usa_graph.db"
	}
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	// 테이블 자동 생성
	db.AutoMigrate(&domain.Metric{}, &domain.ScoreResult{})

	// 클라이언트 및 서비스 초기화
	fredKey := os.Getenv("FRED_API_KEY")
	fredClient := infra.NewFredClient(fredKey)
	yahooClient := infra.NewYahooFinanceClient()
	cnnClient := infra.NewCnnClient()
	scoringService := app.NewScoringService()

	// 서버 시작 시 즉시 1회 수집
	log.Println("Initial data collection...")
	fetchAndCalculate(db, fredClient, yahooClient, cnnClient, scoringService)

	// 백그라운드 데이터 수집기
	// 1. Yahoo Finance 실시간 데이터 (1분 주기)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			log.Println("Syncing real-time data from Yahoo & CNN...")
			fetchAndCalculate(db, fredClient, yahooClient, cnnClient, scoringService)
		}
	}()

	// 2. FRED 거시 데이터 (15분 주기 - 갱신이 느리므로 주기를 늘림)
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		for range ticker.C {
			log.Println("Syncing macro data from FRED...")
			// FRED 데이터만 명시적으로 갱신하는 로직은 fetchAndCalculate 내부에서 처리
		}
	}()

	// Gin 서버 설정
	r := gin.Default()

	// CORS 설정
	r.Use(func(c *gin.Context) {
		// 배포 시 보안을 위해 특정 도메인(예: https://usa-liquidity-dashboard.netlify.app)만 허용하는 것이 좋음
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*") 
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/api/status", func(c *gin.Context) {
		var result domain.ScoreResult
		db.Order("calculated_at desc").First(&result)
		
		position := "보수적 관망"
		if result.Regime == "긴축" {
			position = "방어적 포지션"
		} else if result.Regime == "완화" {
			position = "공격적 포지션"
		}

		var metricsDetails map[string]interface{}
		if err := json.Unmarshal([]byte(result.MetricsJSON), &metricsDetails); err != nil {
			log.Printf("Error unmarshaling metrics_json: %v", err)
			metricsDetails = make(map[string]interface{})
		}

		oneYearAgoDate := time.Now().AddDate(-1, 0, 0)

		for id, detail := range metricsDetails {
			var rawRecords []domain.Metric
			
			// 1. 해당 지표의 1년치 모든 데이터를 DB에서 긁어옴 (메모리 필터링을 위해)
			// SQLite의 날짜 함수 리스크를 방지하기 위해 단순 쿼리 후 Go에서 정제함
			db.Where("(series_id = ? OR series_id = ?) AND date >= ?", id, "YF_"+id, oneYearAgoDate).
				Order("date asc").
				Find(&rawRecords)
			
			// 2. 하루에 하나씩만 남기도록 메모리에서 정제 (Daily Sampling)
			var history []float64
			seenDates := make(map[string]bool)
			for _, rec := range rawRecords {
				dateStr := rec.Date.Format("2006-01-02")
				if !seenDates[dateStr] {
					// 지표별 스케일링 적용
					val := rec.Value
					if id == "WRESBAL" {
						val = val / 1000.0
					}
					history = append(history, val)
					seenDates[dateStr] = true
				} else {
					// 같은 날짜면 마지막 데이터로 계속 업데이트 (Daily Close 효과)
					val := rec.Value
					if id == "WRESBAL" {
						val = val / 1000.0
					}
					if len(history) > 0 {
						history[len(history)-1] = val
					}
				}
			}

			if history == nil {
				history = []float64{}
			}

			if m, ok := detail.(map[string]interface{}); ok {
				m["history"] = history
			} else {
				metricsDetails[id] = map[string]interface{}{
					"history": history,
				}
			}
		}

		updatedMetricsJSON, _ := json.Marshal(metricsDetails)

		c.JSON(http.StatusOK, gin.H{
			"total_score":   result.TotalScore,
			"regime":        result.Regime,
			"position":      position,
			"calculated_at": result.CalculatedAt,
			"metrics_json":  string(updatedMetricsJSON),
		})
	})

	r.GET("/api/metrics", func(c *gin.Context) {
		var metrics []domain.Metric
		db.Order("date desc").Limit(100).Find(&metrics)
		c.JSON(http.StatusOK, metrics)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}

func fetchAndCalculate(db *gorm.DB, fred *infra.FredClient, yf *infra.YahooFinanceClient, cnn *infra.CnnClient, svc *app.ScoringService) {
	// 핵심 11개 지표 정의 (Score Standard.md 기준)
	seriesIDs := []string{
		"RRPONTSYD",     // RRP 잔고 (8%)
		"WRESBAL",       // 은행 준비금 (10%)
		"WTREGEN",       // TGA 잔고 (8%)
		"M2SL",          // M2 통화량 (8%)
		"RMFNS",         // MMF 자산 (6%)
		"SOFR",          // SOFR 금리 (6%)
		"T10Y2Y",        // 장단기 금리차 (8%)
		"DTWEXBGS",      // DXY 대신 실효달러인덱스 (6%)
		"BAMLH0A0HYM2",  // 하이일드 스프레드 (10%)
		"VIXCLS",        // VIX 지수 (8%)
	}
	
	currentData := make(map[string]float64)
	prevData := make(map[string]float64)

	// 그래프와 분석을 위해 1년치 데이터를 수집함 (사용자 요청 반영)
	oneYearAgo := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")

	for _, id := range seriesIDs {
		// FRED 시리즈 수집 (1년치)
		metrics, err := fred.FetchObservations(id, oneYearAgo)
		if err != nil {
			log.Printf("Error fetching FRED %s: %v", id, err)
			continue
		}

		if len(metrics) > 0 {
			// 1. 모든 과거 데이터를 DB에 전수로 안전하게 개별 Upsert
			processedCount := 0
			for _, m := range metrics {
				// 중복 체크 (SeriesID + Date)
				var count int64
				db.Model(&domain.Metric{}).Where("series_id = ? AND date = ?", m.SeriesID, m.Date).Count(&count)
				if count == 0 {
					db.Create(&m)
					processedCount++
				}
			}
			if processedCount > 0 {
				log.Printf("FRED Backfill for %s: %d new points saved", id, processedCount)
			}

			latest := metrics[len(metrics)-1]
			
			// 지표별 스케일링 (계산용)
			val := latest.Value
			if id == "WRESBAL" {
				val = val / 1000.0
			}
			
			currentData[id] = val
			
			// 비교 대상을 직전 데이터로 설정 (이미 수집된 metrics 배열 활용)
			if len(metrics) > 1 {
				prevVal := metrics[len(metrics)-2].Value
				if id == "WRESBAL" {
					prevVal = prevVal / 1000.0
				}
				prevData[id] = prevVal
			} else {
				prevData[id] = val
			}
		}
	}

	// Yahoo Finance 실시간 + 1년치 히스토리 수집
	yfSymbols := map[string]string{
		"DXY":    "DX-Y.NYB",
		"VIXCLS": "^VIX",
	}

	for appID, sym := range yfSymbols {
		// 히스토리 수집 (1년치 일봉)
		hist, err := yf.FetchHistoryPrice(sym, "1y")
		if err == nil && len(hist) > 0 {
			// SeriesID를 앱 식별자로 변환하여 저장
			for i := range hist {
				hist[i].SeriesID = "YF_" + appID
			}
			db.Clauses(clause.OnConflict{DoNothing: true}).Create(&hist)
			
			// 최신 가격 및 이전 가격 추출
			latest := hist[len(hist)-1]
			currentData[appID] = latest.Value
			if len(hist) > 1 {
				prevData[appID] = hist[len(hist)-2].Value
			}
			
			// DXY의 경우 DTWEXBGS(실효달러인덱스) 대용으로도 사용
			if appID == "DXY" {
				currentData["DTWEXBGS"] = latest.Value
				if len(hist) > 1 {
					prevData["DTWEXBGS"] = hist[len(hist)-2].Value
				}
			}
			log.Printf("Yahoo History Backfilled for %s: %d points", appID, len(hist))
		}
	}

	// Fear & Greed Index (12%) - CNN 내부 API 수집 및 히스토리 백필
	// 히스토리는 그래프용으로, 실시간 API는 증감분(PreviousClose) 확인용으로 모두 사용
	fgData, err := cnn.FetchFearGreed()
	if err == nil {
		currentData["FEAR_GREED"] = fgData.Score
		prevData["FEAR_GREED"] = fgData.PreviousClose
		log.Printf("CNN Realtime: Current = %.2f, Prev = %.2f", fgData.Score, fgData.PreviousClose)
	}

	cnnHist, err := cnn.FetchFearGreedHistorical()
	if err == nil && len(cnnHist) > 0 {
		var fgMetrics []domain.Metric
		for _, p := range cnnHist {
			fgMetrics = append(fgMetrics, domain.Metric{
				SeriesID:  "FEAR_GREED",
				Value:     p.Y,
				Date:      time.Unix(int64(p.X)/1000, 0),
				CreatedAt: time.Now(),
			})
		}
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(&fgMetrics)
		
		// 만약 실시간 API가 실패했다면 히스토리에서 보완
		if _, ok := currentData["FEAR_GREED"]; !ok {
			latest := fgMetrics[len(fgMetrics)-1]
			currentData["FEAR_GREED"] = latest.Value
			if len(fgMetrics) > 1 {
				prevData["FEAR_GREED"] = fgMetrics[len(fgMetrics)-2].Value
			}
		}
		log.Printf("CNN History Backfilled: %d points", len(fgMetrics))
	} else if err != nil && currentData["FEAR_GREED"] == 0 {
		// 모두 실패 시 Fallback
		currentData["FEAR_GREED"] = 45.0
		prevData["FEAR_GREED"] = 50.0
	}

	// 9개 지표별 점수 산출
	indicatorScores := svc.CalculateIndicatorScores(currentData, prevData)
	
	// 가중치 적용 (Score Standard.md 기준 100% 동기화)
	weights := map[string]float64{
		"RRPONTSYD":    0.08,
		"WRESBAL":      0.10,
		"WTREGEN":      0.08,
		"M2SL":         0.08,
		"RMFNS":        0.06,
		"SOFR":         0.06,
		"T10Y2Y":       0.08,
		"DXY":          0.06,
		"BAMLH0A0HYM2": 0.10,
		"VIXCLS":       0.08,
		"FEAR_GREED":   0.12,
	}

	totalScore := 0.0
	for id, score := range indicatorScores {
		totalScore += score * weights[id]
	}
	totalScore = totalScore * 10.0 // 10점 -> 100점 환산

	regime, _ := svc.CalculateRegime(totalScore, indicatorScores, currentData)

	// 개별 지표 정보 및 점수를 JSON으로 저장
	metricDetails := make(map[string]interface{})
	for id, val := range currentData {
		diff := val - prevData[id]
		percent := 0.0
		if prevData[id] != 0 {
			percent = (diff / math.Abs(prevData[id])) * 100
		}
		metricDetails[id] = map[string]interface{}{
			"value":   val,
			"diff":    diff,
			"percent": percent,
			"score":   indicatorScores[id], // 개별 10점 만점 점수 포함
		}
	}
	metricsJSON, _ := json.Marshal(metricDetails)

	res := domain.ScoreResult{
		TotalScore:   totalScore,
		Regime:       regime,
		// 핵심군 점수는 이제 가중 점수의 합으로 대체하거나 유연하게 처리
		NetLiquidity: (indicatorScores["RRPONTSYD"]*0.08 + indicatorScores["WTREGEN"]*0.08) / 0.16 * 10,
		BankSystem:   (indicatorScores["WRESBAL"]*0.10 + indicatorScores["SOFR"]*0.06) / 0.16 * 10,
		Monetary:     (indicatorScores["M2SL"]*0.08 + indicatorScores["RMFNS"]*0.06) / 0.14 * 10,
		RiskAppetite: (indicatorScores["VIXCLS"]*0.08 + indicatorScores["FEAR_GREED"]*0.12 + indicatorScores["BAMLH0A0HYM2"]*0.10) / 0.30 * 10,
		DollarGlobal: (indicatorScores["DXY"]*0.06 + indicatorScores["T10Y2Y"]*0.08) / 0.14 * 10,
		CalculatedAt: time.Now(),
		MetricsJSON:  string(metricsJSON),
	}
	db.Create(&res)
	log.Printf("Calculation complete (9 indicators): Total Score %.2f, Regime: %s", totalScore, regime)
}
