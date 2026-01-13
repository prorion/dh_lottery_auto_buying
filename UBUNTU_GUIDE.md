# 🐧 Ubuntu/Linux 사용 가이드

## 📦 빌드된 파일

빌드 후 다음 파일들이 생성됩니다:

- `build/linux/dhlottery-amd64` - Ubuntu/Linux (64비트) 전용
- `build/linux/dhlottery-arm64` - ARM64 (라즈베리파이 등)

## 🚀 Ubuntu 서버에 배포하기

### 1. 파일 업로드

#### SCP 사용
```bash
# Windows에서 실행
scp build/linux/dhlottery-amd64 user@your-server:/home/user/dhlottery
```

#### SFTP 사용
```bash
sftp user@your-server
put build/linux/dhlottery-amd64 /home/user/dhlottery
```

#### WinSCP, FileZilla 등 FTP 클라이언트 사용 가능

### 2. 서버에서 설정

```bash
# SSH로 서버 접속
ssh user@your-server

# 실행 권한 부여
chmod +x dhlottery

# 설정 파일 생성
nano config.json
```

### 3. config.json 작성

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

저장: `Ctrl+O` → `Enter` → `Ctrl+X`

## 🎯 실행 방법

### 기본 실행 (예치금 확인 후 구매)
```bash
./dhlottery
```

### 예치금만 확인
```bash
./dhlottery -check
```

### 테스트 모드 (실제 구매 안함)
```bash
./dhlottery -dryrun
```

### 즉시 구매 (예치금 확인 생략)
```bash
./dhlottery -once
```

### 백그라운드 실행
```bash
nohup ./dhlottery &
```

### 로그 확인
```bash
# 실시간 로그 확인
tail -f logs/lottery_$(date +%Y-%m-%d).log

# 최근 로그 보기
cat logs/lottery_$(date +%Y-%m-%d).log
```

## ⏰ 자동 실행 (Cron 설정)

### 1. Cron 편집
```bash
crontab -e
```

### 2. 스케줄 추가

```bash
# 매주 토요일 오전 6시에 실행
0 6 * * 6 cd /home/user && ./dhlottery >> /home/user/logs/cron.log 2>&1

# 매주 월요일 오전 8시에 예치금 확인
0 8 * * 1 cd /home/user && ./dhlottery -check >> /home/user/logs/cron.log 2>&1
```

### 3. Cron 로그 확인
```bash
tail -f ~/logs/cron.log
```

## 🔧 systemd 서비스로 등록 (스케줄러 모드)

### 1. 서비스 파일 생성
```bash
sudo nano /etc/systemd/system/dhlottery.service
```

### 2. 서비스 내용
```ini
[Unit]
Description=DH Lottery Auto Buy Service
After=network.target

[Service]
Type=simple
User=your_username
WorkingDirectory=/home/your_username
ExecStart=/home/your_username/dhlottery -service
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### 3. 서비스 시작
```bash
# 서비스 리로드
sudo systemctl daemon-reload

# 서비스 시작
sudo systemctl start dhlottery

# 부팅 시 자동 시작
sudo systemctl enable dhlottery

# 상태 확인
sudo systemctl status dhlottery

# 로그 확인
sudo journalctl -u dhlottery -f
```

## 📊 로그 관리

### 로그 파일 위치
```
logs/lottery_YYYY-MM-DD.log
```

### 로그 자동 정리 (30일 이상 삭제)
```bash
# crontab에 추가
0 3 * * * find /home/user/logs -name "lottery_*.log" -mtime +30 -delete
```

## 🔍 문제 해결

### 실행 권한 오류
```bash
chmod +x dhlottery
```

### 설정 파일 오류
```bash
# JSON 문법 검증
cat config.json | jq .
```

### 네트워크 오류
```bash
# DNS 확인
ping www.dhlottery.co.kr

# 방화벽 확인
sudo ufw status
```

### 프로세스 확인
```bash
# 실행 중인 프로세스 찾기
ps aux | grep dhlottery

# 프로세스 종료
pkill dhlottery
```

## 💡 팁

### 1. 백그라운드 실행 + 로그
```bash
nohup ./dhlottery > output.log 2>&1 &
```

### 2. 화면 세션 사용 (screen)
```bash
# screen 설치
sudo apt install screen

# 세션 시작
screen -S lottery

# 프로그램 실행
./dhlottery -service

# 세션 분리: Ctrl+A, D

# 세션 재접속
screen -r lottery
```

### 3. tmux 사용
```bash
# tmux 설치
sudo apt install tmux

# 세션 시작
tmux new -s lottery

# 프로그램 실행
./dhlottery -service

# 세션 분리: Ctrl+B, D

# 세션 재접속
tmux attach -t lottery
```

## 🔒 보안 팁

### 1. 설정 파일 권한 설정
```bash
chmod 600 config.json
```

### 2. 전용 사용자 생성
```bash
sudo useradd -m -s /bin/bash lottery
sudo su - lottery
```

### 3. 환경변수 사용
```bash
# 환경변수 설정
export DH_LOTTERY_ID="your_id"
export DH_LOTTERY_PW="your_password"
export TELEGRAM_BOT_TOKEN="your_token"
export TELEGRAM_CHAT_ID="your_chat_id"

# 실행
./dhlottery
```

## 📱 텔레그램 봇 설정 확인

```bash
# 텔레그램 API 테스트
curl -X GET "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getMe"

# 메시지 전송 테스트
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/sendMessage" \
  -d "chat_id=<YOUR_CHAT_ID>" \
  -d "text=테스트 메시지"
```

## ⚙️ 의존성

프로그램은 정적 빌드되어 별도의 의존성이 필요 없습니다!

- ✅ 추가 라이브러리 설치 불필요
- ✅ Go 런타임 설치 불필요
- ✅ 독립 실행 파일

## 📞 문제 발생 시

1. 로그 파일 확인: `cat logs/lottery_$(date +%Y-%m-%d).log`
2. 네트워크 연결 확인
3. 설정 파일 문법 확인
4. 텔레그램 봇 토큰 확인

---

**Happy Lottery! 🎱**
