package handler

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/google/uuid"
	"github.com/pikachu0310/livekit-server/internal/pkg/util"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/auth"
	"github.com/pikachu0310/livekit-server/openapi/models"
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

// GetLiveKitToken GET /token?room=UUID
// Bearerトークン(ES256)で認証後、LiveKit接続用JWTを生成して返す。
// さらに canUpdateOwnMetadata を付与するため、UpdateParticipant を呼ぶ。
func (h *Handler) GetLiveKitToken(c echo.Context, _ models.GetLiveKitTokenParams) error {
	// 1) roomクエリパラメータ取得 (必須)
	room := c.QueryParam("room")
	if room == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "room query parameter is required",
		})
	}

	// 2) AuthorizationヘッダからJWTを取得
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authorization header is required",
		})
	}
	tokenString := authHeader[len("Bearer "):]
	parsedToken, err := jwt.ParseSigned(tokenString)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid token",
		})
	}

	// 3) Verify algorithm is ES256
	if len(parsedToken.Headers) == 0 || parsedToken.Headers[0].Algorithm != "ES256" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid token algorithm",
		})
	}

	// 4) 署名検証 (本番key / dev key)
	var claims map[string]interface{}
	if err := verifyWithECDSA(parsedToken, publicKeyPEM, devPublicKeyPEM, &claims); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	// 5) exp と name クレームをチェック
	exp, ok := claims["exp"].(float64)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Token missing expiration",
		})
	}
	if time.Unix(int64(exp), 0).Before(time.Now()) {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Token has expired",
		})
	}
	name, ok := claims["name"].(string)
	if !ok || name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "name claim is required in JWT",
		})
	}
	userID := name

	// 6) LiveKit用APIキー/シークレット (環境変数より)
	apiKey := os.Getenv("LIVEKIT_API_KEY")
	apiSecret := os.Getenv("LIVEKIT_API_SECRET")
	if apiKey == "" || apiSecret == "" {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "API key and secret must be set in environment variables",
		})
	}

	// 7) VideoGrant にルーム名、CanPublishData=true を設定
	at := auth.NewAccessToken(apiKey, apiSecret)
	grant := &auth.VideoGrant{
		RoomJoin:             true,
		Room:                 room,
		CanPublishData:       util.BoolPtr(true),
		CanUpdateOwnMetadata: util.BoolPtr(true),
	}
	randomUUID := uuid.New().String()
	userIdentity := fmt.Sprintf("%s_%s", userID, randomUUID)
	at.SetVideoGrant(grant).
		SetIdentity(userIdentity).
		SetName(userID).
		SetValidFor(24 * time.Hour)

	livekitToken, err := at.ToJWT()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate livekit token",
		})
	}

	// 9) 最終的にトークンをJSONで返す
	return c.JSON(http.StatusOK, map[string]string{
		"token": livekitToken,
	})
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
