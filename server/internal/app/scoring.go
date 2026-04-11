package app

import (
	"math"
)

type ScoringService struct{}

func NewScoringService() *ScoringService {
	return &ScoringService{}
}

// CalculateRegime 11개 지표 체계 및 확인조건에 따른 레짐 판별
func (s *ScoringService) CalculateRegime(totalScore float64, scores map[string]float64, currentData map[string]float64) (string, string) {
	// 리턴값: (Regime Name, Position Recommendation)
	
	// 5.2.2 긴축 우선 조건 (Red)
	// 1. 총점 <= 39
	// 2. (은행 준비금 + M2) 평균 점수 < 5
	// 3. RRP, TGA, 하이일드, VIX, DXY 중 3개 이상이 2.5 이하
	isRed := false
	if totalScore <= 39 {
		isRed = true
	} else if (scores["WRESBAL"]+scores["M2SL"])/2 < 5.0 {
		isRed = true
	} else {
		smallCount := 0
		if scores["RRPONTSYD"] <= 2.5 { smallCount++ }
		if scores["WTREGEN"] <= 2.5 { smallCount++ }
		if scores["BAMLH0A0HYM2"] <= 2.5 { smallCount++ }
		if scores["VIXCLS"] <= 2.5 { smallCount++ }
		if scores["DXY"] <= 2.5 { smallCount++ }
		if smallCount >= 3 { isRed = true }
	}

	if isRed {
		return "긴축", "방어적 포지션"
	}

	// 5.2.1 완화 확정 조건 (Green)
	// 1. 총점 >= 70
	// 2. (은행 준비금 + M2) 평균 >= 7.5
	// 3. (RRP, TGA 중 최소 1개) >= 7.5
	// 4. VIX >= 5 AND F&G >= 5
	if totalScore >= 70 && 
	   (scores["WRESBAL"]+scores["M2SL"])/2 >= 7.5 &&
	   (scores["WTREGEN"] >= 7.5 || scores["RRPONTSYD"] >= 7.5) &&
	   scores["VIXCLS"] >= 5.0 && scores["FEAR_GREED"] >= 5.0 {
		return "완화", "공격적 포지션"
	}

	// 5.2.3 중립 판제 (Yellow)
	return "중립", "보수적 관망"
}

// MapToScore 0~10점 5단계 점수 매핑
func (s *ScoringService) MapToScore(val float64, steps [4]float64, reverse bool) float64 {
	if !reverse {
		if val >= steps[0] { return 10.0 }
		if val >= steps[1] { return 7.5 }
		if val >= steps[2] { return 5.0 }
		if val >= steps[3] { return 2.5 }
		return 0.0
	} else {
		if val <= steps[0] { return 10.0 }
		if val <= steps[1] { return 7.5 }
		if val <= steps[2] { return 5.0 }
		if val <= steps[3] { return 2.5 }
		return 0.0
	}
}

// CalculateIndicatorScores 11개 지표별 점수 산출
func (s *ScoringService) CalculateIndicatorScores(current, prev map[string]float64) map[string]float64 {
	scores := make(map[string]float64)

	// 기초 유동성
	scores["RRPONTSYD"] = s.MapToScore(getRatio(current["RRPONTSYD"], prev["RRPONTSYD"]), [4]float64{-0.15, -0.05, 0.05, 0.15}, true)
	scores["WRESBAL"] = s.MapToScore(getRatio(current["WRESBAL"], prev["WRESBAL"]), [4]float64{0.05, 0.02, -0.02, -0.05}, false)
	scores["WTREGEN"] = s.MapToScore(getRatio(current["WTREGEN"], prev["WTREGEN"]), [4]float64{-0.15, -0.05, 0.05, 0.15}, true)
	scores["M2SL"] = s.MapToScore(getRatio(current["M2SL"], prev["M2SL"]), [4]float64{0.025, 0.010, -0.010, -0.025}, false)
	scores["RMFNS"] = s.MapToScore(getRatio(current["RMFNS"], prev["RMFNS"]), [4]float64{-0.05, -0.01, 0.01, 0.05}, true)

	// 금리·달러
	sofrDiff := current["SOFR"] - prev["SOFR"]
	scores["SOFR"] = s.MapToScore(sofrDiff, [4]float64{-0.10, -0.03, 0.03, 0.10}, true)
	scores["T10Y2Y"] = s.MapToScore(current["T10Y2Y"], [4]float64{0.75, 0.25, 0, -0.50}, false)
	scores["DXY"] = s.MapToScore(getRatio(current["DXY"], prev["DXY"]), [4]float64{-0.03, -0.01, 0.01, 0.03}, true)

	// 크레딧·심리
	// BAMLH0A0HYM2는 FRED에서 % 단위로 올 수 있으므로(예: 4.54), bp 환산(454)을 위해 100을 곱함
	scores["BAMLH0A0HYM2"] = s.MapToScore(current["BAMLH0A0HYM2"]*100, [4]float64{300, 400, 500, 600}, true)
	scores["VIXCLS"] = s.MapToScore(current["VIXCLS"], [4]float64{20, 25, 30, 35}, true)
	scores["FEAR_GREED"] = s.MapToScore(current["FEAR_GREED"], [4]float64{70, 55, 40, 25}, false)

	return scores
}

func getRatio(curr, prev float64) float64 {
	if prev == 0 { return 0 }
	return (curr - prev) / math.Abs(prev)
}
