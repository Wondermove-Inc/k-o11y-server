-- ============================================================
-- Watermark Update Query
-- ============================================================
-- 목적: 배치 처리 완료 후 마지막 처리 시점(watermark) 갱신
--
-- 출력 테이블: signoz_traces.network_batch_watermark (워터마크 상태 관리)
--
-- 주요 처리:
--   1. Go에서 계산한 upperBound를 직접 watermark로 저장
--   2. INSERT가 (watermark, upperBound] 범위를 처리했으므로
--      다음 watermark는 정확히 upperBound
--   3. updated_at 현재 시각으로 기록
--
-- 주의사항:
--   - INSERT INTO VALUES로 직접 삽입 (재조회 불필요)
--   - ReplacingMergeTree이므로 id=1 기존 레코드 덮어쓰기
--   - {{UPPER_BOUND_TS}}는 Go에서 주입 (network_insert.sql과 동일 값)
-- ============================================================

INSERT INTO signoz_traces.network_batch_watermark (id, last_processed_ts, updated_at)
VALUES (1, toDateTime64('{{UPPER_BOUND_TS}}', 9), now64(9));
