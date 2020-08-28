package cryption

import (
	"crypto/md5"
	"crypto/hmac"
)

func TimeHMACHash(key []byte, timeSec int64) []byte {
	return HMACHash(key, int64ToByteSlice(timeSec))
}

func HMACHash(key, data []byte) []byte {
	hash := hmac.New(md5.New, key)
	hash.Write(data)
	return hash.Sum(nil)
}

func int64ToByteSlice(value int64) []byte {
	return []byte{
		byte(value >> 56),
		byte(value >> 48),
		byte(value >> 40),
		byte(value >> 32),
		byte(value >> 24),
		byte(value >> 16),
		byte(value >> 8),
		byte(value),
	}
}

func Int64Hash(value int64) []byte {
	md5hash := md5.New()
	buffer := int64ToByteSlice(value)
	md5hash.Write(buffer)
	md5hash.Write(buffer)
	md5hash.Write(buffer)
	md5hash.Write(buffer)
	return md5hash.Sum(nil)
}