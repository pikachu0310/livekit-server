package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/pikachu0310/livekit-server/internal/pkg/util"
)

// AuthTraQMiddlewareWithPathSkipper は AuthTraQClient 関数をミドルウェアとして呼び出し、
// 成功した場合はユーザ名を c.Set("traqUserID", userName) に設定します。
func AuthTraQMiddlewareWithPathSkipper(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// パスベースでスキップ
		skipPaths := map[string]bool{
			"/api/ping":    true,
			"/api/webhook": true,
			"/api/rooms":   true,
			"/api/ws":      true,
		}
		if skipPaths[c.Path()] {
			return next(c)
		}

		// AuthTraQClient で Bearerトークンを検証し、ユーザ名を取得
		userName, err := util.AuthTraQClient(c)
		if err != nil {
			// HTTPError が返ってきた場合はそのまま返す
			return err
		}

		// 検証成功なら、Echoのコンテキストにユーザ名をセット
		c.Set("traqUserID", userName)

		// 次のハンドラへ
		return next(c)
	}
}
