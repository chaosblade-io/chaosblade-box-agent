package transport

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestNewRequest(t *testing.T) {
	request := NewRequest()
	bytes, _ := json.Marshal(request)
	logrus.Info(string(bytes))
}
