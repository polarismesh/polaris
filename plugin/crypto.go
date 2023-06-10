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

package plugin

import (
	"fmt"
	"os"
	"sync"
)

var (
	cryptoManagerOnce sync.Once
	cryptoManager     *defaultCryptoManager
)

// Crypto Crypto interface
type Crypto interface {
	Plugin
	GenerateKey() ([]byte, error)
	Encrypt(plaintext string, key []byte) (cryptotext string, err error)
	Decrypt(cryptotext string, key []byte) (string, error)
}

// GetCrypto get the crypto plugin
func GetCryptoManager() CryptoManager {
	if cryptoManager != nil {
		return cryptoManager
	}

	cryptoManagerOnce.Do(func() {
		var (
			entries []ConfigEntry
		)
		if len(config.Crypto.Entries) != 0 {
			entries = append(entries, config.Crypto.Entries...)
		} else {
			entries = append(entries, ConfigEntry{
				Name:   config.Crypto.Name,
				Option: config.Crypto.Option,
			})
		}
		cryptoManager = &defaultCryptoManager{
			cryptos: make(map[string]Crypto),
			options: entries,
		}

		if err := cryptoManager.Initialize(); err != nil {
			log.Errorf("Crypto plugin init err: %s", err.Error())
			os.Exit(-1)
		}
	})
	return cryptoManager
}

// CryptoManager crypto algorithm manager
type CryptoManager interface {
	Name() string
	Initialize() error
	Destroy() error
	GetCryptoAlgoNames() []string
	GetCrypto(algo string) (Crypto, error)
}

// defaultCryptoManager crypto algorithm manager
type defaultCryptoManager struct {
	cryptos map[string]Crypto
	options []ConfigEntry
}

func (c *defaultCryptoManager) Name() string {
	return "CryptoManager"
}

func (c *defaultCryptoManager) Initialize() error {
	for i := range c.options {
		entry := c.options[i]
		item, exist := pluginSet[entry.Name]
		if !exist {
			log.Errorf("plugin Crypto not found target: %s", entry.Name)
			continue
		}
		crypto, ok := item.(Crypto)
		if !ok {
			log.Errorf("plugin target: %s not Crypto", entry.Name)
			continue
		}
		if err := crypto.Initialize(&entry); err != nil {
			return err
		}
		c.cryptos[entry.Name] = crypto
	}
	return nil
}

func (c *defaultCryptoManager) Destroy() error {
	for i := range c.cryptos {
		if err := c.cryptos[i].Destroy(); err != nil {
			return err
		}
	}
	return nil
}

func (c *defaultCryptoManager) GetCryptoAlgoNames() []string {
	var names []string
	for name := range c.cryptos {
		names = append(names, name)
	}
	return names
}

func (c *defaultCryptoManager) GetCrypto(algo string) (Crypto, error) {
	crypto, ok := c.cryptos[algo]
	if !ok {
		log.Errorf("plugin Crypto not found target: %s", algo)
		return nil, fmt.Errorf("plugin Crypto not found target: %s", algo)
	}
	return crypto, nil
}
