/*
 * Copyright 2025 The ChaosBlade Authors
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

package options

import (
	"strings"
	"testing"
)

func Test_parseVersionFromOutput(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		want      string
		wantErr   bool
		errString string
	}{
		{
			name: "第一种格式 - 标准输出",
			output: `ChaosBlade Version Information:
==============================
Version:     1.8.0
Git Tag:     v1.8.0
Git Commit:  7dd785f
Git Branch:  HEAD
Build Time:  Mon Oct 20 03:37:24 UTC 2025
Release:     Yes (Production)
==============================`,
			want:    "1.8.0",
			wantErr: false,
		},
		{
			name: "第一种格式 - 带额外空格",
			output: `ChaosBlade Version Information:
==============================
Version:     1.8.0    
Git Tag:     v1.8.0
==============================`,
			want:    "1.8.0",
			wantErr: false,
		},
		{
			name: "第二种格式 - 标准输出",
			output: `version: 1.7.3
env: #1 SMP Thu Mar 17 17:08:06 UTC 2022 x86_64
build-time: Tue Jan  2 08:01:18 UTC 2024`,
			want:    "1.7.3",
			wantErr: false,
		},
		{
			name: "第二种格式 - 大写VERSION",
			output: `VERSION: 1.7.3
env: #1 SMP Thu Mar 17 17:08:06 UTC 2022 x86_64
build-time: Tue Jan  2 08:01:18 UTC 2024`,
			want:    "1.7.3",
			wantErr: false,
		},
		{
			name: "第二种格式 - 带额外空格",
			output: `version:    1.7.3
env: #1 SMP Thu Mar 17 17:08:06 UTC 2022 x86_64`,
			want:    "1.7.3",
			wantErr: false,
		},
		{
			name: "第一种格式优先 - 同时存在两种格式",
			output: `ChaosBlade Version Information:
==============================
Version:     1.8.0
Git Tag:     v1.8.0
version: 1.7.3
env: #1 SMP Thu Mar 17 17:08:06 UTC 2022 x86_64
==============================`,
			want:    "1.8.0",
			wantErr: false,
		},
		{
			name:      "空输出",
			output:    ``,
			want:      "",
			wantErr:   true,
			errString: "cannot get blade version",
		},
		{
			name:      "只有换行符",
			output:    "\n\n\n",
			want:      "",
			wantErr:   true,
			errString: "cannot get blade version",
		},
		{
			name: "无法解析的格式",
			output: `some random text
no version info here
another line`,
			want:      "",
			wantErr:   true,
			errString: "cannot parse version info from output",
		},
		{
			name:      "Version行格式错误 - 多个冒号",
			output:    `Version: 1.8.0:extra`,
			want:      "",
			wantErr:   true,
			errString: "cannot parse version info from output",
		},
		{
			name:      "version行格式错误 - 多个冒号",
			output:    `version: 1.7.3:extra`,
			want:      "",
			wantErr:   true,
			errString: "cannot parse version info from output",
		},
		{
			name:    "第一种格式 - 版本号前后有空格",
			output:  `Version:     1.8.0    `,
			want:    "1.8.0",
			wantErr: false,
		},
		{
			name:    "第二种格式 - 版本号前后有空格",
			output:  `version:    1.7.3    `,
			want:    "1.7.3",
			wantErr: false,
		},
		{
			name: "第一种格式 - 多行，Version不在第一行",
			output: `ChaosBlade Version Information:
==============================
Some other info
Version:     2.0.0
Git Tag:     v2.0.0
==============================`,
			want:    "2.0.0",
			wantErr: false,
		},
		{
			name: "第二种格式 - 多行，version不在第一行",
			output: `some header
version: 2.1.0
env: #1 SMP Thu Mar 17 17:08:06 UTC 2022 x86_64`,
			want:    "2.1.0",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVersionFromOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVersionFromOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err != nil && tt.errString != "" {
					if !strings.Contains(err.Error(), tt.errString) {
						t.Errorf("parseVersionFromOutput() error = %v, want error contains %v", err.Error(), tt.errString)
					}
				}
			} else {
				if got != tt.want {
					t.Errorf("parseVersionFromOutput() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func Test_parseVersionFromOutput_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    string
		wantErr bool
	}{
		{
			name:    "只有Version:，没有值",
			output:  "Version:",
			want:    "",
			wantErr: true,
		},
		{
			name:    "只有version:，没有值",
			output:  "version:",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Version:后只有空格",
			output:  "Version:     ",
			want:    "",
			wantErr: true,
		},
		{
			name:    "version:后只有空格",
			output:  "version:     ",
			want:    "",
			wantErr: true,
		},
		{
			name:    "包含Version:但格式不完整 - 应该只取版本号",
			output:  "Version: 1.8.0 extra text",
			want:    "1.8.0",
			wantErr: false,
		},
		{
			name:    "包含version:但格式不完整 - 应该只取版本号",
			output:  "version: 1.7.3 extra text",
			want:    "1.7.3",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVersionFromOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVersionFromOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseVersionFromOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_GetChaosBladeVersion_ErrorCases(t *testing.T) {
	// 保存原始值
	originalBladeBinPath := BladeBinPath
	originalBladeHome := BladeHome

	// 测试：blade bin文件不存在
	t.Run("blade bin file not exist", func(t *testing.T) {
		// 设置一个不存在的路径
		BladeBinPath = "/nonexistent/path/blade"
		defer func() {
			BladeBinPath = originalBladeBinPath
		}()

		version, err := GetChaosBladeVersion()
		if err == nil {
			t.Errorf("GetChaosBladeVersion() expected error, got nil")
		}
		if version != "" {
			t.Errorf("GetChaosBladeVersion() expected empty version, got %v", version)
		}
		if err != nil && err.Error() != "blade bin file not exist" {
			t.Errorf("GetChaosBladeVersion() error = %v, want 'blade bin file not exist'", err)
		}
	})

	// 恢复原始值
	BladeBinPath = originalBladeBinPath
	BladeHome = originalBladeHome
}
