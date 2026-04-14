package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"time"
	"usa-graph/server/internal/domain"

	"gorm.io/gorm"
)

func FetchOrGenerateAiReport(db *gorm.DB) (*domain.AiReport, error) {
	var currentScore domain.ScoreResult
	if err := db.Order("calculated_at desc").First(&currentScore).Error; err != nil {
		return nil, errors.New("no score data available yet")
	}

	var lastReport domain.AiReport
	err := db.Order("created_at desc").First(&lastReport).Error

	needsNewReport := false
	if err != nil {
		// 보고서가 아예 없으면 무조건 생성
		needsNewReport = true
	} else {
		// 4단계 캐싱 무효화 판독
		needsNewReport = checkCacheInvalidation(lastReport, currentScore)
	}

	if !needsNewReport {
		// 캐시 반환 전에 점수는 최신 대시보드 점수로 일치시키기 위해 업데이트 
		// (UI 통일성을 위해 내용물의 점수가 변동된 것은 아니지만 API 응답 메타 구조상 최신화)
		lastReport.TotalScore = currentScore.TotalScore
		lastReport.Regime = currentScore.Regime
		return &lastReport, nil
	}

	// Perplexity API 호출 필요
	newContent, err := callPerplexityAPI(currentScore)
	if err != nil {
		return nil, err
	}

	// 새 리포트 DB 저장
	newReport := domain.AiReport{
		TotalScore:  currentScore.TotalScore,
		Regime:      currentScore.Regime,
		Content:     newContent,
		MetricsJSON: currentScore.MetricsJSON,
		CreatedAt:   time.Now(),
	}

	if err := db.Create(&newReport).Error; err != nil {
		return nil, err
	}

	return &newReport, nil
}

func checkCacheInvalidation(lastReport domain.AiReport, currentScore domain.ScoreResult) bool {
	// Rule 1: Time Expiration (24h)
	if time.Since(lastReport.CreatedAt).Hours() > 24 {
		return true
	}

	// Rule 2: Regime Change
	if lastReport.Regime != currentScore.Regime {
		return true
	}

	// Rule 3: Significant Total Score Change (±5)
	if math.Abs(lastReport.TotalScore-currentScore.TotalScore) >= 5 {
		return true
	}

	// Rule 4: Key Indicator Variance
	var oldMetrics, newMetrics map[string]interface{}
	json.Unmarshal([]byte(lastReport.MetricsJSON), &oldMetrics)
	json.Unmarshal([]byte(currentScore.MetricsJSON), &newMetrics)

	// VIX Check (10% variance)
	if oldVixI, ok1 := oldMetrics["VIXCLS"]; ok1 {
		if newVixI, ok2 := newMetrics["VIXCLS"]; ok2 {
			oldVixMap := oldVixI.(map[string]interface{})
			newVixMap := newVixI.(map[string]interface{})
			
			oldVix := oldVixMap["value"].(float64)
			newVix := newVixMap["value"].(float64)

			// 10% 이상 변동 시
			if oldVix > 0 && math.Abs((newVix-oldVix)/oldVix) >= 0.10 {
				return true
			}
			// 핵심 임계치(20)를 크로스 한 경우
			if (oldVix < 20 && newVix >= 20) || (oldVix >= 20 && newVix < 20) {
				return true
			}
		}
	}

	// Fear & Greed Check (±5)
	if oldFearI, ok1 := oldMetrics["FEAR_GREED"]; ok1 {
		if newFearI, ok2 := newMetrics["FEAR_GREED"]; ok2 {
			oldFearMap := oldFearI.(map[string]interface{})
			newFearMap := newFearI.(map[string]interface{})
			
			oldFear := oldFearMap["value"].(float64)
			newFear := newFearMap["value"].(float64)

			if math.Abs(newFear-oldFear) >= 5 {
				return true
			}
		}
	}

	return false
}

func callPerplexityAPI(currentScore domain.ScoreResult) (string, error) {
	apiKey := os.Getenv("PERPLEXITY_API_KEY")
	if apiKey == "" {
		return "", errors.New("PERPLEXITY_API_KEY is not set")
	}

	url := "https://api.perplexity.ai/chat/completions"

	promptText := fmt.Sprintf(`너는 미국 경제 전문가야. 현재 주어진 지표 데이터를 바탕으로 미국 주식 유동성 대시보드 분석 보고서를 작성해줘.

현재 총점: %.2f점
현재 레짐: %s
지표 현황 (JSON): %s

반드시 포함할 내용 (마크다운 형식 준수):
1. 현재 레짐 판정
2. 핵심 지표별 분석 (RRP 잔고, 금리차, VIX, Fear&Greed 중심)
3. 긍정적 기여 지표와 부정적 기여 지표
4. 향후 1~4주 전망 및 미국 주식 투자 전략 가이드
5. 마지막 문장은 "현재 시장을 한 문장으로 요약: [문장]" 형식으로 끝내기

어투는 논리적이고 객관적인 한국어로 작성할 것. JSON 수치 데이터 자체를 나열하기보단 그 '의미'를 해석하는 데 집중할 것.`, currentScore.TotalScore, currentScore.Regime, currentScore.MetricsJSON)

	payload := map[string]interface{}{
		"model": "llama-3.1-sonar-small-128k-online", // Perplexity 최신 스탠다드 모델
		"messages": []map[string]string{
			{"role": "system", "content": "You are a professional financial analyst producing precise economic reports in Korean."},
			{"role": "user", "content": promptText},
		},
		"temperature": 0.2,
	}

	jsonPayload, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("perplexity API failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	choices := result["choices"].([]interface{})
	if len(choices) > 0 {
		firstChoice := choices[0].(map[string]interface{})
		message := firstChoice["message"].(map[string]interface{})
		return message["content"].(string), nil
	}

	return "보고서 생성 내용이 없습니다.", nil
}
