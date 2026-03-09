package persistence

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

var fallbackIDSequence atomic.Uint64

// NewID 生成带业务前缀的唯一标识，兼顾本地开发可读性和跨重启唯一性。
func NewID(prefix string) string {
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err == nil {
		return fmt.Sprintf("%s-%d-%s", prefix, time.Now().UnixMilli(), hex.EncodeToString(randomBytes))
	}

	sequence := fallbackIDSequence.Add(1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixMilli(), sequence)
}

// NullableTime 把零值时间转换为数据库 NULL，避免把“未开始/未完成”误写成业务时间点。
func NullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

func FromNullableTime(value sql.NullTime) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}
