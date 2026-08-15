package auth

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/textproto"
	"strings"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/emersion/go-message"
)

func EncryptEmail(rawKey string, email *message.Entity) error {
	// skip PGP messages
	body, err := io.ReadAll(email.Body)
	if err != nil {
		return err
	}
	defer func() {
		// because io.ReadAll leaves an empty body
		if err != nil {
			email.Body = bytes.NewBuffer(body)
		}
	}()
	if crypto.IsPGPMessage(string(body)) {
		return nil
	}
	key, err := crypto.NewKeyFromArmored(rawKey)
	if err != nil {
		return err
	}
	pgp := crypto.PGP()
	enc, err := pgp.Encryption().Recipient(key).New()
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("Content-Type: ")
	sb.WriteString(email.Header.Get("Content-Type"))
	sb.WriteRune('\n')
	if email.Header.Has("Content-Transfer-Encoding") {
		sb.WriteString("Content-Transfer-Encoding: ")
		sb.WriteString(email.Header.Get("Content-Transfer-Encoding"))
		sb.WriteRune('\n')
	}
	sb.WriteRune('\n')
	sb.Write(body)
	msg, err := enc.Encrypt([]byte(sb.String()))
	if err != nil {
		return err
	}
	bodyEnc, err := msg.ArmorBytes()
	if err != nil {
		return err
	}
	// encoding in PGP/Mime format
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", "application/pgp-encrypted")
	part, err := w.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = part.Write([]byte("Version: 1\n"))
	if err != nil {
		return err
	}
	h = make(textproto.MIMEHeader)
	h.Set("Content-Type", "application/octet-stream")
	part, err = w.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = part.Write(bodyEnc)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	email.Header.Set("Content-Type", "multipart/encrypted; boundary="+w.Boundary()+`; protocol="application/pgp-encrypted"`)
	email.Header.Set("Content-Transfer-Encoding", "7bit")
	email.Body = &buf
	return nil
}
