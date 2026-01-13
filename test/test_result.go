package main

import (
	"dhlottery/lottery"
	"log"
)

func main() {
	log.Println("=== 당첨번호 조회 테스트 ===")
	log.Println()

	result, err := lottery.GetLatestResult()
	if err != nil {
		log.Fatalf("❌ 실패: %v", err)
	}

	log.Println("✅ 조회 성공!")
	log.Println()
	log.Printf("회차: %s회\n", result.Round)
	log.Printf("추첨일: %s\n", result.DrawDate)
	log.Printf("당첨번호: %v\n", result.Numbers)
	log.Printf("보너스번호: %d\n", result.BonusNumber)
}
