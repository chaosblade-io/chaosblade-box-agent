package transport

import (
	"fmt"

	"github.com/chaosblade-io/chaos-agent/pkg/options"
)

type Response struct {
	Code    int32
	Success bool
	Error   string
	Result  interface{}
}

const (
	OK = 200

	InvalidTimestamp   = 401
	Forbidden          = 403
	HandlerNotFound    = 404
	TokenNotFound      = 405
	ParameterEmpty     = 406
	ParameterLess      = 407
	ParameterTypeError = 408

	ServerError          = 500
	ServiceNotOpened     = 501
	ServiceNotAuthorized = 502
	EncodeError          = 503
	ServiceSwitchError   = 504
	HandlerClosed        = 505
	ServiceNotSupport    = 506
	CtlFileNotFound      = 507
	CtlExecFailed        = 508

	ChaosbladeFileNotFound = 600
	ResultUnmarshalFailed  = 601
	Helm3ExecError         = 602
)

var Errors = map[int32]string{
	OK: "success",

	InvalidTimestamp:   "invalid timestamp",
	Forbidden:          "forbidden, err: %s",
	HandlerNotFound:    "request handler not found",
	TokenNotFound:      "access token not found",
	ParameterEmpty:     "`%s`: parameter is empty",
	ParameterLess:      "`%s`: parameter less",
	ParameterTypeError: "`%s` parameter data error",

	ServerError:          "server error, err: %s",
	ServiceNotOpened:     "chaos service not opened",
	ServiceNotAuthorized: "chaos service not authorized",
	EncodeError:          "encode error, err: %s",
	ServiceSwitchError:   "service switch error, err: %s",
	HandlerClosed:        "service handler closed",
	ServiceNotSupport:    "service not support: %s",
	CtlFileNotFound:      "`%s`: ctl file not found",
	CtlExecFailed:        "exec ctl file failed: %s",

	ChaosbladeFileNotFound: fmt.Sprintf("%s, chaosblade file not found", options.BladeBinPath),
	ResultUnmarshalFailed:  "`%s`: exec result unmarshal failed, err: %s",
	Helm3ExecError:         "helm3 exec error, err: %s",
}

func ReturnFail(errCode int32, args ...interface{}) *Response {
	return &Response{Code: errCode, Success: false, Error: fmt.Sprintf(Errors[errCode], args)}
}

func ReturnSuccess() *Response {
	return &Response{Code: OK, Success: true, Result: "success"}
}

func ReturnSuccessWithResult(result interface{}) *Response {
	return &Response{Code: OK, Success: true, Result: result}
}
