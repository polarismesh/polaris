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

package boltdb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	bolt "go.etcd.io/bbolt"
)

const (
	typeUnknown byte = iota
	typeString
	typeBool
	typeTime
	typeProtobuf
	typeInt
	typeInt8
	typeInt16
	typeInt32
	typeInt64
	typeUint
	typeUint8
	typeUint16
	typeUint32
	typeUint64
)

var (
	timeType         = reflect.TypeOf(time.Now())
	messageType      = reflect.TypeOf((*proto.Message)(nil)).Elem()
	numberKindToType = buildNumberKindToType()
)

func buildNumberKindToType() map[reflect.Kind]byte {
	values := make(map[reflect.Kind]byte)
	values[reflect.Int] = typeInt
	values[reflect.Int8] = typeInt8
	values[reflect.Int16] = typeInt16
	values[reflect.Int32] = typeInt32
	values[reflect.Int64] = typeInt64
	values[reflect.Uint] = typeUint
	values[reflect.Uint8] = typeUint8
	values[reflect.Uint16] = typeUint16
	values[reflect.Uint32] = typeUint32
	values[reflect.Uint64] = typeUint64
	return values
}

func encodeStringBuffer(strValue string) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, len(strValue)+1))
	buf.WriteByte(typeString)
	buf.WriteString(strValue)
	return buf.Bytes()
}

func decodeStringBuffer(name string, buf []byte) (string, error) {
	if len(buf) == 0 {
		return "", nil
	}
	byteType := buf[0]
	if byteType != typeString {
		return "",
			fmt.Errorf("invalid type field %s, want string(%v), actual is %v", name, typeString, byteType)
	}
	strBytes := buf[1:]
	return string(strBytes), nil
}

func encodeIntBuffer(intValue int64, typeByte byte) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 9))
	buf.WriteByte(typeByte)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(intValue))
	buf.Write(b)
	return buf.Bytes()
}

func encodeUintBuffer(intValue uint64, typeByte byte) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 9))
	buf.WriteByte(typeByte)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, intValue)
	buf.Write(b)
	return buf.Bytes()
}

func decodeIntBuffer(name string, buf []byte, typeByte byte) (int64, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	byteType := buf[0]
	if byteType != typeByte {
		return 0, fmt.Errorf("invalid type field %s, want int(%v), actual is %v", name, typeByte, byteType)
	}
	intBytes := buf[1:]
	value := binary.LittleEndian.Uint64(intBytes)
	return int64(value), nil
}

func decodeUintBuffer(name string, buf []byte, typeByte byte) (uint64, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	byteType := buf[0]
	if byteType != typeByte {
		return 0, fmt.Errorf("invalid type field %s, want uint(%v), actual is %v", name, typeByte, byteType)
	}
	intBytes := buf[1:]
	value := binary.LittleEndian.Uint64(intBytes)
	return value, nil
}

func encodeBoolBuffer(boolValue bool) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 2))
	buf.WriteByte(typeBool)
	if boolValue {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
	return buf.Bytes()
}

func decodeBoolBuffer(name string, buf []byte) (bool, error) {
	if len(buf) == 0 {
		return false, nil
	}
	byteType := buf[0]
	if byteType != typeBool {
		return false,
			fmt.Errorf("invalid type field %s, want bool(%v), actual is %v", name, typeBool, byteType)
	}
	boolByte := buf[1]
	return boolByte > 0, nil
}

func encodeRawMap(parent *bolt.Bucket, name string, values map[string]string) error {
	nameByte := []byte(name)
	var bucket *bolt.Bucket
	var err error
	if bucket = parent.Bucket(nameByte); bucket != nil {
		err = parent.DeleteBucket(nameByte)
		if err != nil {
			return err
		}
	}
	bucket, err = parent.CreateBucket(nameByte)
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}
	for mapKey, mapValue := range values {
		if err = bucket.Put([]byte(mapKey), []byte(mapValue)); err != nil {
			return err
		}
	}
	return nil
}

func encodeMapBuffer(parent *bolt.Bucket, name string, mapKeys []reflect.Value, fieldValue *reflect.Value) error {
	nameByte := []byte(name)
	var bucket *bolt.Bucket
	var err error
	if bucket = parent.Bucket(nameByte); bucket != nil {
		err = parent.DeleteBucket(nameByte)
		if err != nil {
			return err
		}
	}
	bucket, err = parent.CreateBucket(nameByte)
	if err != nil {
		return err
	}
	if len(mapKeys) == 0 {
		return nil
	}
	for _, mapKey := range mapKeys {
		keyStr := mapKey.String()
		mapValue := fieldValue.MapIndex(mapKey)
		valueStr := mapValue.String()
		if err = bucket.Put([]byte(keyStr), []byte(valueStr)); err != nil {
			return err
		}
	}
	return nil
}

func decodeMapBuffer(bucket *bolt.Bucket) (map[string]string, error) {
	values := make(map[string]string)
	if bucket == nil {
		return values, nil
	}
	err := bucket.ForEach(func(k, v []byte) error {
		values[string(k)] = string(v)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return values, nil
}

func encodeTimeBuffer(timeValue time.Time) []byte {
	intValue := timeValue.UnixNano()
	buf := bytes.NewBuffer(make([]byte, 0, 9))
	buf.WriteByte(typeTime)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(intValue))
	buf.Write(b)
	return buf.Bytes()
}

func decodeTimeBuffer(name string, buf []byte) (time.Time, error) {
	if len(buf) == 0 {
		return time.Unix(0, 0), nil
	}
	byteType := buf[0]
	if byteType != typeTime {
		return time.Unix(0, 0),
			fmt.Errorf("invalid type field %s, want time(%v), actual is %v", name, typeTime, byteType)
	}
	intBytes := buf[1:]
	value := binary.LittleEndian.Uint64(intBytes)
	return time.Unix(0, int64(value)), nil
}

func encodeMessageBuffer(msg proto.Message) ([]byte, error) {
	protoBuf, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(make([]byte, 0, len(protoBuf)+1))
	buf.WriteByte(typeProtobuf)
	buf.Write(protoBuf)
	return buf.Bytes(), nil
}

func decodeMessageBuffer(msg proto.Message, name string, buf []byte) (proto.Message, error) {
	if len(buf) == 0 {
		return nil, nil
	}
	byteType := buf[0]
	if byteType != typeProtobuf {
		return nil, fmt.Errorf("invalid type field %s, want protoBuf(%v), actual is %v", name, typeProtobuf, byteType)
	}
	protoBytes := buf[1:]
	err := proto.Unmarshal(protoBytes, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func serializeObject(parent *bolt.Bucket, value interface{}) (map[string][]byte, error) {
	rawValue := reflect.ValueOf(value)
	into := indirect(rawValue)
	if !into.IsValid() {
		// nil object
		return nil, nil
	}
	values := make(map[string][]byte)
	intoType := indirectType(into.Type())
	nums := intoType.NumField()
	var err error
	for i := 0; i < nums; i++ {
		field := intoType.Field(i)
		rawFieldType := field.Type
		name := toBucketField(field.Name)
		if field.Anonymous {
			// 不处理匿名属性
			log.Warnf("[BlobStore] anonymous field %s, type is %v", name, rawFieldType)
			continue
		}
		fieldValue := into.FieldByName(field.Name)
		if !fieldValue.CanAddr() {
			log.Warnf("[BlobStore] addressable field %s, type is %v", name, rawFieldType)
			continue
		}
		if !fieldValue.IsValid() {
			log.Warnf("[BlobStore] invalid field %s, type is %v", name, rawFieldType)
			continue
		}
		if !fieldValue.CanInterface() {
			log.Warnf("[BlobStore] private field %s, type is %v", name, rawFieldType)
			continue
		}
		kind := rawFieldType.Kind()
		switch kind {
		case reflect.String:
			strValue := fieldValue.String()
			values[name] = encodeStringBuffer(strValue)
		case reflect.Bool:
			boolValue := fieldValue.Bool()
			values[name] = encodeBoolBuffer(boolValue)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intValue := fieldValue.Int()
			values[name] = encodeIntBuffer(intValue, numberKindToType[kind])
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			intValue := fieldValue.Uint()
			values[name] = encodeUintBuffer(intValue, numberKindToType[kind])
		case reflect.Map:
			mapKeys := fieldValue.MapKeys()
			err := encodeMapBuffer(parent, name, mapKeys, &fieldValue)
			if err != nil {
				return nil, err
			}
		case reflect.Ptr:
			if rawFieldType.Implements(messageType) {
				// protobuf类型
				elem := fieldValue.Addr().Elem()
				if elem.IsNil() {
					continue
				}
				msgValue := elem.Interface().(proto.Message)
				values[name], err = encodeMessageBuffer(msgValue)
				if err != nil {
					return nil, err
				}
			}
		case reflect.Struct:
			if rawFieldType.AssignableTo(timeType) {
				// 时间类型
				timeValue := fieldValue.Addr().Elem().Interface().(time.Time)
				values[name] = encodeTimeBuffer(timeValue)
			}
		default:
			log.Warnf(
				"[BlobStore] serialize unrecognized field %s, type is %v, kind is %s", name, rawFieldType, kind)
			continue
		}
	}
	return values, nil
}

func toBucketField(field string) string {
	return strings.ToLower(field)
}

func deserializeObject(bucket *bolt.Bucket, value interface{}) (interface{}, error) {
	fromObj := indirect(reflect.ValueOf(value))
	if !fromObj.IsValid() {
		// nil object
		return nil, nil
	}
	toValue := reflect.New(reflect.TypeOf(value).Elem()).Interface()
	toObj := indirect(reflect.ValueOf(toValue))
	intoType := indirectType(toObj.Type())
	nums := intoType.NumField()
	for i := 0; i < nums; i++ {
		field := intoType.Field(i)
		rawFieldType := field.Type
		name := toBucketField(field.Name)
		if field.Anonymous {
			// 不处理匿名属性
			log.Warnf("[BlobStore] anonymous field %s, type is %v", name, rawFieldType)
			continue
		}
		fieldValue := toObj.FieldByName(field.Name)
		if !fieldValue.CanAddr() {
			log.Warnf("[BlobStore] addressable field %s, type is %v", name, rawFieldType)
			continue
		}
		if !fieldValue.IsValid() {
			log.Warnf("[BlobStore] invalid field %s, type is %v", name, rawFieldType)
			continue
		}
		if !fieldValue.CanSet() {
			log.Warnf("[BlobStore] private field %s, type is %v", name, rawFieldType)
			continue
		}
		kind := rawFieldType.Kind()
		switch kind {
		case reflect.String:
			buf := bucket.Get([]byte(name))
			value, err := decodeStringBuffer(name, buf)
			if err != nil {
				return nil, err
			}
			fieldValue.Set(reflect.ValueOf(value))
		case reflect.Bool:
			buf := bucket.Get([]byte(name))
			value, err := decodeBoolBuffer(name, buf)
			if err != nil {
				return nil, err
			}
			fieldValue.Set(reflect.ValueOf(value))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			buf := bucket.Get([]byte(name))
			value, err := decodeIntBuffer(name, buf, numberKindToType[kind])
			if err != nil {
				return nil, err
			}
			setIntValue(value, &fieldValue, kind)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			buf := bucket.Get([]byte(name))
			value, err := decodeUintBuffer(name, buf, numberKindToType[kind])
			if err != nil {
				return nil, err
			}
			setUintValue(value, &fieldValue, kind)
		case reflect.Map:
			subBucket := bucket.Bucket([]byte(name))
			values, err := decodeMapBuffer(subBucket)
			if err != nil {
				return nil, err
			}
			fieldValue.Set(reflect.ValueOf(values))
		case reflect.Ptr:
			if rawFieldType.Implements(messageType) {
				// protobuf类型
				buf := bucket.Get([]byte(name))
				if nil == buf {
					continue
				}
				toMsgValue := reflect.New(rawFieldType.Elem()).Interface().(proto.Message)
				msg, err := decodeMessageBuffer(toMsgValue, name, buf)
				if err != nil {
					return nil, err
				}
				fieldValue.Set(reflect.ValueOf(msg))
			}
		case reflect.Struct:
			if rawFieldType.AssignableTo(timeType) {
				// 时间类型
				buf := bucket.Get([]byte(name))
				value, err := decodeTimeBuffer(name, buf)
				if err != nil {
					return nil, err
				}
				fieldValue.Set(reflect.ValueOf(value))
			}
		default:
			log.Warnf("[BlobStore] deserialize unrecognized field %s, type is %v", name, rawFieldType)
			continue
		}
	}
	return toValue, nil
}

func setIntValue(value int64, fieldValue *reflect.Value, kind reflect.Kind) {
	switch kind {
	case reflect.Int:
		fieldValue.Set(reflect.ValueOf(int(value)))
	case reflect.Int8:
		fieldValue.Set(reflect.ValueOf(int8(value)))
	case reflect.Int16:
		fieldValue.Set(reflect.ValueOf(int16(value)))
	case reflect.Int32:
		fieldValue.Set(reflect.ValueOf(int32(value)))
	case reflect.Int64:
		fieldValue.Set(reflect.ValueOf(value))
	default:
		// do nothing
	}
}

func setUintValue(value uint64, fieldValue *reflect.Value, kind reflect.Kind) {
	switch kind {
	case reflect.Uint:
		fieldValue.Set(reflect.ValueOf(uint(value)))
	case reflect.Uint8:
		fieldValue.Set(reflect.ValueOf(uint8(value)))
	case reflect.Uint16:
		fieldValue.Set(reflect.ValueOf(uint16(value)))
	case reflect.Uint32:
		fieldValue.Set(reflect.ValueOf(uint32(value)))
	case reflect.Uint64:
		fieldValue.Set(reflect.ValueOf(value))
	default:
		// do nothing
	}
}

func indirect(reflectValue reflect.Value) reflect.Value {
	for reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}
	return reflectValue
}

func indirectType(reflectType reflect.Type) reflect.Type {
	for reflectType.Kind() == reflect.Ptr || reflectType.Kind() == reflect.Slice {
		reflectType = reflectType.Elem()
	}
	return reflectType
}
