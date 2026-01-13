package scheduler

import (
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler는 크론 스케줄러입니다
type Scheduler struct {
	cron *cron.Cron
}

// New는 새로운 스케줄러를 생성합니다
func New() *Scheduler {
	// 한국 시간대 설정
	location, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		log.Printf("⚠️  시간대 로드 실패, UTC 사용: %v\n", err)
		location = time.UTC
	}

	return &Scheduler{
		cron: cron.New(cron.WithLocation(location)),
	}
}

// AddFunc는 크론 작업을 추가합니다
func (s *Scheduler) AddFunc(spec string, cmd func()) error {
	_, err := s.cron.AddFunc(spec, cmd)
	return err
}

// Start는 스케줄러를 시작합니다
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop은 스케줄러를 중지합니다
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// Wait는 무한 대기합니다
func (s *Scheduler) Wait() {
	select {}
}
