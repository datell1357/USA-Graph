# RUNNING.md

본 문서는 USA Liquidity Dashboard 프로젝트를 로컬 및 서버 환경에서 실행하기 위한 가이드입니다.

## 1. 실행 개요
본 프로젝트는 Go 기반의 백엔드 서버와 React(Vite) 기반의 프론트엔드 클라이언트로 구성되어 있습니다. 5분 주기로 FRED API에서 데이터를 수집하여 SQLite 데이터베이스에 저장하고 대시보드에 시각화합니다.

## 2. 필수 프로그램
- **Go**: v1.20 이상
- **Node.js**: v18.0 이상 (npm 포함)
- **FRED API Key**: [St. Louis Fed](https://fred.stlouisfed.org/docs/api/api_key.html)에서 발급 가능

## 3. 환경 변수 설정
프로젝트 루트 디렉토리에 `.env` 파일을 생성하고 아래 내용을 설정합니다.

```env
FRED_API_KEY=your_api_key_here
PORT=8080
```

## 4. 실행 방법 (로컬)

### 4.1 백엔드 서버 실행
```bash
cd server
go mod tidy
go run main.go
```
서버는 기본적으로 `http://localhost:8080`에서 실행됩니다.

### 4.2 프론트엔드 클라이언트 실행
> [!IMPORTANT]
> 반드시 `client` 디렉토리로 이동한 후 명령어를 실행해야 합니다.

```bash
cd client
npm install
npm run dev
```
클라이언트는 기본적으로 `http://localhost:5173`에서 실행됩니다.

## 5. Docker 실행
(추후 지원 예정) 현재는 로컬 실행을 권장합니다.

## 6. 정상 동작 확인
1. 브라우저에서 `http://localhost:5173` 접속.
2. 상단 '종합 유동성 점수'가 숫자로 표시되는지 확인.
3. 하단 11개 지표 카드가 실제 데이터와 함께 표시되는지 확인.
4. 백엔드 터미널 로그에 `Initial data collection...` 및 `Calculation complete` 메시지가 출력되는지 확인.

## 7. 테스트 실행
### 7.1 백엔드 테스트
```bash
cd server
go test ./...
```

## 8. 자주 발생하는 오류
- **.env file not found**: 루트 디렉토리에 `.env` 파일이 있는지 확인하세요. 서버는 `../.env` 경로를 참조합니다.
- **FRED API 403 Error**: API 키가 유효한지 확인하고 대기 시간이 초과되었는지 체크하세요.
- **CORS Error**: 프론트엔드에서 API 호출 시 포트 번호(`8080`)가 일치하는지 확인하세요.
