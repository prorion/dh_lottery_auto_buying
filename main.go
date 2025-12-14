package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/robfig/cron/v3"
	"golang.org/x/net/publicsuffix"
)

// Config는 로그인 및 구매 설정을 담는 구조체입니다
type Config struct {
	UserID           string `json:"userId"`
	Password         string `json:"password"`
	TelegramBotToken string `json:"telegramBotToken,omitempty"`
	TelegramChatID   string `json:"telegramChatId,omitempty"`
}

// TelegramBot은 텔레그램 봇 설정입니다
type TelegramBot struct {
	Token  string
	ChatID string
}

// NewTelegramBot은 텔레그램 봇을 생성합니다
func NewTelegramBot(token, chatID string) *TelegramBot {
	return &TelegramBot{
		Token:  token,
		ChatID: chatID,
	}
}

// SendMessage는 텔레그램 메시지를 전송합니다
func (t *TelegramBot) SendMessage(message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.Token)

	payload := map[string]interface{}{
		"chat_id":    t.ChatID,
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

// DhLottery는 동행복권 클라이언트 구조체입니다
type DhLottery struct {
	client *http.Client
	config Config
}

// NewDhLottery는 새로운 DhLottery 클라이언트를 생성합니다
func NewDhLottery(config Config) (*DhLottery, error) {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, fmt.Errorf("쿠키 저장소 생성 실패: %w", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 리다이렉트를 자동으로 따라가도록 설정
			if len(via) >= 10 {
				return fmt.Errorf("리다이렉트가 너무 많습니다")
			}
			return nil
		},
	}

	return &DhLottery{
		client: client,
		config: config,
	}, nil
}

// Login은 동행복권 사이트에 로그인합니다
func (d *DhLottery) Login() error {
	loginURL := "https://www.dhlottery.co.kr/user.do?method=login&returnUrl="

	// 로그인 페이지 접속
	resp, err := d.client.Get(loginURL)
	if err != nil {
		return fmt.Errorf("로그인 페이지 접속 실패: %w", err)
	}
	defer resp.Body.Close()

	// HTML 파싱
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("HTML 파싱 실패: %w", err)
	}

	// 로그인 폼 데이터 준비
	formData := url.Values{}
	formData.Set("returnUrl", "")
	formData.Set("userId", d.config.UserID)
	formData.Set("password", d.config.Password)

	// hidden 필드들 추출 (CSRF 토큰 등)
	doc.Find("form input[type='hidden']").Each(func(i int, s *goquery.Selection) {
		if name, exists := s.Attr("name"); exists {
			if value, exists := s.Attr("value"); exists {
				formData.Set(name, value)
			}
		}
	})

	// POST 요청 생성
	req, err := http.NewRequest("POST", "https://www.dhlottery.co.kr/userSsl.do?method=login",
		strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("로그인 요청 생성 실패: %w", err)
	}

	// 헤더 설정
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", loginURL)
	req.Header.Set("Origin", "https://www.dhlottery.co.kr")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7")

	// 로그인 요청 전송
	loginResp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("로그인 요청 실패: %w", err)
	}
	defer loginResp.Body.Close()

	// 응답 확인
	body, _ := io.ReadAll(loginResp.Body)
	bodyStr := string(body)

	// 로그인 실패 체크
	if strings.Contains(bodyStr, "아이디 또는 비밀번호를 확인해주세요") ||
		strings.Contains(bodyStr, "아이디") && strings.Contains(bodyStr, "비밀번호") && strings.Contains(bodyStr, "확인") {
		return fmt.Errorf("로그인 실패: 아이디 또는 비밀번호가 올바르지 않습니다")
	}

	// 로그인 성공 체크 - loginResult 페이지 확인
	if strings.Contains(loginResp.Request.URL.String(), "loginResult") && loginResp.StatusCode == 200 {
		log.Println("✅ 로그인 성공")
	}

	// 쿠키 확인
	cookies := d.client.Jar.Cookies(loginResp.Request.URL)

	// 세션 쿠키 확인
	hasSession := false
	for _, cookie := range cookies {
		if cookie.Name == "JSESSIONID" {
			hasSession = true
			break
		}
	}

	if !hasSession {
		return fmt.Errorf("로그인 실패: 세션 쿠키를 찾을 수 없습니다")
	}

	log.Println("✅ 로그인 완료! 세션이 정상적으로 생성되었습니다")
	return nil
}

// CheckBalance는 예치금 잔액을 확인합니다
func (d *DhLottery) CheckBalance() (int, error) {
	log.Println("예치금 확인 중...")

	// 메인 페이지 접속 (실제 메인 페이지)
	resp, err := d.client.Get("https://www.dhlottery.co.kr/common.do?method=main")
	if err != nil {
		return 0, fmt.Errorf("메인 페이지 접속 실패: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// HTML 파싱
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return 0, fmt.Errorf("HTML 파싱 실패: %w", err)
	}

	// 예치금 요소 찾기 (여러 방법 시도)
	balance := 0

	// 방법 1: .topAccount .information .money a strong
	doc.Find(".topAccount .information .money a strong").Each(func(i int, s *goquery.Selection) {
		balanceText := strings.TrimSpace(s.Text())
		balanceText = strings.ReplaceAll(balanceText, ",", "")
		balanceText = strings.ReplaceAll(balanceText, "원", "")
		balanceText = strings.TrimSpace(balanceText)
		if balanceText != "" {
			fmt.Sscanf(balanceText, "%d", &balance)
		}
	})

	// 방법 2: .money strong (첫 번째 시도 실패시)
	if balance == 0 {
		doc.Find(".money strong").Each(func(i int, s *goquery.Selection) {
			balanceText := strings.TrimSpace(s.Text())
			if strings.Contains(balanceText, "원") {
				balanceText = strings.ReplaceAll(balanceText, ",", "")
				balanceText = strings.ReplaceAll(balanceText, "원", "")
				balanceText = strings.TrimSpace(balanceText)
				if balanceText != "" {
					fmt.Sscanf(balanceText, "%d", &balance)
					log.Printf("   (방법2) 추출: %s -> %d\n", s.Text(), balance)
				}
			}
		})
	}

	// 방법 3: a href에 depositListView가 있는 strong
	if balance == 0 {
		doc.Find("a[href*='depositListView'] strong").Each(func(i int, s *goquery.Selection) {
			balanceText := strings.TrimSpace(s.Text())
			balanceText = strings.ReplaceAll(balanceText, ",", "")
			balanceText = strings.ReplaceAll(balanceText, "원", "")
			balanceText = strings.TrimSpace(balanceText)
			if balanceText != "" {
				fmt.Sscanf(balanceText, "%d", &balance)
				log.Printf("   (방법3) 추출: %s -> %d\n", s.Text(), balance)
			}
		})
	}

	log.Printf("✅ 예치금 확인 완료: %s원\n", formatMoney(balance))
	return balance, nil
}

// formatMoney는 숫자를 천 단위 구분자가 있는 문자열로 변환합니다
func formatMoney(amount int) string {
	if amount < 1000 {
		return fmt.Sprintf("%d", amount)
	}

	str := fmt.Sprintf("%d", amount)
	result := ""
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

// min 함수 (Go 1.21 미만 호환성)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NavigateToLottoBuyPage는 로또 6/45 구매 페이지로 이동합니다
func (d *DhLottery) NavigateToLottoBuyPage() error {
	log.Println("로또 6/45 구매 페이지로 이동 중...")

	// 로또 6/45 구매 페이지 URL (실제 구매 팝업)
	buyPageURL := "https://el.dhlottery.co.kr/game/TotalGame.jsp?LottoId=LO40"

	// 메인 페이지 먼저 방문 (세션 유지)
	_, err := d.client.Get("https://www.dhlottery.co.kr/")
	if err != nil {
		return fmt.Errorf("메인 페이지 접속 실패: %w", err)
	}

	time.Sleep(1 * time.Second)

	// 로또 구매 페이지 접속
	req, err := http.NewRequest("GET", buyPageURL, nil)
	if err != nil {
		return fmt.Errorf("구매 페이지 요청 생성 실패: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.dhlottery.co.kr/")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("구매 페이지 접속 실패: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	log.Printf("구매 페이지 상태 코드: %d\n", resp.StatusCode)
	log.Printf("구매 페이지 URL: %s\n", resp.Request.URL.String())
	log.Printf("페이지 내용 길이: %d bytes\n", len(bodyStr))

	// 구매 페이지 확인 (더 유연한 체크)
	if resp.StatusCode == 200 && len(bodyStr) > 1000 {
		log.Println("✅ 로또 6/45 구매 페이지 접근 성공!")

		// 페이지 내용 일부 출력 (디버깅용)
		if strings.Contains(bodyStr, "LO40") ||
			strings.Contains(bodyStr, "자동번호발급") ||
			strings.Contains(bodyStr, "로또") ||
			strings.Contains(bodyStr, "복권") {
			log.Println("   → 로또 구매 페이지로 확인됨")
		}

		return nil
	}

	// 실패 시 페이지 내용 일부 출력
	log.Printf("페이지 내용 샘플 (처음 500자):\n%s\n", bodyStr[:min(500, len(bodyStr))])

	return fmt.Errorf("구매 페이지 확인 실패: 예상하지 못한 페이지입니다")
}

// LottoGameInfo는 로또 구매에 필요한 정보를 담는 구조체입니다
type LottoGameInfo struct {
	CurRound         string
	RoundDrawDate    string
	WamtPayTlmtEndDt string
	MoneyBalance     string
}

// BuyLottoAuto는 로또를 자동으로 구매합니다
func (d *DhLottery) BuyLottoAuto(quantity int) error {
	// 1단계: 실제 로또 구매 페이지 접근 (iframe 내부 페이지)
	buyPageURL := "https://ol.dhlottery.co.kr/olotto/game/game645.do"

	req, err := http.NewRequest("GET", buyPageURL, nil)
	if err != nil {
		return fmt.Errorf("구매 페이지 요청 생성 실패: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://el.dhlottery.co.kr/game/TotalGame.jsp?LottoId=LO40")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("구매 페이지 접속 실패: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// 2단계: HTML 파싱하여 구매에 필요한 정보 추출
	log.Println("2단계: 구매 정보 추출 중...")

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return fmt.Errorf("HTML 파싱 실패: %w", err)
	}

	gameInfo := LottoGameInfo{}

	// 현재 회차
	doc.Find("#curRound").Each(func(i int, s *goquery.Selection) {
		gameInfo.CurRound = strings.TrimSpace(s.Text())
	})

	// 추첨일
	doc.Find("#ROUND_DRAW_DATE").Each(func(i int, s *goquery.Selection) {
		if val, exists := s.Attr("value"); exists {
			gameInfo.RoundDrawDate = val
		}
	})

	// 지급기한
	doc.Find("#WAMT_PAY_TLMT_END_DT").Each(func(i int, s *goquery.Selection) {
		if val, exists := s.Attr("value"); exists {
			gameInfo.WamtPayTlmtEndDt = val
		}
	})

	// 예치금 잔액
	doc.Find("#moneyBalance").Each(func(i int, s *goquery.Selection) {
		gameInfo.MoneyBalance = strings.TrimSpace(s.Text())
	})

	// 대기열 체크
	directIP, err := d.checkReadySocket()
	if err != nil {
		return fmt.Errorf("대기열 확인 실패: %w", err)
	}

	// 실제 구매 요청
	result, err := d.executeBuy(gameInfo, directIP, quantity)
	if err != nil {
		return fmt.Errorf("구매 실패: %w", err)
	}

	// 구매 결과 출력
	d.printBuyResult(result)

	return nil
}

// BuyLottoAutoWithResult는 로또를 자동으로 구매하고 텔레그램용 메시지를 반환합니다
func (d *DhLottery) BuyLottoAutoWithResult(quantity int) (map[string]interface{}, string, error) {
	// 실제 로또 구매 페이지 접근
	buyPageURL := "https://ol.dhlottery.co.kr/olotto/game/game645.do"

	req, err := http.NewRequest("GET", buyPageURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("구매 페이지 요청 생성 실패: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://el.dhlottery.co.kr/game/TotalGame.jsp?LottoId=LO40")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := d.client.Do(req)
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

	// 3단계: 대기열 체크
	log.Println("3단계: 구매 대기열 확인 중...")

	directIP, err := d.checkReadySocket()
	if err != nil {
		return nil, "", fmt.Errorf("대기열 확인 실패: %w", err)
	}

	if directIP != "" {
		log.Printf("   → 대기열 없음, 즉시 구매 가능 (IP: %s)\n", directIP)
	}

	// 4단계: 실제 구매 요청
	log.Println("4단계: 로또 구매 요청 중...")
	log.Printf("   💰 구매 금액: %d원\n", quantity*1000)

	result, err := d.executeBuy(gameInfo, directIP, quantity)
	if err != nil {
		return nil, "", fmt.Errorf("구매 실패: %w", err)
	}

	// 5단계: 텔레그램용 메시지 생성
	telegramMsg := d.formatTelegramMessage(result, quantity)

	return result, telegramMsg, nil
}

// formatTelegramMessage는 구매 결과를 텔레그램 메시지로 포맷합니다
func (d *DhLottery) formatTelegramMessage(result map[string]interface{}, quantity int) string {
	// 로그인 체크
	if loginYn, ok := result["loginYn"].(string); ok && loginYn == "N" {
		return "❌ <b>로그인 세션 만료</b>\n\n다시 로그인해주세요."
	}

	// 기기 제한 체크
	if isAllowed, ok := result["isAllowed"].(string); ok && isAllowed == "N" {
		return "❌ <b>구매 실패</b>\n\n모바일에서는 구매할 수 없습니다."
	}

	// 판매시간 체크
	if checkTime, ok := result["checkOltSaleTime"].(bool); ok && !checkTime {
		return "❌ <b>구매 실패</b>\n\n현재 판매 시간이 아닙니다."
	}

	// 결과 확인
	if resultData, ok := result["result"].(map[string]interface{}); ok {
		resultCode := resultData["resultCode"].(string)

		if resultCode == "100" {
			// 구매 성공
			msg := "✅ <b>로또 구매 성공!</b>\n\n"
			msg += fmt.Sprintf("💰 구매 금액: <b>%s원</b>\n", formatMoney(quantity*1000))
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

			msg := "❌ <b>구매 실패</b>\n\n"
			msg += fmt.Sprintf("사유: %s\n\n", resultMsg)

			if strings.Contains(resultMsg, "한도") || strings.Contains(resultMsg, "5000") {
				msg += "💡 이번 회차에 이미 최대 한도(5,000원)를 구매하셨습니다."
			} else if strings.Contains(resultMsg, "예치금") || strings.Contains(resultMsg, "잔액") {
				msg += "💡 예치금이 부족합니다. 충전 후 다시 시도해주세요."
			}

			return msg
		}
	}

	return "❌ 구매 결과를 확인할 수 없습니다."
}

// checkReadySocket은 구매 대기열을 확인합니다
func (d *DhLottery) checkReadySocket() (string, error) {
	readyURL := "https://ol.dhlottery.co.kr/olotto/game/egovUserReadySocket.json"

	req, err := http.NewRequest("POST", readyURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://ol.dhlottery.co.kr/olotto/game/game645.do")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := d.client.Do(req)
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
func (d *DhLottery) executeBuy(gameInfo LottoGameInfo, directIP string, quantity int) (map[string]interface{}, error) {
	buyURL := "https://ol.dhlottery.co.kr/olotto/game/execBuy.do"

	// 자동 구매 파라미터 생성 (genType: "0" = 자동)
	alpabet := []string{"A", "B", "C", "D", "E"}
	param := make([]map[string]interface{}, quantity)

	for i := 0; i < quantity; i++ {
		param[i] = map[string]interface{}{
			"genType":          "0", // 자동
			"arrGameChoiceNum": nil, // 자동이므로 null
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

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var buyResult map[string]interface{}
	if err := json.Unmarshal(body, &buyResult); err != nil {
		return nil, fmt.Errorf("구매 응답 파싱 실패: %w", err)
	}

	return buyResult, nil
}

// printBuyResult는 구매 결과를 출력합니다
func (d *DhLottery) printBuyResult(result map[string]interface{}) {
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
				log.Printf("    구매 게임 수: %d 게임 (총 %,d원)\n", len(arrGameChoiceNum), len(arrGameChoiceNum)*1000)
				log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
				log.Println()

				alpabet := []string{"A", "B", "C", "D", "E"}

				for i, numData := range arrGameChoiceNum {
					numStr := numData.(string)
					// 마지막 문자는 genType (자동/수동 구분)
					genType := numStr[len(numStr)-1:]
					numStr = numStr[:len(numStr)-1]

					// 번호 파싱
					numbers := strings.Split(numStr, "|")

					// 게임 레이블
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

					// 번호 출력
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

			// 일반적인 실패 원인 안내
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

// PurchaseLotto는 실제로 로또를 구매합니다 (최종 확인)
func (d *DhLottery) PurchaseLotto() error {
	log.Println("⚠️  주의: 실제 구매 기능은 신중하게 사용해야 합니다!")
	log.Println("이 기능은 실제 금액이 결제됩니다.")

	// TODO: 실제 구매 로직 구현
	// 현재는 안전을 위해 구현하지 않음

	return fmt.Errorf("실제 구매 기능은 아직 구현되지 않았습니다 (안전을 위함)")
}

// GetLoginStatus는 현재 로그인 상태를 반환합니다
func (d *DhLottery) GetLoginStatus() (bool, error) {
	resp, err := d.client.Get("https://www.dhlottery.co.kr/")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	return strings.Contains(bodyStr, "로그아웃"), nil
}

// PrintConfig는 설정 정보를 출력합니다 (보안상 비밀번호는 마스킹)
func (c *Config) PrintConfig() {
	maskedPw := ""
	if len(c.Password) > 0 {
		maskedPw = strings.Repeat("*", len(c.Password))
	}

	configJSON, _ := json.MarshalIndent(map[string]string{
		"UserID":   c.UserID,
		"Password": maskedPw,
	}, "", "  ")

	log.Println("=== 설정 정보 ===")
	log.Println(string(configJSON))
}

// LoadConfigFromEnv는 환경변수에서 설정을 로드합니다
func LoadConfigFromEnv() (Config, error) {
	userID := os.Getenv("DH_LOTTERY_ID")
	password := os.Getenv("DH_LOTTERY_PW")
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramChatID := os.Getenv("TELEGRAM_CHAT_ID")

	if userID == "" || password == "" {
		return Config{}, fmt.Errorf("환경변수가 설정되지 않았습니다 (DH_LOTTERY_ID, DH_LOTTERY_PW)")
	}

	return Config{
		UserID:           userID,
		Password:         password,
		TelegramBotToken: telegramToken,
		TelegramChatID:   telegramChatID,
	}, nil
}

// LoadConfigFromFile은 파일에서 설정을 로드합니다
func LoadConfigFromFile(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("설정 파일 읽기 실패: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("설정 파일 파싱 실패: %w", err)
	}

	if config.UserID == "" || config.Password == "" {
		return Config{}, fmt.Errorf("설정 파일에 필수 정보가 없습니다")
	}

	return config, nil
}

// LoadConfigInteractive는 사용자 입력으로 설정을 로드합니다
func LoadConfigInteractive() (Config, error) {
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
		UserID:   userID,
		Password: password,
	}, nil
}

func main() {
	// 커맨드 라인 플래그
	serviceMode := flag.Bool("service", false, "서비스 모드로 실행 (스케줄러 활성화)")
	onceMode := flag.Bool("once", false, "즉시 1회 구매 실행")
	checkBalanceMode := flag.Bool("check", false, "예치금만 확인")
	flag.Parse()

	log.Println("==============================================")
	log.Println("   동행복권 로또 자동 구매 프로그램 v2.0")
	log.Println("==============================================")
	log.Println()

	// 설정 로드
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("❌ 설정 로드 실패: %v", err)
	}

	config.PrintConfig()
	log.Println()

	// 텔레그램 봇 초기화
	var telegramBot *TelegramBot
	if config.TelegramBotToken != "" && config.TelegramChatID != "" {
		telegramBot = NewTelegramBot(config.TelegramBotToken, config.TelegramChatID)
		log.Println("✅ 텔레그램 봇 활성화됨")
	} else {
		log.Println("⚠️  텔레그램 봇 설정이 없습니다 (알림 비활성화)")
	}

	// 서비스 모드
	if *serviceMode {
		log.Println("🔄 서비스 모드로 시작합니다...")
		runServiceMode(config, telegramBot)
		return
	}

	// 예치금 확인 모드
	if *checkBalanceMode {
		log.Println("💰 예치금 확인 모드")
		checkBalanceTask(config, telegramBot)
		return
	}

	// 1회 구매 모드 (기존 동작)
	if *onceMode {
		log.Println("🎯 1회 구매 모드")
		buyLottoTask(config, telegramBot)
		return
	}

	// 기본: 1회 구매 모드
	log.Println("🎯 기본 모드: 1회 구매 실행")
	buyLottoTask(config, telegramBot)
}

// loadConfig는 설정을 로드합니다
func loadConfig() (Config, error) {
	// 1. 환경변수에서 로드 시도
	config, err := LoadConfigFromEnv()
	if err == nil {
		return config, nil
	}

	log.Printf("환경변수 로드 실패: %v\n", err)

	// 2. 설정 파일에서 로드 시도
	config, err = LoadConfigFromFile("config.json")
	if err == nil {
		return config, nil
	}

	log.Printf("설정 파일 로드 실패: %v\n", err)

	// 3. 대화형 입력
	log.Println("설정 정보를 입력해주세요:")
	return LoadConfigInteractive()
}

// runServiceMode는 서비스 모드를 실행합니다
func runServiceMode(config Config, telegramBot *TelegramBot) {
	log.Println("┌────────────────────────────────────────┐")
	log.Println("│     스케줄러 설정                      │")
	log.Println("├────────────────────────────────────────┤")
	log.Println("│ 매주 월요일 13:00 - 예치금 확인        │")
	log.Println("│ 매주 월요일 19:00 - 로또 자동 구매     │")
	log.Println("└────────────────────────────────────────┘")
	log.Println()

	// Cron 스케줄러 생성 (Asia/Seoul 타임존)
	location, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		log.Fatalf("❌ 타임존 로드 실패: %v", err)
	}

	c := cron.New(cron.WithLocation(location))

	// 매주 월요일 13:00 - 예치금 확인
	c.AddFunc("0 13 * * 1", func() {
		log.Println("⏰ [스케줄] 예치금 확인 작업 시작")
		checkBalanceTask(config, telegramBot)
	})

	// 매주 월요일 19:00 - 로또 구매
	c.AddFunc("0 19 * * 1", func() {
		log.Println("⏰ [스케줄] 로또 구매 작업 시작")
		buyLottoTask(config, telegramBot)
	})

	// 테스트용: 매분 실행 (주석 처리)
	// c.AddFunc("* * * * *", func() {
	// 	log.Println("⏰ [테스트] 1분마다 실행")
	// })

	// 스케줄러 시작
	c.Start()
	log.Println("✅ 스케줄러가 시작되었습니다")
	log.Println("💡 프로그램을 중지하려면 Ctrl+C를 누르세요")
	log.Println()

	// 다음 실행 시간 표시
	entries := c.Entries()
	for _, entry := range entries {
		log.Printf("   다음 실행: %s\n", entry.Next.Format("2006-01-02 15:04:05 (Mon)"))
	}
	log.Println()

	// 시그널 대기 (Ctrl+C로 종료)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println()
	log.Println("🛑 종료 신호를 받았습니다. 스케줄러를 중지합니다...")
	c.Stop()
	log.Println("✅ 프로그램이 정상적으로 종료되었습니다")
}

// checkBalanceTask는 예치금 확인 작업을 수행합니다
func checkBalanceTask(config Config, telegramBot *TelegramBot) {
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("          💰 예치금 확인 작업")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 클라이언트 생성
	client, err := NewDhLottery(config)
	if err != nil {
		log.Printf("❌ 클라이언트 생성 실패: %v\n", err)
		return
	}

	// 로그인
	if err := client.Login(); err != nil {
		log.Printf("❌ 로그인 실패: %v\n", err)
		if telegramBot != nil {
			telegramBot.SendMessage(fmt.Sprintf("❌ <b>동행복권 로그인 실패</b>\n\n%v", err))
		}
		return
	}

	// 예치금 확인
	balance, err := client.CheckBalance()
	if err != nil {
		log.Printf("❌ 예치금 확인 실패: %v\n", err)
		if telegramBot != nil {
			telegramBot.SendMessage(fmt.Sprintf("❌ <b>예치금 확인 실패</b>\n\n%v", err))
		}
		return
	}

	// 예치금이 10,000원 미만인 경우 알림
	if balance < 10000 {
		log.Printf("⚠️  예치금 부족: %s원 (10,000원 미만)\n", formatMoney(balance))

		if telegramBot != nil {
			message := fmt.Sprintf(
				"⚠️ <b>예치금 부족 알림</b>\n\n"+
					"현재 예치금: <b>%s원</b>\n"+
					"기준 금액: 10,000원\n\n"+
					"💡 예치금을 충전해주세요!",
				formatMoney(balance),
			)
			telegramBot.SendMessage(message)
		}
	} else {
		log.Printf("✅ 예치금 충분: %s원\n", formatMoney(balance))
		// 10,000원 이상이면 텔레그램 알림 보내지 않음
	}

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println()
}

// buyLottoTask는 로또 구매 작업을 수행합니다
func buyLottoTask(config Config, telegramBot *TelegramBot) {
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("          🎱 로또 구매 작업")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 클라이언트 생성
	client, err := NewDhLottery(config)
	if err != nil {
		log.Printf("❌ 클라이언트 생성 실패: %v\n", err)
		if telegramBot != nil {
			telegramBot.SendMessage(fmt.Sprintf("❌ <b>로또 구매 실패</b>\n\n클라이언트 생성 오류: %v", err))
		}
		return
	}

	// 로그인
	log.Println("=== 로그인 시작 ===")
	if err := client.Login(); err != nil {
		log.Printf("❌ 로그인 실패: %v\n", err)
		if telegramBot != nil {
			telegramBot.SendMessage(fmt.Sprintf("❌ <b>로또 구매 실패</b>\n\n로그인 오류: %v", err))
		}
		return
	}

	// 구매 페이지 접근
	log.Println()
	log.Println("=== 로또 6/45 구매 페이지 접근 ===")
	if err := client.NavigateToLottoBuyPage(); err != nil {
		log.Printf("❌ 구매 페이지 접근 실패: %v\n", err)
		if telegramBot != nil {
			telegramBot.SendMessage(fmt.Sprintf("❌ <b>로또 구매 실패</b>\n\n페이지 접근 오류: %v", err))
		}
		return
	}

	// 로또 구매 (5게임)
	log.Println()
	log.Println("=== 로또 자동 구매 (5게임) ===")
	result, resultMsg, err := client.BuyLottoAutoWithResult(5)
	if err != nil {
		log.Printf("❌ 구매 실패: %v\n", err)
		if telegramBot != nil {
			telegramBot.SendMessage(fmt.Sprintf("❌ <b>로또 구매 실패</b>\n\n%v", err))
		}
		return
	}

	// 구매 결과 출력
	client.printBuyResult(result)

	// 텔레그램 알림 전송
	if telegramBot != nil {
		telegramBot.SendMessage(resultMsg)
	}

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println()
}
