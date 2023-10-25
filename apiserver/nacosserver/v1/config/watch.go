package config

import (
	"context"
	"sync"
	"time"

	"github.com/polarismesh/specification/source/go/api/v1/config_manage"
	"go.uber.org/zap"
)

type LongPollWatchContext struct {
	clientId         string
	once             sync.Once
	finishTime       time.Time
	finishChan       chan *config_manage.ConfigClientResponse
	watchConfigFiles map[string]*config_manage.ClientConfigFileInfo
}

// GetNotifieResult .
func (c *LongPollWatchContext) GetNotifieResult() *config_manage.ConfigClientResponse {
	return <-c.finishChan
}

// GetNotifieResultWithTime .
func (c *LongPollWatchContext) GetNotifieResultWithTime(timeout time.Duration) (*config_manage.ConfigClientResponse, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case ret := <-c.finishChan:
		return ret, nil
	case <-timer.C:
		return nil, context.DeadlineExceeded
	}
}

// IsOnce
func (c *LongPollWatchContext) IsOnce() bool {
	return true
}

// ShouldExpire .
func (c *LongPollWatchContext) ShouldExpire(now time.Time) bool {
	return now.After(c.finishTime)
}

// ClientID .
func (c *LongPollWatchContext) ClientID() string {
	return c.clientId
}

// ShouldNotify .
func (c *LongPollWatchContext) ShouldNotify(resp *config_manage.ClientConfigFileInfo) bool {
	key := resp.GetNamespace().GetValue() + "@" +
		resp.GetGroup().GetValue() + "@" + resp.GetFileName().GetValue()
	watchFile, ok := c.watchConfigFiles[key]
	if !ok {
		return false
	}
	isChange := watchFile.GetMd5().GetValue() != resp.GetMd5().GetValue()
	nacoslog.Info("watch ctx is change", zap.String("client", c.ClientID()), zap.String("file", key),
		zap.String("clientMd5", resp.GetMd5().GetValue()), zap.String("serverMd5", resp.GetMd5().GetValue()))
	return isChange
}

func (c *LongPollWatchContext) ListWatchFiles() []*config_manage.ClientConfigFileInfo {
	ret := make([]*config_manage.ClientConfigFileInfo, 0, len(c.watchConfigFiles))
	for _, v := range c.watchConfigFiles {
		ret = append(ret, v)
	}
	return ret
}

// AppendInterest .
func (c *LongPollWatchContext) AppendInterest(item *config_manage.ClientConfigFileInfo) {
	key := item.GetNamespace().GetValue() + "@" +
		item.GetGroup().GetValue() + "@" + item.GetFileName().GetValue()

	c.watchConfigFiles[key] = item
}

// RemoveInterest .
func (c *LongPollWatchContext) RemoveInterest(item *config_manage.ClientConfigFileInfo) {
	key := item.GetNamespace().GetValue() + "@" +
		item.GetGroup().GetValue() + "@" + item.GetFileName().GetValue()

	delete(c.watchConfigFiles, key)
}

// Close .
func (c *LongPollWatchContext) Close() error {
	c.once.Do(func() {
		close(c.finishChan)
	})
	return nil
}

func (c *LongPollWatchContext) Reply(rsp *config_manage.ConfigClientResponse) {
	c.once.Do(func() {
		c.finishChan <- rsp
		close(c.finishChan)
		key := rsp.GetConfigFile().GetNamespace().GetValue() + "@" +
			rsp.GetConfigFile().GetGroup().GetValue() + "@" + rsp.GetConfigFile().GetFileName().GetValue()
		nacoslog.Info("watch ctx do reply", zap.String("client", c.ClientID()), zap.String("file", key),
			zap.Uint32("code", rsp.GetCode().GetValue()))
	})
}
