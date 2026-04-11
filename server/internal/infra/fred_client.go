package infra

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"usa-graph/server/internal/domain"
)

type FredClient struct {
	ApiKey string
	BaseUrl string
}

func NewFredClient(apiKey string) *FredClient {
	return &FredClient{
		ApiKey: apiKey,
		BaseUrl: "https://api.stlouisfed.org/fred",
	}
}

// FetchObservations 특정 Series ID의 최신 데이터를 가져옴
func (c *FredClient) FetchObservations(seriesID string, startTime string) ([]domain.Metric, error) {
	url := fmt.Sprintf("%s/series/observations?series_id=%s&api_key=%s&file_type=json&observation_start=%s", 
		c.BaseUrl, seriesID, c.ApiKey, startTime)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call FRED API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FRED API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Observations []struct {
			Date  string `json:"date"`
			Value string `json:"value"`
		} `json:"observations"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode FRED response: %w", err)
	}

	var metrics []domain.Metric
	for _, obs := range result.Observations {
		if obs.Value == "." {
			continue // 누락된 데이터 처리
		}
		var val float64
		_, err := fmt.Sscanf(obs.Value, "%f", &val)
		if err != nil {
			continue
		}

		t, _ := time.Parse("2006-01-02", obs.Date)
		metrics = append(metrics, domain.Metric{
			SeriesID: seriesID,
			Value:    val,
			Date:     t,
		})
	}

	return metrics, nil
}
