package router

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
)

const swaggerIndexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="robots" content="noindex, nofollow">
  <title>API Documentation</title>
  <link rel="icon" type="image/png" sizes="32x32" href="./assets/favicon-32x32.png">
  <link rel="icon" type="image/png" sizes="16x16" href="./assets/favicon-16x16.png">
  <link rel="stylesheet" href="./assets/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="./assets/swagger-ui-bundle.js"></script>
  <script src="./assets/swagger-ui-standalone-preset.js"></script>
  <script src="./swagger-initializer.js"></script>
</body>
</html>
`

const swaggerInitializerJS = `window.onload = function () {
  window.ui = SwaggerUIBundle({
    dom_id: '#swagger-ui',
    urls: [
      { name: '后台管理接口', url: './openapi/api.json' },
      { name: 'AI 模型接口', url: './openapi/relay.json' }
    ],
    'urls.primaryName': 'AI 模型接口',
    deepLinking: true,
    displayRequestDuration: true,
    filter: true,
    validatorUrl: null,
    withCredentials: true,
    requestInterceptor: function (request) {
      request.headers = request.headers || {};
      request.headers['New-Api-User'] = '%d';
      return request;
    },
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    plugins: [SwaggerUIBundle.plugins.DownloadUrl],
    layout: 'StandaloneLayout'
  });
};
`

type swaggerAsset struct {
	contentType string
	data        []byte
}

var swaggerAssets = map[string]swaggerAsset{
	"swagger-ui.css": {
		contentType: "text/css; charset=utf-8",
		data:        swaggerFiles.FileSwaggerUICSS,
	},
	"swagger-ui-bundle.js": {
		contentType: "application/javascript; charset=utf-8",
		data:        swaggerFiles.FileSwaggerUIBundleJs,
	},
	"swagger-ui-standalone-preset.js": {
		contentType: "application/javascript; charset=utf-8",
		data:        swaggerFiles.FileSwaggerUIStandalonePresetJs,
	},
	"favicon-16x16.png": {
		contentType: "image/png",
		data:        swaggerFiles.FileFavicon16x16Png,
	},
	"favicon-32x32.png": {
		contentType: "image/png",
		data:        swaggerFiles.FileFavicon32x32Png,
	},
}

func setDocsSecurityHeaders(c *gin.Context) {
	c.Header("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; frame-ancestors 'none'")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "DENY")
}

func disableDocsCache(c *gin.Context) {
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
}

// SetDocsRouter serves the embedded OpenAPI specifications through a
// session-protected Swagger UI. The specifications and UI assets are protected
// by the same administrator check so they cannot be fetched independently.
func SetDocsRouter(router *gin.Engine, assets ThemeAssets) {
	docsRouter := router.Group("/docs")
	docsRouter.Use(middleware.RouteTag("docs"))
	docsRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	docsRouter.Use(middleware.AdminSessionAuth())
	{
		docsRouter.GET("", func(c *gin.Context) {
			c.Redirect(http.StatusTemporaryRedirect, c.Request.URL.Path+"/")
		})
		docsRouter.GET("/", func(c *gin.Context) {
			setDocsSecurityHeaders(c)
			disableDocsCache(c)
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerIndexHTML))
		})
		docsRouter.GET("/swagger-initializer.js", func(c *gin.Context) {
			setDocsSecurityHeaders(c)
			disableDocsCache(c)
			initializer := fmt.Sprintf(swaggerInitializerJS, c.GetInt("id"))
			c.Data(http.StatusOK, "application/javascript; charset=utf-8", []byte(initializer))
		})
		docsRouter.GET("/openapi/api.json", func(c *gin.Context) {
			disableDocsCache(c)
			c.Data(http.StatusOK, "application/json; charset=utf-8", assets.AdminOpenAPISpec)
		})
		docsRouter.GET("/openapi/relay.json", func(c *gin.Context) {
			disableDocsCache(c)
			c.Data(http.StatusOK, "application/json; charset=utf-8", assets.RelayOpenAPISpec)
		})
		docsRouter.GET("/assets/:filename", func(c *gin.Context) {
			asset, ok := swaggerAssets[c.Param("filename")]
			if !ok {
				c.Status(http.StatusNotFound)
				return
			}
			setDocsSecurityHeaders(c)
			c.Header("Cache-Control", "private, max-age=86400")
			c.Data(http.StatusOK, asset.contentType, asset.data)
		})
	}
}
