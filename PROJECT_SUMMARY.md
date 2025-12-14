# 동행복권 로또 자동 구매 프로그램 - 프로젝트 요약

## 개발 완료 날짜
2024년 12월 14일

## 프로젝트 개요
동행복권(dhlottery.co.kr) 사이트에 자동으로 로그인하고 로또 6/45를 **실제로 구매**하는 Go 기반 자동화 프로그램입니다.

## 구현된 기능

### ✅ 완료된 기능 (Phase 1 & 2)
1. **자동 로그인**
   - CSRF 토큰 자동 추출
   - 세션 쿠키 관리 (JSESSIONID)
   - 로그인 성공/실패 검증

2. **로또 6/45 구매 페이지 접근**
   - 인증된 세션으로 구매 페이지 접근
   - iframe 내부 페이지 (`game645.do`) 직접 접근
   - 페이지 내용 검증 및 정보 추출

3. **다중 설정 방식 지원**
   - 환경변수 (`DH_LOTTERY_ID`, `DH_LOTTERY_PW`)
   - JSON 설정 파일 (`config.json`)
   - 대화형 입력 모드

4. **보안 기능**
   - 비밀번호 마스킹 (로그 출력)
   - `.gitignore`에 민감 정보 제외
   - 설정 파일 예시 제공

5. **✨ 실제 로또 구매 기능 (NEW!)**
   - 자동번호 5게임 구매
   - 구매 대기열 체크
   - 실시간 예치금 잔액 확인
   - 구매 API 호출 (`/olotto/game/execBuy.do`)
   - 구매 결과 상세 출력

6. **✨ 구매 결과 처리 (NEW!)**
   - 구매 성공 시 번호 출력
   - 추첨일, 지급기한 정보 표시
   - 바코드 정보 출력
   - 오류 처리 (한도 초과, 예치금 부족, 판매시간 등)

### 🚧 향후 구현 예정 (Phase 3)
- 수동 번호 선택 기능
- 당첨 번호 확인 자동화
- 구매 내역 저장 (JSON/CSV)
- 스케줄링 기능

## 기술 스택
- **언어**: Go 1.24.0
- **주요 라이브러리**:
  - `github.com/PuerkitoBio/goquery` - HTML 파싱
  - `golang.org/x/net/publicsuffix` - 쿠키 관리
  - `net/http` - HTTP 클라이언트
  - `net/http/cookiejar` - 쿠키 저장소

## 프로젝트 구조
```
DhLottery/
├── main.go                 # 메인 프로그램 (약 420줄)
├── go.mod                  # Go 모듈 정의
├── go.sum                  # 의존성 체크섬
├── config.example.json     # 설정 파일 예시
├── .gitignore             # Git 제외 목록
├── README.md              # 프로젝트 문서
└── dhlottery.exe          # 빌드된 실행 파일 (9.7MB)
```

## 주요 구현 사항

### 1. 로그인 프로세스
```
사용자 → 로그인 페이지 접속 → Hidden 필드 추출 
       → POST 요청 (계정 정보 + CSRF 토큰)
       → 세션 쿠키 저장 → 로그인 확인
```

### 2. 설정 로드 우선순위
```
환경변수 → 설정 파일 (config.json) → 대화형 입력
```

### 3. HTTP 클라이언트 특징
- 쿠키 자동 저장/전송 (cookiejar)
- 리다이렉트 자동 추적 (최대 10회)
- 타임아웃 30초
- User-Agent 설정 (브라우저 에뮬레이션)

## 테스트 결과
✅ 로그인 성공
✅ 세션 유지 확인
✅ 로또 구매 페이지 접근 성공
✅ **실제 구매 성공 (5게임, 5,000원)**
✅ 구매 번호 출력 확인
✅ 오류 처리 확인 (한도 초과 메시지)
✅ 빌드 성공 (실행 파일 생성)

## 보안 고려사항
1. **구현된 보안 기능**
   - 비밀번호 마스킹 (로그)
   - `.gitignore`로 민감 정보 제외
   - 환경변수 사용 권장

2. **추가 권장 사항**
   - HTTPS 통신만 사용
   - 과도한 요청 방지 (Rate Limiting)
   - 로컬 환경에서만 사용

## 사용 방법

### 환경변수 방식 (권장)
```powershell
$env:DH_LOTTERY_ID="your_id"
$env:DH_LOTTERY_PW="your_password"
go run main.go
```

### 설정 파일 방식
```powershell
Copy-Item config.example.json config.json
notepad config.json  # 계정 정보 입력
go run main.go
```

### 빌드 및 실행
```powershell
go build -o dhlottery.exe main.go
.\dhlottery.exe
```

## 실행 결과 예시
```
==============================================
   동행복권 로또 자동 구매 프로그램 v1.0
==============================================

=== 설정 정보 ===
{
  "Password": "*********",
  "UserID": "your_id"
}

=== 로그인 시작 ===
1단계: 로그인 페이지 접속 중...
로그인 페이지 로드 완료 (상태 코드: 200)
2단계: 로그인 요청 전송 중...
✅ 로그인 성공! (loginResult 페이지 확인)
✅ 로그인 완료! 세션이 정상적으로 생성되었습니다

=== 로또 6/45 구매 페이지 접근 ===
로또 6/45 구매 페이지로 이동 중...
✅ 로또 6/45 구매 페이지 접근 성공!

==============================================
   ✅ 모든 작업이 완료되었습니다!
==============================================
```

## 알려진 제한사항
1. 실제 구매 기능은 아직 미구현
2. 동행복권 사이트 구조 변경 시 수정 필요
3. Windows 환경 최적화 (다른 OS에서는 테스트 필요)

## 개발 참고 사항

### 주요 URL 및 API
- 로그인 페이지: `https://www.dhlottery.co.kr/user.do?method=login&returnUrl=`
- 로그인 처리: `https://www.dhlottery.co.kr/userSsl.do?method=login`
- 구매 페이지 (래퍼): `https://el.dhlottery.co.kr/game/TotalGame.jsp?LottoId=LO40`
- 구매 페이지 (실제): `https://ol.dhlottery.co.kr/olotto/game/game645.do`
- **대기열 체크**: `https://ol.dhlottery.co.kr/olotto/game/egovUserReadySocket.json`
- **구매 API**: `https://ol.dhlottery.co.kr/olotto/game/execBuy.do`

### 중요 쿠키
- `JSESSIONID`: 세션 식별자
- `WMONID`: 웹 모니터링 ID

### HTTP 헤더
- `Content-Type: application/x-www-form-urlencoded`
- `User-Agent`: Chrome 120 에뮬레이션
- `Referer`: 이전 페이지 URL
- `X-Requested-With: XMLHttpRequest` (AJAX 요청용)

### 구매 API 파라미터
```go
{
    "round": "1203",              // 현재 회차
    "direct": "INTCOM1",          // 서버 IP
    "nBuyAmount": "5000",         // 구매 금액
    "param": [{                   // 게임 정보 배열
        "genType": "0",           // 0: 자동, 1: 수동, 2: 반자동
        "arrGameChoiceNum": null, // 자동일 경우 null
        "alpabet": "A"            // 게임 레이블 (A-E)
    }],
    "ROUND_DRAW_DATE": "2025/12/20",
    "WAMT_PAY_TLMT_END_DT": "2026/12/21",
    "gameCnt": "5"                // 게임 수
}
```

## 향후 개발 로드맵

### ✅ Phase 1 완료 (로그인 및 페이지 접근)
- [x] 로그인 자동화
- [x] 세션 관리
- [x] 구매 페이지 접근

### ✅ Phase 2 완료 (실제 구매 기능)
- [x] HTML 파싱 및 정보 추출
- [x] 자동번호 구매 구현
- [x] 대기열 처리
- [x] 구매 결과 출력
- [x] 오류 처리 (한도, 예치금, 시간 등)

### Phase 3 (예정)
- [ ] 수동 번호 선택 기능
- [ ] 반자동 (일부 수동, 일부 자동) 구매
- [ ] 구매 내역 저장 (JSON/CSV)
- [ ] 당첨 번호 확인 자동화

### Phase 4 (예정)
- [ ] 스케줄링 기능 (cron)
- [ ] 웹 GUI 인터페이스
- [ ] 다중 계정 지원
- [ ] 당첨 알림 기능

## 라이선스 및 주의사항
⚠️ **개인 사용 목적으로만 사용하세요**
- 동행복권의 공식 API가 아닙니다
- 과도한 요청은 계정 제재를 받을 수 있습니다
- 책임감 있는 복권 구매를 권장합니다

## 개발자
- 개발 환경: Windows 11 + Go 1.24.0
- 개발 도구: Cursor IDE
- 테스트 완료: 2024-12-14

---
**프로젝트 상태**: ✅ Phase 2 완료 (실제 구매 기능 구현 완료)

**⚠️ 중요 공지**
- 이 프로그램은 **실제 금액이 결제**됩니다
- 온라인 구매 한도: **1회차당 최대 5,000원 (5게임)**
- 테스트 시에도 실제 구매가 진행되니 주의하세요
- 예치금 잔액을 확인한 후 사용하세요

