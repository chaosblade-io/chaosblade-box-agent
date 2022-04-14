/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
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
	"io"
	"net/http"
	"os"
)

func Download(destFileFullPath, url string) error {
	// 1. create destination path
	file, err := os.Create(destFileFullPath)
	if err != nil {
		return err
	}
	os.Chmod(destFileFullPath, 0744)
	defer file.Close()

	// 2. get body from url
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("response code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	// 3. copy body to file
	_, err = io.Copy(file, resp.Body)
	return err
}
