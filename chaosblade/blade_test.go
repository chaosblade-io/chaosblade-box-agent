package chaosblade

import (
	"testing"

	"encoding/json"
	"fmt"
	"github.com/chaosblade-io/chaos-agent/transport"
	"github.com/sirupsen/logrus"
	"strings"
)

func TestChaosBlade_exec(t *testing.T) {
	var response transport.Response
	result := `shell-init: error retrieving current directory: getcwd: cannot access parent directories: No such file or directory {"code":200,"success":true, "result":"ae152af7138727ce"}`
	excludeInfo := "getcwd: cannot access parent directories"
	errIndex := strings.Index(result, excludeInfo)
	if errIndex < 0 {
		response = *transport.ReturnFail(transport.Code[transport.ServerError],
			fmt.Sprintf("execute success, but unmarshal result err, result: %s", result))
	} else {
		bladeIndex := strings.Index(result, "{")
		if bladeIndex < 0 {
			response = *transport.ReturnFail(transport.Code[transport.ServerError],
				fmt.Sprintf("execute success, but parse result err, result: %s", result))
		}
		result = result[bladeIndex:]
		err := json.Unmarshal([]byte(result), &response)
		if err != nil {
			response = *transport.ReturnFail(transport.Code[transport.ServerError],
				fmt.Sprintf("execute success, but unmarshal result err with parsing, result: %s", result))
		}
	}
	logrus.Infof("%+v", response)
}
