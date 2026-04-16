package clickhouseReader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type GetStatusFiltersTest struct {
	query        string
	statusParams []string
	excludeMap   map[string]struct{}
	expected     string
}

func TestGetStatusFilters(t *testing.T) {
	assert := assert.New(t)
	var tests = []GetStatusFiltersTest{
		{"", make([]string, 0), map[string]struct{}{}, ""},
		{"test", []string{"error"}, map[string]struct{}{}, "test AND hasError = true"},
		{"test", []string{"ok"}, map[string]struct{}{}, "test AND hasError = false"},
		{"test", []string{"error"}, map[string]struct{}{"status": {}}, "test AND hasError = false"},
		{"test", []string{"ok"}, map[string]struct{}{"status": {}}, "test AND hasError = true"},
		{"test", []string{"error", "ok"}, map[string]struct{}{}, "test"},
	}
	for _, test := range tests {
		assert.Equal(getStatusFilters(test.query, test.statusParams, test.excludeMap), test.expected)
	}
}

func init() {
	// parseTTLFromEngineFullStr uses zap.L() internally
	zap.ReplaceGlobals(zap.NewNop())
}

func TestParseTTLFromEngineFullStr(t *testing.T) {
	tests := []struct {
		name       string
		engineFull string
		wantDel    int
		wantMove   int
	}{
		{
			name:       "should return hours when toIntervalSecond DELETE only",
			engineFull: `ReplicatedMergeTree('/clickhouse/tables/{uuid}/{shard}', '{replica}') TTL toDateTime(timestamp) + toIntervalSecond(1296000) SETTINGS index_granularity = 1024`,
			wantDel:    360, // 1296000 / 3600
			wantMove:   -1,
		},
		{
			name:       "should return hours when toIntervalSecond DELETE and MOVE",
			engineFull: `ReplicatedMergeTree('/clickhouse/tables/{uuid}/{shard}', '{replica}') TTL toDateTime(timestamp / 1000000000) + toIntervalSecond(2592000), toDateTime(timestamp / 1000000000) + toIntervalSecond(604800) TO VOLUME 's3' SETTINGS index_granularity = 8192`,
			wantDel:    720, // 2592000 / 3600
			wantMove:   168, // 604800 / 3600
		},
		{
			name:       "should return hours when toIntervalDay DELETE only",
			engineFull: `ReplicatedMergeTree('/clickhouse/tables/{uuid}/{shard}', '{replica}') TTL toDateTime(timestamp / 1000000000) + toIntervalDay(97) SETTINGS index_granularity = 8192`,
			wantDel:    2328, // 97 * 24
			wantMove:   -1,
		},
		{
			name:       "should return hours when toIntervalDay DELETE and MOVE",
			engineFull: `ReplicatedMergeTree('/clickhouse/tables/{uuid}/{shard}', '{replica}') TTL toDateTime(timestamp / 1000000000) + toIntervalDay(97), toDateTime(timestamp / 1000000000) + toIntervalDay(7) TO VOLUME 's3', toDateTime(timestamp / 1000000000) + toIntervalDay(7) RECOMPRESS CODEC(ZSTD(3)) SETTINGS index_granularity = 8192`,
			wantDel:    2328, // 97 * 24
			wantMove:   168,  // 7 * 24
		},
		{
			name:       "should return -1 when no TTL in engine_full",
			engineFull: `ReplicatedMergeTree('/clickhouse/tables/{uuid}/{shard}', '{replica}') ORDER BY (timestamp) SETTINGS index_granularity = 8192`,
			wantDel:    -1,
			wantMove:   -1,
		},
		{
			name:       "should return -1 when empty string",
			engineFull: ``,
			wantDel:    -1,
			wantMove:   -1,
		},
		{
			name:       "should return hours when metrics toIntervalSecond format",
			engineFull: `ReplicatedMergeTree('/clickhouse/tables/{uuid}/{shard}', '{replica}') TTL toDateTime(toUInt32(unix_milli / 1000), 'UTC') + toIntervalSecond(2592000), toDateTime(toUInt32(unix_milli / 1000), 'UTC') + toIntervalSecond(604800) TO VOLUME 's3', toDateTime(toUInt32(unix_milli / 1000), 'UTC') + toIntervalSecond(604800) RECOMPRESS CODEC(ZSTD(3)) SETTINGS index_granularity = 8192`,
			wantDel:    720,
			wantMove:   168,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDel, gotMove := parseTTLFromEngineFullStr(tt.engineFull)
			assert.Equal(t, tt.wantDel, gotDel, "delTTL mismatch")
			assert.Equal(t, tt.wantMove, gotMove, "moveTTL mismatch")
		})
	}
}
