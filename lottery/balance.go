package lottery

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// userMndpData는 /mypage/selectUserMndp.do 응답의 예치금 정보입니다
type userMndpData struct {
	PntDpstAmt   int `json:"pntDpstAmt"`   // 포인트 입금
	PntTkmnyAmt  int `json:"pntTkmnyAmt"`  // 포인트 출금
	NcsblDpstAmt int `json:"ncsblDpstAmt"` // 비현금성 입금
	NcsblTkmnyAmt int `json:"ncsblTkmnyAmt"` // 비현금성 출금
	CsblDpstAmt  int `json:"csblDpstAmt"`  // 현금성 입금
	CsblTkmnyAmt int `json:"csblTkmnyAmt"` // 현금성 출금
	CrntEntrsAmt int `json:"crntEntrsAmt"` // 구매가능금액
	RsvtOrdrAmt  int `json:"rsvtOrdrAmt"`  // 예약구매금액
	DawAplyAmt   int `json:"dawAplyAmt"`   // 출금신청중금액
	FeeAmt       int `json:"feeAmt"`       // 수수료
}

// CheckBalance는 예치금 잔액(구매가능금액)을 확인합니다.
// 동행복권 사이트 리뉴얼 이후 HTML 파싱이 깨지므로 JSON API를 우선 사용합니다.
func (c *Client) CheckBalance() (int, error) {
	log.Println("예치금 확인 중...")

	// 1순위: 마이페이지 JSON API (사이트 리뉴얼 대응)
	balance, err := c.checkBalanceFromAPI()
	if err == nil {
		log.Printf("✅ 예치금 확인 완료: %s원 (구매가능금액)\n", FormatMoney(balance))
		return balance, nil
	}
	log.Printf("   ⚠️  JSON API 조회 실패: %v\n", err)
	log.Println("   → HTML 파싱으로 폴백 시도 중...")

	// 2순위: 기존 HTML 파싱 폴백
	balance, err = c.checkBalanceFromHTML()
	if err != nil {
		return 0, err
	}

	log.Printf("✅ 예치금 확인 완료: %s원\n", FormatMoney(balance))
	return balance, nil
}

// checkBalanceFromAPI는 /mypage/selectUserMndp.do JSON API로 구매가능금액을 조회합니다.
func (c *Client) checkBalanceFromAPI() (int, error) {
	// Referer용 마이페이지 선방문
	homeReq, err := http.NewRequest("GET", "https://www.dhlottery.co.kr/mypage/home", nil)
	if err != nil {
		return 0, fmt.Errorf("마이페이지 요청 생성 실패: %w", err)
	}
	homeReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	homeReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	homeResp, err := c.httpClient.Do(homeReq)
	if err != nil {
		return 0, fmt.Errorf("마이페이지 접속 실패: %w", err)
	}
	homeResp.Body.Close()

	apiURL := "https://www.dhlottery.co.kr/mypage/selectUserMndp.do"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("예치금 API 요청 생성 실패: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", "https://www.dhlottery.co.kr/mypage/home")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("예치금 API 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("예치금 API 응답 코드: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "json") {
		return 0, fmt.Errorf("예치금 API가 JSON이 아님 (Content-Type: %s, 세션 만료 가능)", contentType)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("예치금 API 응답 읽기 실패: %w", err)
	}

	var apiResp struct {
		Data struct {
			UserMndp userMndpData `json:"userMndp"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return 0, fmt.Errorf("예치금 API 파싱 실패: %w", err)
	}

	mndp := apiResp.Data.UserMndp

	// 구매가능금액 (구매 판단에 사용)
	purchasable := mndp.CrntEntrsAmt

	// 참고용 총예치금 (웹사이트 JS와 동일 계산)
	total := (mndp.PntDpstAmt - mndp.PntTkmnyAmt) +
		(mndp.NcsblDpstAmt - mndp.NcsblTkmnyAmt) +
		(mndp.CsblDpstAmt - mndp.CsblTkmnyAmt)

	log.Printf("   → 총예치금: %s원 / 구매가능: %s원 / 예약구매: %s원 / 출금신청중: %s원\n",
		FormatMoney(total),
		FormatMoney(purchasable),
		FormatMoney(mndp.RsvtOrdrAmt),
		FormatMoney(mndp.DawAplyAmt),
	)

	return purchasable, nil
}

// checkBalanceFromHTML은 기존 HTML 파싱 방식입니다 (폴백용).
func (c *Client) checkBalanceFromHTML() (int, error) {
	if err := c.EnsureLottoEntryPath(); err != nil {
		log.Printf("   ⚠️  진입 경로 방문 실패(무시하고 진행): %v\n", err)
	}

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

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return 0, fmt.Errorf("HTML 파싱 실패: %w", err)
	}

	balance := 0

	doc.Find("#moneyBalance").Each(func(i int, s *goquery.Selection) {
		balanceText := strings.TrimSpace(s.Text())
		balanceText = strings.ReplaceAll(balanceText, ",", "")
		balanceText = strings.ReplaceAll(balanceText, "원", "")
		balanceText = strings.TrimSpace(balanceText)
		if balanceText != "" {
			fmt.Sscanf(balanceText, "%d", &balance)
		}
	})

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
