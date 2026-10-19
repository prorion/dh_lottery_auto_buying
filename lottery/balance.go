package lottery

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// flexInt는 숫자 또는 "15,000" 같은 문자열로 내려오는 금액 필드를 모두 받습니다.
// Valid가 false면 응답에 해당 필드가 없었다는 뜻입니다 (0원과 구분).
type flexInt struct {
	Value int
	Valid bool
}

func (f *flexInt) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		return nil
	}
	s = strings.Trim(s, `"`)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSuffix(strings.TrimSpace(s), "원")
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("금액 파싱 실패 (%s): %w", string(b), err)
	}
	f.Value = int(v)
	f.Valid = true
	return nil
}

// userMndpData는 /mypage/selectUserMndp.do 응답의 예치금 정보입니다
type userMndpData struct {
	TotalAmt      flexInt `json:"totalAmt"`      // 총예치금 (응답에 있을 때)
	PntDpstAmt    flexInt `json:"pntDpstAmt"`    // 포인트 입금
	PntTkmnyAmt   flexInt `json:"pntTkmnyAmt"`   // 포인트 출금
	NcsblDpstAmt  flexInt `json:"ncsblDpstAmt"`  // 비현금성 입금
	NcsblTkmnyAmt flexInt `json:"ncsblTkmnyAmt"` // 비현금성 출금
	CsblDpstAmt   flexInt `json:"csblDpstAmt"`   // 현금성 입금
	CsblTkmnyAmt  flexInt `json:"csblTkmnyAmt"`  // 현금성 출금
	CrntEntrsAmt  flexInt `json:"crntEntrsAmt"`  // 구매가능금액
	RsvtOrdrAmt   flexInt `json:"rsvtOrdrAmt"`   // 예약구매금액
	DawAplyAmt    flexInt `json:"dawAplyAmt"`    // 출금신청중금액
	FeeAmt        flexInt `json:"feeAmt"`        // 수수료
}

// totalDeposit은 총예치금을 계산합니다. totalAmt가 있으면 그대로 쓰고,
// 없으면 웹사이트 JS와 동일하게 입금-출금 합으로 계산합니다.
func (m userMndpData) totalDeposit() (int, bool) {
	if m.TotalAmt.Valid {
		return m.TotalAmt.Value, true
	}
	parts := []flexInt{m.PntDpstAmt, m.PntTkmnyAmt, m.NcsblDpstAmt, m.NcsblTkmnyAmt, m.CsblDpstAmt, m.CsblTkmnyAmt}
	found := false
	for _, p := range parts {
		found = found || p.Valid
	}
	if !found {
		return 0, false
	}
	total := (m.PntDpstAmt.Value - m.PntTkmnyAmt.Value) +
		(m.NcsblDpstAmt.Value - m.NcsblTkmnyAmt.Value) +
		(m.CsblDpstAmt.Value - m.CsblTkmnyAmt.Value)
	return total, true
}

// resolveBalance는 응답에서 구매 판단에 쓸 예치금을 고릅니다.
// 금액 필드가 하나도 없으면 0원으로 처리하지 않고 에러를 돌려줍니다.
func (m userMndpData) resolveBalance() (int, error) {
	total, hasTotal := m.totalDeposit()

	if m.CrntEntrsAmt.Valid {
		// 구매가능금액이 0인데 총예치금이 남아 있으면 필드 의미가 바뀌었을 수 있으므로 총예치금 사용
		if m.CrntEntrsAmt.Value == 0 && hasTotal && total > 0 {
			log.Printf("   ⚠️  구매가능금액(crntEntrsAmt)=0, 총예치금=%s원 → 총예치금 기준으로 판단\n", FormatMoney(total))
			return total, nil
		}
		return m.CrntEntrsAmt.Value, nil
	}
	if hasTotal {
		log.Println("   ⚠️  응답에 구매가능금액(crntEntrsAmt)이 없어 총예치금 기준으로 판단")
		return total, nil
	}
	return 0, fmt.Errorf("예치금 API 응답에 금액 필드가 없음")
}

// CheckBalance는 예치금 잔액(구매가능금액)을 확인합니다.
// 동행복권 사이트 리뉴얼 이후 HTML 파싱이 깨지므로 JSON API를 우선 사용합니다.
func (c *Client) CheckBalance() (int, error) {
	log.Println("예치금 확인 중...")

	// 1순위: 마이페이지 JSON API (사이트 리뉴얼 대응)
	balance, apiErr := c.checkBalanceFromAPI()
	if apiErr == nil {
		log.Printf("✅ 예치금 확인 완료: %s원\n", FormatMoney(balance))
		return balance, nil
	}
	log.Printf("   ⚠️  JSON API 조회 실패: %v\n", apiErr)
	log.Println("   → HTML 파싱으로 폴백 시도 중...")

	// 2순위: 기존 HTML 파싱 폴백
	balance, err := c.checkBalanceFromHTML()
	if err != nil {
		return 0, fmt.Errorf("예치금 조회 실패 (API: %v / HTML: %v)", apiErr, err)
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

	apiURL := fmt.Sprintf("https://www.dhlottery.co.kr/mypage/selectUserMndp.do?_=%d", time.Now().UnixMilli())
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("예치금 API 요청 생성 실패: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("AJAX", "true")
	req.Header.Set("requestMenuUri", "/mypage/home")
	req.Header.Set("Referer", "https://www.dhlottery.co.kr/mypage/home")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("예치금 API 호출 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("예치금 API 응답 코드: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("예치금 API 응답 읽기 실패: %w", err)
	}

	balance, err := parseBalanceResponse(body)
	if err != nil {
		return 0, fmt.Errorf("%w (응답 URL: %s, Content-Type: %s, 응답: %s)",
			err, resp.Request.URL.Path, resp.Header.Get("Content-Type"), snippet(body, 300))
	}
	return balance, nil
}

// parseBalanceResponse는 selectUserMndp.do 응답 본문에서 예치금을 추출합니다.
// Content-Type과 무관하게 본문이 JSON이면 파싱합니다.
func parseBalanceResponse(body []byte) (int, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return 0, fmt.Errorf("예치금 API 응답이 비어 있음 (세션 만료 가능)")
	}
	if strings.HasPrefix(trimmed, "<") {
		return 0, fmt.Errorf("예치금 API가 HTML을 반환함 (로그인 세션 없음 가능)")
	}

	var apiResp struct {
		Data struct {
			UserMndp *userMndpData `json:"userMndp"`
		} `json:"data"`
		UserMndp *userMndpData `json:"userMndp"`
	}
	if err := json.Unmarshal([]byte(trimmed), &apiResp); err != nil {
		return 0, fmt.Errorf("예치금 API 파싱 실패: %w", err)
	}

	mndp := apiResp.Data.UserMndp
	if mndp == nil {
		mndp = apiResp.UserMndp
	}
	if mndp == nil {
		return 0, fmt.Errorf("예치금 API 응답에 userMndp가 없음")
	}

	total, _ := mndp.totalDeposit()
	log.Printf("   → 총예치금: %s원 / 구매가능: %s원 / 예약구매: %s원 / 출금신청중: %s원\n",
		FormatMoney(total),
		FormatMoney(mndp.CrntEntrsAmt.Value),
		FormatMoney(mndp.RsvtOrdrAmt.Value),
		FormatMoney(mndp.DawAplyAmt.Value),
	)

	return mndp.resolveBalance()
}

func snippet(b []byte, n int) string {
	s := strings.Join(strings.Fields(string(b)), " ")
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "..."
	}
	return s
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
		// 0원과 '못 찾음'을 구분하기 위해 에러로 돌려줌 (잘못된 0원 알림 방지)
		return 0, fmt.Errorf("HTML에서 예치금을 찾지 못함")
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
