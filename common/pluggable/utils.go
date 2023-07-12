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

package pluggable

import (
	"os"

	"github.com/pkg/errors"

	"github.com/polarismesh/polaris/common/log"
)

var (
	// errNotSocket is returned when the file is not a socket.
	errNotSocket = errors.New("not a socket")
)

// pluginFiles returns the plugin files in the socket folder.
func pluginFiles(sockFolder string) ([]os.DirEntry, error) {
	_, err := os.Stat(sockFolder)
	if os.IsNotExist(err) {
		log.Infof("socket folder %s does not exist, skip plugin discovery", sockFolder)
		return nil, nil
	}

	if err != nil {
		log.Errorf("failed to stat socket folder %s: %v", sockFolder, err)
		return nil, err
	}

	var files []os.DirEntry
	files, err = os.ReadDir(sockFolder)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read socket folder %s", sockFolder)
	}

	return files, nil
}

// socketName returns true if the file is a socket.
func socketName(entry os.DirEntry) (string, error) {
	if entry.IsDir() {
		return "", errNotSocket
	}

	f, err := entry.Info()
	if err != nil {
		return "", err
	}

	// skip non-socket files.
	if f.Mode()&os.ModeSocket == 0 {
		return "", errNotSocket
	}

	return entry.Name(), nil
}
