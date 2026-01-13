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
