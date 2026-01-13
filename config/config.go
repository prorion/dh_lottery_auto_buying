package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// Account는 개별 계정 정보를 담는 구조체입니다
type Account struct {
	UserID   string `json:"userId"`
	Password string `json:"password"`
}

// Config는 전체 설정을 담는 구조체입니다
type Config struct {
	Accounts         []Account `json:"accounts"`
	TelegramBotToken string    `json:"telegramBotToken,omitempty"`
	TelegramChatID   string    `json:"telegramChatId,omitempty"`
}

// Load는 설정을 로드합니다
func Load() (Config, error) {
	// 1. 환경변수에서 로드 시도
	config, err := LoadFromEnv()
	if err == nil {
		return config, nil
	}

	log.Printf("환경변수 로드 실패: %v\n", err)

	// 2. 설정 파일에서 로드 시도
	config, err = LoadFromFile("config.json")
	if err == nil {
		return config, nil
	}

	log.Printf("설정 파일 로드 실패: %v\n", err)

	// 3. 대화형 입력
	log.Println("설정 정보를 입력해주세요:")
	return LoadInteractive()
}

// LoadFromEnv는 환경변수에서 설정을 로드합니다 (단일 계정)
func LoadFromEnv() (Config, error) {
	userID := os.Getenv("DH_LOTTERY_ID")
	password := os.Getenv("DH_LOTTERY_PW")
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramChatID := os.Getenv("TELEGRAM_CHAT_ID")

	if userID == "" || password == "" {
		return Config{}, fmt.Errorf("환경변수가 설정되지 않았습니다 (DH_LOTTERY_ID, DH_LOTTERY_PW)")
	}

	return Config{
		Accounts: []Account{
			{
				UserID:   userID,
				Password: password,
			},
		},
		TelegramBotToken: telegramToken,
		TelegramChatID:   telegramChatID,
	}, nil
}

// LoadFromFile은 파일에서 설정을 로드합니다
func LoadFromFile(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("설정 파일 읽기 실패: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("설정 파일 파싱 실패: %w", err)
	}

	if len(config.Accounts) == 0 {
		return Config{}, fmt.Errorf("설정 파일에 계정 정보가 없습니다")
	}

	// 각 계정의 필수 정보 검증
	for i, account := range config.Accounts {
		if account.UserID == "" || account.Password == "" {
			return Config{}, fmt.Errorf("계정 %d: 아이디와 비밀번호가 필요합니다", i+1)
		}
	}

	return config, nil
}

// LoadInteractive는 사용자 입력으로 설정을 로드합니다 (단일 계정)
func LoadInteractive() (Config, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("동행복권 아이디: ")
	userID, _ := reader.ReadString('\n')
	userID = strings.TrimSpace(userID)

	fmt.Print("비밀번호: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	if userID == "" || password == "" {
		return Config{}, fmt.Errorf("아이디와 비밀번호를 모두 입력해주세요")
	}

	return Config{
		Accounts: []Account{
			{
				UserID:   userID,
				Password: password,
			},
		},
	}, nil
}

// Print는 설정 정보를 출력합니다 (보안상 비밀번호는 마스킹)
func (c *Config) Print() {
	log.Println("=== 설정 정보 ===")
	log.Printf("등록된 계정 수: %d\n", len(c.Accounts))

	for i, account := range c.Accounts {
		maskedPw := strings.Repeat("*", len(account.Password))
		log.Printf("  [계정 %d] %s / %s\n", i+1, account.UserID, maskedPw)
	}

	if c.TelegramBotToken != "" && c.TelegramChatID != "" {
		log.Println("  텔레그램 알림: 활성화")
	} else {
		log.Println("  텔레그램 알림: 비활성화")
	}
}
