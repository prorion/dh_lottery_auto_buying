package lottery

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// PurchaseHistory는 구매 내역을 관리하는 구조체
type PurchaseHistory struct {
	Round        string                  `json:"round"`        // 회차
	PurchaseDate string                  `json:"purchaseDate"` // 구매일
	Users        map[string]UserPurchase `json:"users"`        // 사용자별 구매 내역
}

// UserPurchase는 사용자별 구매 정보
type UserPurchase struct {
	Success bool           `json:"success"` // 구매 성공 여부
	Games   []GamePurchase `json:"games"`   // 게임별 번호
}

// GamePurchase는 게임별 구매 번호
type GamePurchase struct {
	Type    string `json:"type"`    // A, B, C, D, E
	Numbers []int  `json:"numbers"` // 선택된 번호들
}

const historyFilePath = "logs/last_purchase.json"

// SavePurchaseHistory는 구매 내역을 저장합니다
func SavePurchaseHistory(userID string, round string, purchaseDate string, result map[string]interface{}) error {
	// logs 디렉토리 생성
	if err := os.MkdirAll("logs", 0755); err != nil {
		return fmt.Errorf("logs 디렉토리 생성 실패: %w", err)
	}

	// 기존 파일 읽기
	history := &PurchaseHistory{
		Round:        round,
		PurchaseDate: purchaseDate,
		Users:        make(map[string]UserPurchase),
	}

	// 기존 파일이 있으면 읽기
	if data, err := os.ReadFile(historyFilePath); err == nil {
		var existingHistory PurchaseHistory
		if json.Unmarshal(data, &existingHistory) == nil {
			// 같은 회차면 기존 데이터 유지
			if existingHistory.Round == round {
				history = &existingHistory
			}
			// 다른 회차면 새로 시작 (덮어쓰기)
		}
	}

	// 구매 결과 파싱
	userPurchase := UserPurchase{
		Success: false,
		Games:   []GamePurchase{},
	}

	if resultData, ok := result["result"].(map[string]interface{}); ok {
		resultCode, _ := resultData["resultCode"].(string)

		if resultCode == "100" {
			// 구매 성공
			userPurchase.Success = true

			// 번호 추출
			if arrGameChoiceNum, ok := resultData["arrGameChoiceNum"].([]interface{}); ok {
				for _, gameData := range arrGameChoiceNum {
					gameStr, _ := gameData.(string)
					game := parseGameNumbers(gameStr)
					if game != nil {
						userPurchase.Games = append(userPurchase.Games, *game)
					}
				}
			}
		}
	}

	// 사용자 데이터 추가/업데이트
	history.Users[userID] = userPurchase

	// 파일에 저장
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 마샬링 실패: %w", err)
	}

	if err := os.WriteFile(historyFilePath, data, 0644); err != nil {
		return fmt.Errorf("파일 저장 실패: %w", err)
	}

	return nil
}

// parseGameNumbers는 "A|20|21|27|29|30|383" 형식의 문자열을 파싱합니다
func parseGameNumbers(gameStr string) *GamePurchase {
	parts := strings.Split(gameStr, "|")
	if len(parts) < 7 {
		return nil
	}

	game := &GamePurchase{
		Type:    parts[0],
		Numbers: make([]int, 0, 6),
	}

	// 번호 6개 추출 (마지막은 genType이므로 제외)
	for i := 1; i <= 6; i++ {
		if num, err := strconv.Atoi(parts[i]); err == nil {
			game.Numbers = append(game.Numbers, num)
		}
	}

	return game
}

// GetLastPurchaseHistory는 마지막 구매 내역을 읽어옵니다
func GetLastPurchaseHistory() (*PurchaseHistory, error) {
	data, err := os.ReadFile(historyFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // 파일이 없으면 nil 반환
		}
		return nil, fmt.Errorf("파일 읽기 실패: %w", err)
	}

	var history PurchaseHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("JSON 파싱 실패: %w", err)
	}

	return &history, nil
}
