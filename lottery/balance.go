package lottery

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// CheckBalance는 예치금 잔액을 확인합니다
func (c *Client) CheckBalance() (int, error) {
	log.Println("예치금 확인 중...")

	// 로또 구매 페이지에서 예치금을 확인 (가장 안정적)
	buyPageURL := "https://ol.dhlottery.co.kr/olotto/game/game645.do"

	req, err := http.NewRequest("GET", buyPageURL, nil)
	if err != nil {
		return 0, fmt.Errorf("구매 페이지 요청 생성 실패: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.dhlottery.co.kr/")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("구매 페이지 접속 실패: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// HTML 파싱
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return 0, fmt.Errorf("HTML 파싱 실패: %w", err)
	}

	// 예치금 요소 찾기
	balance := 0

	// 방법 1: #moneyBalance (구매 페이지의 예치금 표시)
	doc.Find("#moneyBalance").Each(func(i int, s *goquery.Selection) {
		balanceText := strings.TrimSpace(s.Text())
		balanceText = strings.ReplaceAll(balanceText, ",", "")
		balanceText = strings.ReplaceAll(balanceText, "원", "")
		balanceText = strings.TrimSpace(balanceText)
		if balanceText != "" {
			fmt.Sscanf(balanceText, "%d", &balance)
		}
	})

	// 방법 2: input#moneyBalance (hidden 필드일 수도 있음)
	if balance == 0 {
		doc.Find("input#moneyBalance").Each(func(i int, s *goquery.Selection) {
			if val, exists := s.Attr("value"); exists {
				balanceText := strings.ReplaceAll(val, ",", "")
				balanceText = strings.ReplaceAll(balanceText, "원", "")
				balanceText = strings.TrimSpace(balanceText)
				if balanceText != "" {
					fmt.Sscanf(balanceText, "%d", &balance)
				}
			}
		})
	}

	// 방법 3: 마이페이지 시도 (폴백)
	if balance == 0 {
		log.Println("   → 구매 페이지에서 예치금을 찾지 못했습니다. 마이페이지 시도 중...")

		mypageResp, err := c.httpClient.Get("https://www.dhlottery.co.kr/mypage/home")
		if err == nil {
			defer mypageResp.Body.Close()
			mypageBody, _ := io.ReadAll(mypageResp.Body)
			mypageDoc, err := goquery.NewDocumentFromReader(strings.NewReader(string(mypageBody)))
			if err == nil {
				mypageDoc.Find("#totalAmt, span.deposit-num").Each(func(i int, s *goquery.Selection) {
					balanceText := strings.TrimSpace(s.Text())
					balanceText = strings.ReplaceAll(balanceText, ",", "")
					balanceText = strings.ReplaceAll(balanceText, "원", "")
					balanceText = strings.TrimSpace(balanceText)
					if balanceText != "" && balance == 0 {
						fmt.Sscanf(balanceText, "%d", &balance)
					}
				})
			}
		}
	}

	if balance == 0 {
		log.Println("   ⚠️  예치금 정보를 찾을 수 없습니다.")
		log.Printf("   페이지 내용 샘플 (처음 300자):\n%s\n", bodyStr[:min(300, len(bodyStr))])
	}

	log.Printf("✅ 예치금 확인 완료: %s원\n", FormatMoney(balance))
	return balance, nil
}

// NavigateToLottoBuyPage는 로또 6/45 구매 페이지로 이동합니다
func (c *Client) NavigateToLottoBuyPage() error {
	log.Println("로또 6/45 구매 페이지로 이동 중...")

	buyPageURL := "https://el.dhlottery.co.kr/game/TotalGame.jsp?LottoId=LO40"

	// 메인 페이지 먼저 방문 (세션 유지)
	_, err := c.httpClient.Get("https://www.dhlottery.co.kr/")
	if err != nil {
		return fmt.Errorf("메인 페이지 접속 실패: %w", err)
	}

	// 로또 구매 페이지 접속
	req, err := http.NewRequest("GET", buyPageURL, nil)
	if err != nil {
		return fmt.Errorf("구매 페이지 요청 생성 실패: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.dhlottery.co.kr/")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("구매 페이지 접속 실패: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	log.Printf("구매 페이지 상태 코드: %d\n", resp.StatusCode)
	log.Printf("구매 페이지 URL: %s\n", resp.Request.URL.String())
	log.Printf("페이지 내용 길이: %d bytes\n", len(bodyStr))

	// 구매 페이지 확인
	if resp.StatusCode == 200 && len(bodyStr) > 1000 {
		log.Println("✅ 로또 6/45 구매 페이지 접근 성공!")

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
