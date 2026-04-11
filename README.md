# USA Liquidity Dashboard

미국 시장의 유동성 지표를 종합적으로 분석하여 현재 시장의 '유동성 레짐(완화·중립·긴축)'을 시각화하는 대시보드입니다.

## 🌟 주요 기능

- **11개 종합 지표 분석**: RRP, 은행 준비금, TGA, M2, MMF, SOFR, 금리차, DXY, 하이일드 스프레드, VIX, Fear & Greed 등 핵심 지표 실시간 추적.
- **유동성 신호등**: 종합 점수(0-100점)를 기반으로 시장 상황을 완화(Green), 중립(Yellow), 긴축(Red)으로 분류.
- **자동 데이터 수집**: FRED(Federal Reserve Economic Data) API를 통해 5분 주기로 데이터 자동 갱신.
- **반응형 대시보드**: 고성능, 노스크롤(Compact) 설계를 통한 한눈에 들어오는 데이터 시각화.

## 🛠 기술 스택

### Backend
- **Language**: Go (v1.20+)
- **Framework**: Gin Gonic
- **ORM**: GORM
- **Database**: SQLite (Glebarez)
- **Data Source**: FRED API

### Frontend
- **Framework**: React (Vite)
- **Styling**: Vanilla CSS (Premium Dark Mode)
- **Charts**: Chart.js / react-chartjs-2

## 📂 프로젝트 구조

```text
/
├── client/          # React 프론트엔드
├── server/          # Go 백엔드
├── docs/            # 컴포넌트 및 로직 상세 문서
├── Score Standard.md # 점수 산정 기준 문서
├── ARCHITECTURE.md  # 시스템 설계 문서
├── RUNNING.md       # 실행 및 환경 설정 가이드
└── TroubleShoot.md  # 이슈 해결 이력
```

## 📜 라이선스

본 프로젝트의 데이터 원천인 FRED®는 세인트루이스 연방준비은행의 등록 상표입니다.
