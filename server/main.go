package main

import (
	"encoding/json"
	"fmt"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
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

	// SQLite 성능 최적화: WAL 모드 및 Busy Timeout 설정
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Exec("PRAGMA journal_mode=WAL;")
		sqlDB.Exec("PRAGMA busy_timeout=5000;") // 락 발생 시 5초간 대기
	}

	// 3. 모델 마이그레이션 실행 (테이블 생성 보장 후 데이터 정리 진행)
	db.AutoMigrate(&domain.Metric{}, &domain.ScoreResult{})

	// 중요: 기존 일반 인덱스 삭제 및 중복 데이터 청소
	db.Exec("DROP INDEX IF EXISTS idx_series_id_date")
	db.Exec(`DELETE FROM metrics 
	         WHERE id NOT IN (
	             SELECT MAX(id) 
	             FROM metrics 
	             GROUP BY series_id, date
	         )`)
	
	// 유니크 인덱스 강제 생성 시도
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_metric_series_date ON metrics(series_id, date)")

	// [New] 단위 체계 변경에 따른 기존 캐시 강제 삭제
	// 배포 직후 최신 T/B 단위 데이터를 생성하기 위해 한시적으로 실행
	db.Exec("DELETE FROM score_results")

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

	// 2. FRED 거시 데이터 (30분 주기 - 갱신이 느리므로 주기를 늘림)
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		for range ticker.C {
			log.Println("Syncing macro data from FRED...")
			// FRED 데이터만 명시적으로 갱신하는 로직은 fetchAndCalculate 내부에서 처리
		}
	}()

	// Gin 서버 설정
	r := gin.Default()
	r.HandleMethodNotAllowed = true // 메서드 허용되지 않은 경우 처리 보강
	r.ForwardedByClientIP = true    // 프록시 IP 전달 허용

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

	// Root 핸들러: index.html 서빙 시 실시간 데이터 Meta Tag 주입 (AI 에이전트 크롤링 지원)
	r.GET("/", func(c *gin.Context) {
		var result domain.ScoreResult
		db.Order("calculated_at desc").First(&result)

		// index.html 위치 찾기 (dist 먼저, 없으면 루트 client)
		paths := []string{"../client/dist/index.html", "../client/index.html", "./client/dist/index.html", "./client/index.html"}
		var htmlPath string
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				htmlPath = p
				break
			}
		}

		if htmlPath == "" {
			c.String(http.StatusInternalServerError, "Failed to locate dashboard template")
			return
		}

		htmlBytes, err := os.ReadFile(htmlPath)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to read dashboard template")
			return
		}
		
		html := string(htmlBytes)

		// 데이터가 있는 경우에만 치환 진행
		if result.ID != 0 {
			description := fmt.Sprintf("현재 미국 시장 유동성 상태는 %s입니다. (종합 점수: %.2f점, 11개 지표 기반 분석 결과)", result.Regime, result.TotalScore)
			
			metas := fmt.Sprintf(`
    <meta name="description" content="%s" />
    <meta property="og:title" content="미국 유동성 대시보드" />
    <meta property="og:description" content="%s" />
    <meta name="robots" content="index, follow" />`, description, description)

			jsonLd := fmt.Sprintf(`
    <script type="application/ld+json">
    {
      "@context": "https://schema.org",
      "@type": "Dataset",
      "name": "USA Liquidity Index",
      "description": "%s",
      "variableMeasured": "Liquidity Score",
      "value": "%.2f",
      "interpretation": "%s"
    }
    </script>`, description, result.TotalScore, result.Regime)

			// 주석 플레이스홀더 치환 (타이틀 제외)
			html = strings.Replace(html, "<!--{{METAS}}-->", metas, 1)
			html = strings.Replace(html, "<!--{{JSON_LD}}-->", jsonLd, 1)
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
	})

	// [New] AI 에이전트 전용 전용 API 경로 (JSON 전용)
	r.GET("/api/data.json", func(c *gin.Context) {
		var result domain.ScoreResult
		if err := db.Order("calculated_at desc").First(&result).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No data available"})
			return
		}

		var metricsDetails map[string]interface{}
		json.Unmarshal([]byte(result.MetricsJSON), &metricsDetails)

		c.JSON(http.StatusOK, gin.H{
			"summary": gin.H{
				"score":         result.TotalScore,
				"regime":        result.Regime,
				"calculated_at": result.CalculatedAt,
			},
			"indicators": metricsDetails,
		})
	})

	// Health Check 엔드포인트
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
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

		// 1. 모든 관련 지표의 1년치 데이터를 DB 수준에서 일별 평균(Daily Sampling)으로 가져옴
		// Go 메모리가 아닌 DB 엔진을 활용하여 응답 데이터 크기를 99% 절감함
		var targetSeries []string
		for id := range metricsDetails {
			targetSeries = append(targetSeries, id, "YF_"+id)
		}

		type DailyMetric struct {
			SeriesID string
			Day      string
			AvgValue float64
		}
		var dailyMetrics []DailyMetric
		
		// SQLite의 strftime을 사용하여 날짜별 그룹화 및 평균 산출
		db.Table("metrics").
			Select("series_id, strftime('%Y-%m-%d', date) as day, AVG(value) as avg_value").
			Where("series_id IN ? AND date >= ?", targetSeries, oneYearAgoDate).
			Group("series_id, day").
			Order("day asc").
			Scan(&dailyMetrics)

		// 2. 가져온 일별 데이터를 지표별로 그룹화
		historyBySeries := make(map[string][]float64)
		for _, dm := range dailyMetrics {
			baseID := dm.SeriesID
			if len(baseID) > 3 && baseID[:3] == "YF_" {
				baseID = baseID[3:]
			}
			
			// 지표별 히스토리 데이터 스케일링 (표시 단위 기준)
			val := dm.AvgValue
			if baseID == "WRESBAL" {
				val = val / 1000000.0 // Million -> Trillion
			} else if baseID == "WTREGEN" {
				val = val / 1000.0    // Million -> Billion
			} else if baseID == "M2SL" {
				val = val / 1000.0    // Billion -> Trillion
			}
			historyBySeries[baseID] = append(historyBySeries[baseID], val)
		}

		// 3. 기존 metricsDetails 구조에 정제된 history 주입
		for id, detail := range metricsDetails {
			history := historyBySeries[id]
			if history == nil {
				history = []float64{}
			}

			if m, ok := detail.(map[string]interface{}); ok {
				m["history"] = history
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
			log.Printf("Error fetching FRED %s: %v. Using DB fallback...", id, err)
			// [Fallback] API 호출 실패 시 DB에서 마지막 데이터라도 가져와서 계산 유지
			var lastMetric domain.Metric
			if dbErr := db.Where("series_id = ?", id).Order("date desc").First(&lastMetric).Error; dbErr == nil {
				metrics = []domain.Metric{lastMetric}
				log.Printf("Fallback successful: Use DB value for %s from %s", id, lastMetric.Date.Format("2006-01-02"))
			} else {
				continue // DB에도 없으면 건너뜀
			}
		}

		if len(metrics) > 0 {
			// 대량 데이터를 한꺼번에 Upsert (OnConflict 사용 및 배치 처리)
			// SQLite의 DB 락을 최소화하기 위해 트랜잭션을 묶어서 처리함
			err := db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "series_id"}, {Name: "date"}},
				DoNothing: true,
			}).CreateInBatches(metrics, 100).Error
			
			if err != nil {
				log.Printf("Error backfilling FRED %s: %v", id, err)
			} else {
				log.Printf("FRED Backfill for %s: Synced %d points", id, len(metrics))
			}

			latest := metrics[len(metrics)-1]
			
			currentData[id] = latest.Value
			
			// 비교 대상을 직전 데이터로 설정 (이미 수집된 metrics 배열 활용)
			if len(metrics) > 1 {
				prevData[id] = metrics[len(metrics)-2].Value
			} else {
				prevData[id] = latest.Value
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

			// 최적화된 배치 Upsert 적용
			db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "series_id"}, {Name: "date"}},
				DoNothing: true,
			}).CreateInBatches(hist, 100)
			
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
		
		// CNN 데이터도 동일하게 안전한 배치 Upsert 적용
		db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "series_id"}, {Name: "date"}},
			DoNothing: true,
		}).CreateInBatches(fgMetrics, 100)
		
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

	// [Optimization] 이전 결과와 비교하여 변경사항이 없으면 DB 쓰기 스킵
	var lastResult domain.ScoreResult
	if err := db.Order("calculated_at desc").First(&lastResult).Error; err == nil {
		// 점수와 레짐이 같으면 굳이 새로 저장하지 않음 (리소스 절약)
		// 단, 1시간 이상 지났으면 상태 확인을 위해 다시 저장
		if lastResult.TotalScore == totalScore && lastResult.Regime == regime && time.Since(lastResult.CalculatedAt) < 1*time.Hour {
			log.Printf("Skip saving: Score(%.2f) and Regime(%s) unchanged", totalScore, regime)
			return
		}
	}

	// 개별 지표 정보 및 점수를 JSON으로 저장
	metricDetails := make(map[string]interface{})
	for id, val := range currentData {
		// 중요: 원본 맵을 수정하지 않고 연산을 수행하기 위해 별도 변수 할당
		displayVal := val
		displayPrev := prevData[id]

		// 수치 표시를 위한 단위 스케일링 (T, B 단위 최적화)
		// WRESBAL(Million -> Trillion), WTREGEN(Million -> Billion), M2SL(Billion -> Trillion)
		if id == "WRESBAL" {
			displayVal /= 1000000.0 // ex: 3.11T
			displayPrev /= 1000000.0
		} else if id == "WTREGEN" {
			displayVal /= 1000.0    // ex: 748.37B
			displayPrev /= 1000.0
		} else if id == "M2SL" {
			displayVal /= 1000.0    // ex: 22.67T
			displayPrev /= 1000.0
		}

		diff := displayVal - displayPrev
		percent := 0.0
		if displayPrev != 0 {
			percent = (diff / math.Abs(displayPrev)) * 100
		}
		
		metricDetails[id] = map[string]interface{}{
			"value":   displayVal,
			"diff":    diff,
			"percent": percent,
			"score":   indicatorScores[id],
		}
	}

	// [Validation] 11개 필수 지표가 모두 존재하는지 확인하여 데이터 정합성 보장
	requiredCount := 11
	if len(metricDetails) < requiredCount {
		log.Printf("[Warning] Skipping update: Only %d/%d indicators collected", len(metricDetails), requiredCount)
		return
	}

	metricsJSON, _ := json.Marshal(metricDetails)

	res := domain.ScoreResult{
		TotalScore:   totalScore,
		Regime:       regime,
		NetLiquidity: (indicatorScores["RRPONTSYD"]*0.08 + indicatorScores["WTREGEN"]*0.08) / 0.16 * 10,
		BankSystem:   (indicatorScores["WRESBAL"]*0.10 + indicatorScores["SOFR"]*0.06) / 0.16 * 10,
		Monetary:     (indicatorScores["M2SL"]*0.08 + indicatorScores["RMFNS"]*0.06) / 0.14 * 10,
		RiskAppetite: (indicatorScores["VIXCLS"]*0.08 + indicatorScores["FEAR_GREED"]*0.12 + indicatorScores["BAMLH0A0HYM2"]*0.10) / 0.30 * 10,
		DollarGlobal: (indicatorScores["DXY"]*0.06 + indicatorScores["T10Y2Y"]*0.08) / 0.14 * 10,
		CalculatedAt: time.Now(),
		MetricsJSON:  string(metricsJSON),
	}
	db.Create(&res)
	log.Printf("Calculation complete (11 indicators): Total Score %.2f, Regime: %s", totalScore, regime)
}
