package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// Bot은 텔레그램 봇 구조체입니다
type Bot struct {
	Token  string
	ChatID string
}

// New는 텔레그램 봇을 생성합니다
func New(token, chatID string) *Bot {
	return &Bot{
		Token:  token,
		ChatID: chatID,
	}
}

// SendMessage는 텔레그램 메시지를 전송합니다
func (b *Bot) SendMessage(message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.Token)

	payload := map[string]interface{}{
		"chat_id":    b.ChatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("JSON 마샬링 실패: %w", err)
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("텔레그램 API 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("텔레그램 메시지 전송 실패 (상태: %d): %s", resp.StatusCode, string(body))
	}

	log.Println("✅ 텔레그램 메시지 전송 완료")
	return nil
}

// SendMessageSafe는 텔레그램 메시지를 전송하고 에러를 로그로 출력합니다
func (b *Bot) SendMessageSafe(message string) {
	if err := b.SendMessage(message); err != nil {
		log.Printf("⚠️  텔레그램 메시지 전송 실패: %v\n", err)
	}
}
