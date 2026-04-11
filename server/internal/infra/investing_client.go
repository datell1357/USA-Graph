package infra

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"usa-graph/server/internal/domain"
)

type InvestingClient struct {
	BaseUrl string
}

func NewInvestingClient() *InvestingClient {
	return &InvestingClient{
		BaseUrl: "https://ssltsw.investing.com/api.php",
	}
}

// FetchRealtimeData Investing.com 위젯 API를 통해 실시간 데이터를 가져옴
func (c *InvestingClient) FetchRealtimeData(pairs string) ([]domain.Metric, error) {
	// action=refresher 엔드포인트 사용 (더 안정적이며 JSON 구조가 다름)
	url := fmt.Sprintf("%s?action=refresher&pairs=%s&lang=1", c.BaseUrl, pairs)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.investing.com/")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("investing api status error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// action=refresher의 응답은 {"data": {"ID": {"row": {"last": "VALUE"}}}} 형태임
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal investing data: %w", err)
	}

	var metrics []domain.Metric
	now := time.Now()

	for id, val := range result.Data {
		// val은 {"row": {"last": "98.44"}} 형태
		idMap, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		row, ok := idMap["row"].(map[string]interface{})
		if !ok {
			continue
		}
		lastValRaw, exists := row["last"]
		if !exists {
			continue
		}

		// 문자열에서 컴마 제거 후 플로트로 변환
		lastValStr := strings.ReplaceAll(fmt.Sprintf("%v", lastValRaw), ",", "")
		log.Printf("Parsing Investing ID %s: raw=%v, sanitized=%s", id, lastValRaw, lastValStr)
		var lastVal float64
		_, err := fmt.Sscanf(lastValStr, "%f", &lastVal)
		if err != nil {
			continue
		}

		metrics = append(metrics, domain.Metric{
			SeriesID:  "INV_" + id,
			Value:     lastVal,
			Date:      now,
			CreatedAt: now,
		})
	}

	return metrics, nil
}
