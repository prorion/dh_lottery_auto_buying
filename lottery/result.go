package lottery

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

// LottoResult는 당첨 결과 정보
type LottoResult struct {
	Round       string // 회차 (예: "1206")
	DrawDate    string // 추첨일 (예: "2026-01-10")
	Numbers     []int  // 당첨번호 6개
	BonusNumber int    // 보너스번호
}

// GetLatestResult는 최근 당첨번호를 가져옵니다
func GetLatestResult() (*LottoResult, error) {
	url := "https://www.dhlottery.co.kr/lt645/selectPstLt645Info.do"

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("API 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %w", err)
	}

	// JSON 파싱
	var apiResponse struct {
		Data struct {
			List []struct {
				LtEpsd   int    `json:"ltEpsd"`   // 회차
				LtRflYmd string `json:"ltRflYmd"` // 추첨일 (YYYYMMDD)
				Tm1WnNo  int    `json:"tm1WnNo"`  // 당첨번호 1
				Tm2WnNo  int    `json:"tm2WnNo"`  // 당첨번호 2
				Tm3WnNo  int    `json:"tm3WnNo"`  // 당첨번호 3
				Tm4WnNo  int    `json:"tm4WnNo"`  // 당첨번호 4
				Tm5WnNo  int    `json:"tm5WnNo"`  // 당첨번호 5
				Tm6WnNo  int    `json:"tm6WnNo"`  // 당첨번호 6
				BnsWnNo  int    `json:"bnsWnNo"`  // 보너스번호
			} `json:"list"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("JSON 파싱 실패: %w", err)
	}

	if len(apiResponse.Data.List) == 0 {
		return nil, fmt.Errorf("당첨 정보가 없습니다")
	}

	data := apiResponse.Data.List[0]

	// 날짜 포맷 변환 (YYYYMMDD -> YYYY-MM-DD)
	dateStr := data.LtRflYmd
	if len(dateStr) == 8 {
		dateStr = fmt.Sprintf("%s-%s-%s", dateStr[0:4], dateStr[4:6], dateStr[6:8])
	}

	result := &LottoResult{
		Round:       strconv.Itoa(data.LtEpsd),
		DrawDate:    dateStr,
		Numbers:     []int{data.Tm1WnNo, data.Tm2WnNo, data.Tm3WnNo, data.Tm4WnNo, data.Tm5WnNo, data.Tm6WnNo},
		BonusNumber: data.BnsWnNo,
	}

	log.Printf("✅ 당첨번호 조회 완료: %s회 (%s)\n", result.Round, result.DrawDate)
	log.Printf("   당첨번호: %v, 보너스: %d\n", result.Numbers, result.BonusNumber)

	return result, nil
}

// CheckWinning은 구매 번호와 당첨번호를 비교하여 등수를 판정합니다
func CheckWinning(purchaseNumbers []int, result *LottoResult) (rank int, matchCount int, hasBonus bool) {
	matchCount = 0
	hasBonus = false

	// 당첨번호와 일치하는 개수 확인
	for _, pNum := range purchaseNumbers {
		for _, wNum := range result.Numbers {
			if pNum == wNum {
				matchCount++
				break
			}
		}
	}

	// 보너스번호 확인
	for _, pNum := range purchaseNumbers {
		if pNum == result.BonusNumber {
			hasBonus = true
			break
		}
	}

	// 등수 판정
	switch matchCount {
	case 6:
		rank = 1 // 1등: 6개 일치
	case 5:
		if hasBonus {
			rank = 2 // 2등: 5개 일치 + 보너스
		} else {
			rank = 3 // 3등: 5개 일치
		}
	case 4:
		rank = 4 // 4등: 4개 일치
	case 3:
		rank = 5 // 5등: 3개 일치
	default:
		rank = 0 // 낙첨
	}

	return rank, matchCount, hasBonus
}

// FormatWinningMessage는 당첨 결과 메시지를 포맷합니다
func FormatWinningMessage(userID string, result *LottoResult, history *PurchaseHistory) string {
	if history == nil {
		return fmt.Sprintf("(%s) ℹ️ <b>당첨 확인 불가</b>\n\n저장된 구매 내역이 없습니다.", userID)
	}

	// 회차 확인
	if history.Round != result.Round {
		return fmt.Sprintf("(%s) ℹ️ <b>당첨 확인 불가</b>\n\n구매 회차(%s회)와 추첨 회차(%s회)가 다릅니다.",
			userID, history.Round, result.Round)
	}

	userPurchase, exists := history.Users[userID]
	if !exists {
		return fmt.Sprintf("(%s) ℹ️ <b>당첨 확인 불가</b>\n\n%s회 구매 내역이 없습니다.", userID, result.Round)
	}

	if !userPurchase.Success || len(userPurchase.Games) == 0 {
		return fmt.Sprintf("(%s) ℹ️ <b>당첨 확인 불가</b>\n\n%s회 구매가 실패했습니다.", userID, result.Round)
	}

	// 당첨 확인
	msg := fmt.Sprintf("(%s) 🎰 <b>로또 %s회 당첨 결과</b>\n\n", userID, result.Round)
	msg += fmt.Sprintf("🗓 추첨일: %s\n", result.DrawDate)
	msg += "🎱 당첨번호: "
	for i, num := range result.Numbers {
		if i > 0 {
			msg += ", "
		}
		msg += fmt.Sprintf("<b>%02d</b>", num)
	}
	msg += fmt.Sprintf("\n➕ 보너스: <b>%02d</b>\n\n", result.BonusNumber)
	msg += "━━━━━━━━━━━━━━━━━━━━\n\n"

	bestRank := 0
	totalWinnings := 0

	for _, game := range userPurchase.Games {
		rank, matchCount, hasBonus := CheckWinning(game.Numbers, result)

		// 게임 정보
		msg += fmt.Sprintf("🎲 [%s 게임]\n", game.Type)
		msg += "   번호: "
		for i, num := range game.Numbers {
			if i > 0 {
				msg += ", "
			}
			// 일치하는 번호는 강조
			isMatch := false
			for _, wNum := range result.Numbers {
				if num == wNum {
					isMatch = true
					break
				}
			}
			if isMatch {
				msg += fmt.Sprintf("✅<b>%02d</b>", num)
			} else {
				msg += fmt.Sprintf("%02d", num)
			}
		}
		msg += "\n"

		if rank > 0 {
			msg += fmt.Sprintf("   🎉 <b>%d등 당첨!</b> (%d개 일치", rank, matchCount)
			if hasBonus && rank == 2 {
				msg += " + 보너스"
			}
			msg += ")\n"

			if bestRank == 0 || rank < bestRank {
				bestRank = rank
			}
			totalWinnings++
		} else {
			msg += fmt.Sprintf("   ❌ 낙첨 (%d개 일치)\n", matchCount)
		}
		msg += "\n"
	}

	msg += "━━━━━━━━━━━━━━━━━━━━\n"

	if totalWinnings > 0 {
		msg += fmt.Sprintf("\n🎊 <b>총 %d게임 당첨!</b>\n", totalWinnings)
		if bestRank <= 3 {
			msg += "💰 <b>고액 당첨! 축하합니다!</b> 🎉\n"
		}
	} else {
		msg += "\n아쉽지만 다음 기회에! 😊\n"
	}

	return msg
}
