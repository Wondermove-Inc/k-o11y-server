package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/batch"
	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/config"
	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/handler"
	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/infrastructure"
	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/pkg"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// @title K-O11y Backend v2 API
// @version 1.0.0
// @description This is a K-O11y Backend v2 api server
// @contact.name Request permission of K-O11y Backend v2 API
// @contact.email support@example.com
// @host localhost:3001
func main() {
	log.Printf("Start K-O11y Backend")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// ClickHouse connection (공식 문서 권장 Native API 패턴)
	clickHouseConfig := cfg.GetClickHouseConfig()
	err = infrastructure.ConnectClickHouse(*clickHouseConfig)
	if err != nil {
		log.Fatalf("Failed to connect ClickHouse (will use fallback): %v", err)
		// ClickHouse 연결 실패 시에도 애플리케이션은 계속 실행 (mock 데이터 사용)
	} else {
		log.Printf("✅ [Database] ClickHouse connected successfully")

		// 연결 검증: SELECT 1 테스트
		result, testErr := infrastructure.QueryRowClickHouse("SELECT 1")
		if testErr != nil {
			log.Fatalf("❌ ClickHouse test failed: %v", testErr)
		} else {
			log.Printf("✅ ClickHouse test SUCCESS - result: %v", result)
		}
	}

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// WaitGroup to wait for all goroutines
	var wg sync.WaitGroup

	// AC1: BatchConfig 로드
	// AC2: BATCH_SERVICEMAP_ENABLED=true일 때만 ServiceMapBatchProcessor 시작
	batchConfig := cfg.GetBatchConfig()
	if batchConfig.ServiceMapEnabled {
		log.Printf("🔄 [Batch] ServiceMapBatchProcessor enabled (interval: %v)", batchConfig.ServiceMapInterval)

		// Initialize logger for batch processor
		pkg.InitLogger()
		logger := pkg.GetLogger()

		// Get ClickHouse connection
		clickHouseConn := infrastructure.GetClickHouseConn()
		if clickHouseConn == nil {
			log.Printf("⚠️  [Batch] ClickHouse connection not available, batch processor will not start")
		} else {
			// Initialize metrics (nil registry uses default Prometheus registry)
			batchMetrics := batch.NewBatchMetrics(nil)

			// Create ServiceMapBatchProcessor
			processor, err := batch.NewServiceMapBatchProcessor(
				clickHouseConn,
				batchConfig.ServiceMapInterval,
				batchConfig.InsertTimeout,
				batchConfig.SafetyBuffer,
				batchConfig.MaxWindow,
				logger,
				batchMetrics,
			)
			if err != nil {
				log.Fatalf("❌ [Batch] Failed to create ServiceMapBatchProcessor: %v", err)
			}

			// AC3: Context 전파 (SIGTERM/SIGINT 수신 시 취소)
			// Start ServiceMapBatchProcessor in goroutine
			wg.Add(1)
			go func() {
				defer wg.Done()
				processor.Run(ctx)
			}()

			// AC5: 시작 로그 출력 (INFO 레벨)
			log.Printf("✅ [Batch] ServiceMapBatchProcessor started successfully")
		}
	} else {
		log.Printf("ℹ️  [Batch] ServiceMapBatchProcessor disabled (BATCH_SERVICEMAP_ENABLED=false)")
	}

	// UsageReporter: collects cluster node counts and sends to mgmt portal
	usageCfg := cfg.GetUsageReporterConfig()
	if usageCfg.Enabled {
		log.Printf("🔄 [UsageReporter] enabled (schedule: every hour at :30, tenant: %s)", usageCfg.TenantID)

		pkg.InitLogger()
		usageLogger := pkg.GetLogger()

		clickHouseConn := infrastructure.GetClickHouseConn()
		if clickHouseConn == nil {
			log.Printf("⚠️  [UsageReporter] ClickHouse connection not available, will not start")
		} else {
			usageReporter, err := batch.NewUsageReporter(
				clickHouseConn,
				usageCfg,
				usageLogger,
			)
			if err != nil {
				log.Fatalf("❌ [UsageReporter] Failed to create: %v", err)
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				usageReporter.Run(ctx)
			}()

			log.Printf("✅ [UsageReporter] started successfully")
		}
	} else {
		log.Printf("ℹ️  [UsageReporter] disabled (USAGE_REPORTER_ENABLED=false)")
	}

	// Signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Start HTTP router in goroutine to allow graceful shutdown
	go func() {
		handler.StartRouter()
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("🛑 [Shutdown] Received signal: %v", sig)

	// AC4: Graceful shutdown 대기 (최대 20초)
	// Cancel context to stop batch processor
	cancel()

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// AC4: Graceful shutdown with 20 second timeout
	shutdownTimeout := 20 * time.Second
	select {
	case <-done:
		// AC5: 종료 로그 출력 (INFO 레벨)
		log.Printf("✅ [Shutdown] Graceful shutdown completed")
	case <-time.After(shutdownTimeout):
		log.Printf("⚠️  [Shutdown] Shutdown timeout after %v, forcing exit", shutdownTimeout)
	}

	log.Printf("👋 [Shutdown] K-O11y Backend stopped")
}
