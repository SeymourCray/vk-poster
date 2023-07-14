package controller

import (
	"context"
	"fmt"
	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/vk"
	"log"
	"net/http"
	"vk-poster/config"
)

const userKey = "user"

type authRoutes struct {
	username string
	password string
	conf     *oauth2.Config
	g        GroupsUseCase
}

func NewAuthRoutes(r *gin.Engine, g GroupsUseCase, cfg config.Config) {
	conf := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  fmt.Sprintf("http://%s:%d/private/auth", cfg.IP, cfg.HTTPPort),
		Scopes:       []string{"friends", "wall", "video", "photos", "offline", "groups"},
		Endpoint:     vk.Endpoint,
	}

	h := authRoutes{
		username: cfg.ServiceUser,
		password: cfg.ServicePassword,
		conf:     conf,
		g:        g,
	}

	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("mysession", store))
	// Setup the cookie store for session management
	private := r.Group("/private")
	private.Use(AuthRequired)
	{
		private.GET("/token", h.doRedirect)
		private.GET("/auth", h.getToken)
		NewGroupsRoutes(private, g)
	}

	r.POST("/login", h.login)
	r.GET("/login", loginPage)
}

func (r *authRoutes) doRedirect(c *gin.Context) {
	url := r.conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (r *authRoutes) getToken(c *gin.Context) {
	ctx := context.TODO()

	authCode := c.Request.URL.Query()["code"]

	tok, err := r.conf.Exchange(ctx, authCode[0])
	if err != nil {
		log.Println(err)
	}

	client := api.NewVK(tok.AccessToken)

	r.g.AddVkProvider(client)

	c.Redirect(http.StatusSeeOther, "/private/groups")
}

func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userKey)
	if user == nil {
		c.Redirect(http.StatusUnauthorized, "/login")
	}

	c.Next()
}

type credentials struct {
	username string `form:"username"`
	password string `form:"password"`
}

func (r *authRoutes) login(c *gin.Context) {
	cr := credentials{}
	c.Bind(&cr)

	if cr.username != r.username || cr.password != r.password {
		c.Redirect(http.StatusSeeOther, "/login")
	}

	session := sessions.Default(c)
	session.Set(userKey, cr.username) // In real world usage you'd set this to the users ID
	if err := session.Save(); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}

	c.Redirect(http.StatusSeeOther, "/private/token")
}

func loginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.tmpl", gin.H{})
}
