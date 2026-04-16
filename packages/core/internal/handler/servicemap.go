package handler

import (
	"fmt"
	"net/http"

	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/domain/servicemap"
	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/service"
	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/pkg"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ServiceMapController interface {
	GetTopology() gin.HandlerFunc
	GetWorkloadHover() gin.HandlerFunc // 호버 전용 API (기존 TopPeers 대체)
	GetWorkloadDetails() gin.HandlerFunc
	GetEdgeTraceDetails() gin.HandlerFunc
}

type ServiceMapControllerImpl struct {
	serviceMapService service.ServiceMapService
}

var serviceMapControllerInstance ServiceMapController = nil

func ServiceMapControllerInstance() ServiceMapController {
	if serviceMapControllerInstance == nil {
		serviceMapControllerInstance = &ServiceMapControllerImpl{
			serviceMapService: service.ServiceMapServiceInstance(),
		}
	}
	return serviceMapControllerInstance
}

// @Summary Get ServiceMap Topology (POST 방식 - 대용량 멀티클러스터 지원)
// @Description This API gets the service map topology data for React Flow visualization. POST 방식으로 변경하여 대용량 클러스터/네임스페이스 필터를 지원합니다.
// @Tags ServiceMap
// @Accept json
// @Produce json
// @Param request body servicemap.TopologyRequest true "토폴로지 필터링 요청"
// @Success 200 {object} pkg.APIResponse
// @Failure 400 {object} pkg.APIResponse
// @Failure 401 {object} pkg.APIResponse
// @Failure 500 {object} pkg.APIResponse
// @Router /api/v1/servicemap/topology [post]
func (c *ServiceMapControllerImpl) GetTopology() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		logger := pkg.GetLogger()
		// logger.Info("GetTopology 요청 시작")

		// JSON 요청 바디 바인딩 (POST 방식으로 변경 - 대용량 멀티클러스터 필터 지원)
		var req servicemap.TopologyRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			logger.Error("Request body 바인딩 실패", zap.Error(err))
			pkg.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		// 필수 파라미터 검증 (StartTime/EndTime은 DTO binding:"required"로 검증됨)
		if req.StartTime == "" || req.EndTime == "" {
			pkg.ErrorResponse(ctx, http.StatusBadRequest, "startTime and endTime are required", fmt.Errorf("missing required time parameters"))
			return
		}

		logger.Info("토폴로지 요청 파라미터",
			zap.String("startTime", req.StartTime),
			zap.String("endTime", req.EndTime),
			zap.Strings("cluster", req.Cluster),
			zap.Strings("namespace", req.Namespace),
			zap.Strings("protocol", req.Protocol),
			zap.Strings("status", req.Status),
		)

		// 서비스 호출
		response, err := c.serviceMapService.GetTopology(&req)
		if err != nil {
			logger.Error("토폴로지 조회 실패", zap.Error(err))
			pkg.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to get topology", err)
			return
		}

		// logger.Info("토폴로지 조회 성공",
		// 	zap.Int("nodeCount", len(response.Nodes)),
		// 	zap.Int("edgeCount", len(response.Edges)),
		// )

		pkg.SuccessResponse(ctx, http.StatusOK, response, "Topology retrieved successfully")
	}
}

// @Summary Get Service Hover Info (호버용 최적화된 API)
// @Description This API gets service hover information including RED metrics and top peers for UI hover tooltip
// @Tags ServiceMap
// @Accept json
// @Produce json
// @Param request body servicemap.ServiceHoverRequest true "Service hover request with 3-tier identifier"
// @Success 200 {object} pkg.APIResponse
// @Failure 400 {object} pkg.APIResponse
// @Failure 401 {object} pkg.APIResponse
// @Failure 500 {object} pkg.APIResponse
// @Router /api/v1/servicemap/workload/hover-info [post]
func (c *ServiceMapControllerImpl) GetWorkloadHover() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		logger := pkg.GetLogger()
		// logger.Info("GetWorkloadHover 요청 시작")

		// JSON 바인딩
		var req servicemap.WorkloadHoverRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			logger.Error("Request body 바인딩 실패", zap.Error(err))
			pkg.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		// 3-tier 식별자 검증
		if req.Cluster == "" || req.Namespace == "" || req.WorkloadName == "" {
			pkg.ErrorResponse(ctx, http.StatusBadRequest, "cluster, namespace, and serviceName are required", fmt.Errorf("missing 3-tier identifier"))
			return
		}

		// 시간 범위 검증
		if req.StartTime == "" || req.EndTime == "" {
			pkg.ErrorResponse(ctx, http.StatusBadRequest, "startTime and endTime are required", fmt.Errorf("missing time range"))
			return
		}

		logger.Info("Workload Hover 요청 파라미터",
			zap.String("cluster", req.Cluster),
			zap.String("namespace", req.Namespace),
			zap.String("workloadName", req.WorkloadName),
			zap.String("startTime", req.StartTime),
			zap.String("endTime", req.EndTime),
		)

		// 서비스 호출
		response, err := c.serviceMapService.GetWorkloadHover(&req)
		if err != nil {
			logger.Error("Workload Hover 조회 실패", zap.Error(err))
			pkg.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to get workload hover info", err)
			return
		}

		// logger.Info("Workload Hover 조회 성공",
		// 	zap.String("cluster", req.Cluster),
		// 	zap.String("namespace", req.Namespace),
		// 	zap.String("workloadName", req.WorkloadName),
		// 	zap.Int("topPeerCount", len(response.TopPeers)),
		// )

		pkg.SuccessResponse(ctx, http.StatusOK, response, "Service hover info retrieved successfully")
	}
}

// @Summary Get Service Details
// @Description This API gets detailed information about a specific service using 3-tier identifier (cluster, namespace, serviceName)
// @Tags ServiceMap
// @Accept json
// @Produce json
// @Param request body servicemap.ServiceDetailRequest true "Service detail request with 3-tier identifier"
// @Success 200 {object} pkg.APIResponse
// @Failure 400 {object} pkg.APIResponse
// @Failure 401 {object} pkg.APIResponse
// @Failure 500 {object} pkg.APIResponse
// @Router /api/v1/servicemap/service/details [post]
func (c *ServiceMapControllerImpl) GetWorkloadDetails() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		logger := pkg.GetLogger()
		// logger.Info("GetWorkloadDetails 요청 시작")

		// JSON 바인딩
		var req servicemap.WorkloadDetailRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			logger.Error("Request body 바인딩 실패", zap.Error(err))
			pkg.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		// 3-tier 식별자 검증 (binding:"required"로도 검증되지만 명시적 추가)
		if req.Cluster == "" || req.Namespace == "" || req.WorkloadName == "" {
			pkg.ErrorResponse(ctx, http.StatusBadRequest, "cluster, namespace, and serviceName are required", fmt.Errorf("missing 3-tier identifier"))
			return
		}

		// 시간 범위 검증
		if req.StartTime == "" || req.EndTime == "" {
			pkg.ErrorResponse(ctx, http.StatusBadRequest, "startTime and endTime are required", fmt.Errorf("missing time range"))
			return
		}

		logger.Info("서비스 상세 요청 파라미터",
			zap.String("cluster", req.Cluster),
			zap.String("namespace", req.Namespace),
			zap.String("workloadName", req.WorkloadName),
			zap.String("startTime", req.StartTime),
			zap.String("endTime", req.EndTime),
		)

		// 서비스 호출
		response, err := c.serviceMapService.GetWorkloadDetails(&req)
		if err != nil {
			logger.Error("서비스 상세 조회 실패", zap.Error(err))
			pkg.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to get workload details", err)
			return
		}

		// logger.Info("서비스 상세 조회 성공",
		// 	zap.String("cluster", req.Cluster),
		// 	zap.String("namespace", req.Namespace),
		// 	zap.String("workloadName", req.WorkloadName),
		// )

		pkg.SuccessResponse(ctx, http.StatusOK, response, "Service details retrieved successfully")
	}
}

// @Summary Get Connection Details (POST 방식 - URL 인코딩 문제 해결)
// @Description This API gets detailed information about a specific connection. POST 방식으로 변경하여 ConnectionID의 특수문자 URL 인코딩 문제를 해결합니다.
// @Tags ServiceMap
// @Accept json
// @Produce json
// @Param request body servicemap.ConnectionDetailRequest true "연결 상세 정보 요청"
// @Success 200 {object} pkg.APIResponse
// @Failure 400 {object} pkg.APIResponse
// @Failure 401 {object} pkg.APIResponse
// @Failure 500 {object} pkg.APIResponse
// @Router /api/v1/servicemap/edge/trace/details [post]
func (c *ServiceMapControllerImpl) GetEdgeTraceDetails() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		logger := pkg.GetLogger()
		// logger.Info("GetEdgeTraceDetails 요청 시작")

		// JSON 요청 바디 바인딩 (POST 방식으로 변경 - URL 인코딩 문제 해결)
		var req servicemap.EdgeTraceDetailRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			logger.Error("Request body 바인딩 실패", zap.Error(err))
			pkg.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		// 필수 파라미터 검증
		if req.EdgeId == "" {
			pkg.ErrorResponse(ctx, http.StatusBadRequest, "edgeId is required", fmt.Errorf("missing edgeId parameter"))
			return
		}

		if req.StartTime == "" || req.EndTime == "" {
			pkg.ErrorResponse(ctx, http.StatusBadRequest, "startTime and endTime are required", fmt.Errorf("missing timeRange parameter"))
			return
		}

		logger.Info("연결 상세 요청 파라미터",
			zap.String("edgeId", req.EdgeId),
			zap.String("source", req.Source),
			zap.String("destination", req.Destination),
			zap.String("startTime", req.StartTime),
			zap.String("endTime", req.EndTime),
		)

		// 서비스 호출
		response, err := c.serviceMapService.GetEdgeTraceDetails(&req)
		if err != nil {
			logger.Error("edge trace details 조회 실패", zap.Error(err))
			pkg.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to get edge trace details", err)
			return
		}

		// logger.Info("edge trace details 조회 성공", zap.String("edgeId", req.EdgeId))

		pkg.SuccessResponse(ctx, http.StatusOK, response, "Connection details retrieved successfully")
	}
}
