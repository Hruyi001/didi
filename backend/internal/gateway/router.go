package gateway

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

type Clients struct {
	APIBase   string
	AuthBase  string
	AdminBase string
}

func NewRouter(clients *Clients) *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"data": gin.H{"status": "ok", "service": "gateway"}})
	})
	if clients != nil && clients.APIBase != "" {
		apiProxy := newReverseProxy(clients.APIBase)
		authProxy := apiProxy
		if clients.AuthBase != "" {
			authProxy = newReverseProxy(clients.AuthBase)
		}
		adminProxy := apiProxy
		if clients.AdminBase != "" {
			adminProxy = newReverseProxy(clients.AdminBase)
		}
		r.Any("/api/*path", func(c *gin.Context) {
			if shouldProxyToAuth(c) {
				gin.WrapH(authProxy)(c)
				return
			}
			if shouldProxyToAdmin(c) {
				gin.WrapH(adminProxy)(c)
				return
			}
			gin.WrapH(apiProxy)(c)
		})
		r.Any("/ws", gin.WrapH(apiProxy))
	}
	return r
}

func shouldProxyToAuth(c *gin.Context) bool {
	if c.Request.Method != http.MethodPost {
		return false
	}
	switch c.Request.URL.Path {
	case "/api/auth/login", "/api/auth/send-code":
		return true
	default:
		return false
	}
}

func shouldProxyToAdmin(c *gin.Context) bool {
	if c.Request.Method != http.MethodGet {
		return false
	}
	switch c.Request.URL.Path {
	case "/api/admin/drivers":
		return true
	default:
		return false
	}
}

func newReverseProxy(target string) *httputil.ReverseProxy {
	urlValue, err := url.Parse(target)
	if err != nil {
		panic(err)
	}
	return httputil.NewSingleHostReverseProxy(urlValue)
}
