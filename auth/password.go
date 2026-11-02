package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/scrypt"
)

type Password interface {
	Hash(pass string) (string, error)
	Compare(hash, pass string) (bool, error)
}

var (
	ErrInvalidPasswordFormat = errors.New("invalid password format")
	defaultEncoder           = base64.StdEncoding.WithPadding(base64.NoPadding)
)

type PasswordEncoder interface {
	Encode(hash []byte, salt []byte) string
	Decode(string) (hash, salt []byte, err error)
}

type Bcrypt struct {
	Cost int
}

func (bc Bcrypt) Hash(pass string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pass), bc.Cost)
	return string(b), err
}

func (Bcrypt) Compare(hash, pass string) (bool, error) {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass)) == nil, nil
}

type Scrypt struct {
	// N is the number of iterations.
	N int
	// R is the block size.
	R int
	// P is the parallelism.
	P int
}

func (s Scrypt) Hash(pass string) (string, error) {
	var salt [16]byte
	rand.Read(salt[:])
	hash, err := scrypt.Key([]byte(pass), salt[:], s.N, s.R, s.P, 32)
	if err != nil {
		return "", err
	}
	return s.Encode(hash, salt[:]), nil
}

func (s Scrypt) Encode(hash []byte, salt []byte) string {
	return fmt.Sprintf(
		"$scrypt$N=%d,r=%d,p=%d$%s$%s",
		s.N, s.R, s.P,
		defaultEncoder.EncodeToString(salt),
		defaultEncoder.EncodeToString(hash),
	)
}

var scryptFormat = regexp.MustCompile(`^\$scrypt\$N=(\d+),r=(\d+),p=(\d+)\$([^$]+)\$([^$]+)$`)

func (s *Scrypt) Decode(raw string) (hash []byte, salt []byte, err error) {
	f := scryptFormat.FindStringSubmatch(raw)
	if len(f) != 6 || f[0] != raw {
		return nil, nil, ErrInvalidPasswordFormat
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("%w: %w", ErrInvalidPasswordFormat, err)
		}
	}()
	s.N, err = strconv.Atoi(f[1])
	if err != nil {
		return
	}
	s.R, err = strconv.Atoi(f[2])
	if err != nil {
		return
	}
	s.P, err = strconv.Atoi(f[3])
	if err != nil {
		return
	}
	salt, err = defaultEncoder.DecodeString(f[4])
	if err != nil {
		return
	}
	hash, err = defaultEncoder.DecodeString(f[5])
	return
}

func (s Scrypt) Compare(encodedHash, pass string) (bool, error) {
	hash, salt, err := s.Decode(encodedHash)
	if err != nil {
		return false, err
	}
	got, err := scrypt.Key([]byte(pass), salt, s.N, s.R, s.P, 32)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(hash, got) == 0, nil
}

type Argon2id struct {
	// M is the maximum memory to use.
	M uint32
	// T is the number of iterations (parameter time in [argon2.IDKey]).
	T uint32
	// P is the number of threads.
	P uint8
}

func (a Argon2id) Hash(pass string) (string, error) {
	var salt [16]byte
	rand.Read(salt[:])
	key := argon2.IDKey([]byte(pass), salt[:], a.T, a.M, a.P, 32)
	return a.Encode(key, salt[:]), nil
}

func (a Argon2id) Encode(hash, salt []byte) string {
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		a.M, a.T, a.P,
		defaultEncoder.EncodeToString(salt),
		defaultEncoder.EncodeToString(hash),
	)
}

var argon2idFormat = regexp.MustCompile(`^\$argon2id\$v=19$m=(\d+),t=(\d+),p=(\d+)\$([^$]+)\$([^$]+)$`)

func (a *Argon2id) Decode(raw string) (hash, salt []byte, err error) {
	f := argon2idFormat.FindStringSubmatch(raw)
	if len(f) != 6 || f[0] != raw {
		err = ErrInvalidPasswordFormat
		return
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("%w: %w", ErrInvalidPasswordFormat, err)
		}
	}()
	var r uint64
	r, err = strconv.ParseUint(f[1], 10, 32)
	if err != nil {
		return
	}
	a.M = uint32(r)
	r, err = strconv.ParseUint(f[2], 10, 32)
	if err != nil {
		return
	}
	a.T = uint32(r)
	r, err = strconv.ParseUint(f[3], 10, 8)
	if err != nil {
		return
	}
	a.P = uint8(r)
	salt, err = defaultEncoder.DecodeString(f[4])
	if err != nil {
		return
	}
	hash, err = defaultEncoder.DecodeString(f[5])
	return
}

func (a Argon2id) Compare(encodedHash, pass string) (bool, error) {
	hash, salt, err := a.Decode(encodedHash)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(pass), salt, a.T, a.M, a.P, 32)
	return subtle.ConstantTimeCompare(hash, got) == 0, nil
}

var (
	DefaultArgon2id = Argon2id{
		M: 16 * 1024,
		T: 5,
		P: 2,
	}
	DefaultScrypt = Scrypt{
		N: 2 << 15,
		P: 2,
		R: 8,
	}
	DefaultBcrypt = Bcrypt{Cost: 12}
)

func detectPasswordAlgorithm(pass string) Password {
	switch {
	case strings.HasPrefix(pass, "$argon2id"):
		return DefaultArgon2id
	case strings.HasPrefix(pass, "$scrypt"):
		return DefaultScrypt
	case strings.HasPrefix(pass, "$2y"):
		return DefaultBcrypt
	default:
		panic("unsupported password algorithm: " + pass)
	}
}
