package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	logFile *os.File
)

// Init는 로거를 초기화하고 로그 파일을 생성합니다
func Init() error {
	// logs 디렉토리 생성
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("로그 디렉토리 생성 실패: %w", err)
	}

	// 로그 파일명: logs/lottery_2026-01-13.log
	logFileName := fmt.Sprintf("lottery_%s.log", time.Now().Format("2006-01-02"))
	logFilePath := filepath.Join(logsDir, logFileName)

	// 로그 파일 열기 (append 모드)
	var err error
	logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("로그 파일 생성 실패: %w", err)
	}

	// 로그를 콘솔과 파일 둘 다에 출력
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime)

	log.Printf("✅ 로그 파일 초기화 완료: %s\n", logFilePath)

	return nil
}

// Close는 로그 파일을 닫습니다
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

// Info는 정보 로그를 출력합니다
func Info(format string, v ...interface{}) {
	log.Printf("[INFO] "+format, v...)
}

// Error는 에러 로그를 출력합니다
func Error(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

// Warning은 경고 로그를 출력합니다
func Warning(format string, v ...interface{}) {
	log.Printf("[WARNING] "+format, v...)
}

// Debug는 디버그 로그를 출력합니다
func Debug(format string, v ...interface{}) {
	log.Printf("[DEBUG] "+format, v...)
}
