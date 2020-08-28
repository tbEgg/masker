package cryption

import (
	"io"
	"crypto/aes"
	"crypto/cipher"

	"../log"
)

type AESEncryptWriter struct {
	stream	cipher.Stream
	writer	io.Writer
}

func NewAESEncryptWriter(writer io.Writer, key []byte, iv []byte) (*AESEncryptWriter, error) {
	aesStream, err := NewAESEncryptStream(key, iv)
	if err != nil {
		return nil, err
	}
	return &AESEncryptWriter{
		stream:	aesStream,
		writer:	writer,
	}, nil
}

func NewAESEncryptStream(key []byte, iv []byte) (cipher.Stream, error) {
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return cipher.NewCFBEncrypter(aesBlock, iv), nil
}

// encrypt blocks in place
func (encrytWriter *AESEncryptWriter) Encrypt(blocks []byte) {
	encrytWriter.stream.XORKeyStream(blocks, blocks)
}

// implement io.Writer interface
func (encrytWriter *AESEncryptWriter) Write(blocks []byte) (int, error) {
	encrytWriter.Encrypt(blocks)
	return encrytWriter.writer.Write(blocks)
}


type AESDecryptReader struct {
	stream	cipher.Stream
	reader	io.Reader
}

func NewAESDecryptReader(reader io.Reader, key []byte, iv []byte) (*AESDecryptReader, error) {
	aesStream, err := NewAESDecryptStream(key, iv)
	if err != nil {
		return nil, err
	}
	return &AESDecryptReader{
		stream:	aesStream,
		reader:	reader,
	}, nil
}

func NewAESDecryptStream(key []byte, iv []byte) (cipher.Stream, error) {
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return cipher.NewCFBDecrypter(aesBlock, iv), nil
}

func (decryptReader *AESDecryptReader) Decrypt(blocks []byte) {
	decryptReader.stream.XORKeyStream(blocks, blocks)
}

// implement io.Reader interface
func (decryptReader *AESDecryptReader) Read(blocks []byte) (int, error) {
	nBytes, err := decryptReader.reader.Read(blocks)
	if nBytes > 0 {
		decryptReader.Decrypt(blocks[:nBytes])
	}
	if err != nil && err != io.EOF {
		log.Error("Err in reading blocks: %v", err)
	}
	return nBytes, err
}