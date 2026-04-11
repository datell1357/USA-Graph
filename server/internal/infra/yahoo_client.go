package infra

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"usa-graph/server/internal/domain"
)

type YahooFinanceClient struct {
	BaseUrl string
}

func NewYahooFinanceClient() *YahooFinanceClient {
	return &YahooFinanceClient{
		BaseUrl: "https://query1.finance.yahoo.com/v8/finance/chart",
	}
}

// FetchRealtimePrice 야후 파이낸스 JSON API를 통해 개별 심볼의 현재가를 가져옴
func (c *YahooFinanceClient) FetchRealtimePrice(symbol string) (float64, error) {
	url := fmt.Sprintf("%s/%s?interval=1m&range=1d", c.BaseUrl, symbol)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	// 야후는 브라우저 헤더가 필수적임
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("yahoo api status error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice float64 `json:"regularMarketPrice"`
				} `json:"meta"`
			} `json:"result"`
		} `json:"chart"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("failed to unmarshal yahoo data: %w", err)
	}

	if len(result.Chart.Result) == 0 {
		return 0, fmt.Errorf("no data found for symbol: %s", symbol)
	}

	return result.Chart.Result[0].Meta.RegularMarketPrice, nil
}

// FetchHistoryPrice 야후 파이낸스 JSON API를 통해 특정 기간의 일별 종가 히스토리를 가져옴
func (c *YahooFinanceClient) FetchHistoryPrice(symbol string, rangeStr string) ([]domain.Metric, error) {
	url := fmt.Sprintf("%s/%s?interval=1d&range=%s", c.BaseUrl, symbol, rangeStr)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Chart struct {
			Result []struct {
				Timestamp []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Close []float64 `json:"close"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
		} `json:"chart"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Chart.Result) == 0 || len(result.Chart.Result[0].Timestamp) == 0 {
		return nil, fmt.Errorf("no history data found for symbol: %s", symbol)
	}

	res := result.Chart.Result[0]
	var metrics []domain.Metric
	for i, ts := range res.Timestamp {
		val := res.Indicators.Quote[0].Close[i]
		if val == 0 { continue } // 데이터 누락 방지
		
		date := time.Unix(ts, 0)
		metrics = append(metrics, domain.Metric{
			SeriesID:  "YF_" + symbol, // 실제로는 호출 시 변경될 수 있음
			Value:     val,
			Date:      date,
			CreatedAt: time.Now(),
		})
	}

	return metrics, nil
}

// FetchMetrics 매칭된 심볼 목록을 Metrics 구조체로 반환
func (c *YahooFinanceClient) FetchMetrics(symbolMap map[string]string) ([]domain.Metric, error) {
	var metrics []domain.Metric
	now := time.Now()

	for appID, yahooSymbol := range symbolMap {
		price, err := c.FetchRealtimePrice(yahooSymbol)
		if err != nil {
			continue // 일부 지표 실패 시 건너뛰기
		}

		metrics = append(metrics, domain.Metric{
			SeriesID:  "YF_" + appID,
			Value:     price,
			Date:      now,
			CreatedAt: now,
		})
	}

	return metrics, nil
}
