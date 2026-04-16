package pkg

import (
	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Status     string      `json:"status"`     // "success" 또는 "error"
	StatusCode int         `json:"statusCode"` // HTTP 상태 코드
	Message    string      `json:"message"`    // 응답 메시지
	Result     interface{} `json:"result"`     // 성공 시 응답 데이터
	Error      interface{} `json:"error"`      // 실패 시 오류 정보
}

// SuccessResponse: 성공 응답 처리
func SuccessResponse(c *gin.Context, statusCode int, result interface{}, message string) {
	response := APIResponse{
		Status:     "200",
		StatusCode: statusCode,
		Message:    message,
		Result:     result,
		Error:      nil,
	}
	c.JSON(statusCode, response)
}

// ErrorResponse: 오류 응답 처리
func ErrorResponse(c *gin.Context, statusCode int, message string, err error) {
	response := APIResponse{
		Status:     "error",
		StatusCode: statusCode,
		Message:    message,
		Result:     nil,
		Error:      err.Error(),
	}
	c.JSON(statusCode, response)
}
