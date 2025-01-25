package util

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"gopkg.in/square/go-jose.v2/jwt"
)

// サンプルの公開鍵(本番では適切に管理)
const publicKeyPEM = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAErNkbjzyMz81Np8sBb8Jr3bUOkLW4
H41Ugac0eSzPyemDvmaCIDpRofi3Rb0EgaSRSqC3IoBgVmQ+bPLtueUtUg==
-----END PUBLIC KEY-----`
const devPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEsif3xPZ/ObY12BCB2SfC3045eSkq
G9Kw2nD2DYgoJHFCPTzCLUqOKDpig4H0tYXH4RaSy6+apfgfeE/TJagHuw==
-----END PUBLIC KEY-----`

func BoolPtr(b bool) *bool {
	return &b
}

// Metadataに収容されるJSONの構造体
type Metadata struct {
	// ルームのメタデータ
	Status string `json:"status"`

	// webinarかどうか
	IsWebinar bool `json:"isWebinar"`
}

func AuthTraQClient(c echo.Context) (string, *echo.HTTPError) {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "Authorization header is required")
	}
	tokenString := authHeader[len("Bearer "):]
	parsedToken, err := jwt.ParseSigned(tokenString)
	if err != nil {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
	}

	// 3) Verify algorithm is ES256
	if len(parsedToken.Headers) == 0 || parsedToken.Headers[0].Algorithm != "ES256" {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "Invalid token algorithm")
	}

	// 4) 署名検証 (本番key / dev key)
	var claims map[string]interface{}
	if err := verifyWithECDSA(parsedToken, publicKeyPEM, devPublicKeyPEM, &claims); err != nil {
		return "", echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	// 5) exp と name クレームをチェック
	exp, ok := claims["exp"].(float64)
	if !ok {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "Token missing expiration")
	}
	if time.Unix(int64(exp), 0).Before(time.Now()) {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "Token has expired")
	}
	name, ok := claims["name"].(string)
	if !ok || name == "" {
		return "", echo.NewHTTPError(http.StatusBadRequest, "name claim is required in JWT")
	}
	return name, nil
}

// verifyWithECDSA は ECDSA 公開鍵2種類(本番鍵 / 開発用鍵)で検証を試みるユーティリティ
func verifyWithECDSA(parsedToken *jwt.JSONWebToken, primaryKey, devKey string, claims interface{}) error {
	// 1) primary key
	if err := verifyECDSA(parsedToken, primaryKey, claims); err == nil {
		return nil // 成功
	}
	// 2) dev key
	if err := verifyECDSA(parsedToken, devKey, claims); err == nil {
		return nil
	}
	return fmt.Errorf("failed to verify with both primary & dev public key")
}

func verifyECDSA(parsedToken *jwt.JSONWebToken, keyPEM string, claims interface{}) error {
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return fmt.Errorf("failed to decode PEM block")
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse PKIX public key: %v", err)
	}
	ecdsaPubKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("not an ECDSA public key")
	}
	if err := parsedToken.Claims(ecdsaPubKey, claims); err != nil {
		return err
	}
	return nil
}
