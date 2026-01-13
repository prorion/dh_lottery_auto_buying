package lottery

// LottoGameInfo는 로또 구매에 필요한 정보를 담는 구조체입니다
type LottoGameInfo struct {
	CurRound         string
	RoundDrawDate    string
	WamtPayTlmtEndDt string
	MoneyBalance     string
}

// RSAModulusResponse는 RSA 공개키 응답 구조체입니다
type RSAModulusResponse struct {
	RsaModulus     string `json:"rsaModulus"`
	PublicExponent string `json:"publicExponent"`
}
