# 🎰 동행복권 로또 6/45 자동 구매 프로그램

동행복권 웹사이트에서 로또 6/45를 자동으로 구매하는 Go 프로그램입니다.

## ✨ 주요 기능

- 🔐 **자동 로그인** (RSA 암호화 지원)
- 💰 **예치금 자동 확인**
- 🎱 **로또 자동 구매** (최대 5게임)
- 👥 **멀티 계정 지원** ⭐ NEW (여러 계정 순차 처리)
- 📱 **텔레그램 알림** (구매 성공/실패 알림, 계정별 구분)
- 📊 **실시간 로그 파일 저장** (`logs/` 디렉토리)
- ⏰ **스케줄러 모드** (자동 예약 구매)
- 🔍 **테스트 모드** (실제 구매 없이 테스트)

## 📁 프로젝트 구조

```
dh_lottery_auto_buying/
├── main.go                 # 메인 진입점
├── config/
│   └── config.go          # 설정 관리
├── telegram/
│   └── bot.go             # 텔레그램 봇
├── lottery/
│   ├── client.go          # 로또 클라이언트
│   ├── login.go           # 로그인
│   ├── balance.go         # 예치금 확인
│   ├── buy.go             # 구매 로직
│   └── types.go           # 공통 타입
├── logger/
│   └── logger.go          # 로그 설정
├── scheduler/
│   └── scheduler.go       # 스케줄러
├── tasks/
│   └── tasks.go           # 작업 실행
├── logs/                  # 로그 파일 저장 위치
│   └── lottery_YYYY-MM-DD.log
└── config.json            # 설정 파일
```

## 🚀 설치 및 실행

### 1. 의존성 설치

```bash
go mod tidy
```

### 2. 빌드

#### Windows에서 빌드
```bash
# 모든 플랫폼 빌드 (Windows + Linux)
.\build.bat

# 빠른 빌드 (현재 디렉토리)
.\build-quick.bat

# 수동 빌드
go build -o dhlottery.exe .
```

#### Linux/Ubuntu에서 빌드
```bash
# 모든 플랫폼 빌드
chmod +x build.sh
./build.sh

# 수동 빌드
go build -o dhlottery .
```

빌드 결과:
- `build/windows/dhlottery.exe` - Windows 실행 파일
- `build/linux/dhlottery-amd64` - Linux/Ubuntu (64비트)
- `build/linux/dhlottery-arm64` - Linux ARM64 (라즈베리파이 등)

### 3. 설정 파일 생성

`config.json` 파일을 생성하고 아래 내용을 입력하세요:

#### 단일 계정

```json
{
  "accounts": [
    {
      "userId": "your_id",
      "password": "your_password"
    }
  ],
  "telegramBotToken": "your_telegram_bot_token",
  "telegramChatId": "your_telegram_chat_id"
}
```

#### 여러 계정 (멀티 계정)

```json
{
  "accounts": [
    {
      "userId": "account1",
      "password": "password1"
    },
    {
      "userId": "account2",
      "password": "password2"
    },
    {
      "userId": "account3",
      "password": "password3"
    }
  ],
  "telegramBotToken": "your_telegram_bot_token",
  "telegramChatId": "your_telegram_chat_id"
}
```

> 💡 **여러 계정을 등록하면 순차적으로 예치금 확인 및 구매가 진행됩니다.**
> 각 계정은 독립적인 세션을 사용하므로 로그아웃 처리 없이 자동으로 분리됩니다.

또는 환경변수를 사용할 수 있습니다 (단일 계정만):

```bash
set DH_LOTTERY_ID=your_id
set DH_LOTTERY_PW=your_password
set TELEGRAM_BOT_TOKEN=your_telegram_bot_token
set TELEGRAM_CHAT_ID=your_telegram_chat_id
```

### 4. 실행

```bash
# 기본 모드: 예치금 확인 후 1회 구매
.\dhlottery.exe

# 예치금만 확인
.\dhlottery.exe -check

# 테스트 모드 (실제 구매 안함)
.\dhlottery.exe -dryrun

# 즉시 1회 구매 (예치금 확인 없이)
.\dhlottery.exe -once

# 스케줄러 모드
.\dhlottery.exe -service
```

## 📊 로그 파일

- 모든 로그는 `logs/lottery_YYYY-MM-DD.log` 파일에 자동으로 저장됩니다.
- 로그는 콘솔과 파일에 동시에 출력됩니다.
- 날짜별로 자동으로 파일이 분리됩니다.

## ⏰ 스케줄러 모드

스케줄러 모드로 실행하면 자동으로 예약 구매가 진행됩니다:

- **매주 월요일 오전 8시**: 예치금 확인 (10,000원 미만 시 알림)
- **매주 토요일 오전 6시**: 예치금 확인 후 로또 구매 (5게임)

```bash
.\dhlottery.exe -service
```

## 📱 텔레그램 알림

텔레그램 봇을 설정하면 다음과 같은 알림을 받을 수 있습니다:

- ✅ 구매 성공 (구매한 번호 포함)
- ❌ 구매 실패 (실패 사유)
- ⚠️ 예치금 부족 알림
- ⚠️ 로그인 실패 알림

## 🔧 개발

### 패키지 구조

- **config**: 설정 로드 및 관리
- **logger**: 로그 파일 생성 및 관리
- **telegram**: 텔레그램 봇 API
- **lottery**: 로또 구매 핵심 로직
  - `client.go`: HTTP 클라이언트
  - `login.go`: RSA 암호화 로그인
  - `balance.go`: 예치금 확인
  - `buy.go`: 로또 구매
- **scheduler**: 크론 스케줄러
- **tasks**: 작업 실행 (예치금 확인, 구매 등)

### 테스트

```bash
# 테스트 모드로 실행 (실제 구매 안함)
.\dhlottery.exe -dryrun
```

## 📝 라이센스

MIT License

## ⚠️ 주의사항

- 이 프로그램은 동행복권 공식 API를 사용하지 않으며, 웹 스크래핑을 통해 동작합니다.
- 동행복권 웹사이트 구조가 변경되면 동작하지 않을 수 있습니다.
- 개인 계정 정보는 안전하게 보관하세요.
- 로또 구매는 본인 책임 하에 진행하세요.

## 📞 문의

문제가 발생하면 이슈를 등록해주세요.

## 📅 업데이트 내역

### v2.1.0 (2026-01-13)
- ✨ **멀티 계정 지원** - 여러 계정을 등록하여 순차적으로 처리
- ✨ 텔레그램 메시지에 계정 ID 표시 (예: `(prorion) ✅ 로또 구매 성공!`)
- ✨ 계정별 독립 세션 관리
- 📝 설정 파일 형식 변경 (accounts 배열)

### v2.0.0 (2026-01-13)
- ✨ 프로젝트 구조 개선 (패키지 분리)
- ✨ 실시간 로그 파일 저장 기능 추가
- ✨ 텔레그램 에러 처리 개선
- 🐛 세션 안정성 향상
- 🐛 RSA 암호화 로그인 지원

### v1.0.0
- 🎉 최초 릴리스
