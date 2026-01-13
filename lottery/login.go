package lottery

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
)

// encryptRSA는 문자열을 RSA로 암호화합니다
func encryptRSA(plaintext, modulusHex, exponentHex string) (string, error) {
	// 16진수 modulus를 big.Int로 변환
	modulus := new(big.Int)
	modulus.SetString(modulusHex, 16)

	// 16진수 exponent를 int로 변환
	exponent := new(big.Int)
	exponent.SetString(exponentHex, 16)

	// RSA 공개키 생성
	pubKey := &rsa.PublicKey{
		N: modulus,
		E: int(exponent.Int64()),
	}

	// RSA PKCS1v15로 암호화
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, []byte(plaintext))
	if err != nil {
		return "", fmt.Errorf("RSA 암호화 실패: %w", err)
	}

	// 16진수 문자열로 변환
	return hex.EncodeToString(ciphertext), nil
}

// Login은 동행복권 사이트에 로그인합니다
func (c *Client) Login() error {
	log.Println("1단계: 로그인 페이지 접속 중...")

	loginURL := "https://www.dhlottery.co.kr/login"

	// 로그인 페이지 접속 (쿠키 획득)
	resp, err := c.httpClient.Get(loginURL)
	if err != nil {
		return fmt.Errorf("로그인 페이지 접속 실패: %w", err)
	}
	defer resp.Body.Close()

	log.Println("2단계: RSA 공개키 가져오는 중...")

	// RSA 공개키 가져오기
	rsaURL := "https://www.dhlottery.co.kr/login/selectRsaModulus.do"
	rsaResp, err := c.httpClient.Get(rsaURL)
	if err != nil {
		return fmt.Errorf("RSA 공개키 가져오기 실패: %w", err)
	}
	defer rsaResp.Body.Close()

	rsaBody, _ := io.ReadAll(rsaResp.Body)

	var rsaData struct {
		Data RSAModulusResponse `json:"data"`
	}

	if err := json.Unmarshal(rsaBody, &rsaData); err != nil {
		return fmt.Errorf("RSA 공개키 파싱 실패: %w", err)
	}

	log.Printf("   → RSA Modulus: %s...\n", rsaData.Data.RsaModulus[:20])
	log.Printf("   → Public Exponent: %s\n", rsaData.Data.PublicExponent)

	log.Println("3단계: 아이디/비밀번호 암호화 중...")

	// 아이디와 비밀번호를 RSA로 암호화
	encryptedUserID, err := encryptRSA(c.UserID, rsaData.Data.RsaModulus, rsaData.Data.PublicExponent)
	if err != nil {
		return fmt.Errorf("아이디 암호화 실패: %w", err)
	}

	encryptedPassword, err := encryptRSA(c.Password, rsaData.Data.RsaModulus, rsaData.Data.PublicExponent)
	if err != nil {
		return fmt.Errorf("비밀번호 암호화 실패: %w", err)
	}

	log.Println("4단계: 로그인 요청 전송 중...")

	// 로그인 폼 데이터 준비
	formData := url.Values{}
	formData.Set("userId", encryptedUserID)
	formData.Set("userPswdEncn", encryptedPassword)

	// POST 요청 생성
	loginActionURL := "https://www.dhlottery.co.kr/login/securityLoginCheck.do"
	req, err := http.NewRequest("POST", loginActionURL, strings.NewReader(formData.Encode()))
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
	loginResp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("로그인 요청 실패: %w", err)
	}
	defer loginResp.Body.Close()

	// 응답 확인
	body, _ := io.ReadAll(loginResp.Body)
	bodyStr := string(body)

	log.Printf("   → 응답 상태 코드: %d\n", loginResp.StatusCode)
	log.Printf("   → 응답 URL: %s\n", loginResp.Request.URL.String())

	// 로그인 실패 체크
	if strings.Contains(bodyStr, "아이디 또는 비밀번호를 확인해주세요") ||
		strings.Contains(bodyStr, "로그인에 실패") ||
		strings.Contains(bodyStr, "loginFail") {
		return fmt.Errorf("로그인 실패: 아이디 또는 비밀번호가 올바르지 않습니다")
	}

	// 로그인 성공 체크
	isLoggedIn := false

	// 방법 1: URL 체크 (성공 시 리다이렉트)
	if strings.Contains(loginResp.Request.URL.String(), "main") ||
		strings.Contains(loginResp.Request.URL.String(), "index") ||
		loginResp.Request.URL.Path == "/" {
		isLoggedIn = true
	}

	// 방법 2: 로그아웃 링크 존재 확인
	if strings.Contains(bodyStr, "로그아웃") || strings.Contains(bodyStr, "logout") {
		isLoggedIn = true
	}

	// 방법 3: 세션 쿠키 확인
	cookies := c.httpClient.Jar.Cookies(loginResp.Request.URL)
	for _, cookie := range cookies {
		if cookie.Name == "JSESSIONID" && cookie.Value != "" {
			isLoggedIn = true
			log.Printf("   → 세션 쿠키 획득: %s\n", cookie.Value[:20]+"...")
			break
		}
	}

	if !isLoggedIn {
		// 디버깅을 위해 응답 일부 출력
		log.Printf("응답 내용 샘플 (처음 500자):\n%s\n", bodyStr[:min(500, len(bodyStr))])
		return fmt.Errorf("로그인 실패: 로그인 확인 실패")
	}

	log.Println("✅ 로그인 완료! 세션이 정상적으로 생성되었습니다")
	return nil
}
