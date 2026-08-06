package auth

import (
	"github.com/ProtonMail/gopenpgp/v3/crypto"
)

func EncryptEmail(rawKey string, content []byte) ([]byte, error) {
	// skip PGP messages
	if crypto.IsPGPMessage(string(content)) {
		return content, nil
	}
	key, err := crypto.NewKeyFromArmored(rawKey)
	if err != nil {
		return nil, err
	}
	pgp := crypto.PGP()
	enc, err := pgp.Encryption().Recipient(key).New()
	if err != nil {
		return nil, err
	}
	msg, err := enc.Encrypt(content)
	if err != nil {
		return nil, err
	}
	return msg.ArmorBytes()
}
