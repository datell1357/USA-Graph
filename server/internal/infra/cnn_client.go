package infra

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type CnnClient struct {
	BaseUrl string
}

func NewCnnClient() *CnnClient {
	return &CnnClient{
		BaseUrl: "https://production.dataviz.cnn.io/index/fearandgreed/graphdata",
	}
}

type FearGreedData struct {
	Score         float64 `json:"score"`
	Rating        string  `json:"rating"`
	PreviousClose float64 `json:"previous_close"`
}

type CnnResponse struct {
	FearAndGreed FearGreedData `json:"fear_and_greed"`
}

// FetchFearGreed CNN 내부 API를 통해 공포와 탐욕 지수를 가져옴
func (c *CnnClient) FetchFearGreed() (*FearGreedData, error) {
	req, err := http.NewRequest("GET", c.BaseUrl, nil)
	if err != nil {
		return nil, err
	}

	// CNN은 브라우저 헤더가 필수적일 수 있음
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://www.cnn.com")
	req.Header.Set("Referer", "https://www.cnn.com/")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cnn api status error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result CnnResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cnn data: %w", err)
	}

	return &result.FearAndGreed, nil
}

type HistoricalPoint struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Rating string  `json:"rating"`
}

type CnnHistoricalResponse struct {
	FearAndGreedHistorical struct {
		Data []HistoricalPoint `json:"data"`
	} `json:"fear_and_greed_historical"`
}

// FetchFearGreedHistorical CNN 내부 API를 통해 1년치 역사적 데이터를 가져옴
func (c *CnnClient) FetchFearGreedHistorical() ([]HistoricalPoint, error) {
	req, err := http.NewRequest("GET", c.BaseUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://www.cnn.com")
	req.Header.Set("Referer", "https://www.cnn.com/")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result CnnHistoricalResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cnn historical data: %w", err)
	}

	return result.FearAndGreedHistorical.Data, nil
}
