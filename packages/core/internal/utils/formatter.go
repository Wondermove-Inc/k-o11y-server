package utils

import (
	"log"
	"time"
)

// formatTimeForClickHouse ISO8601 시간을 ClickHouse toDateTime() 함수용 형식으로 변환
func FormatTimeForClickHouse(isoTime string) string {
	// RFC3339 (ISO8601) 형식 파싱: "2025-08-21T01:44:00.000Z"
	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		log.Printf("[WARN] failed to parse ISO8601 time '%s': %v", isoTime, err)
		// 파싱 실패 시 현재 시간 반환
		return time.Now().Format("2006-01-02 15:04:05")
	}

	// ClickHouse toDateTime() 형식으로 변환: "2025-08-21 01:44:00"
	return t.Format("2006-01-02 15:04:05")
}
