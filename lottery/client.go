package lottery

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"time"

	"golang.org/x/net/publicsuffix"
)

// Client는 동행복권 클라이언트 구조체입니다
type Client struct {
	httpClient *http.Client
	UserID     string
	Password   string
}

// NewClient는 새로운 동행복권 클라이언트를 생성합니다
func NewClient(userID, password string) (*Client, error) {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, fmt.Errorf("쿠키 저장소 생성 실패: %w", err)
	}

	httpClient := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("리다이렉트가 너무 많습니다")
			}
			return nil
		},
	}

	return &Client{
		httpClient: httpClient,
		UserID:     userID,
		Password:   password,
	}, nil
}

// GetHTTPClient는 HTTP 클라이언트를 반환합니다
func (c *Client) GetHTTPClient() *http.Client {
	return c.httpClient
}

// FormatMoney는 숫자를 천 단위 구분자가 있는 문자열로 변환합니다
func FormatMoney(amount int) string {
	if amount < 1000 {
		return fmt.Sprintf("%d", amount)
	}

	str := fmt.Sprintf("%d", amount)
	result := ""
	for i, char := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(char)
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

// LottoEntryURLs는 로또 구매/예치금 페이지 접근 전에 거쳐야 할 진입 URL입니다.
// 동행복권이 '추첨식 복권 > 로또 6/45 바로구매' 경로로 들어온 요청만 허용할 수 있어 사용합니다.
const (
	LottoMainURL   = "https://www.dhlottery.co.kr/"
	LottoIntroURL  = "https://www.dhlottery.co.kr/lt645/intro"
	LottoBuyRefURL = "https://www.dhlottery.co.kr/lt645/intro"
)

// EnsureLottoEntryPath는 로그인 후 로또 구매/예치금 페이지(ol 도메인) 접근 전에
// www 메인 → 로또 6/45 소개 페이지를 순서대로 방문하여 진입 경로를 맞춥니다.
func (c *Client) EnsureLottoEntryPath() error {
	reqMain, err := http.NewRequest("GET", LottoMainURL, nil)
	if err != nil {
		return err
	}
	reqMain.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	reqMain.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	respMain, err := c.httpClient.Do(reqMain)
	if err != nil {
		return fmt.Errorf("메인 페이지 방문 실패: %w", err)
	}
	respMain.Body.Close()

	reqIntro, err := http.NewRequest("GET", LottoIntroURL, nil)
	if err != nil {
		return err
	}
	reqIntro.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	reqIntro.Header.Set("Referer", LottoMainURL)
	reqIntro.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	respIntro, err := c.httpClient.Do(reqIntro)
	if err != nil {
		return fmt.Errorf("로또 6/45 소개 페이지 방문 실패: %w", err)
	}
	respIntro.Body.Close()

	return nil
}
