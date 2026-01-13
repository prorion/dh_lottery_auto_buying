package main

import (
	"dhlottery/config"
	"dhlottery/logger"
	"dhlottery/scheduler"
	"dhlottery/tasks"
	"dhlottery/telegram"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 로그 파일 초기화
	if err := logger.Init(); err != nil {
		log.Fatalf("로그 초기화 실패: %v", err)
	}
	defer logger.Close()

	log.Println("╔════════════════════════════════════════╗")
	log.Println("║    동행복권 로또 6/45 자동 구매 프로그램    ║")
	log.Println("╚════════════════════════════════════════╝")
	log.Println()

	// 커맨드 라인 플래그 파싱
	checkBalance := flag.Bool("check", false, "예치금 확인만 수행")
	once := flag.Bool("once", false, "즉시 1회 구매 (기본값: 예치금 확인 후 구매)")
	dryRun := flag.Bool("dryrun", false, "테스트 모드 (실제 구매 안함)")
	serviceMode := flag.Bool("service", false, "스케줄러 모드 (매주 토요일 6시 구매)")

	flag.Parse()

	// 설정 로드
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ 설정 로드 실패: %v\n", err)
	}

	// 텔레그램 봇 초기화
	var bot *telegram.Bot
	if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
		bot = telegram.New(cfg.TelegramBotToken, cfg.TelegramChatID)
		log.Println("✅ 텔레그램 봇 초기화 완료")
	} else {
		log.Println("⚠️  텔레그램 설정이 없습니다. 알림은 전송되지 않습니다.")
	}

	log.Println()

	// 플래그에 따라 실행
	switch {
	case *serviceMode:
		// 스케줄러 모드
		runScheduler(cfg, bot)

	case *checkBalance:
		// 예치금 확인만
		tasks.CheckBalance(cfg, bot)

	case *dryRun:
		// 테스트 모드
		tasks.DryRun(cfg, bot)

	case *once:
		// 즉시 1회 구매 (예치금 확인 없이)
		tasks.BuyLotto(cfg, bot)

	default:
		// 기본값: 예치금 확인 후 구매
		log.Println("🎯 기본 모드: 예치금 확인 후 1회 구매 실행")
		tasks.CheckBalance(cfg, bot)
		tasks.BuyLotto(cfg, bot)
	}
}

// runScheduler는 스케줄러를 실행합니다
func runScheduler(cfg config.Config, bot *telegram.Bot) {
	log.Println("🔄 스케줄러 모드 시작")
	log.Println()
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("    스케줄:")
	log.Println("    - 예치금 확인: 매주 월요일 오전 8시")
	log.Println("    - 로또 구매: 매주 토요일 오전 6시")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println()

	sched := scheduler.New()

	// 예치금 확인: 매주 월요일 오전 8시
	if err := sched.AddFunc("0 8 * * 1", func() {
		tasks.CheckBalance(cfg, bot)
	}); err != nil {
		log.Fatalf("❌ 예치금 확인 스케줄 등록 실패: %v", err)
	}

	// 로또 구매: 매주 토요일 오전 6시
	if err := sched.AddFunc("0 6 * * 6", func() {
		tasks.CheckBalanceAndBuy(cfg, bot)
	}); err != nil {
		log.Fatalf("❌ 로또 구매 스케줄 등록 실패: %v", err)
	}

	sched.Start()

	log.Println("✅ 스케줄러 시작 완료")
	log.Println("   프로그램이 백그라운드에서 실행됩니다.")
	log.Println("   종료하려면 Ctrl+C를 누르세요.")
	log.Println()

	// 시그널 대기
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println()
	log.Println("⚠️  종료 신호를 받았습니다.")
	log.Println("   스케줄러를 중지합니다...")

	sched.Stop()

	log.Println("✅ 프로그램 종료")
}
