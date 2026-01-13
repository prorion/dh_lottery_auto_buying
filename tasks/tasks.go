package tasks

import (
	"dhlottery/config"
	"dhlottery/lottery"
	"dhlottery/telegram"
	"fmt"
	"log"
)

// CheckBalance는 예치금 확인 작업을 수행합니다 (모든 계정)
func CheckBalance(cfg config.Config, bot *telegram.Bot) {
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("          💰 예치금 확인 작업")
	log.Printf("          (총 %d개 계정)\n", len(cfg.Accounts))
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, account := range cfg.Accounts {
		log.Println()
		log.Printf("┌─────────────────────────────────────┐")
		log.Printf("│ 계정 %d/%d: %s", i+1, len(cfg.Accounts), account.UserID)
		log.Printf("└─────────────────────────────────────┘")
		log.Println()

		checkBalanceForAccount(account, bot)
	}

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println()
}

// checkBalanceForAccount는 특정 계정의 예치금을 확인합니다
func checkBalanceForAccount(account config.Account, bot *telegram.Bot) {
	// 클라이언트 생성
	client, err := lottery.NewClient(account.UserID, account.Password)
	if err != nil {
		log.Printf("❌ 클라이언트 생성 실패: %v\n", err)
		if bot != nil {
			bot.SendMessageSafe(fmt.Sprintf("(%s) ❌ <b>예치금 확인 실패</b>\n\n클라이언트 생성 오류: %v", account.UserID, err))
		}
		return
	}

	// 로그인
	if err := client.Login(); err != nil {
		log.Printf("❌ 로그인 실패: %v\n", err)
		if bot != nil {
			bot.SendMessageSafe(fmt.Sprintf("(%s) ❌ <b>동행복권 로그인 실패</b>\n\n%v", account.UserID, err))
		}
		return
	}

	// 예치금 확인
	balance, err := client.CheckBalance()
	if err != nil {
		log.Printf("❌ 예치금 확인 실패: %v\n", err)
		if bot != nil {
			bot.SendMessageSafe(fmt.Sprintf("(%s) ❌ <b>예치금 확인 실패</b>\n\n%v", account.UserID, err))
		}
		return
	}

	// 예치금이 10,000원 미만인 경우 알림
	if balance < 10000 {
		log.Printf("⚠️  예치금 부족: %s원 (10,000원 미만)\n", lottery.FormatMoney(balance))

		if bot != nil {
			message := fmt.Sprintf(
				"(%s) ⚠️ <b>예치금 부족 알림</b>\n\n"+
					"현재 예치금: <b>%s원</b>\n"+
					"기준 금액: 10,000원\n\n"+
					"💡 예치금을 충전해주세요!",
				account.UserID,
				lottery.FormatMoney(balance),
			)
			bot.SendMessageSafe(message)
		}
	} else {
		log.Printf("✅ 예치금 충분: %s원\n", lottery.FormatMoney(balance))
		// 10,000원 이상이면 텔레그램 알림 보내지 않음
	}
}

// BuyLotto는 로또 구매 작업을 수행합니다 (모든 계정)
func BuyLotto(cfg config.Config, bot *telegram.Bot) {
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("          🎱 로또 구매 작업")
	log.Printf("          (총 %d개 계정)\n", len(cfg.Accounts))
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, account := range cfg.Accounts {
		log.Println()
		log.Printf("┌─────────────────────────────────────┐")
		log.Printf("│ 계정 %d/%d: %s", i+1, len(cfg.Accounts), account.UserID)
		log.Printf("└─────────────────────────────────────┘")
		log.Println()

		buyLottoForAccount(account, bot)
	}

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println()
}

// buyLottoForAccount는 특정 계정으로 로또를 구매합니다
func buyLottoForAccount(account config.Account, bot *telegram.Bot) {
	// 클라이언트 생성
	client, err := lottery.NewClient(account.UserID, account.Password)
	if err != nil {
		log.Printf("❌ 클라이언트 생성 실패: %v\n", err)
		if bot != nil {
			bot.SendMessageSafe(fmt.Sprintf("(%s) ❌ <b>로또 구매 실패</b>\n\n클라이언트 생성 오류: %v", account.UserID, err))
		}
		return
	}

	// 로그인
	log.Println("=== 로그인 시작 ===")
	if err := client.Login(); err != nil {
		log.Printf("❌ 로그인 실패: %v\n", err)
		if bot != nil {
			bot.SendMessageSafe(fmt.Sprintf("(%s) ❌ <b>로또 구매 실패</b>\n\n로그인 오류: %v", account.UserID, err))
		}
		return
	}

	// 구매 페이지 접근
	log.Println()
	log.Println("=== 로또 6/45 구매 페이지 접근 ===")
	if err := client.NavigateToLottoBuyPage(); err != nil {
		log.Printf("❌ 구매 페이지 접근 실패: %v\n", err)
		if bot != nil {
			bot.SendMessageSafe(fmt.Sprintf("(%s) ❌ <b>로또 구매 실패</b>\n\n페이지 접근 오류: %v", account.UserID, err))
		}
		return
	}

	// 로또 구매 (5게임)
	log.Println()
	log.Println("=== 로또 자동 구매 (5게임) ===")
	result, resultMsg, err := client.BuyLottoAutoWithResult(account.UserID, 5)
	if err != nil {
		log.Printf("❌ 구매 실패: %v\n", err)
		if bot != nil {
			bot.SendMessageSafe(fmt.Sprintf("(%s) ❌ <b>로또 구매 실패</b>\n\n%v", account.UserID, err))
		}
		return
	}

	// 구매 결과 출력
	client.PrintBuyResult(result)

	// 텔레그램 알림 전송
	if bot != nil {
		bot.SendMessageSafe(resultMsg)
	}
}

// CheckBalanceAndBuy는 예치금 확인 후 로또 구매 작업을 수행합니다 (모든 계정)
func CheckBalanceAndBuy(cfg config.Config, bot *telegram.Bot) {
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("      💰 예치금 확인 및 로또 구매 작업")
	log.Printf("          (총 %d개 계정)\n", len(cfg.Accounts))
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, account := range cfg.Accounts {
		log.Println()
		log.Printf("┌─────────────────────────────────────┐")
		log.Printf("│ 계정 %d/%d: %s", i+1, len(cfg.Accounts), account.UserID)
		log.Printf("└─────────────────────────────────────┘")
		log.Println()

		checkBalanceAndBuyForAccount(account, bot)
	}

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println()
}

// checkBalanceAndBuyForAccount는 특정 계정으로 예치금 확인 후 구매합니다
func checkBalanceAndBuyForAccount(account config.Account, bot *telegram.Bot) {
	// 클라이언트 생성
	client, err := lottery.NewClient(account.UserID, account.Password)
	if err != nil {
		log.Printf("❌ 클라이언트 생성 실패: %v\n", err)
		if bot != nil {
			bot.SendMessageSafe(fmt.Sprintf("(%s) ❌ <b>작업 실패</b>\n\n클라이언트 생성 오류: %v", account.UserID, err))
		}
		return
	}

	// 1단계: 로그인
	log.Println()
	log.Println("=== 1단계: 로그인 ===")
	if err := client.Login(); err != nil {
		log.Printf("❌ 로그인 실패: %v\n", err)
		if bot != nil {
			bot.SendMessageSafe(fmt.Sprintf("(%s) ❌ <b>로그인 실패</b>\n\n%v", account.UserID, err))
		}
		return
	}

	// 2단계: 예치금 확인
	log.Println()
	log.Println("=== 2단계: 예치금 확인 ===")
	balance, err := client.CheckBalance()
	if err != nil {
		log.Printf("❌ 예치금 확인 실패: %v\n", err)
		if bot != nil {
			bot.SendMessageSafe(fmt.Sprintf("(%s) ❌ <b>예치금 확인 실패</b>\n\n%v", account.UserID, err))
		}
		return
	}

	// 예치금 부족 체크
	if balance < 5000 {
		log.Printf("⚠️  예치금 부족: %s원 (최소 5,000원 필요)\n", lottery.FormatMoney(balance))
		if bot != nil {
			message := fmt.Sprintf(
				"(%s) ⚠️ <b>예치금 부족 알림</b>\n\n"+
					"현재 예치금: <b>%s원</b>\n"+
					"필요 금액: 5,000원\n\n"+
					"💡 예치금을 충전해주세요!",
				account.UserID,
				lottery.FormatMoney(balance),
			)
			bot.SendMessageSafe(message)
		}
		return
	}

	log.Printf("✅ 예치금 충분: %s원\n", lottery.FormatMoney(balance))

	// 예치금 알림 (텔레그램)
	if bot != nil && balance < 10000 {
		message := fmt.Sprintf(
			"(%s) ⚠️ <b>예치금 알림</b>\n\n"+
				"현재 예치금: <b>%s원</b>\n\n"+
				"💡 예치금이 10,000원 미만입니다.",
			account.UserID,
			lottery.FormatMoney(balance),
		)
		bot.SendMessageSafe(message)
	}

	// 3단계: 구매 페이지 접근
	log.Println()
	log.Println("=== 3단계: 로또 6/45 구매 페이지 접근 ===")
	if err := client.NavigateToLottoBuyPage(); err != nil {
		log.Printf("❌ 구매 페이지 접근 실패: %v\n", err)
		if bot != nil {
			bot.SendMessageSafe(fmt.Sprintf("(%s) ❌ <b>로또 구매 실패</b>\n\n페이지 접근 오류: %v", account.UserID, err))
		}
		return
	}

	// 4단계: 로또 구매 (5게임)
	log.Println()
	log.Println("=== 4단계: 로또 자동 구매 (5게임) ===")
	result, resultMsg, err := client.BuyLottoAutoWithResult(account.UserID, 5)
	if err != nil {
		log.Printf("❌ 구매 실패: %v\n", err)
		if bot != nil {
			bot.SendMessageSafe(fmt.Sprintf("(%s) ❌ <b>로또 구매 실패</b>\n\n%v", account.UserID, err))
		}
		return
	}

	// 구매 결과 출력
	client.PrintBuyResult(result)

	// 텔레그램 알림 전송
	if bot != nil {
		bot.SendMessageSafe(resultMsg)
	}
}

// DryRun은 구매하지 않고 테스트만 수행합니다 (모든 계정)
func DryRun(cfg config.Config, bot *telegram.Bot) {
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("    🔍 테스트 모드 (실제 구매 안 함)")
	log.Printf("          (총 %d개 계정)\n", len(cfg.Accounts))
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, account := range cfg.Accounts {
		log.Println()
		log.Printf("┌─────────────────────────────────────┐")
		log.Printf("│ 계정 %d/%d: %s", i+1, len(cfg.Accounts), account.UserID)
		log.Printf("└─────────────────────────────────────┘")
		log.Println()

		dryRunForAccount(account)
	}

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println()
}

// dryRunForAccount는 특정 계정으로 테스트를 수행합니다
func dryRunForAccount(account config.Account) {
	// 클라이언트 생성
	client, err := lottery.NewClient(account.UserID, account.Password)
	if err != nil {
		log.Printf("❌ 클라이언트 생성 실패: %v\n", err)
		return
	}

	// 로그인
	log.Println()
	log.Println("=== 1단계: 로그인 ===")
	if err := client.Login(); err != nil {
		log.Printf("❌ 로그인 실패: %v\n", err)
		return
	}

	// 예치금 확인
	log.Println()
	log.Println("=== 2단계: 예치금 확인 ===")
	balance, err := client.CheckBalance()
	if err != nil {
		log.Printf("❌ 예치금 확인 실패: %v\n", err)
		return
	}

	log.Printf("✅ 현재 예치금: %s원\n", lottery.FormatMoney(balance))

	// 구매 페이지 접근
	log.Println()
	log.Println("=== 3단계: 로또 6/45 구매 페이지 접근 ===")
	if err := client.NavigateToLottoBuyPage(); err != nil {
		log.Printf("❌ 구매 페이지 접근 실패: %v\n", err)
		return
	}

	log.Println()
	log.Println("✅ 테스트 완료! (실제 구매는 하지 않았습니다)")
}
