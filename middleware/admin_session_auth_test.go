package middleware

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

func performAdminSessionRequest(t *testing.T, authenticated bool, role, status int) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("admin-session-auth-test"))))
	router.GET("/login", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("username", "tester")
		session.Set("role", role)
		session.Set("id", 42)
		session.Set("status", status)
		session.Set("group", "default")
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	router.GET("/docs/", AdminSessionAuth(), func(c *gin.Context) {
		c.String(http.StatusOK, strconv.Itoa(c.GetInt("id")))
	})

	var cookies []*http.Cookie
	if authenticated {
		loginRecorder := httptest.NewRecorder()
		router.ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodGet, "/login", nil))
		require.Equal(t, http.StatusNoContent, loginRecorder.Code)
		cookies = loginRecorder.Result().Cookies()
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	for _, sessionCookie := range cookies {
		request.AddCookie(sessionCookie)
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestAdminSessionAuthRequiresLogin(t *testing.T) {
	recorder := performAdminSessionRequest(t, false, 0, 0)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAdminSessionAuthRejectsCommonUser(t *testing.T) {
	recorder := performAdminSessionRequest(t, true, common.RoleCommonUser, common.UserStatusEnabled)

	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestAdminSessionAuthRejectsDisabledAdmin(t *testing.T) {
	recorder := performAdminSessionRequest(t, true, common.RoleAdminUser, common.UserStatusDisabled)

	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestAdminSessionAuthAllowsAdminWithoutUserHeader(t *testing.T) {
	recorder := performAdminSessionRequest(t, true, common.RoleAdminUser, common.UserStatusEnabled)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "42", recorder.Body.String())
}
