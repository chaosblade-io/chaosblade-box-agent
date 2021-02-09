/*
 * Copyright 1999-2021 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tools

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	Delimiter = "="

	AppInstanceKeyName = "appInstance"
	AppGroupKeyName    = "appGroup"
)

var AppFile = path.Join(GetCurrentDirectory(), ".chaos.app")
var mutex = sync.RWMutex{}

// RecordApplicationToFile
func RecordApplicationToFile(appInstance, appGroup string, truncate bool) error {
	keys := map[string]string{
		AppInstanceKeyName: appInstance,
		AppGroupKeyName:    appGroup,
	}
	return RecordMapToFile(keys, AppFile, truncate)
}

func RecordMapToFile(data map[string]string, filePath string, truncate bool) error {
	if len(data) == 0 {
		return nil
	}
	mutex.Lock()
	defer mutex.Unlock()
	flag := os.O_WRONLY | os.O_CREATE
	if truncate {
		flag = flag | os.O_TRUNC
	}
	file, err := os.OpenFile(filePath, flag, 0666)
	defer file.Close()
	if err != nil {
		log.WithField("file", filePath).WithError(err).Errorf("record data to file failed")
		return err
	}
	for key, value := range data {
		_, err := file.WriteString(strings.Join([]string{key, value}, Delimiter) + "\n")
		if err != nil {
			log.WithFields(log.Fields{
				"file":  filePath,
				"key":   key,
				"value": value,
			}).WithError(err).Errorf("write data to file failed")
			return err
		}
	}
	return nil
}

// ReadAppInfoFromFile returns the local application record
func ReadAppInfoFromFile() (appInstance, appGroup string, err error) {
	bytes, err := ioutil.ReadFile(AppFile)
	if err != nil {
		return "", "", err
	}
	content := strings.TrimSpace(string(bytes))
	slice := strings.Split(content, "\n")
	if len(slice) == 0 {
		return "", "", fmt.Errorf("empty content")
	}
	for _, value := range slice {
		kv := strings.SplitN(value, Delimiter, 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case AppInstanceKeyName:
			appInstance = kv[1]
		case AppGroupKeyName:
			appGroup = kv[1]
		}
	}
	return
}
