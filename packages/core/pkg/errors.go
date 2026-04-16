package pkg

import (
	"errors"
	"fmt"
)

/*
 code range

 - 00000 ~ 00999 : 일반 에러

 - 01000 ~ 01999 : agent 에러

 - 02000 ~ 02999 : host 에러
*/

type (
	// ErrorCode is a type for error code
	ErrorCode uint16

	// ko11yError is a struct for error
	ReserveError struct {
		SystemCode  ErrorCode
		Description string
		Message     string
		Err         error
	}

	// ErrorHandler handles errors asynchronously
	ErrorHandler struct {
		errorHandler   chan ReserveError
		handlingAction func(ReserveError)
	}
)

const (
	// common error 00000 ~ 00099
	CodeCommonError ErrorCode = 00000

	CodeKubeClientError          ErrorCode = 00010
	CodeKubeClientOperationError ErrorCode = 00011
	CodeMetricClientError        ErrorCode = 00012
	CodeMetricOperationError     ErrorCode = 00013

	CodeGRPCError        ErrorCode = 00020
	CodeGRPCStart        ErrorCode = 00021
	CodeGRPCNetworkError ErrorCode = 00022

	CodeEKSError       ErrorCode = 00210
	CodeAKSError       ErrorCode = 00220
	CodeGKESDKError    ErrorCode = 00230
	CodeOpenShiftError ErrorCode = 00240

	CodeCronError ErrorCode = 00500

	CodeUtilsError        ErrorCode = 00600
	CodeUtilsLoggerError  ErrorCode = 00610
	CodeUtilsJWTError     ErrorCode = 00620
	CodeUtilsSessionError ErrorCode = 00630

	// agent error 01000 ~ 01999
	CodeAgentError ErrorCode = 01000

	// cluster autoscaler error 01100 ~ 01199
	CA_AgentErrCode ErrorCode = 01100

	CA_AgentInitErrCode   ErrorCode = 01110
	CA_AgentBackupErrCode ErrorCode = 01111
	CA_AgentOIDCErrCode   ErrorCode = 01112

	CA_AgentRBACErrCode       ErrorCode = 01120
	CA_AgentRBACCreateErrCode ErrorCode = 01121
	CA_AgentRBACReadErrCode   ErrorCode = 01122
	CA_AgentRBACDeleteErrCode ErrorCode = 01123

	CA_AgentWorkloadErrCode       ErrorCode = 01130
	CA_AgentWorkloadCreateErrCode ErrorCode = 01131
	CA_AgentWorkloadReadErrCode   ErrorCode = 01132
	CA_AgentWorkloadDeleteErrCode ErrorCode = 01133
	CA_AgentWorkloadUpdateErrCode ErrorCode = 01134

	// vertical pod autoscaler error 01200 ~ 01299
	VPA_AgentErrCode ErrorCode = 01200

	VPA_AgentInstallErrCode   ErrorCode = 01211
	VPA_AgentUninstallErrCode ErrorCode = 01212

	VPA_AgentFetchErrCode ErrorCode = 01220

	VPA_AgentApplyErrCode  ErrorCode = 01231
	VPA_AgentRemoveErrCode ErrorCode = 01232

	VPA_AgentResizingErrCode ErrorCode = 01240

	VPA_AgentUpdateErrCode ErrorCode = 01250

	VPA_AgentCheckErrCode ErrorCode = 01260

	// host error
	CodeHostError ErrorCode = 02000
)

var (
	// common error
	ErrCommon = ReserveError{SystemCode: CodeCommonError, Message: "common error"}

	ErrKubeClient          = ReserveError{SystemCode: CodeKubeClientError, Message: "kube client error"}
	ErrKubeClientOperation = ReserveError{SystemCode: CodeKubeClientOperationError, Message: "kube client operation error"}

	ErrMetricClient    = ReserveError{SystemCode: CodeMetricClientError, Message: "metric client error"}
	ErrMetricOperation = ReserveError{SystemCode: CodeMetricOperationError, Message: "metric operation error"}

	ErrGRPC        = ReserveError{SystemCode: CodeGRPCError, Message: "grpc error"}
	ErrGRPCStart   = ReserveError{SystemCode: CodeGRPCStart, Message: "grpc start error"}
	ErrGRPCNetwork = ReserveError{SystemCode: CodeGRPCNetworkError, Message: "grpc network error"}

	ErrEKSSDK       = ReserveError{SystemCode: CodeEKSError, Message: "eks sdk error"}
	ErrAKSSDK       = ReserveError{SystemCode: CodeAKSError, Message: "aks sdk error"}
	ErrGKESDK       = ReserveError{SystemCode: CodeGKESDKError, Message: "gke sdk error"}
	ErrOpenShiftSDK = ReserveError{SystemCode: CodeOpenShiftError, Message: "openshift sdk error"}

	ErrCron = ReserveError{SystemCode: CodeCronError, Message: "cron error"}

	ErrUtils        = ReserveError{SystemCode: CodeUtilsError, Message: "utils error"}
	ErrUtilsLogger  = ReserveError{SystemCode: CodeUtilsLoggerError, Message: "utils logger error"}
	ErrUtilsJWT     = ReserveError{SystemCode: CodeUtilsJWTError, Message: "utils jwt error"}
	ErrUtilsSession = ReserveError{SystemCode: CodeUtilsSessionError, Message: "utils session error"}

	// agent error
	// 에러 코드 범위 01000 ~ 01999
	ErrAgent = ReserveError{SystemCode: CodeAgentError, Message: "agent error"}
	// Cluster Autoscaler error
	// 에러 코드 범위 01100 ~ 01199
	ErrClusterAutoscaler = ReserveError{SystemCode: CA_AgentErrCode, Message: "cluster autoscaler error"}

	ErrCA_AgentInit   = ReserveError{SystemCode: CA_AgentInitErrCode, Message: "cluster autoscaler agent init error"}
	ErrCA_AgentBackup = ReserveError{SystemCode: CA_AgentBackupErrCode, Message: "cluster autoscaler agent backup error"}
	ErrCA_AgentOIDC   = ReserveError{SystemCode: CA_AgentOIDCErrCode, Message: "cluster autoscaler agent oidc error"}

	ErrCA_AgentRBAC       = ReserveError{SystemCode: CA_AgentRBACErrCode, Message: "cluster autoscaler agent rbac error"}
	ErrCA_AgentRBACCreate = ReserveError{SystemCode: CA_AgentRBACCreateErrCode, Message: "cluster autoscaler agent rbac create error"}
	ErrCA_AgentRBACRead   = ReserveError{SystemCode: CA_AgentRBACReadErrCode, Message: "cluster autoscaler agent rbac read error"}
	ErrCA_AgentRBACDelete = ReserveError{SystemCode: CA_AgentRBACDeleteErrCode, Message: "cluster autoscaler agent rbac delete error"}

	ErrCA_AgentWorkload       = ReserveError{SystemCode: CA_AgentWorkloadErrCode, Message: "cluster autoscaler agent workload error"}
	ErrCA_AgentWorkloadCreate = ReserveError{SystemCode: CA_AgentWorkloadCreateErrCode, Message: "cluster autoscaler agent workload create error"}
	ErrCA_AgentWorkloadRead   = ReserveError{SystemCode: CA_AgentWorkloadReadErrCode, Message: "cluster autoscaler agent workload read error"}
	ErrCA_AgentWorkloadDelete = ReserveError{SystemCode: CA_AgentWorkloadDeleteErrCode, Message: "cluster autoscaler agent workload delete error"}
	ErrCA_AgentWorkloadUpdate = ReserveError{SystemCode: CA_AgentWorkloadUpdateErrCode, Message: "cluster autoscaler agent workload update error"}

	// Vertical Pod Autoscaler error
	// 에러 코드 범위 01200 ~ 01299
	ErrVerticalPodAutoscaler = ReserveError{SystemCode: VPA_AgentErrCode, Message: "vertical pod autoscaler error"}

	ErrVPA_AgentInstall   = ReserveError{SystemCode: VPA_AgentInstallErrCode, Message: "vertical pod autoscaler agent install error"}
	ErrVPA_AgentUninstall = ReserveError{SystemCode: VPA_AgentInstallErrCode, Message: "vertical pod autoscaler agent uninstall error"}

	ErrVPA_AgentFetch = ReserveError{SystemCode: VPA_AgentFetchErrCode, Message: "vertical pod autoscaler agent fetch error"}

	ErrVPA_AgentApply  = ReserveError{SystemCode: VPA_AgentFetchErrCode, Message: "vertical pod autoscaler agent smart scaling apply error"}
	ErrVPA_AgentRemove = ReserveError{SystemCode: VPA_AgentFetchErrCode, Message: "vertical pod autoscaler agent smart scaling remove error"}

	ErrVPA_AgentResizing = ReserveError{SystemCode: VPA_AgentFetchErrCode, Message: "vertical pod autoscaler agent resizing error"}

	ErrVPA_AgentUpdate = ReserveError{SystemCode: VPA_AgentFetchErrCode, Message: "vertical pod autoscaler agent update error"}

	ErrVPA_AgentCheck = ReserveError{SystemCode: VPA_AgentCheckErrCode, Message: "vertical pod autoscaler agent check error"}
	// host error
	ErrHost = &ReserveError{SystemCode: CodeHostError, Message: "host error"}

	errorHandler *ErrorHandler
)

/*
에러 처리 요령
 1. 어디서 발생한 에러인지 명확해야 한다
	- 코드로 분리해서 관리하기
	- 어떤 내용으로 발생한 에러인지 기록하기
 2. 퍼블릭 에러와 프라이빗 에러를 구분하기
	- 퍼블릭 에러는 서비스 형태로 제공되는 에러
	 외부에 공개하도 되는 값으로 구성한다
	- 프라이빗 에러는 모듈 내에서 처리되는 에러
	 호출 스택등을 포함해 내부 처리 및 디버깅에 활용한다.
 3. 사용성을 고려한다
    - 각 모듈에서 사용할 수 있는 형태로 구현한다.
*/

// Error returns formatted error message
func (e *ReserveError) Error() string {
	return fmt.Sprintf("\n\033[1;91mERROR\033[0m\n├── \033[1;101;37mCODE: %d\033[0m\n├── \033[1;32mMessage: %s\033[0m\n├── \033[1;32mDescription: %s\033[0m\n└── \033[1;32mError: \n     %v\033[0m\n", e.SystemCode, e.Message, e.Description, e.Err.Error())
}

// Is compares errors by SystemCode
func (e *ReserveError) Is(target error) bool {
	if t, ok := target.(*ReserveError); ok {
		// systemCode 비교를 통해 같은 종류의 에러인지 확인
		return e.SystemCode == t.SystemCode
	}
	return errors.Is(e, target)
}

// Unwrap returns wrapped error
func (e *ReserveError) Unwrap() error {
	if e.Err == nil {
		return nil
	}
	return e.Err
}

// Log returns formatted error log
func Log(e *ReserveError) string {
	return fmt.Sprintf("\n\033[1;91mERROR\033[0m\n├── \033[1;101;37mCODE: %d\033[0m\n├── \033[1;32mMessage: %s\033[0m\n├── \033[1;32mDescription: %s\033[0m\n└── \033[1;32mError: \n     %v\033[0m\n", e.SystemCode, e.Message, e.Description, e.Err.Error())
}

// Desc adds description to error
func (e *ReserveError) Desc(message string) *ReserveError {
	newErr := *e
	if newErr.Description != "" {
		newErr.Description = newErr.Description + " - " + message
	} else {
		newErr.Description = message
	}
	return &newErr
}

// WithErrStack wraps external error
func (e *ReserveError) WithErrStack(err error) *ReserveError {
	newErr := *e
	newErr.Err = err
	return &newErr
}

// InitializeErrorHandler initializes global error handler
func InitializeErrorHandler(action func(ReserveError)) *ErrorHandler {
	if errorHandler != nil {
		return errorHandler
	}

	errorHandler = &ErrorHandler{
		errorHandler:   make(chan ReserveError),
		handlingAction: action,
	}

	return errorHandler
}

// ReleaseErrorHandler closes and releases global error handler
func ReleaseErrorHandler() {
	close(errorHandler.errorHandler)
	errorHandler = nil
}

// Catch handles error asynchronously
func Catch(err ReserveError) {
	if errorHandler == nil {
		return
	}
	errorHandler.handlingAction(err)
}
