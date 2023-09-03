package utils

import (
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	lru "github.com/hashicorp/golang-lru"
)

const (
	enableProtobufCacheKey = "enableCacheProto"
	sizeProtobufCacheKey   = "sizeCacheProto"
)

// MessageToCache
type MessageToCache func(m interface{}) *CacheObject

// Cache
type Cache interface {
	// Get
	Get(cacheType string, key string) *CacheObject
	// Put
	Put(v *CacheObject) (*CacheObject, bool)
}

type CacheObject struct {
	OriginVal proto.Message

	buf []byte

	CacheType string

	Key string
}

func (c *CacheObject) GetBuf() []byte {
	return c.buf
}

func (c *CacheObject) Marshal(m proto.Message) error {
	jsonpbMsg := jsonpb.Marshaler{Indent: " ", EmitDefaults: true}
	msg, err := jsonpbMsg.MarshalToString(m)
	if err != nil {
		return err
	}
	c.buf = []byte(msg)
	return nil
}

type jsonProtoBufferCache struct {
	enabled       bool
	cacheRegistry map[string]*lru.ARCCache
}

// NewCache Component a PB cache pool
func NewCache(options map[string]interface{}, cacheType []string) (Cache, error) {
	enabled, _ := options[enableProtobufCacheKey].(bool)

	if !enabled {
		return nil, nil
	}

	size, _ := options[sizeProtobufCacheKey].(int)
	if size == 0 {
		size = 128
	}

	cacheRegistry := make(map[string]*lru.ARCCache)

	for i := range cacheType {
		cache, err := lru.NewARC(size)
		if err != nil {
			return nil, fmt.Errorf("init protobuf=[%s] cache fail : %+v", cacheType[i], err)
		}
		cacheRegistry[cacheType[i]] = cache
	}

	return &jsonProtoBufferCache{
		enabled:       enabled,
		cacheRegistry: cacheRegistry,
	}, nil
}

func (jbc *jsonProtoBufferCache) Get(cacheType string, key string) *CacheObject {
	c, ok := jbc.cacheRegistry[cacheType]
	if !ok {
		return nil
	}

	val, exist := c.Get(key)
	if !exist {
		return nil
	}
	return val.(*CacheObject)
}

func (jbc *jsonProtoBufferCache) Put(v *CacheObject) (*CacheObject, bool) {
	if v == nil {
		return nil, false
	}
	cacheType := v.CacheType
	key := v.Key
	c, ok := jbc.cacheRegistry[cacheType]
	if !ok {
		return nil, false
	}
	c.Add(key, v)
	return v, true
}
