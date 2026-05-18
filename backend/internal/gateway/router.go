package gateway

import (
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

type Clients struct {
	APIBase  string
	AuthBase string
}

func NewRouter(clients *Clients) *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"data": gin.H{"status": "ok", "service": "gateway"}})
	})
	if clients != nil && clients.APIBase != "" {
		apiProxy := newReverseProxy(clients.APIBase)
		loginProxy := apiProxy
		if clients.AuthBase != "" {
			loginProxy = newReverseProxy(clients.AuthBase)
		}
		r.Any("/api/*path", func(c *gin.Context) {
			if c.Request.URL.Path == "/api/auth/login" && c.Request.Method == "POST" {
				gin.WrapH(loginProxy)(c)
				return
			}
			gin.WrapH(apiProxy)(c)
		})
		r.Any("/ws", gin.WrapH(apiProxy))
	}
	return r
}

func newReverseProxy(target string) *httputil.ReverseProxy {
	urlValue, err := url.Parse(target)
	if err != nil {
		panic(err)
	}
	return httputil.NewSingleHostReverseProxy(urlValue)
}
