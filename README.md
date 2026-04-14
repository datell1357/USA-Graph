# USA Liquidity Dashboard (미국 주식 유동성 대시보드)

미국 거시 경제와 금융 시장의 핵심 유동성 지표 11가지를 실시간(또는 일/분 단위)으로 종합 분석하여, 현재 주식 시장이 위치한 **'유동성 레짐(완화 · 중립 · 긴축)'** 상태를 직관적으로 시각화하는 고성능 대시보드입니다. 

더불어 생성형 AI(Perplexity) 기반의 스마트 요약 리포트를 제공하여, 복잡한 경제 수치들을 매일 읽기 쉬운 인사이트 문장으로 번역해 줍니다. 단순한 데이터 나열을 넘어 **매크로 스코어링 시스템**을 통해 실제 투자 결정에 보탬이 되는 시그널을 추출하는 것을 목표로 합니다.

---

## 🌟 주요 및 특화 기능

### 1. 11개 핵심 유동성 지표 통합 스코어링
미국 연방준비제도(Fed)의 주요 통화 정책 및 시장 심리 지표 11개를 가중치 기반으로 병합하여 `0 ~ 100점`의 단일 '종합 점수'로 환산합니다.
- **순유동성 지표**: 역레포(RRP) 잔고, 연준 실질 예치금(은행 준비금), 재무부 일반계정(TGA 잔고)
- **통화량 및 금융 지표**: M2 통화량, MMF(머니마켓펀드) 자산, 장단기 금리차(10Y - 2Y)
- **글로벌 및 리스크 지표**: SOFR(담보 대출 금리), 달러 인덱스(DXY), 미국 하이일드 스프레드
- **시장 심리 지표**: VIX(공포지수), Fear & Greed Index

### 2. 동적 레짐(Regime) 판별 시스템 (유동성 신호등)
산출된 점수를 바탕으로 현재 시장의 상태를 3구간으로 자동 분류하고, 직관적인 신호등 색상으로 표현합니다.
- 🟢 **완화 (70점 이상)**: 유동성이 풍부하며 위험자산(주식) 투자에 매우 우호적인 환경
- 🟡 **중립 (40 ~ 69점)**: 보수적 관망세, 모멘텀 지표 변화를 예의 주시해야 하는 구간
- 🔴 **긴축 (39점 이하)**: 유동성 흡수 구간, 주식 비중 축소 및 현금/채권 비중 확대 고려 구간

### 3. AI 유동성 요약 보고서 (Perplexity 통합 & 스마트 캐싱)
종합 점수 원형 차트를 클릭 시, Perplexity AI가 생성한 **최신 시황 요약 보고서**를 모달 팝업으로 제공합니다.
- API 비용 폭탄을 막기 위한 **4단계 스마트 캐싱 무효화(Cache Invalidation) 로직** 적용
  - *레짐 변동 시*, *총점 5점 이상 변동 시*, *VIX 10% 이상 등 핵심 지표 트리거 작동 시*, *최대 24시간 경과 시*에만 AI API를 백그라운드에서 신규 호출
  - 그 외의 클릭에는 DB 캐싱 데이터를 0초 로딩으로 응답하되, **모달 내 종합 점수는 무조건 실시간 점수로 덮어씌워** 사용자 경험(UX) 극대화

### 4. 반응형 초경량 프리미엄 UI 
- 스크롤 없이(No-scroll) 데스크탑 모니터 해상도 1화면에 11개 지표와 미니 스파크라인(히스토리 차트)이 꽉 차게 들어오도록 설계 (블룸버그 터미널 스타일)
- 글래스모피즘(Glassmorphism)과 반응형 다크 톤(Premium Dark Mode) 인터페이스 적용

---

## 🛠 기술 스택 (Tech Stack)

안정적인 실시간 백그라운드 데이터 수집 프로세스와 빠른 API 응답을 위해 초경량 아키텍처를 선택했습니다.

### Backend (API & Data Aggregation)
- **Language**: Go (v1.20+) - 강력한 고루틴을 활용한 멀티 소스(FRED, Yahoo, CNN) 동시 비동기 패치
- **Framework**: Gin Gonic - 초고속 HTTP Web Framework
- **ORM & Database**: GORM + SQLite (CGO-free Glebarez) - Serverless/Docker 환경에 최적화된 파일 기반 DB 내장
- **LLM API**: Perplexity (llama-3.1-sonar 모델) - 시장 시황 실시간 분석

### Frontend (UI & Visualization)
- **Library**: React (v19) + Vite
- **Styling**: Vanilla CSS - 불필요한 의존성 없이 변수(Custom Properties)를 활용한 커스텀 테마
- **Charts**: Chart.js / react-chartjs-2 - 가벼운 스파크라인 및 도넛 스코어 차트 구현
- **State & HTTP**: React Hooks(useState, useEffect), Axios

---

## 📂 프로젝트 구조 (Directory Structure)

```text
USA-Graph/
├── client/                 # React 프론트엔드 (Vite)
│   ├── src/
│   │   ├── components/     # 지표 카드(Metric Card), 모달 등 재사용 컴포넌트
│   │   ├── App.jsx         # 메인 대시보드 레이아웃 및 렌더링 로직
│   │   └── index.css       # 글로벌 디자인 시스템 및 글래스모피즘 스타일
├── server/                 # Go 백엔드 (Gin, GORM)
│   ├── internal/
│   │   ├── domain/         # Metric, ScoreResult, AiReport 데이터베이스 구조체 모델
│   │   ├── app/            # 비즈니스 로직(채점 알고리즘, AI 캐싱 전략 로직)
│   │   └── infra/          # FRED, Yahoo, CNN 외부 API 통신 클라이언트
│   └── main.go             # API 라우팅, 서버 초기화, 주기적 백그라운드 스케줄러 실행
├── docs/                   # 프로젝트 컴포넌트 및 로직 상세 기능 명세서
├── ARCHITECTURE.md         # 전체 시스템 아키텍처 다이어그램 및 설계 의도 문서
├── RUNNING.md              # 로컬 및 운영 환경 실행/배포 가이드
├── TroubleShoot.md         # 주요 에러 해결 이력 및 디버깅 노하우 기록
└── Score Standard.md       # 핵심! 11개 지표별 산정 공식 및 가중치 레퍼런스
```

---

## 🌍 데이터 소스 (Data Sources)

이 대시보드는 글로벌 매크로 분석에 필수적인 공신력 있는 데이터를 다중 소스에서 수집합니다.
1. **FRED (Federal Reserve Economic Data)**: 미국 연방준비은행 공식 경제 DB (RRP, 준비금, TGA, M2, SOFR 등)
2. **Yahoo Finance**: DXY(달러 인덱스), VIX(공포지수)의 실시간 체결 데이터 크롤링 기반 파싱
3. **CNN Business**: Fear & Greed Index (시장 탐욕 공포 심리 지수) 실시간 퍼블릭 웹 파싱

---

## 📜 라이선스 및 면책 조항

- 본 프로젝트의 데이터 원천 중 하나인 **FRED®**는 세인트루이스 연방준비은행(Federal Reserve Bank of St. Louis)의 등록 상표입니다.
- 이 대시보드에서 제공하는 총점 시스템과 AI 분석 리포트는 참고 목적일 뿐, 절대적인 투자 권유나 금융 자문을 의미하지 않습니다. 투자의 최종 결정과 책임은 투자자 본인에게 있습니다.
