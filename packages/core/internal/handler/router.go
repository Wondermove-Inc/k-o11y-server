package handler

import (
	"fmt"
	"log"

	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/config"
	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/pkg"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	httpSwagger "github.com/swaggo/http-swagger"
)

var routeDefault *gin.Engine
var routeGroup *gin.RouterGroup

func StartRouter() {
	// description: port는 자유롭게 지정해줍니다.
	cfg, _ := config.LoadConfig()
	port := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("Running on %s port\n", port)

	pkg.InitLogger()
	logger := pkg.GetLogger()

	// gin
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = pkg.NewZapWriter(logger)

	routeDefault = gin.New()

	// description: 정적 파일 제공 (Gin)
	routeDefault.Static("/swagger", "./swagger")
	routeDefault.GET("/swagger-ui/*any", gin.WrapH(httpSwagger.Handler(httpSwagger.URL(fmt.Sprintf("http://localhost%s/swagger/swagger.json", port)))))

	// description: CORS 관련 설정입니다
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	// description: 특정 origin만 허용하려면 다음 코드를 사용합니다
	// config.AllowOrigins = []string{
	// 	// string slice 형식으로 허용하고자 하는 목록을 작성합니다.
	// }
	// description: method는 GET과 POST만 사용합니다.
	corsConfig.AllowMethods = []string{"GET", "POST", "DELETE", "PUT"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}

	// description: middleware 설정
	routeDefault.Use(LoggingMiddleware(logger))

	routeDefault.Use(gin.Recovery())
	routeDefault.Use(cors.New(corsConfig))

	// description: Swaggo 테스트를 위한 API 핸들러 등록

	// description: api group을 생성합니다. 그룹 생성시 Parameter로 넘겨준 값이 api endpoint의 앞에 붙게 됩니다.
	routeGroup = routeDefault.Group("/api/v1")
	// routeGroup.Use(middlewares.JWTMiddleware()) // 글로벌 JWT 미들웨어와 중복으로 주석 처리

	serviceMapHandler()
	lifecycleHandler()

	if err := routeDefault.Run(port); err != nil {
		log.Fatalf("Failed to run route: %v", err)
	}

}

func serviceMapHandler() {
	serviceMapRouter := routeGroup.Group("/servicemap")
	serviceMapController := ServiceMapControllerInstance()

	// 메인 토폴로지 API (POST 방식으로 변경 - 대용량 멀티클러스터 필터 지원)
	serviceMapRouter.POST("/topology", serviceMapController.GetTopology())

	// 서비스 관련 API (POST 방식으로 변경 - 3-tier 식별자 지원)
	serviceMapRouter.POST("/workload/details", serviceMapController.GetWorkloadDetails())
	serviceMapRouter.POST("/workload/hover-info", serviceMapController.GetWorkloadHover()) // 호버 전용 API 상대 노드 top 5

	// 연결 상세 API (POST 방식으로 변경 - URL 인코딩 문제 해결)
	serviceMapRouter.POST("/edge/trace/details", serviceMapController.GetEdgeTraceDetails())
}
