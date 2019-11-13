package storage

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWriteBatch(t *testing.T) {
	s := NewMemMetadataStorage()
	wb := s.NewWriteBatch()

	key1 := []byte("k1")
	value1 := []byte("v1")

	key2 := []byte("k2")
	value2 := []byte("v2")

	key3 := []byte("k3")
	value3 := []byte("v3")

	key4 := []byte("k4")
	value4 := []byte("v4")

	wb.Set(key1, value1)
	wb.Set(key2, value2)
	wb.Set(key3, value3)
	wb.Set(key4, value4)
	wb.Delete(key4)

	err := s.Write(wb, true)
	assert.NoError(t, err, "TestWriteBatch failed")

	value, err := s.Get(key1)
	assert.NoError(t, err, "TestWriteBatch failed")
	assert.Equal(t, string(value1), string(value), "TestWriteBatch failed")

	value, err = s.Get(key2)
	assert.NoError(t, err, "TestWriteBatch failed")
	assert.Equal(t, string(value2), string(value), "TestWriteBatch failed")

	value, err = s.Get(key3)
	assert.NoError(t, err, "TestWriteBatch failed")
	assert.Equal(t, string(value3), string(value), "TestWriteBatch failed")

	value, err = s.Get(key4)
	assert.NoError(t, err, "TestWriteBatch failed")
	assert.Equal(t, "", string(value), "TestWriteBatch failed")
}

func TestSetAndGet(t *testing.T) {
	s := NewMemMetadataStorage()

	key1 := []byte("k1")
	value1 := []byte("v1")

	key2 := []byte("k2")
	value2 := []byte("v2")

	key3 := []byte("k3")
	value3 := []byte("v3")

	key4 := []byte("k4")

	s.Set(key1, value1)
	s.Set(key2, value2)
	s.Set(key3, value3)

	value, err := s.Get(key1)
	assert.NoError(t, err, "TestSetAndGet failed")
	assert.Equal(t, string(value1), string(value), "TestSetAndGet failed")

	value, err = s.Get(key2)
	assert.NoError(t, err, "TestSetAndGet failed")
	assert.Equal(t, string(value2), string(value), "TestSetAndGet failed")

	value, err = s.Get(key3)
	assert.NoError(t, err, "TestSetAndGet failed")
	assert.Equal(t, string(value3), string(value), "TestSetAndGet failed")

	value, err = s.Get(key4)
	assert.NoError(t, err, "TestSetAndGet failed")
	assert.Equal(t, "", string(value), "TestSetAndGet failed")
}

func TestDelete(t *testing.T) {
	s := NewMemMetadataStorage()

	key1 := []byte("k1")
	value1 := []byte("v1")

	key2 := []byte("k2")
	value2 := []byte("v2")

	s.Set(key1, value1)
	s.Set(key2, value2)

	s.Delete(key2)

	value, err := s.Get(key1)
	assert.NoError(t, err, "TestDelete failed")
	assert.Equal(t, string(value1), string(value), "TestDelete failed")

	value, err = s.Get(key2)
	assert.NoError(t, err, "TestDelete failed")
	assert.Equal(t, "", string(value), "TestDelete failed")
}

func TestMetaRangeDelete(t *testing.T) {
	s := NewMemMetadataStorage()

	key1 := []byte("k1")
	value1 := []byte("v1")

	key2 := []byte("k2")
	value2 := []byte("v2")

	key3 := []byte("k3")
	value3 := []byte("v3")

	s.Set(key1, value1)
	s.Set(key2, value2)
	s.Set(key3, value3)

	err := s.RangeDelete(key1, key3)
	assert.NoError(t, err, "TestMetaRangeDelete failed")

	value, err := s.Get(key1)
	assert.NoError(t, err, "TestMetaRangeDelete failed")
	assert.Equal(t, "", string(value), "TestMetaRangeDelete failed")

	value, err = s.Get(key2)
	assert.NoError(t, err, "TestMetaRangeDelete failed")
	assert.Equal(t, "", string(value), "TestMetaRangeDelete failed")

	value, err = s.Get(key3)
	assert.NoError(t, err, "TestMetaRangeDelete failed")
	assert.Equal(t, string(value3), string(value), "TestMetaRangeDelete failed")
}

func TestSeek(t *testing.T) {
	s := NewMemMetadataStorage()

	key1 := []byte("k1")
	value1 := []byte("v1")

	key3 := []byte("k3")
	value3 := []byte("v3")

	s.Set(key1, value1)
	s.Set(key3, value3)

	key, value, err := s.Seek(key1)
	assert.NoError(t, err, "TestSeek failed")
	assert.Equal(t, string(key1), string(key), "TestSeek failed")
	assert.Equal(t, string(value1), string(value), "TestSeek failed")

	key, value, err = s.Seek([]byte("k2"))
	assert.NoError(t, err, "TestSeek failed")
	assert.Equal(t, string(key3), string(key), "TestSeek failed")
	assert.Equal(t, string(value3), string(value), "TestSeek failed")
}

func TestScan(t *testing.T) {
	s := NewMemMetadataStorage()

	key1 := []byte("k1")
	value1 := []byte("v1")

	key2 := []byte("k2")
	value2 := []byte("v2")

	key3 := []byte("k3")
	value3 := []byte("v3")

	s.Set(key1, value1)
	s.Set(key2, value2)
	s.Set(key3, value3)

	count := 0
	err := s.Scan(key1, key3, func(key, value []byte) (bool, error) {
		count++
		return true, nil
	}, false)
	assert.NoError(t, err, "TestScan failed")
	assert.Equal(t, 2, count, "TestScan failed")

	count = 0
	err = s.Scan(key1, key3, func(key, value []byte) (bool, error) {
		count++
		if count == 1 {
			return false, nil
		}
		return true, nil
	}, false)
	assert.NoError(t, err, "TestScan failed")
	assert.Equal(t, 1, count, "TestScan failed")
}