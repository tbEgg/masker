package cryption

type DecryptReader interface {
	Decrypt([]byte)
	Read([]byte) (int, error)
}

type EncryptWriter interface {
	Encrypt([]byte)
	Write([]byte) (int, error)
}