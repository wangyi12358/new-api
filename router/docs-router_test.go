package router

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newDocsTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("docs-router-test"))))
	router.GET("/login/:role", func(c *gin.Context) {
		role, err := strconv.Atoi(c.Param("role"))
		require.NoError(t, err)
		session := sessions.Default(c)
		session.Set("username", "docs-tester")
		session.Set("role", role)
		session.Set("id", 73)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	SetDocsRouter(router, ThemeAssets{
		AdminOpenAPISpec: []byte(`{"openapi":"3.0.1","info":{"title":"admin","version":"1"},"paths":{}}`),
		RelayOpenAPISpec: []byte(`{"openapi":"3.0.1","info":{"title":"relay","version":"1"},"paths":{}}`),
	})
	return router
}

func docsSessionCookies(t *testing.T, router *gin.Engine, role int) []*http.Cookie {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/login/"+strconv.Itoa(role), nil)
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusNoContent, recorder.Code)
	return recorder.Result().Cookies()
}

func performDocsRequest(router *gin.Engine, target string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	for _, sessionCookie := range cookies {
		request.AddCookie(sessionCookie)
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestDocsRouterProtectsPageAndSpecifications(t *testing.T) {
	router := newDocsTestRouter(t)

	require.Equal(t, http.StatusUnauthorized, performDocsRequest(router, "/docs/", nil).Code)
	require.Equal(t, http.StatusUnauthorized, performDocsRequest(router, "/docs/openapi/api.json", nil).Code)

	userCookies := docsSessionCookies(t, router, common.RoleCommonUser)
	require.Equal(t, http.StatusForbidden, performDocsRequest(router, "/docs/", userCookies).Code)

	adminCookies := docsSessionCookies(t, router, common.RoleAdminUser)
	page := performDocsRequest(router, "/docs/", adminCookies)
	require.Equal(t, http.StatusOK, page.Code)
	require.Contains(t, page.Body.String(), "swagger-ui")
	require.Equal(t, "no-store, no-cache, must-revalidate, private, max-age=0", page.Header().Get("Cache-Control"))

	adminSpec := performDocsRequest(router, "/docs/openapi/api.json", adminCookies)
	require.Equal(t, http.StatusOK, adminSpec.Code)
	require.Contains(t, adminSpec.Body.String(), `"title":"admin"`)

	relaySpec := performDocsRequest(router, "/docs/openapi/relay.json", adminCookies)
	require.Equal(t, http.StatusOK, relaySpec.Code)
	require.Contains(t, relaySpec.Body.String(), `"title":"relay"`)
}

func TestDocsRouterServesInitializerAndEmbeddedAssets(t *testing.T) {
	router := newDocsTestRouter(t)
	adminCookies := docsSessionCookies(t, router, common.RoleAdminUser)

	redirect := performDocsRequest(router, "/docs", adminCookies)
	require.Equal(t, http.StatusTemporaryRedirect, redirect.Code)
	require.Equal(t, "/docs/", redirect.Header().Get("Location"))

	initializer := performDocsRequest(router, "/docs/swagger-initializer.js", adminCookies)
	require.Equal(t, http.StatusOK, initializer.Code)
	require.Contains(t, initializer.Body.String(), "'New-Api-User'] = '73'")
	require.Contains(t, initializer.Body.String(), "./openapi/relay.json")

	stylesheet := performDocsRequest(router, "/docs/assets/swagger-ui.css", adminCookies)
	require.Equal(t, http.StatusOK, stylesheet.Code)
	require.NotEmpty(t, stylesheet.Body.Bytes())

	missingAsset := performDocsRequest(router, "/docs/assets/unknown.js", adminCookies)
	require.Equal(t, http.StatusNotFound, missingAsset.Code)
}
