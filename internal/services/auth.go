package services

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}

type TokenService struct {
	Secret     []byte
	Issuer     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func (t TokenService) HashPassword(raw string) (string, error) {
	return hashArgon2id(raw)
}

func (t TokenService) VerifyPassword(raw, hashed string) bool {
	if strings.HasPrefix(hashed, "$argon2") {
		return verifyArgon2id(raw, hashed)
	}
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(raw)) == nil
}

func (t TokenService) CreateAccessToken(userID, email string, roles []string) (string, int64, error) {
	now := time.Now().UTC()
	exp := now.Add(t.AccessTTL)
	claims := jwt.MapClaims{
		"iss":   t.Issuer,
		"sub":   userID,
		"typ":   "access",
		"email": email,
		"roles": roles,
		"iat":   now.Unix(),
		"exp":   exp.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.Secret)
	return signed, exp.Unix(), err
}

func (t TokenService) CreateRefreshToken(userID string) (string, error) {
	now := time.Now().UTC()
	exp := now.Add(t.RefreshTTL)
	claims := jwt.MapClaims{
		"iss": t.Issuer,
		"sub": userID,
		"typ": "refresh",
		"iat": now.Unix(),
		"exp": exp.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(t.Secret)
}

func (t TokenService) ParseToken(tokenStr string) (*jwt.Token, jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return t.Secret, nil
	}, jwt.WithIssuer(t.Issuer))
	return token, claims, err
}

type argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  int
	keyLength   int
}

func hashArgon2id(raw string) (string, error) {
	params := argon2Params{
		memory:      65536,
		iterations:  3,
		parallelism: 1,
		saltLength:  16,
		keyLength:   32,
	}
	salt := make([]byte, params.saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(raw), salt, params.iterations, params.memory, params.parallelism, uint32(params.keyLength))
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Key := base64.RawStdEncoding.EncodeToString(key)
	return "$argon2id$v=19$m=" + strconv.FormatUint(uint64(params.memory), 10) +
		",t=" + strconv.FormatUint(uint64(params.iterations), 10) +
		",p=" + strconv.FormatUint(uint64(params.parallelism), 10) +
		"$" + b64Salt + "$" + b64Key, nil
}

func verifyArgon2id(raw, encoded string) bool {
	params, salt, hash, err := decodeArgon2id(encoded)
	if err != nil {
		return false
	}
	key := argon2.IDKey([]byte(raw), salt, params.iterations, params.memory, params.parallelism, uint32(params.keyLength))
	return subtleCompare(hash, key)
}

func decodeArgon2id(encoded string) (argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return argon2Params{}, nil, nil, errors.New("invalid hash format")
	}
	var params argon2Params
	if !strings.HasPrefix(parts[1], "argon2") {
		return argon2Params{}, nil, nil, errors.New("invalid hash type")
	}
	paramValues := strings.Split(parts[3], ",")
	for _, kv := range paramValues {
		pair := strings.SplitN(kv, "=", 2)
		if len(pair) != 2 {
			continue
		}
		switch pair[0] {
		case "m":
			value, _ := strconv.ParseUint(pair[1], 10, 32)
			params.memory = uint32(value)
		case "t":
			value, _ := strconv.ParseUint(pair[1], 10, 32)
			params.iterations = uint32(value)
		case "p":
			value, _ := strconv.ParseUint(pair[1], 10, 8)
			params.parallelism = uint8(value)
		}
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argon2Params{}, nil, nil, err
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argon2Params{}, nil, nil, err
	}
	params.saltLength = len(salt)
	params.keyLength = len(hash)
	return params, salt, hash, nil
}

func subtleCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
