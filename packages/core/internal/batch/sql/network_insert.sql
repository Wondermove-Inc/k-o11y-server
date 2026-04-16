-- ============================================================
-- Network Map INSERT Query (Optimized: Subquery Separation)
-- ============================================================
-- 목적: Beyla network flow trace 데이터를 집계하여 서비스 간 의존성 맵 생성
--
-- 최적화 전략:
--   Inner Query: 42K raw rows에서 dest resolution dictGet 없이 집계 → ~10K grouped rows
--   Outer Query: ~10K grouped rows에서만 dictGet 실행
--   효과: dictGet 호출 756K → 264K (65% 감소), CPU 스파이크 완화
--
-- 정확성 보장:
--   dest_raw가 이미 GROUP BY 컬럼이고, dest(resolved workload)는
--   (dest_raw, k8s_cluster_name)의 결정론적 함수이므로
--   inner GROUP BY에서 dest/is_external/dest_namespace 제거해도
--   그룹 경계 불변 → quantile 값 100% 동일
--
-- 입력 테이블: signoz_traces.signoz_index_v3 (Beyla trace 데이터)
-- 출력 테이블: signoz_traces.network_map_connections (집계된 서비스 맵 데이터)
--
-- 주요 처리:
--   1. Watermark 기반 증분 처리 (Go에서 주입된 watermark 이후 데이터만)
--   2. Inner: 집계 전용 (protocol/method 감지, duration 통계, 에러율)
--   3. Outer: Destination IP → Workload 해석 (Dictionary 활용)
--   4. Outer: Internal/External 트래픽 분류
--
-- 주의사항:
--   - 20초 Safety Buffer 적용 (데이터 안정화 대기)
--   - kubernetes.default.svc, kube-apiserver, node IP 제외 (outer WHERE)
--   - src = dest인 self-call 제외 (outer WHERE)
-- ============================================================

INSERT INTO signoz_traces.network_map_connections
WITH
    -- ============================================================
    -- Outer: dictGet Resolution (~10K grouped rows에서만 실행)
    -- ============================================================
    -- Step 1: dest_raw에서 IP/호스트명 추출
    (CASE
        WHEN position(dest_raw, ':') > 0
        THEN splitByChar(':', dest_raw)[1]
        ELSE dest_raw
    END) AS dest_ip_extracted,
    match(dest_ip_extracted, '^[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+$') AS is_ip_address,
    (position(dest_ip_extracted, '.svc.cluster.local') > 0
        OR position(dest_ip_extracted, '.cluster.local') > 0) AS is_fqdn_svc,
    (position(dest_ip_extracted, '.') > 0) AS has_dot,
    -- Step 2: IP → Pod workload 해석
    (IF(is_ip_address,
        dictGetOrDefault('signoz_traces.pod_workload_map_dict', 'workload_name',
            (dest_ip_extracted, k8s_cluster_name), ''),
        '')) AS pod_workload_name,
    (IF(is_ip_address,
        dictGetOrDefault('signoz_traces.pod_workload_map_dict', 'namespace',
            (dest_ip_extracted, k8s_cluster_name), ''),
        '')) AS pod_workload_namespace,
    -- Step 3: FQDN → Service 해석
    splitByChar('.', dest_ip_extracted)[1] AS fqdn_service_name,
    (IF(is_fqdn_svc AND length(splitByChar('.', dest_ip_extracted)) >= 2,
        splitByChar('.', dest_ip_extracted)[2],
        src_namespace)) AS fqdn_namespace,
    (IF(NOT is_ip_address,
        dictGetOrDefault('signoz_traces.svc_ep_addr_dict', 'pod_ip',
            (k8s_cluster_name, fqdn_service_name, fqdn_namespace), ''),
        '')) AS svc_pod_ip,
    (IF(svc_pod_ip != '',
        dictGetOrDefault('signoz_traces.pod_workload_map_dict', 'workload_name',
            (svc_pod_ip, k8s_cluster_name), ''),
        '')) AS svc_workload_name,
    (IF(svc_pod_ip != '',
        dictGetOrDefault('signoz_traces.pod_workload_map_dict', 'namespace',
            (svc_pod_ip, k8s_cluster_name), ''),
        '')) AS svc_workload_namespace,
    -- Step 4: 제외 대상 판별 (kubernetes svc, node name/IP)
    (dictGetOrDefault('signoz_traces.cluster_service_dict', 'service_name',
        (k8s_cluster_name, dest_ip_extracted), '') = 'kubernetes') AS is_kubernetes_svc,
    has(dictGet('signoz_traces.cluster_nodes_dict', 'node_names',
        k8s_cluster_name), dest_ip_extracted) AS is_node_name,
    has(dictGet('signoz_traces.cluster_nodes_dict', 'node_ips',
        k8s_cluster_name), dest_ip_extracted) AS is_node_ip,
    -- Step 5: 최종 destination IP 결정
    (CASE
        WHEN length(agg_ext_url) > 0 AND match(agg_ext_url, '^[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+') THEN
            CASE WHEN position(agg_ext_url, ':') > 0 THEN
                substring(agg_ext_url, 1, position(agg_ext_url, ':') - 1)
            ELSE agg_ext_url END
        WHEN is_ip_address THEN dest_ip_extracted
        WHEN svc_pod_ip != '' THEN svc_pod_ip
        ELSE ''
    END) AS final_dest_ip,
    -- Step 6: 최종 destination workload name 결정
    (CASE
        WHEN is_ip_address THEN
            CASE WHEN pod_workload_name != '' THEN pod_workload_name ELSE dest_ip_extracted END
        WHEN is_fqdn_svc THEN
            CASE WHEN svc_workload_name != '' THEN svc_workload_name ELSE '' END
        WHEN has_dot AND length(agg_ext_url) = 0 THEN
            CASE WHEN svc_workload_name != '' THEN svc_workload_name ELSE splitByChar('.', dest_ip_extracted)[1] END
        WHEN has_dot AND length(agg_ext_url) > 0 THEN dest_ip_extracted
        ELSE
            CASE WHEN svc_workload_name != '' THEN svc_workload_name ELSE dest_ip_extracted END
    END) AS raw_dest,
    COALESCE(
        nullIf(dictGetOrDefault('signoz_traces.pod_workload_map_dict', 'workload_name',
            (final_dest_ip, k8s_cluster_name), ''), ''),
        raw_dest
    ) AS final_dest,
    -- Step 7: 최종 destination namespace 결정
    dictGetOrDefault('signoz_traces.svc_ep_addr_dict', 'pod_ip',
        (k8s_cluster_name, raw_dest, src_namespace), '') AS final_svc_pod_ip,
    dictGetOrDefault('signoz_traces.svc_ep_addr_dict', 'ns',
        (k8s_cluster_name, raw_dest, src_namespace), '') AS final_svc_namespace,
    (multiIf(
        is_ip_address AND pod_workload_namespace != '', pod_workload_namespace,
        svc_workload_namespace != '', svc_workload_namespace,
        is_fqdn_svc AND length(splitByChar('.', dest_ip_extracted)) >= 2, splitByChar('.', dest_ip_extracted)[2],
        (NOT is_ip_address) AND has_dot AND (match(dest_ip_extracted, '\\.[0-9]') = 0) AND (NOT is_fqdn_svc),
            splitByChar('.', dest_ip_extracted)[2],
        ''
    )) AS raw_dest_namespace,
    COALESCE(
        nullIf(svc_workload_namespace, ''),
        nullIf(raw_dest_namespace, ''),
        dictGetOrDefault('signoz_traces.pod_workload_map_dict', 'namespace',
            (final_svc_pod_ip, k8s_cluster_name), 'unknown')
    ) AS final_dest_namespace,
    -- Step 8: Internal/External 트래픽 분류
    -- RC3 fix: bare hostname(IP 아님, 점 없음)이 svc_ep_addr_dict를 통해 IP를 획득한 경우,
    --   pod_workload_map_dict 리로딩(~10초) 중 IP 검증 실패 시 external 오분류 방지
    (CASE
        WHEN endsWith(final_dest, '.svc.cluster.local') THEN 0
        WHEN endsWith(final_dest, '.svc') THEN 0
        WHEN final_dest = 'kubernetes.default.svc' THEN 0
        WHEN length(final_dest_ip) > 0 THEN
            CASE
                WHEN dictHas('signoz_traces.pod_workload_map_dict', tuple(final_dest_ip, k8s_cluster_name)) THEN 0
                WHEN dictHas('signoz_traces.cluster_service_dict', tuple(k8s_cluster_name, final_dest_ip)) THEN 0
                WHEN has(dictGet('signoz_traces.cluster_nodes_dict', 'node_ips', k8s_cluster_name), final_dest_ip) THEN 0
                WHEN startsWith(final_dest_ip, '127.') THEN 0
                -- RC3 fix: 원본 dest_raw가 bare hostname이면 K8s 내부 서비스 → 항상 internal
                WHEN NOT is_ip_address AND position(dest_ip_extracted, '.') = 0 THEN 0
                ELSE 1
            END
        WHEN position(final_dest, '.') = 0 THEN 0
        WHEN position(final_dest, '.') > 0 THEN 1
        ELSE 0
    END) AS is_external_calc
-- ============================================================
-- Outer SELECT: 19 columns matching network_map_connections schema
-- ============================================================
SELECT
    src,
    src_raw,
    final_dest AS dest,
    dest_raw,
    protocol,
    method,
    is_external_calc AS is_external,
    duration_sum,
    duration_count,
    duration_p50,
    duration_p95,
    duration_p99,
    error_count,
    total_count,
    ts_bucket,
    deployment_environment,
    k8s_cluster_name,
    src_namespace,
    final_dest_namespace AS dest_namespace
FROM (
    -- ============================================================
    -- Inner: Aggregation Only (dest resolution dictGet 없음, ~42K raw rows 처리)
    -- ============================================================
    SELECT
        COALESCE(nullIf(resources_string['k8s.deployment.name'], ''), nullIf(resources_string['k8s.statefulset.name'], ''),
            nullIf(resources_string['k8s.daemonset.name'], ''), nullIf(resources_string['k8s.job.name'], '')) AS src,
        COALESCE(nullIf(resources_string['k8s.deployment.name'], ''), nullIf(resources_string['k8s.statefulset.name'], ''),
            nullIf(resources_string['k8s.daemonset.name'], ''), nullIf(resources_string['k8s.job.name'], '')) AS src_raw,
        attributes_string['server.address'] AS dest_raw,
        COALESCE(nullIf(CASE
            WHEN attributes_string['rpc.system'] = 'grpc' THEN 'gRPC'
            WHEN attributes_string['http.scheme'] = 'http' THEN 'HTTP'
            WHEN length(external_http_url) > 0 THEN 'HTTP'
            WHEN name REGEXP '^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\\s+/' THEN 'HTTP'
            WHEN name IN (
                'PING', 'GET', 'SET', 'SETEX', 'SETNX', 'PSETEX', 'MGET', 'MSET',
                'GETSET', 'APPEND', 'INCR', 'DECR', 'INCRBY', 'DECRBY', 'STRLEN',
                'HGET', 'HSET', 'HMGET', 'HMSET', 'HGETALL', 'HDEL',
                'HEXISTS', 'HLEN', 'HKEYS', 'HVALS', 'HINCRBY',
                'LPUSH', 'RPUSH', 'LPOP', 'RPOP', 'LRANGE', 'LLEN', 'LINDEX', 'LREM',
                'SADD', 'SREM', 'SMEMBERS', 'SISMEMBER', 'SCARD', 'SPOP',
                'ZADD', 'ZRANGE', 'ZREM', 'ZSCORE', 'ZCARD', 'ZRANGEBYSCORE',
                'DEL', 'EXISTS', 'EXPIRE', 'TTL', 'PERSIST', 'TYPE', 'SCAN', 'RENAME',
                'AUTH', 'INFO', 'CLIENT', 'DBSIZE',
                'EVAL', 'EVALSHA',
                'SUBSCRIBE', 'PUBLISH',
                'MULTI', 'EXEC', 'WATCH',
                'PFADD', 'PFCOUNT'
            ) THEN 'Redis'
            WHEN length(attributes_string['db.system.name']) > 0 THEN
                CASE WHEN attributes_string['db.system.name'] = 'redis' THEN 'Redis'
                     WHEN attributes_string['db.system.name'] IN ('mysql', 'postgresql') THEN 'SQL' ELSE 'SQL' END
            WHEN length(attributes_string['db.system']) > 0 THEN
                CASE WHEN attributes_string['db.system'] = 'redis' THEN 'Redis' ELSE 'SQL' END
            WHEN length(attributes_string['messaging.system']) > 0 THEN
                CASE WHEN attributes_string['messaging.system'] = 'kafka' THEN 'Kafka'
                     WHEN attributes_string['messaging.system'] = 'rabbitmq' THEN 'RabbitMQ' ELSE 'Messaging' END
            ELSE 'UNKNOWN'
        END, ''), 'UNKNOWN') AS protocol,
        CASE
            WHEN length(external_http_method) > 0 THEN external_http_method
            WHEN attributes_string['rpc.system'] = 'grpc' AND length(name) > 0 THEN
                CASE WHEN position(reverse(name), '/') > 0 THEN reverse(substring(reverse(name), 1, position(reverse(name), '/') - 1)) ELSE name END
            WHEN name IN (
                'PING', 'GET', 'SET', 'SETEX', 'SETNX', 'PSETEX', 'MGET', 'MSET',
                'GETSET', 'APPEND', 'INCR', 'DECR', 'INCRBY', 'DECRBY', 'STRLEN',
                'HGET', 'HSET', 'HMGET', 'HMSET', 'HGETALL', 'HDEL',
                'HEXISTS', 'HLEN', 'HKEYS', 'HVALS', 'HINCRBY',
                'LPUSH', 'RPUSH', 'LPOP', 'RPOP', 'LRANGE', 'LLEN', 'LINDEX', 'LREM',
                'SADD', 'SREM', 'SMEMBERS', 'SISMEMBER', 'SCARD', 'SPOP',
                'ZADD', 'ZRANGE', 'ZREM', 'ZSCORE', 'ZCARD', 'ZRANGEBYSCORE',
                'DEL', 'EXISTS', 'EXPIRE', 'TTL', 'PERSIST', 'TYPE', 'SCAN', 'RENAME',
                'AUTH', 'INFO', 'CLIENT', 'DBSIZE',
                'EVAL', 'EVALSHA',
                'SUBSCRIBE', 'PUBLISH',
                'MULTI', 'EXEC', 'WATCH',
                'PFADD', 'PFCOUNT'
            ) THEN name
            WHEN length(attributes_string['db.statement']) > 0 THEN
                CASE WHEN attributes_string['db.statement'] LIKE 'SELECT%' THEN 'SELECT'
                     WHEN attributes_string['db.statement'] LIKE 'INSERT%' THEN 'INSERT'
                     WHEN attributes_string['db.statement'] LIKE 'UPDATE%' THEN 'UPDATE'
                     WHEN attributes_string['db.statement'] LIKE 'DELETE%' THEN 'DELETE'
                     WHEN attributes_string['db.statement'] LIKE 'CREATE%' THEN 'CREATE'
                     WHEN attributes_string['db.statement'] LIKE 'DROP%' THEN 'DROP'
                     WHEN attributes_string['db.statement'] LIKE 'ALTER%' THEN 'ALTER' ELSE 'SQL' END
            WHEN length(attributes_string['db.operation.name']) > 0 THEN attributes_string['db.operation.name']
            WHEN name IN ('SELECT', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'ALTER') THEN name
            WHEN name LIKE 'SELECT %' THEN 'SELECT' WHEN name LIKE 'UPDATE %' THEN 'UPDATE'
            WHEN name LIKE 'INSERT %' THEN 'INSERT' WHEN name LIKE 'DELETE %' THEN 'DELETE'
            WHEN name LIKE 'PREPARED STATEMENT%' THEN 'PREPARED STATEMENT'
            WHEN length(attributes_string['db.operation']) > 0 THEN attributes_string['db.operation']
            ELSE 'UNKNOWN'
        END AS method,
        sum(toUInt64(duration_nano/1000000)) AS duration_sum,
        count() AS duration_count,
        quantile(0.5)(duration_nano/1000000) AS duration_p50,
        quantile(0.95)(duration_nano/1000000) AS duration_p95,
        quantile(0.99)(duration_nano/1000000) AS duration_p99,
        sum(CASE WHEN status_code >= 2 THEN 1 ELSE 0 END) AS error_count,
        count() AS total_count,
        -- NOTE: alias를 'ts_bucket'으로 명명 (원본 'timestamp' 컬럼과 동일 이름 사용 시
        --        ClickHouse 24.x optimizer가 partition key toDate(timestamp) 인식 실패 → 전체 테이블 스캔)
        toStartOfMinute(timestamp) AS ts_bucket,
        resources_string['deployment.environment'] AS deployment_environment,
        resources_string['k8s.cluster.name'] AS k8s_cluster_name,
        resources_string['k8s.namespace.name'] AS src_namespace,
        anyIf(external_http_url, length(external_http_url) > 0) AS agg_ext_url
    FROM signoz_traces.signoz_index_v3
    WHERE 1=1
        AND kind = 3
        AND timestamp > toDateTime64('{{WATERMARK_TS}}', 9)
        AND timestamp <= toDateTime64('{{UPPER_BOUND_TS}}', 9)
        AND length(attributes_string['server.address']) > 0
        AND attributes_string['server.address'] != '127.0.0.1'
        AND NOT (attributes_string['server.address'] LIKE 'localhost:%')
        AND NOT (attributes_string['server.address'] LIKE 'kubernetes.%')
        -- NOTE: is_kubernetes_svc, is_node_name, is_node_ip moved to outer WHERE
        AND NOT has(dictGet('signoz_traces.cluster_nodes_dict', 'node_names', resources_string['k8s.cluster.name']), http_host)
        AND length(COALESCE(nullIf(resources_string['k8s.deployment.name'], ''), nullIf(resources_string['k8s.statefulset.name'], ''),
            nullIf(resources_string['k8s.daemonset.name'], ''), nullIf(resources_string['k8s.job.name'], ''))) > 0
        AND (
            length(external_http_url) > 0 OR attributes_string['rpc.system'] = 'grpc' OR length(db_name) > 0
            -- RC2 fix: OTel Semantic Conventions v1.27+ uses db.system.name instead of db.system
            OR length(attributes_string['db.system.name']) > 0
            OR length(attributes_string['db.system']) > 0
            OR length(attributes_string['db.statement']) > 0
            OR name IN ('SELECT', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'ALTER')
            OR name LIKE 'SELECT %' OR name LIKE 'UPDATE %' OR name LIKE 'INSERT %' OR name LIKE 'DELETE %'
            OR name LIKE 'PREPARED STATEMENT%'
            OR attributes_string['db.operation.name'] IN ('SELECT', 'INSERT', 'UPDATE', 'DELETE')
            -- RC1 fix: Expanded Redis command list (13 → 50+)
            OR name IN (
                -- String
                'PING', 'GET', 'SET', 'SETEX', 'SETNX', 'PSETEX', 'MGET', 'MSET',
                'GETSET', 'APPEND', 'INCR', 'DECR', 'INCRBY', 'DECRBY', 'STRLEN',
                -- Hash
                'HGET', 'HSET', 'HMGET', 'HMSET', 'HGETALL', 'HDEL',
                'HEXISTS', 'HLEN', 'HKEYS', 'HVALS', 'HINCRBY',
                -- List
                'LPUSH', 'RPUSH', 'LPOP', 'RPOP', 'LRANGE', 'LLEN', 'LINDEX', 'LREM',
                -- Set
                'SADD', 'SREM', 'SMEMBERS', 'SISMEMBER', 'SCARD', 'SPOP',
                -- Sorted Set
                'ZADD', 'ZRANGE', 'ZREM', 'ZSCORE', 'ZCARD', 'ZRANGEBYSCORE',
                -- Key management
                'DEL', 'EXISTS', 'EXPIRE', 'TTL', 'PERSIST', 'TYPE', 'SCAN', 'RENAME',
                -- Connection/Server
                'AUTH', 'INFO', 'CLIENT', 'DBSIZE',
                -- Scripting
                'EVAL', 'EVALSHA',
                -- Pub/Sub
                'SUBSCRIBE', 'PUBLISH',
                -- Transaction
                'MULTI', 'EXEC', 'WATCH',
                -- HyperLogLog
                'PFADD', 'PFCOUNT'
            )
            OR length(attributes_string['messaging.system']) > 0
            OR attributes_string['http.scheme'] IN ('http', 'https')
            OR length(attributes_string['http.url']) > 0
            OR length(attributes_string['http.target']) > 0
            OR (match(splitByChar(':', attributes_string['server.address'])[1], '^[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+$')
                AND NOT dictHas('signoz_traces.pod_workload_map_dict',
                    tuple(splitByChar(':', attributes_string['server.address'])[1], resources_string['k8s.cluster.name'])))
        )
    GROUP BY ts_bucket, src, src_raw, dest_raw, protocol, method, deployment_environment, k8s_cluster_name, src_namespace
    HAVING length(src) > 0
) AS agg
-- ============================================================
-- Outer WHERE: dictGet-based filters + HAVING conditions (from original)
-- ============================================================
WHERE
    -- Exclusions (moved from inner WHERE - dictGet per grouped row only)
    NOT is_kubernetes_svc
    AND NOT is_node_name
    AND NOT is_node_ip
    -- Validity (moved from original HAVING)
    AND length(final_dest) > 0
    AND src != final_dest
    -- Namespace/external validation (moved from original HAVING)
    AND (
        length(nullIf(svc_workload_namespace, '')) > 0
        OR length(nullIf(final_svc_namespace, '')) > 0
        OR (svc_pod_ip != '' AND dictHas('signoz_traces.pod_workload_map_dict', tuple(svc_pod_ip, k8s_cluster_name)))
        OR dictHas('signoz_traces.pod_workload_map_dict', tuple(final_svc_pod_ip, k8s_cluster_name))
        OR dictHas('signoz_traces.cluster_service_dict', tuple(k8s_cluster_name, final_dest_ip))
        OR is_external_calc = 1
    );
