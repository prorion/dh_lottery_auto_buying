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

	// 진입 경로: www 메인 → 로또 6/45 소개 (페이지 구조 변경 대응)
	if err := c.EnsureLottoEntryPath(); err != nil {
		log.Printf("   ⚠️  진입 경로 방문 실패(무시하고 진행): %v\n", err)
	}

	// 로또 구매 페이지에서 예치금을 확인 (가장 안정적)
	buyPageURL := "https://ol.dhlottery.co.kr/olotto/game/game645.do"

	req, err := http.NewRequest("GET", buyPageURL, nil)
	if err != nil {
		return 0, fmt.Errorf("구매 페이지 요청 생성 실패: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", LottoBuyRefURL)
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

// NavigateToLottoBuyPage는 로또 6/45 구매 페이지 진입 경로를 거칩니다.
// 동행복권 구조: 메인 → 추첨식 복권 → 로또 6/45 소개(바로구매) 경로를 시뮬레이션합니다.
func (c *Client) NavigateToLottoBuyPage() error {
	log.Println("로또 6/45 구매 페이지 진입 경로 이동 중...")

	if err := c.EnsureLottoEntryPath(); err != nil {
		return fmt.Errorf("진입 경로 방문 실패: %w", err)
	}

	log.Println("✅ 로또 6/45 진입 경로 완료 (메인 → 로또 6/45 소개)")
	return nil
}
