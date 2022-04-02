/**
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package time

import (
	"encoding/json"
	"errors"
	"syscall"
	"time"
)

// CurrentMillisecond 获取低精度的微秒时间
func CurrentMillisecond() int64 {
	var tv syscall.Timeval
	if err := syscall.Gettimeofday(&tv); err != nil {
		return time.Now().UnixNano() / 1e6
	}
	return int64(tv.Sec)*1e3 + int64(tv.Usec)/1e3
}

// Duration duration alias
type Duration time.Duration

// MarshalJSON marshal duration to json
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON unmarshal json text to struct
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

// Time2String Convert time.Time to string time
func Time2String(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// Int64Time2String Convert time stamp of Int64 to string time
func Int64Time2String(t int64) string {
	return time.Unix(t, 0).Format("2006-01-02 15:04:05")
}
