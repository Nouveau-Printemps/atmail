package auth

import (
	"errors"
	"fmt"
)

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

type PasswordConfig struct {
	Default  *string   `toml:"default"`
	Bcrypt   *Bcrypt   `toml:"bcrypt"`
	Argon2id *Argon2id `toml:"argon2id"`
	Scrypt   *Scrypt   `toml:"scrypt"`
}

func (cfg *PasswordConfig) Load() error {
	if cfg.Default != nil {
		switch *cfg.Default {
		case "argon2id", "bcrypt", "scrypt":
		default:
			return errors.New("password config: unknown default algorithm " + *cfg.Default)
		}
	}
	if cfg.Argon2id != nil {
		cfg.Argon2id.M *= 1024
		_, err := cfg.Argon2id.Hash("test")
		if err != nil {
			return fmt.Errorf("argon2id config: %w", err)
		}
		DefaultArgon2id = *cfg.Argon2id
	}
	if cfg.Scrypt != nil {
		cfg.Scrypt.N = 2<<cfg.Scrypt.N - 1
		_, err := cfg.Scrypt.Hash("test")
		if err != nil {
			return fmt.Errorf("scrypt config: %w", err)
		}
		DefaultScrypt = *cfg.Scrypt
	}
	if cfg.Bcrypt != nil {
		_, err := cfg.Bcrypt.Hash("test")
		if err != nil {
			return fmt.Errorf("bcrypt config: %w", err)
		}
		DefaultBcrypt = *cfg.Bcrypt
	}
	return nil
}
