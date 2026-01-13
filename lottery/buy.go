package lottery

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// BuyLottoAutoWithResult는 로또를 자동으로 구매하고 텔레그램용 메시지를 반환합니다
func (c *Client) BuyLottoAutoWithResult(userID string, quantity int) (map[string]interface{}, string, error) {
	// 실제 로또 구매 페이지 접근
	buyPageURL := "https://ol.dhlottery.co.kr/olotto/game/game645.do"

	req, err := http.NewRequest("GET", buyPageURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("구매 페이지 요청 생성 실패: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://el.dhlottery.co.kr/game/TotalGame.jsp?LottoId=LO40")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("구매 페이지 접속 실패: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	log.Printf("구매 페이지 응답 상태: %d\n", resp.StatusCode)

	// 2단계: HTML 파싱하여 구매에 필요한 정보 추출
	log.Println("2단계: 구매 정보 추출 중...")

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return nil, "", fmt.Errorf("HTML 파싱 실패: %w", err)
	}

	gameInfo := LottoGameInfo{}

	doc.Find("#curRound").Each(func(i int, s *goquery.Selection) {
		gameInfo.CurRound = strings.TrimSpace(s.Text())
	})

	doc.Find("#ROUND_DRAW_DATE").Each(func(i int, s *goquery.Selection) {
		if val, exists := s.Attr("value"); exists {
			gameInfo.RoundDrawDate = val
		}
	})

	doc.Find("#WAMT_PAY_TLMT_END_DT").Each(func(i int, s *goquery.Selection) {
		if val, exists := s.Attr("value"); exists {
			gameInfo.WamtPayTlmtEndDt = val
		}
	})

	doc.Find("#moneyBalance").Each(func(i int, s *goquery.Selection) {
		gameInfo.MoneyBalance = strings.TrimSpace(s.Text())
	})

	// 필수 정보 검증
	if gameInfo.CurRound == "" || gameInfo.RoundDrawDate == "" {
		return nil, "", fmt.Errorf("구매 정보 추출 실패: 회차 또는 추첨일 정보가 없습니다")
	}

	log.Printf("   → 현재 회차: %s회\n", gameInfo.CurRound)
	log.Printf("   → 추첨일: %s\n", gameInfo.RoundDrawDate)
	log.Printf("   → 예치금: %s원\n", gameInfo.MoneyBalance)

	// 3단계: 대기열 체크
	log.Println("3단계: 구매 대기열 확인 중...")

	directIP, err := c.checkReadySocket()
	if err != nil {
		return nil, "", fmt.Errorf("대기열 확인 실패: %w", err)
	}

	if directIP != "" {
		log.Printf("   → 대기열 없음, 즉시 구매 가능 (IP: %s)\n", directIP)
	}

	// 4단계: 구매 직전 세션 확인을 위해 구매 페이지 재방문
	log.Println("4단계: 구매 전 세션 확인 중...")

	sessionCheckReq, err := http.NewRequest("GET", buyPageURL, nil)
	if err == nil {
		sessionCheckReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		sessionCheckReq.Header.Set("Referer", "https://el.dhlottery.co.kr/game/TotalGame.jsp?LottoId=LO40")
		sessionCheckResp, err := c.httpClient.Do(sessionCheckReq)
		if err == nil {
			defer sessionCheckResp.Body.Close()
			io.ReadAll(sessionCheckResp.Body)
			log.Println("   → 세션 갱신 완료")
		}
	}

	// 5단계: 실제 구매 요청
	log.Println("5단계: 로또 구매 요청 중...")
	log.Printf("   💰 구매 금액: %d원\n", quantity*1000)

	result, err := c.executeBuy(gameInfo, directIP, quantity)
	if err != nil {
		return nil, "", fmt.Errorf("구매 실패: %w", err)
	}

	// 6단계: 텔레그램용 메시지 생성
	telegramMsg := c.formatTelegramMessage(userID, result, quantity)

	return result, telegramMsg, nil
}

// checkReadySocket은 구매 대기열을 확인합니다
func (c *Client) checkReadySocket() (string, error) {
	readyURL := "https://ol.dhlottery.co.kr/olotto/game/egovUserReadySocket.json"

	req, err := http.NewRequest("POST", readyURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://ol.dhlottery.co.kr/olotto/game/game645.do")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var readyResult map[string]interface{}
	if err := json.Unmarshal(body, &readyResult); err != nil {
		return "", fmt.Errorf("대기열 응답 파싱 실패: %w", err)
	}

	// ready_cnt가 0이면 바로 구매 가능
	if readyCnt, ok := readyResult["ready_cnt"].(float64); ok && readyCnt > 0 {
		log.Printf("   ⚠️  대기 인원: %.0f명\n", readyCnt)
		if readyTime, ok := readyResult["ready_time"].(float64); ok {
			log.Printf("   ⏱️  예상 대기시간: %.0f초\n", readyTime)
		}
		return "", fmt.Errorf("현재 대기 인원이 있습니다. 잠시 후 다시 시도해주세요")
	}

	// direct IP 반환
	if readyIP, ok := readyResult["ready_ip"].(string); ok {
		return readyIP, nil
	}

	return "", nil
}

// executeBuy는 실제 구매를 실행합니다
func (c *Client) executeBuy(gameInfo LottoGameInfo, directIP string, quantity int) (map[string]interface{}, error) {
	buyURL := "https://ol.dhlottery.co.kr/olotto/game/execBuy.do"

	// 자동 구매 파라미터 생성
	alpabet := []string{"A", "B", "C", "D", "E"}
	param := make([]map[string]interface{}, quantity)

	for i := 0; i < quantity; i++ {
		param[i] = map[string]interface{}{
			"genType":          "0",
			"arrGameChoiceNum": nil,
			"alpabet":          alpabet[i],
		}
	}

	paramJSON, _ := json.Marshal(param)

	// 폼 데이터 생성
	formData := url.Values{}
	formData.Set("round", gameInfo.CurRound)
	formData.Set("direct", directIP)
	formData.Set("nBuyAmount", fmt.Sprintf("%d", quantity*1000))
	formData.Set("param", string(paramJSON))
	formData.Set("ROUND_DRAW_DATE", gameInfo.RoundDrawDate)
	formData.Set("WAMT_PAY_TLMT_END_DT", gameInfo.WamtPayTlmtEndDt)
	formData.Set("gameCnt", fmt.Sprintf("%d", quantity))

	req, err := http.NewRequest("POST", buyURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://ol.dhlottery.co.kr/olotto/game/game645.do")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	log.Printf("   → 구매 응답 상태 코드: %d\n", resp.StatusCode)
	log.Printf("   → 구매 응답 URL: %s\n", resp.Request.URL.String())
	log.Printf("   → 구매 응답 길이: %d bytes\n", len(bodyStr))

	// JSON 파싱
	var buyResult map[string]interface{}
	if err := json.Unmarshal(body, &buyResult); err != nil {
		log.Printf("❌ JSON 파싱 실패!\n")
		log.Printf("   응답 내용 샘플 (처음 500자):\n%s\n", bodyStr[:min(500, len(bodyStr))])

		if strings.Contains(bodyStr, "<html") || strings.Contains(bodyStr, "<!DOCTYPE") {
			return nil, fmt.Errorf("구매 실패: 세션이 만료되었거나 로그인이 필요합니다 (HTML 응답 수신)")
		}

		return nil, fmt.Errorf("구매 응답 파싱 실패: %w", err)
	}

	log.Printf("   → 구매 응답 파싱 성공\n")

	return buyResult, nil
}

// formatTelegramMessage는 구매 결과를 텔레그램 메시지로 포맷합니다
func (c *Client) formatTelegramMessage(userID string, result map[string]interface{}, quantity int) string {
	// 로그인 체크
	if loginYn, ok := result["loginYn"].(string); ok && loginYn == "N" {
		return fmt.Sprintf("(%s) ❌ <b>로그인 세션 만료</b>\n\n다시 로그인해주세요.", userID)
	}

	// 기기 제한 체크
	if isAllowed, ok := result["isAllowed"].(string); ok && isAllowed == "N" {
		return fmt.Sprintf("(%s) ❌ <b>구매 실패</b>\n\n모바일에서는 구매할 수 없습니다.", userID)
	}

	// 판매시간 체크
	if checkTime, ok := result["checkOltSaleTime"].(bool); ok && !checkTime {
		return fmt.Sprintf("(%s) ❌ <b>구매 실패</b>\n\n현재 판매 시간이 아닙니다.", userID)
	}

	// 결과 확인
	if resultData, ok := result["result"].(map[string]interface{}); ok {
		resultCode := resultData["resultCode"].(string)

		if resultCode == "100" {
			// 구매 성공
			msg := fmt.Sprintf("(%s) ✅ <b>로또 구매 성공!</b>\n\n", userID)
			msg += fmt.Sprintf("💰 구매 금액: <b>%s원</b>\n", FormatMoney(quantity*1000))
			msg += fmt.Sprintf("🎱 구매 게임: <b>%d게임</b>\n\n", quantity)

			// 번호 출력
			if arrGameChoiceNum, ok := resultData["arrGameChoiceNum"].([]interface{}); ok {
				alpabet := []string{"A", "B", "C", "D", "E"}

				for i, numData := range arrGameChoiceNum {
					numStr := numData.(string)
					genType := numStr[len(numStr)-1:]
					numStr = numStr[:len(numStr)-1]

					numbers := strings.Split(numStr, "|")

					typeLabel := ""
					if genType == "3" {
						typeLabel = " (자동)"
					}

					msg += fmt.Sprintf("[%s%s] ", alpabet[i], typeLabel)
					for j, num := range numbers {
						if j > 0 {
							msg += " - "
						}
						msg += strings.TrimSpace(num)
					}
					msg += "\n"
				}
			}

			msg += "\n"

			// 추첨일
			if drawDate, ok := resultData["drawDate"].(string); ok {
				msg += fmt.Sprintf("📅 추첨일: %s\n", drawDate)
			}

			msg += "\n💡 행운을 빕니다!"

			return msg

		} else {
			// 구매 실패
			resultMsg := ""
			if msg, ok := resultData["resultMsg"].(string); ok {
				resultMsg = msg
			}

			msg := fmt.Sprintf("(%s) ❌ <b>구매 실패</b>\n\n", userID)
			msg += fmt.Sprintf("사유: %s\n\n", resultMsg)

			if strings.Contains(resultMsg, "한도") || strings.Contains(resultMsg, "5000") {
				msg += "💡 이번 회차에 이미 최대 한도(5,000원)를 구매하셨습니다."
			} else if strings.Contains(resultMsg, "예치금") || strings.Contains(resultMsg, "잔액") {
				msg += "💡 예치금이 부족합니다. 충전 후 다시 시도해주세요."
			}

			return msg
		}
	}

	return fmt.Sprintf("(%s) ❌ 구매 결과를 확인할 수 없습니다.", userID)
}

// PrintBuyResult는 구매 결과를 출력합니다
func (c *Client) PrintBuyResult(result map[string]interface{}) {
	log.Println()
	log.Println("╔════════════════════════════════════════╗")
	log.Println("║          로또 6/45 구매 결과           ║")
	log.Println("╚════════════════════════════════════════╝")
	log.Println()

	// 로그인 체크
	if loginYn, ok := result["loginYn"].(string); ok && loginYn == "N" {
		log.Println("❌ 로그인 세션이 만료되었습니다.")
		log.Println("   다시 로그인해주세요.")
		return
	}

	// 기기 제한 체크
	if isAllowed, ok := result["isAllowed"].(string); ok && isAllowed == "N" {
		log.Println("❌ 모바일에서는 구매할 수 없습니다.")
		log.Println("   PC 환경에서 시도해주세요.")
		return
	}

	// 판매시간 체크
	if checkTime, ok := result["checkOltSaleTime"].(bool); ok && !checkTime {
		log.Println("❌ 현재 판매 시간이 아닙니다.")
		log.Println("   판매 시간을 확인해주세요.")
		return
	}

	// 결과 확인
	if resultData, ok := result["result"].(map[string]interface{}); ok {
		resultCode := resultData["resultCode"].(string)

		if resultCode == "100" {
			// 구매 성공
			log.Println("✅ 구매가 성공적으로 완료되었습니다!")
			log.Println()

			// 구매 번호 출력
			if arrGameChoiceNum, ok := resultData["arrGameChoiceNum"].([]interface{}); ok {
				log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
				log.Printf("    구매 게임 수: %d 게임 (총 %s원)\n", len(arrGameChoiceNum), FormatMoney(len(arrGameChoiceNum)*1000))
				log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
				log.Println()

				alpabet := []string{"A", "B", "C", "D", "E"}

				for i, numData := range arrGameChoiceNum {
					numStr := numData.(string)
					genType := numStr[len(numStr)-1:]
					numStr = numStr[:len(numStr)-1]

					numbers := strings.Split(numStr, "|")

					gameLabel := alpabet[i]
					typeLabel := ""
					if genType == "3" {
						typeLabel = " (자동)"
					} else if genType == "1" {
						typeLabel = " (수동)"
					} else if genType == "2" {
						typeLabel = " (반자동)"
					}

					log.Printf("  🎱 [%s 게임%s]  ", gameLabel, typeLabel)

					for j, num := range numbers {
						if j > 0 {
							log.Printf(" - ")
						}
						log.Printf("%s", strings.TrimSpace(num))
					}
					log.Println()
				}
			}

			log.Println()
			log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

			// 당첨금 수령 정보
			if drawDate, ok := resultData["drawDate"].(string); ok {
				log.Printf("    추첨일: %s\n", drawDate)
			}

			if payLimitDate, ok := resultData["payLimitDate"].(string); ok {
				log.Printf("    당첨금 지급기한: %s\n", payLimitDate)
			}

			// 바코드 정보
			if barCode, ok := resultData["barCode"].([]interface{}); ok && len(barCode) > 0 {
				log.Println()
				log.Print("    바코드: ")
				for i, code := range barCode {
					if i > 0 {
						log.Printf(" ")
					}
					log.Printf("%v", code)
				}
				log.Println()
			}

			log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			log.Println()
			log.Println("💡 구매가 완료되었습니다. 행운을 빕니다!")

		} else {
			// 구매 실패
			resultMsg := ""
			if msg, ok := resultData["resultMsg"].(string); ok {
				resultMsg = msg
			}

			log.Println("❌ 구매 실패")
			log.Println()
			log.Printf("   사유: %s\n", resultMsg)
			log.Println()

			if strings.Contains(resultMsg, "한도") || strings.Contains(resultMsg, "5000") {
				log.Println("   💡 이번 회차에 이미 최대 한도(5,000원)를 구매하셨습니다.")
				log.Println("      온라인으로는 1회차당 최대 5게임까지만 구매 가능합니다.")
			} else if strings.Contains(resultMsg, "예치금") || strings.Contains(resultMsg, "잔액") {
				log.Println("   💡 예치금이 부족합니다.")
				log.Println("      예치금을 충전한 후 다시 시도해주세요.")
			} else if strings.Contains(resultMsg, "시간") {
				log.Println("   💡 현재 구매 가능한 시간이 아닙니다.")
				log.Println("      판매 시간을 확인해주세요.")
			}
		}
	}

	log.Println()
}

// GetLoginStatus는 현재 로그인 상태를 반환합니다
func (c *Client) GetLoginStatus() (bool, error) {
	resp, err := c.httpClient.Get("https://www.dhlottery.co.kr/")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	return strings.Contains(bodyStr, "로그아웃"), nil
}
