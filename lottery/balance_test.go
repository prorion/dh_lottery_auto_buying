package lottery

import "testing"

func TestParseBalanceResponse(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		want    int
		wantErr bool
	}{
		{"구매가능금액", `{"data":{"userMndp":{"crntEntrsAmt":20000,"csblDpstAmt":4237000,"csblTkmnyAmt":4207000,"rsvtOrdrAmt":10000}}}`, 20000, false},
		{"문자열 금액", `{"data":{"userMndp":{"crntEntrsAmt":"15,000"}}}`, 15000, false},
		{"실수 금액", `{"data":{"userMndp":{"crntEntrsAmt":15000.0}}}`, 15000, false},
		{"totalAmt만 있음", `{"data":{"userMndp":{"totalAmt":15000}}}`, 15000, false},
		{"구매가능 0, 총예치금 있음", `{"data":{"userMndp":{"crntEntrsAmt":0,"totalAmt":15000}}}`, 15000, false},
		{"실제 0원", `{"data":{"userMndp":{"crntEntrsAmt":0,"csblDpstAmt":5000,"csblTkmnyAmt":5000}}}`, 0, false},
		{"userMndp 없음", `{"data":{}}`, 0, true},
		{"금액 필드 없음", `{"data":{"userMndp":{}}}`, 0, true},
		{"HTML 응답", `<html>login</html>`, 0, true},
		{"빈 응답", ``, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseBalanceResponse([]byte(tc.body))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}
