// Copyright 2023 Antrea Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"k8s.io/client-go/dynamic"

	"antrea.io/antrea-ui/pkg/auth"
	"antrea.io/antrea-ui/pkg/handlers/traceflow"
	"antrea.io/antrea-ui/pkg/password"
	"antrea.io/antrea-ui/pkg/version"
)

type server struct {
	logger                   logr.Logger
	k8sClient                dynamic.Interface
	traceflowRequestsHandler traceflow.RequestsHandler
	passwordStore            password.Store
	tokenManager             auth.TokenManager
	config                   serverConfig
}

type serverConfig struct {
	cookieSecure bool
}

type ServerOptions func(c *serverConfig)

func SetCookieSecure(v bool) ServerOptions {
	return func(c *serverConfig) {
		c.cookieSecure = v
	}
}

func NewServer(
	logger logr.Logger,
	k8sClient dynamic.Interface,
	traceflowRequestsHandler traceflow.RequestsHandler,
	passwordStore password.Store,
	tokenManager auth.TokenManager,
	options ...ServerOptions,
) *server {
	config := serverConfig{}
	for _, fn := range options {
		fn(&config)
	}
	return &server{
		logger:                   logger,
		k8sClient:                k8sClient,
		traceflowRequestsHandler: traceflowRequestsHandler,
		passwordStore:            passwordStore,
		tokenManager:             tokenManager,
		config:                   config,
	}
}

func (s *server) checkBearerToken(c *gin.Context) {
	if sError := func() *serverError {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			return &serverError{
				code:    http.StatusUnauthorized,
				message: "Missing Authorization header",
			}
		}
		t := strings.Split(auth, " ")
		if len(t) != 2 || t[0] != "Bearer" {
			return &serverError{
				code:    http.StatusUnauthorized,
				message: "Authorization header does not have valid format",
			}
		}
		if err := s.tokenManager.VerifyToken(t[1]); err != nil {
			return &serverError{
				code:    http.StatusUnauthorized,
				message: "Invalid Bearer token",
				err:     err,
			}
		}
		return nil
	}(); sError != nil {
		s.HandleError(c, sError)
		c.Abort()
		return
	}
}

func (s *server) AddRoutes(router *gin.Engine) {
	router.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	apiv1 := router.Group("/api/v1")
	apiv1.GET("/version", func(c *gin.Context) {
		c.String(http.StatusOK, version.GetFullVersionWithRuntimeInfo())
	})
	s.AddTraceflowRoutes(apiv1)
	s.AddInfoRoutes(apiv1)
	s.AddAccountRoutes(apiv1)
	s.AddAuthRoutes(apiv1)
}
