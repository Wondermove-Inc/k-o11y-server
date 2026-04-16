package queries

// ==================================================================================== //
// 기존 : Internal to Internal (내부 → 내부 서비스)
// ==================================================================================== //

// EdgeTraceDetail 1. TopSlowRequest
func BuildQueryTopSlowRequest() string {
	query := `
		-- 1단계: 기본 필터링으로 데이터 대폭 줄이기
		WITH filtered_base AS (
			SELECT
				timestamp,
				trace_id,
				name,
				duration_nano,
				status_code,
				attributes_string,
				attributes_number,
				resources_string,
				external_http_url,
				http_host,
				-- 미리 src_workload 계산
				COALESCE(
					nullIf(resources_string['k8s.deployment.name'], ''),
					nullIf(resources_string['k8s.statefulset.name'], ''),
					nullIf(resources_string['k8s.daemonset.name'], ''),
					nullIf(resources_string['k8s.rollout.name'], ''),
					nullIf(resources_string['k8s.job.name'], '')
				) AS src_workload,
				-- 미리 server_address 정제
				attributes_string['server.address'] AS server_addr
			FROM signoz_traces.signoz_index_v3
			WHERE 1=1
				-- ✅ 가장 선택적인 조건부터 (시간 범위)
				AND timestamp >= ?
				AND timestamp <= ?
				-- ✅ 클러스터/네임스페이스 필터링
				AND resources_string['k8s.cluster.name'] = ?
				AND (resources_string['k8s.namespace.name'] = ? OR resources_string['k8s.namespace.name'] = ?)
				-- ✅ 기본 조건들 (복잡한 계산 전에)
				AND length(attributes_string['server.address']) > 0
				AND attributes_string['server.address'] NOT IN ('127.0.0.1', 'kubernetes.default')
		),

		-- 2단계: 워크로드 필터링 및 프로토콜 분류
		protocol_classified AS (
			SELECT *,
				-- ✅ 완전한 프로토콜 분류 (SQL, Redis 포함)
				COALESCE(
					nullIf(
						CASE
							-- gRPC
							WHEN attributes_string['rpc.system'] = 'grpc' THEN 'gRPC'
							-- HTTP
							WHEN attributes_string['http.scheme'] = 'http' THEN 'HTTP'
							WHEN length(external_http_url) > 0 THEN 'HTTP'
							WHEN name REGEXP '^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\\s+/' THEN 'HTTP'
							-- ✅ Redis
							WHEN name IN ('PING', 'GET', 'SET', 'HGET', 'HSET', 'SETEX', 'DEL', 'EXISTS', 'LPUSH', 'RPOP', 'INFO', 'CLIENT', 'EVALSHA') THEN 'Redis'
							-- ✅ Database (SQL)
							WHEN length(attributes_string['db.system.name']) > 0 THEN
								CASE
									WHEN attributes_string['db.system.name'] = 'redis' THEN 'Redis'
									WHEN attributes_string['db.system.name'] = 'mysql' THEN 'SQL'
									WHEN attributes_string['db.system.name'] = 'postgresql' THEN 'SQL'
									ELSE 'SQL'
								END
							WHEN length(attributes_string['db.system']) > 0 THEN 'SQL'
							-- ✅ Messaging Queue
							WHEN length(attributes_string['messaging.system']) > 0 THEN
								CASE
									WHEN attributes_string['messaging.system'] = 'kafka' THEN 'Kafka'
									WHEN attributes_string['messaging.system'] = 'rabbitmq' THEN 'RabbitMQ'
									ELSE 'Messaging'
								END
							ELSE 'UNKNOWN'
						END,
						''
					),
					'UNKNOWN'
				) AS protocol
			FROM filtered_base
			WHERE src_workload = ?  -- 미리 계산된 값으로 필터링
				AND length(src_workload) > 0
		)

		-- 3단계: 최종 결과 조합
		SELECT
			protocol,

			-- ✅ NULL 처리된 status 계산
			CAST(
				COALESCE(
					CASE
						WHEN protocol = 'HTTP' THEN attributes_number['http.response.status_code']
						WHEN protocol = 'gRPC' THEN attributes_number['rpc.grpc.status_code']
						WHEN protocol = 'Redis' THEN status_code
						WHEN protocol = 'SQL' THEN status_code
						ELSE NULL
					END,
					0  -- ✅ NULL일 때 기본값 0
				) AS UInt16
			) AS status,

			-- ✅ OpenTelemetry Span Status + 프로토콜별 에러 처리
			CAST(
				COALESCE(
					CASE
						-- 🔴 최우선: OpenTelemetry span status 체크
						WHEN status_code = 2 THEN 1  -- STATUS_CODE_ERROR
						-- 프로토콜별 status code 체크
						WHEN protocol = 'HTTP' AND status >= 400 THEN 1
						WHEN protocol = 'gRPC' AND status != 0 THEN 1
						WHEN protocol = 'Redis' AND status != 0 THEN 1
						WHEN protocol = 'SQL' AND status != 0 THEN 1
						ELSE 0
					END,
					0  -- ✅ NULL일 때 기본값 0
				) AS UInt16
			) AS is_error,

			timestamp,
			trace_id,

			-- ✅ 모든 프로토콜 method 처리
			COALESCE(
				nullIf(
					CASE
						WHEN protocol = 'gRPC' THEN 'gRPC'
						WHEN protocol = 'HTTP' THEN splitByChar(' ', name)[1]
						WHEN protocol = 'SQL' THEN 'SQL'
						WHEN protocol = 'Redis' THEN 'Redis'
					END,
					''
				),
				''
			) AS method,

			-- ✅ 모든 프로토콜 path 처리
			CASE
				WHEN protocol = 'gRPC' THEN attributes_string['rpc.method']
				WHEN protocol = 'HTTP' THEN splitByChar(' ', name)[2]
				WHEN protocol = 'SQL' THEN name
				WHEN protocol = 'Redis' THEN name
			END AS path,

			duration_nano / 1000000 AS latency_ms,
			src_workload,

			multiIf(
				-- ✅ 조건 1: IP 주소 패턴인가
				match(server_addr, '^[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+(:[0-9]+){0,1}$'),

				-- ✅ 조건 1 참일 때: Dictionary 조회
				dictGetOrDefault(
					'signoz_traces.pod_workload_map_dict',
					'workload_name',
					tuple(
						replaceRegexpOne(server_addr, ':[0-9]+$', ''),
						resources_string['k8s.cluster.name']
					),
					replaceRegexpOne(server_addr, ':[0-9]+$', '')  -- ❌ 못 찾으면 IP 반환
				),

				-- ✅ 조건 2: FQDN 형식인가 (점이 있는가)
				position(server_addr, '.') > 0,

				-- ✅ 조건 2 참일 때: 첫 번째 세그먼트만 추출
				splitByChar('.', replaceRegexpOne(server_addr, ':[0-9]+$', ''))[1],

				-- ✅ 위 조건 모두 거짓일 때: 짧은 이름 (포트만 제거)
				replaceRegexpOne(server_addr, ':[0-9]+$', '')
			) AS dest_workload,

			? AS k8s_cluster_name,
			? AS src_namespace,
			? AS dest_namespace

		FROM protocol_classified
		-- ✅ dest_raw 값으로 직접 매칭 (정확한 server_addr 값)
		WHERE server_addr = ?
		ORDER BY latency_ms DESC
		LIMIT ?
		;
	`
	return query
}

// BuildQueryParamsTopSlowRequest I2I TopSlowRequest 쿼리 파라미터
// dstWorkloadRaw: network_map_connections.dest_raw 값 (signoz_index_v3.server_addr와 매칭)
func BuildQueryParamsTopSlowRequest(startTime, endTime, srcCluster, srcNamespace, srcWorkload, dstCluster, dstNamespace, dstWorkloadRaw string, limit int) []interface{} {
	var params []interface{}

	// clusterList := []string{srcCluster, dstCluster}
	// namespaceList := []string{srcNamespace, dstNamespace}  // ❌ 배열 사용 불가 (IN 절 파라미터 문제)
	// workloadList := []string{srcWorkload, dstWorkload}

	// ✅ dstWorkloadRaw: dest_raw 값으로 server_addr와 직접 매칭
	params = append(params, startTime, endTime, srcCluster, srcNamespace, dstNamespace, srcWorkload, srcCluster, srcNamespace, dstNamespace, dstWorkloadRaw, limit)

	return params
}

// EdgeTraceDetail 2. RecentError
func BuildQueryRecentError() string {
	query := `
	-- 1단계: 기본 필터링으로 데이터 대폭 줄이기
		WITH filtered_base AS (
			SELECT
				timestamp,
				trace_id,
				name,
				duration_nano,
				status_code,
				attributes_string,
				attributes_number,
				resources_string,
				external_http_url,
				http_host,
				-- 미리 src_workload 계산
				COALESCE(
					nullIf(resources_string['k8s.deployment.name'], ''),
					nullIf(resources_string['k8s.statefulset.name'], ''),
					nullIf(resources_string['k8s.daemonset.name'], ''),
					nullIf(resources_string['k8s.rollout.name'], ''),
					nullIf(resources_string['k8s.job.name'], '')
				) AS src_workload,
				-- 미리 server_address 정제
				attributes_string['server.address'] AS server_addr
			FROM signoz_traces.signoz_index_v3
			WHERE 1=1
				-- ✅ 가장 선택적인 조건부터 (시간 범위)
				AND timestamp >= ?
				AND timestamp <= ?
				-- ✅ 클러스터/네임스페이스 필터링
				AND resources_string['k8s.cluster.name'] = ?
				AND (resources_string['k8s.namespace.name'] = ? OR resources_string['k8s.namespace.name'] = ?)
				-- ✅ 기본 조건들 (복잡한 계산 전에)
				AND length(attributes_string['server.address']) > 0
				AND attributes_string['server.address'] NOT IN ('127.0.0.1', 'kubernetes.default')
                AND kind_string != 'Internal'
		),

		-- 2단계: 워크로드 필터링 및 프로토콜 분류
		protocol_classified AS (
			SELECT *,
				-- ✅ 완전한 프로토콜 분류 (SQL, Redis 포함)
				COALESCE(
					nullIf(
						CASE
							-- gRPC
							WHEN attributes_string['rpc.system'] = 'grpc' THEN 'gRPC'
							-- HTTP
							WHEN attributes_string['http.scheme'] = 'http' THEN 'HTTP'
							WHEN length(external_http_url) > 0 THEN 'HTTP'
							WHEN name REGEXP '^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\\s+/' THEN 'HTTP'
							-- ✅ Redis
							WHEN name IN ('PING', 'GET', 'SET', 'HGET', 'HSET', 'SETEX', 'DEL', 'EXISTS', 'LPUSH', 'RPOP', 'INFO', 'CLIENT', 'EVALSHA') THEN 'Redis'
							-- ✅ Database (SQL)
							WHEN length(attributes_string['db.system.name']) > 0 THEN
								CASE
									WHEN attributes_string['db.system.name'] = 'redis' THEN 'Redis'
									WHEN attributes_string['db.system.name'] = 'mysql' THEN 'SQL'
									WHEN attributes_string['db.system.name'] = 'postgresql' THEN 'SQL'
									ELSE 'SQL'
								END
							WHEN length(attributes_string['db.system']) > 0 THEN 'SQL'
							-- ✅ Messaging Queue
							WHEN length(attributes_string['messaging.system']) > 0 THEN
								CASE
									WHEN attributes_string['messaging.system'] = 'kafka' THEN 'Kafka'
									WHEN attributes_string['messaging.system'] = 'rabbitmq' THEN 'RabbitMQ'
									ELSE 'Messaging'
								END
							ELSE 'UNKNOWN'
						END,
						''
					),
					'UNKNOWN'
				) AS protocol
			FROM filtered_base
			WHERE src_workload = ?  -- 미리 계산된 값으로 필터링
				AND length(src_workload) > 0
		)

		-- 3단계: 최종 결과 조합
		SELECT
			protocol,

			-- ✅ NULL 처리된 status 계산
			CAST(
				COALESCE(
					CASE
						WHEN protocol = 'HTTP' THEN attributes_number['http.response.status_code']
						WHEN protocol = 'gRPC' THEN attributes_number['rpc.grpc.status_code']
						WHEN protocol = 'Redis' THEN status_code
						WHEN protocol = 'SQL' THEN status_code
						ELSE NULL
					END,
					0  -- ✅ NULL일 때 기본값 0
				) AS UInt16
			) AS status,

			-- ✅ OpenTelemetry Span Status + 프로토콜별 에러 처리
			CAST(
				COALESCE(
					CASE
						-- 🔴 최우선: OpenTelemetry span status 체크
						WHEN status_code = 2 THEN 1  -- STATUS_CODE_ERROR
						-- 프로토콜별 status code 체크
						WHEN protocol = 'HTTP' AND status >= 400 THEN 1
						WHEN protocol = 'gRPC' AND status != 0 THEN 1
						WHEN protocol = 'Redis' AND status != 0 THEN 1
						WHEN protocol = 'SQL' AND status != 0 THEN 1
						ELSE 0
					END,
					0  -- ✅ NULL일 때 기본값 0
				) AS UInt16
			) AS is_error,

			timestamp,
			trace_id,

			-- ✅ 모든 프로토콜 method 처리
			COALESCE(
				nullIf(
					CASE
						WHEN protocol = 'gRPC' THEN 'gRPC'
						WHEN protocol = 'HTTP' THEN splitByChar(' ', name)[1]
						WHEN protocol = 'SQL' THEN 'SQL'
						WHEN protocol = 'Redis' THEN 'Redis'
					END,
					''
				),
				''
			) AS method,

			-- ✅ 모든 프로토콜 path 처리
			CASE
				WHEN protocol = 'gRPC' THEN attributes_string['rpc.method']
				WHEN protocol = 'HTTP' THEN splitByChar(' ', name)[2]
				WHEN protocol = 'SQL' THEN name
				WHEN protocol = 'Redis' THEN name
			END AS path,

			duration_nano / 1000000 AS latency_ms,
			src_workload,

			multiIf(
				-- ✅ 조건 1: IP 주소 패턴인가
				match(server_addr, '^[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+(:[0-9]+){0,1}$'),

				-- ✅ 조건 1 참일 때: Dictionary 조회
				dictGetOrDefault(
					'signoz_traces.pod_workload_map_dict',
					'workload_name',
					tuple(
						replaceRegexpOne(server_addr, ':[0-9]+$', ''),
						resources_string['k8s.cluster.name']
					),
					replaceRegexpOne(server_addr, ':[0-9]+$', '')  -- ❌ 못 찾으면 IP 반환
				),

				-- ✅ 조건 2: FQDN 형식인가 (점이 있는가)
				position(server_addr, '.') > 0,

				-- ✅ 조건 2 참일 때: 첫 번째 세그먼트만 추출
				splitByChar('.', replaceRegexpOne(server_addr, ':[0-9]+$', ''))[1],

				-- ✅ 위 조건 모두 거짓일 때: 짧은 이름 (포트만 제거)
				replaceRegexpOne(server_addr, ':[0-9]+$', '')
			) AS dest_workload,

			? AS k8s_cluster_name,
			? AS src_namespace,
			? AS dest_namespace

		FROM protocol_classified
		WHERE 1=1
          -- ✅ dest_raw 값으로 직접 매칭 (정확한 server_addr 값)
          AND server_addr = ?
          AND is_error != 0
		ORDER BY latency_ms DESC
		LIMIT ?;
	`
	return query
}

// BuildQueryParamsRecentError I2I RecentError 쿼리 파라미터
// dstWorkloadRaw: network_map_connections.dest_raw 값 (signoz_index_v3.server_addr와 매칭)
func BuildQueryParamsRecentError(startTime, endTime, srcCluster, srcNamespace, srcWorkload, dstCluster, dstNamespace, dstWorkloadRaw string, limit int) []interface{} {
	var params []interface{}

	// clusterList := []string{srcCluster, dstCluster}
	// namespaceList := []string{srcNamespace, dstNamespace}  // ❌ 배열 사용 불가 (IN 절 파라미터 문제)
	// workloadList := []string{srcWorkload, dstWorkload}

	// ✅ dstWorkloadRaw: dest_raw 값으로 server_addr와 직접 매칭
	params = append(params, startTime, endTime, srcCluster, srcNamespace, dstNamespace, srcWorkload, srcCluster, srcNamespace, dstNamespace, dstWorkloadRaw, limit)

	return params
}

// EdgeTraceDetail 3. Requests
func BuildQueryRequests() string {
	query := `
		-- 1단계: 기본 필터링으로 데이터 대폭 줄이기
		WITH filtered_base AS (
			SELECT
				timestamp,
				trace_id,
				name,
				duration_nano,
				status_code,
				attributes_string,
				attributes_number,
				resources_string,
				external_http_url,
				http_host,
				-- 미리 src_workload 계산
				COALESCE(
					nullIf(resources_string['k8s.deployment.name'], ''),
					nullIf(resources_string['k8s.statefulset.name'], ''),
					nullIf(resources_string['k8s.daemonset.name'], ''),
					nullIf(resources_string['k8s.rollout.name'], ''),
					nullIf(resources_string['k8s.job.name'], '')
				) AS src_workload,
				-- 미리 server_address 정제
				attributes_string['server.address'] AS server_addr
			FROM signoz_traces.signoz_index_v3
			WHERE 1=1
				-- ✅ 가장 선택적인 조건부터 (시간 범위)
				AND timestamp >= ?
				AND timestamp <= ?
				-- ✅ 클러스터/네임스페이스 필터링
				AND resources_string['k8s.cluster.name'] = ?
				AND (resources_string['k8s.namespace.name'] = ? OR resources_string['k8s.namespace.name'] = ?)
				-- ✅ 기본 조건들 (복잡한 계산 전에)
				AND length(attributes_string['server.address']) > 0
				AND attributes_string['server.address'] NOT IN ('127.0.0.1', 'kubernetes.default')
		),

		-- 2단계: 워크로드 필터링 및 프로토콜 분류
		protocol_classified AS (
			SELECT *,
				-- ✅ 완전한 프로토콜 분류 (SQL, Redis 포함)
				COALESCE(
					nullIf(
						CASE
							-- gRPC
							WHEN attributes_string['rpc.system'] = 'grpc' THEN 'gRPC'
							-- HTTP
							WHEN attributes_string['http.scheme'] = 'http' THEN 'HTTP'
							WHEN length(external_http_url) > 0 THEN 'HTTP'
							WHEN name REGEXP '^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\\s+/' THEN 'HTTP'
							-- ✅ Redis
							WHEN name IN ('PING', 'GET', 'SET', 'HGET', 'HSET', 'SETEX', 'DEL', 'EXISTS', 'LPUSH', 'RPOP', 'INFO', 'CLIENT', 'EVALSHA') THEN 'Redis'
							-- ✅ Database (SQL)
							WHEN length(attributes_string['db.system.name']) > 0 THEN
								CASE
									WHEN attributes_string['db.system.name'] = 'redis' THEN 'Redis'
									WHEN attributes_string['db.system.name'] = 'mysql' THEN 'SQL'
									WHEN attributes_string['db.system.name'] = 'postgresql' THEN 'SQL'
									ELSE 'SQL'
								END
							WHEN length(attributes_string['db.system']) > 0 THEN 'SQL'
							-- ✅ Messaging Queue
							WHEN length(attributes_string['messaging.system']) > 0 THEN
								CASE
									WHEN attributes_string['messaging.system'] = 'kafka' THEN 'Kafka'
									WHEN attributes_string['messaging.system'] = 'rabbitmq' THEN 'RabbitMQ'
									ELSE 'Messaging'
								END
							ELSE 'UNKNOWN'
						END,
						''
					),
					'UNKNOWN'
				) AS protocol
			FROM filtered_base
			WHERE src_workload = ?  -- 미리 계산된 값으로 필터링
				AND length(src_workload) > 0
		)

		-- 3단계: 최종 결과 조합
		SELECT
			protocol,

			-- ✅ NULL 처리된 status 계산
			CAST(
				COALESCE(
					CASE
						WHEN protocol = 'HTTP' THEN attributes_number['http.response.status_code']
						WHEN protocol = 'gRPC' THEN attributes_number['rpc.grpc.status_code']
						WHEN protocol = 'Redis' THEN status_code
						WHEN protocol = 'SQL' THEN status_code
						ELSE NULL
					END,
					0  -- ✅ NULL일 때 기본값 0
				) AS UInt16
			) AS status,

			-- ✅ OpenTelemetry Span Status + 프로토콜별 에러 처리
			CAST(
				COALESCE(
					CASE
						-- 🔴 최우선: OpenTelemetry span status 체크
						WHEN status_code = 2 THEN 1  -- STATUS_CODE_ERROR
						-- 프로토콜별 status code 체크
						WHEN protocol = 'HTTP' AND status >= 400 THEN 1
						WHEN protocol = 'gRPC' AND status != 0 THEN 1
						WHEN protocol = 'Redis' AND status != 0 THEN 1
						WHEN protocol = 'SQL' AND status != 0 THEN 1
						ELSE 0
					END,
					0  -- ✅ NULL일 때 기본값 0
				) AS UInt16
			) AS is_error,

			timestamp,
			trace_id,

			-- ✅ 모든 프로토콜 method 처리
			COALESCE(
				nullIf(
					CASE
						WHEN protocol = 'gRPC' THEN 'gRPC'
						WHEN protocol = 'HTTP' THEN splitByChar(' ', name)[1]
						WHEN protocol = 'SQL' THEN 'SQL'
						WHEN protocol = 'Redis' THEN 'Redis'
					END,
					''
				),
				''
			) AS method,

			-- ✅ 모든 프로토콜 path 처리
			CASE
				WHEN protocol = 'gRPC' THEN attributes_string['rpc.method']
				WHEN protocol = 'HTTP' THEN splitByChar(' ', name)[2]
				WHEN protocol = 'SQL' THEN name
				WHEN protocol = 'Redis' THEN name
			END AS path,

			duration_nano / 1000000 AS latency_ms,
			src_workload,

			multiIf(
				-- ✅ 조건 1: IP 주소 패턴인가
				match(server_addr, '^[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+(:[0-9]+){0,1}$'),

				-- ✅ 조건 1 참일 때: Dictionary 조회
				dictGetOrDefault(
					'signoz_traces.pod_workload_map_dict',
					'workload_name',
					tuple(
						replaceRegexpOne(server_addr, ':[0-9]+$', ''),
						resources_string['k8s.cluster.name']
					),
					replaceRegexpOne(server_addr, ':[0-9]+$', '')  -- ❌ 못 찾으면 IP 반환
				),

				-- ✅ 조건 2: FQDN 형식인가 (점이 있는가)
				position(server_addr, '.') > 0,

				-- ✅ 조건 2 참일 때: 첫 번째 세그먼트만 추출
				splitByChar('.', replaceRegexpOne(server_addr, ':[0-9]+$', ''))[1],

				-- ✅ 위 조건 모두 거짓일 때: 짧은 이름 (포트만 제거)
				replaceRegexpOne(server_addr, ':[0-9]+$', '')
			) AS dest_workload,

			? AS k8s_cluster_name,
			? AS src_namespace,
			? AS dest_namespace

		FROM protocol_classified
		-- ✅ dest_raw 값으로 직접 매칭 (정확한 server_addr 값)
		WHERE server_addr = ?
		ORDER BY timestamp DESC
		LIMIT 100 -- TODO 임시 설정
		;
	`
	return query
}

// BuildQueryParamsRequests I2I Requests 쿼리 파라미터
// dstWorkloadRaw: network_map_connections.dest_raw 값 (signoz_index_v3.server_addr와 매칭)
func BuildQueryParamsRequests(startTime, endTime, srcCluster, srcNamespace, srcWorkload, dstCluster, dstNamespace, dstWorkloadRaw string) []interface{} {
	var params []interface{}

	// clusterList := []string{srcCluster, dstCluster}
	// namespaceList := []string{srcNamespace, dstNamespace}  // ❌ 배열 사용 불가 (IN 절 파라미터 문제)
	// workloadList := []string{srcWorkload, dstWorkload}

	// ✅ dstWorkloadRaw: dest_raw 값으로 server_addr와 직접 매칭
	params = append(params, startTime, endTime, srcCluster, srcNamespace, dstNamespace, srcWorkload, srcCluster, srcNamespace, dstNamespace, dstWorkloadRaw)

	return params
}

// ==================================================================================== //
// Query B: Internal to External (내부 → 외부 서비스)
// ==================================================================================== //

// EdgeTraceDetail 1. TopSlowRequest
func BuildQueryTopSlowInternalToExternal() string {
	query := `
		-- 1단계: 기본 필터링으로 데이터 대폭 줄이기
		WITH filtered_base AS (
			SELECT
				timestamp,
				trace_id,
				name,
				duration_nano,
				status_code,
				attributes_string,
				attributes_number,
				resources_string,
				external_http_url,
				http_host,
				-- 미리 src_workload 계산
				COALESCE(
					nullIf(resources_string['k8s.deployment.name'], ''),
					nullIf(resources_string['k8s.statefulset.name'], ''),
					nullIf(resources_string['k8s.daemonset.name'], ''),
					nullIf(resources_string['k8s.rollout.name'], ''),
					nullIf(resources_string['k8s.job.name'], '')
				) AS src_workload,
				-- 미리 server_address 정제
				attributes_string['server.address'] AS server_addr
			FROM signoz_traces.signoz_index_v3
			WHERE 1=1
				-- ✅ 가장 선택적인 조건부터 (시간 범위)
				AND timestamp >= ?
				AND timestamp <= ?
				-- ✅ 클러스터/네임스페이스 필터링
				AND resources_string['k8s.cluster.name'] = ?
				AND (resources_string['k8s.namespace.name'] = ? OR resources_string['k8s.namespace.name'] = ?)
				-- ✅ 기본 조건들 (복잡한 계산 전에)
				AND length(attributes_string['server.address']) > 0
				AND attributes_string['server.address'] NOT IN ('127.0.0.1', 'kubernetes.default')
		),

		-- 2단계: 워크로드 필터링 및 프로토콜 분류
		protocol_classified AS (
			SELECT *,
				-- ✅ 완전한 프로토콜 분류 (SQL, Redis 포함)
				COALESCE(
					nullIf(
						CASE
							-- gRPC
							WHEN attributes_string['rpc.system'] = 'grpc' THEN 'gRPC'
							-- HTTP
							WHEN attributes_string['http.scheme'] = 'http' THEN 'HTTP'
							WHEN length(external_http_url) > 0 THEN 'HTTP'
							WHEN name REGEXP '^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\\s+/' THEN 'HTTP'
							-- ✅ Redis
							WHEN name IN ('PING', 'GET', 'SET', 'HGET', 'HSET', 'SETEX', 'DEL', 'EXISTS', 'LPUSH', 'RPOP', 'INFO', 'CLIENT', 'EVALSHA') THEN 'Redis'
							-- ✅ Database (SQL)
							WHEN length(attributes_string['db.system.name']) > 0 THEN
								CASE
									WHEN attributes_string['db.system.name'] = 'redis' THEN 'Redis'
									WHEN attributes_string['db.system.name'] = 'mysql' THEN 'SQL'
									WHEN attributes_string['db.system.name'] = 'postgresql' THEN 'SQL'
									ELSE 'SQL'
								END
							WHEN length(attributes_string['db.system']) > 0 THEN 'SQL'
							-- ✅ Messaging Queue
							WHEN length(attributes_string['messaging.system']) > 0 THEN
								CASE
									WHEN attributes_string['messaging.system'] = 'kafka' THEN 'Kafka'
									WHEN attributes_string['messaging.system'] = 'rabbitmq' THEN 'RabbitMQ'
									ELSE 'Messaging'
								END
							ELSE 'UNKNOWN'
						END,
						''
					),
					'UNKNOWN'
				) AS protocol
			FROM filtered_base
			WHERE src_workload = ?  -- 미리 계산된 값으로 필터링
				AND length(src_workload) > 0
		)

		-- 3단계: 최종 결과 조합
		SELECT
			protocol,

			-- ✅ NULL 처리된 status 계산
			CAST(
				COALESCE(
					CASE
						WHEN protocol = 'HTTP' THEN attributes_number['http.response.status_code']
						WHEN protocol = 'gRPC' THEN attributes_number['rpc.grpc.status_code']
						WHEN protocol = 'Redis' THEN status_code
						WHEN protocol = 'SQL' THEN status_code
						ELSE NULL
					END,
					0  -- ✅ NULL일 때 기본값 0
				) AS UInt16
			) AS status,

			-- ✅ OpenTelemetry Span Status + 프로토콜별 에러 처리
			CAST(
				COALESCE(
					CASE
						-- 🔴 최우선: OpenTelemetry span status 체크
						WHEN status_code = 2 THEN 1  -- STATUS_CODE_ERROR
						-- 프로토콜별 status code 체크
						WHEN protocol = 'HTTP' AND status >= 400 THEN 1
						WHEN protocol = 'gRPC' AND status != 0 THEN 1
						WHEN protocol = 'Redis' AND status != 0 THEN 1
						WHEN protocol = 'SQL' AND status != 0 THEN 1
						ELSE 0
					END,
					0  -- ✅ NULL일 때 기본값 0
				) AS UInt16
			) AS is_error,

			timestamp,
			trace_id,

			-- ✅ 모든 프로토콜 method 처리
			COALESCE(
				nullIf(
					CASE
						WHEN protocol = 'gRPC' THEN 'gRPC'
						WHEN protocol = 'HTTP' THEN splitByChar(' ', name)[1]
						WHEN protocol = 'SQL' THEN 'SQL'
						WHEN protocol = 'Redis' THEN 'Redis'
					END,
					''
				),
				''
			) AS method,

			-- ✅ 모든 프로토콜 path 처리
			CASE
				WHEN protocol = 'gRPC' THEN attributes_string['rpc.method']
				WHEN protocol = 'HTTP' THEN splitByChar(' ', name)[2]
				WHEN protocol = 'SQL' THEN name
				WHEN protocol = 'Redis' THEN name
			END AS path,

			duration_nano / 1000000 AS latency_ms,
			src_workload,

			server_addr AS dest_workload,

			? AS k8s_cluster_name,
			? AS src_namespace,
			? AS dest_namespace

		FROM protocol_classified
		WHERE src_workload = ?  -- 내부 워크로드 필터
		  -- ✅ 핵심: 외부 서비스는 원본 필드로 매칭
		  AND (
			  server_addr = ?              -- 'grafana.com'
			  OR external_http_url = ?     -- 'grafana.com'
			  OR http_host = ?             -- 'grafana.com'
		  )
		ORDER BY latency_ms DESC
		LIMIT ?
		;
	`
	return query
}

func BuildQueryParamsTopSlowInternalToExternal(startTime, endTime, srcCluster, srcNamespace, srcWorkload, dstCluster, dstNamespace, dstWorkload string, limit int) []interface{} {
	var params []interface{}

	params = append(params,
		// filtered_base WHERE
		startTime, endTime,
		srcCluster,
		srcNamespace, dstNamespace,

		// protocol_classified WHERE
		srcWorkload,

		// 최종 SELECT
		srcCluster, srcNamespace, dstNamespace,

		// 최종 WHERE 조건
		srcWorkload,                           // src 필터
		dstWorkload, dstWorkload, dstWorkload, // dest 필터 (3개 OR 조건: server_addr, external_http_url, http_host)
		limit)

	return params
}

// EdgeTraceDetail 2. RecentError
func BuildQueryRecentErrorInternalToExternal() string {
	query := `
		-- 1단계: 기본 필터링으로 데이터 대폭 줄이기
		WITH filtered_base AS (
			SELECT
				timestamp,
				trace_id,
				name,
				duration_nano,
				status_code,
				attributes_string,
				attributes_number,
				resources_string,
				external_http_url,
				http_host,
				-- 미리 src_workload 계산
				COALESCE(
					nullIf(resources_string['k8s.deployment.name'], ''),
					nullIf(resources_string['k8s.statefulset.name'], ''),
					nullIf(resources_string['k8s.daemonset.name'], ''),
					nullIf(resources_string['k8s.rollout.name'], ''),
					nullIf(resources_string['k8s.job.name'], '')
				) AS src_workload,
				-- 미리 server_address 정제
				attributes_string['server.address'] AS server_addr
			FROM signoz_traces.signoz_index_v3
			WHERE 1=1
				-- ✅ 가장 선택적인 조건부터 (시간 범위)
				AND timestamp >= ?
				AND timestamp <= ?
				-- ✅ 클러스터/네임스페이스 필터링
				AND resources_string['k8s.cluster.name'] = ?
				AND (resources_string['k8s.namespace.name'] = ? OR resources_string['k8s.namespace.name'] = ?)
				-- ✅ 기본 조건들 (복잡한 계산 전에)
				AND length(attributes_string['server.address']) > 0
				AND attributes_string['server.address'] NOT IN ('127.0.0.1', 'kubernetes.default')
		),

		-- 2단계: 워크로드 필터링 및 프로토콜 분류
		protocol_classified AS (
			SELECT *,
				-- ✅ 완전한 프로토콜 분류 (SQL, Redis 포함)
				COALESCE(
					nullIf(
						CASE
							-- gRPC
							WHEN attributes_string['rpc.system'] = 'grpc' THEN 'gRPC'
							-- HTTP
							WHEN attributes_string['http.scheme'] = 'http' THEN 'HTTP'
							WHEN length(external_http_url) > 0 THEN 'HTTP'
							WHEN name REGEXP '^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\\s+/' THEN 'HTTP'
							-- ✅ Redis
							WHEN name IN ('PING', 'GET', 'SET', 'HGET', 'HSET', 'SETEX', 'DEL', 'EXISTS', 'LPUSH', 'RPOP', 'INFO', 'CLIENT', 'EVALSHA') THEN 'Redis'
							-- ✅ Database (SQL)
							WHEN length(attributes_string['db.system.name']) > 0 THEN
								CASE
									WHEN attributes_string['db.system.name'] = 'redis' THEN 'Redis'
									WHEN attributes_string['db.system.name'] = 'mysql' THEN 'SQL'
									WHEN attributes_string['db.system.name'] = 'postgresql' THEN 'SQL'
									ELSE 'SQL'
								END
							WHEN length(attributes_string['db.system']) > 0 THEN 'SQL'
							-- ✅ Messaging Queue
							WHEN length(attributes_string['messaging.system']) > 0 THEN
								CASE
									WHEN attributes_string['messaging.system'] = 'kafka' THEN 'Kafka'
									WHEN attributes_string['messaging.system'] = 'rabbitmq' THEN 'RabbitMQ'
									ELSE 'Messaging'
								END
							ELSE 'UNKNOWN'
						END,
						''
					),
					'UNKNOWN'
				) AS protocol
			FROM filtered_base
			WHERE src_workload = ?  -- 미리 계산된 값으로 필터링
				AND length(src_workload) > 0
		)

		-- 3단계: 최종 결과 조합
		SELECT
			protocol,

			-- ✅ NULL 처리된 status 계산
			CAST(
				COALESCE(
					CASE
						WHEN protocol = 'HTTP' THEN attributes_number['http.response.status_code']
						WHEN protocol = 'gRPC' THEN attributes_number['rpc.grpc.status_code']
						WHEN protocol = 'Redis' THEN status_code
						WHEN protocol = 'SQL' THEN status_code
						ELSE NULL
					END,
					0  -- ✅ NULL일 때 기본값 0
				) AS UInt16
			) AS status,

			-- ✅ OpenTelemetry Span Status + 프로토콜별 에러 처리
			CAST(
				COALESCE(
					CASE
						-- 🔴 최우선: OpenTelemetry span status 체크
						WHEN status_code = 2 THEN 1  -- STATUS_CODE_ERROR
						-- 프로토콜별 status code 체크
						WHEN protocol = 'HTTP' AND status >= 400 THEN 1
						WHEN protocol = 'gRPC' AND status != 0 THEN 1
						WHEN protocol = 'Redis' AND status != 0 THEN 1
						WHEN protocol = 'SQL' AND status != 0 THEN 1
						ELSE 0
					END,
					0  -- ✅ NULL일 때 기본값 0
				) AS UInt16
			) AS is_error,

			timestamp,
			trace_id,

			-- ✅ 모든 프로토콜 method 처리
			COALESCE(
				nullIf(
					CASE
						WHEN protocol = 'gRPC' THEN 'gRPC'
						WHEN protocol = 'HTTP' THEN splitByChar(' ', name)[1]
						WHEN protocol = 'SQL' THEN 'SQL'
						WHEN protocol = 'Redis' THEN 'Redis'
					END,
					''
				),
				''
			) AS method,

			-- ✅ 모든 프로토콜 path 처리
			CASE
				WHEN protocol = 'gRPC' THEN attributes_string['rpc.method']
				WHEN protocol = 'HTTP' THEN splitByChar(' ', name)[2]
				WHEN protocol = 'SQL' THEN name
				WHEN protocol = 'Redis' THEN name
			END AS path,

			duration_nano / 1000000 AS latency_ms,
			src_workload,

			server_addr AS dest_workload,

			? AS k8s_cluster_name,
			? AS src_namespace,
			? AS dest_namespace

		FROM protocol_classified
		WHERE src_workload = ?  -- 내부 워크로드 필터
		  -- ✅ 핵심: 외부 서비스는 원본 필드로 매칭
		  AND (
			  server_addr = ?
			  OR external_http_url = ?
			  OR http_host = ?
		  )
		  AND is_error != 0  -- ✅ 에러만 필터링
		ORDER BY timestamp DESC
		LIMIT ?
		;
	`
	return query
}

func BuildQueryParamsRecentErrorInternalToExternal(startTime, endTime, srcCluster, srcNamespace, srcWorkload, dstCluster, dstNamespace, dstWorkload string, limit int) []interface{} {
	var params []interface{}

	params = append(params,
		// filtered_base WHERE
		startTime, endTime,
		srcCluster,
		srcNamespace, dstNamespace,

		// protocol_classified WHERE
		srcWorkload,

		// 최종 SELECT
		srcCluster, srcNamespace, dstNamespace,

		// 최종 WHERE 조건
		srcWorkload,                           // src 필터
		dstWorkload, dstWorkload, dstWorkload, // dest 필터 (3개 OR 조건: server_addr, external_http_url, http_host)
		limit)

	return params
}

// EdgeTraceDetail 3. Requests
func BuildQueryRequestsInternalToExternal() string {
	query := `
		-- 1단계: 기본 필터링으로 데이터 대폭 줄이기
		WITH filtered_base AS (
			SELECT
				timestamp,
				trace_id,
				name,
				duration_nano,
				status_code,
				attributes_string,
				attributes_number,
				resources_string,
				external_http_url,
				http_host,
				-- 미리 src_workload 계산
				COALESCE(
					nullIf(resources_string['k8s.deployment.name'], ''),
					nullIf(resources_string['k8s.statefulset.name'], ''),
					nullIf(resources_string['k8s.daemonset.name'], ''),
					nullIf(resources_string['k8s.rollout.name'], ''),
					nullIf(resources_string['k8s.job.name'], '')
				) AS src_workload,
				-- 미리 server_address 정제
				attributes_string['server.address'] AS server_addr
			FROM signoz_traces.signoz_index_v3
			WHERE 1=1
				-- ✅ 가장 선택적인 조건부터 (시간 범위)
				AND timestamp >= ?
				AND timestamp <= ?
				-- ✅ 클러스터/네임스페이스 필터링
				AND resources_string['k8s.cluster.name'] = ?
				AND (resources_string['k8s.namespace.name'] = ? OR resources_string['k8s.namespace.name'] = ?)
				-- ✅ 기본 조건들 (복잡한 계산 전에)
				AND length(attributes_string['server.address']) > 0
				AND attributes_string['server.address'] NOT IN ('127.0.0.1', 'kubernetes.default')
		),

		-- 2단계: 워크로드 필터링 및 프로토콜 분류
		protocol_classified AS (
			SELECT *,
				-- ✅ 완전한 프로토콜 분류 (SQL, Redis 포함)
				COALESCE(
					nullIf(
						CASE
							-- gRPC
							WHEN attributes_string['rpc.system'] = 'grpc' THEN 'gRPC'
							-- HTTP
							WHEN attributes_string['http.scheme'] = 'http' THEN 'HTTP'
							WHEN length(external_http_url) > 0 THEN 'HTTP'
							WHEN name REGEXP '^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\\s+/' THEN 'HTTP'
							-- ✅ Redis
							WHEN name IN ('PING', 'GET', 'SET', 'HGET', 'HSET', 'SETEX', 'DEL', 'EXISTS', 'LPUSH', 'RPOP', 'INFO', 'CLIENT', 'EVALSHA') THEN 'Redis'
							-- ✅ Database (SQL)
							WHEN length(attributes_string['db.system.name']) > 0 THEN
								CASE
									WHEN attributes_string['db.system.name'] = 'redis' THEN 'Redis'
									WHEN attributes_string['db.system.name'] = 'mysql' THEN 'SQL'
									WHEN attributes_string['db.system.name'] = 'postgresql' THEN 'SQL'
									ELSE 'SQL'
								END
							WHEN length(attributes_string['db.system']) > 0 THEN 'SQL'
							-- ✅ Messaging Queue
							WHEN length(attributes_string['messaging.system']) > 0 THEN
								CASE
									WHEN attributes_string['messaging.system'] = 'kafka' THEN 'Kafka'
									WHEN attributes_string['messaging.system'] = 'rabbitmq' THEN 'RabbitMQ'
									ELSE 'Messaging'
								END
							ELSE 'UNKNOWN'
						END,
						''
					),
					'UNKNOWN'
				) AS protocol
			FROM filtered_base
			WHERE src_workload = ?  -- 미리 계산된 값으로 필터링
				AND length(src_workload) > 0
		)

		-- 3단계: 최종 결과 조합
		SELECT
			protocol,

			-- ✅ NULL 처리된 status 계산
			CAST(
				COALESCE(
					CASE
						WHEN protocol = 'HTTP' THEN attributes_number['http.response.status_code']
						WHEN protocol = 'gRPC' THEN attributes_number['rpc.grpc.status_code']
						WHEN protocol = 'Redis' THEN status_code
						WHEN protocol = 'SQL' THEN status_code
						ELSE NULL
					END,
					0  -- ✅ NULL일 때 기본값 0
				) AS UInt16
			) AS status,

			-- ✅ OpenTelemetry Span Status + 프로토콜별 에러 처리
			CAST(
				COALESCE(
					CASE
						-- 🔴 최우선: OpenTelemetry span status 체크
						WHEN status_code = 2 THEN 1  -- STATUS_CODE_ERROR
						-- 프로토콜별 status code 체크
						WHEN protocol = 'HTTP' AND status >= 400 THEN 1
						WHEN protocol = 'gRPC' AND status != 0 THEN 1
						WHEN protocol = 'Redis' AND status != 0 THEN 1
						WHEN protocol = 'SQL' AND status != 0 THEN 1
						ELSE 0
					END,
					0  -- ✅ NULL일 때 기본값 0
				) AS UInt16
			) AS is_error,

			timestamp,
			trace_id,

			-- ✅ 모든 프로토콜 method 처리
			COALESCE(
				nullIf(
					CASE
						WHEN protocol = 'gRPC' THEN 'gRPC'
						WHEN protocol = 'HTTP' THEN splitByChar(' ', name)[1]
						WHEN protocol = 'SQL' THEN 'SQL'
						WHEN protocol = 'Redis' THEN 'Redis'
					END,
					''
				),
				''
			) AS method,

			-- ✅ 모든 프로토콜 path 처리
			CASE
				WHEN protocol = 'gRPC' THEN attributes_string['rpc.method']
				WHEN protocol = 'HTTP' THEN splitByChar(' ', name)[2]
				WHEN protocol = 'SQL' THEN name
				WHEN protocol = 'Redis' THEN name
			END AS path,

			duration_nano / 1000000 AS latency_ms,
			src_workload,

			server_addr AS dest_workload,

			? AS k8s_cluster_name,
			? AS src_namespace,
			? AS dest_namespace

		FROM protocol_classified
		WHERE src_workload = ?  -- 내부 워크로드 필터
		  -- ✅ 핵심: 외부 서비스는 원본 필드로 매칭
		  AND (
			  server_addr = ?
			  OR external_http_url = ?
			  OR http_host = ?
		  )
		ORDER BY timestamp DESC
		LIMIT 100
		;
	`
	return query
}

func BuildQueryParamsRequestsInternalToExternal(startTime, endTime, srcCluster, srcNamespace, srcWorkload, dstCluster, dstNamespace, dstWorkload string) []interface{} {
	var params []interface{}

	params = append(params,
		// filtered_base WHERE
		startTime, endTime,
		srcCluster,
		srcNamespace, dstNamespace,

		// protocol_classified WHERE
		srcWorkload,

		// 최종 SELECT
		srcCluster, srcNamespace, dstNamespace,

		// 최종 WHERE 조건
		srcWorkload,                           // src 필터
		dstWorkload, dstWorkload, dstWorkload) // dest 필터 (3개 OR 조건: server_addr, external_http_url, http_host)

	return params
}
