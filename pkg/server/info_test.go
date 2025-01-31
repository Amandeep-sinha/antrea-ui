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
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

func createTestControllerInfo(ctx context.Context, k8sClient dynamic.Interface, name string) error {
	controllerInfo := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": controllerInfoGVR.Group + "/" + controllerInfoGVR.Version,
			"kind":       "AntreaControllerInfo",
			"metadata": map[string]interface{}{
				"name": name,
			},
			// an empty Spec is fine for the sake of the test
			"spec": map[string]interface{}{},
		},
	}
	_, err := k8sClient.Resource(controllerInfoGVR).Create(ctx, controllerInfo, metav1.CreateOptions{})
	return err
}

func TestGetControllerInfo(t *testing.T) {
	ctx := context.Background()
	ts := newTestServer(t)
	require.NoError(t, createTestControllerInfo(ctx, ts.k8sClient, "antrea-controller"), "failed to create test ControllerInfo")

	req, err := http.NewRequest("GET", "/api/v1/info/controller", nil)
	require.NoError(t, err)
	ts.authorizeRequest(req)
	rr := httptest.NewRecorder()
	ts.router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "antrea-controller", gjson.GetBytes(rr.Body.Bytes(), "metadata.name").String())
}

func createTestAgentInfo(ctx context.Context, k8sClient dynamic.Interface, name string) error {
	agentInfo := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": agentInfoGVR.Group + "/" + agentInfoGVR.Version,
			"kind":       "AntreaAgentInfo",
			"metadata": map[string]interface{}{
				"name": name,
			},
			// an empty Spec is fine for the sake of the test
			"spec": map[string]interface{}{},
		},
	}
	_, err := k8sClient.Resource(agentInfoGVR).Create(ctx, agentInfo, metav1.CreateOptions{})
	return err
}

func TestGetAgentInfo(t *testing.T) {
	ctx := context.Background()
	ts := newTestServer(t)
	require.NoError(t, createTestAgentInfo(ctx, ts.k8sClient, "node-A"), "failed to create test AgentInfo")

	t.Run("valid name", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/info/agents/node-A", nil)
		require.NoError(t, err)
		ts.authorizeRequest(req)
		rr := httptest.NewRecorder()
		ts.router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "node-A", gjson.GetBytes(rr.Body.Bytes(), "metadata.name").String())
	})

	t.Run("invalid name", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/info/agents/node-B", nil)
		require.NoError(t, err)
		ts.authorizeRequest(req)
		rr := httptest.NewRecorder()
		ts.router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestGetAgentInfos(t *testing.T) {
	ctx := context.Background()
	ts := newTestServer(t)
	require.NoError(t, createTestAgentInfo(ctx, ts.k8sClient, "node-A"), "failed to create test AgentInfo")
	require.NoError(t, createTestAgentInfo(ctx, ts.k8sClient, "node-B"), "failed to create test AgentInfo")

	req, err := http.NewRequest("GET", "/api/v1/info/agents", nil)
	require.NoError(t, err)
	ts.authorizeRequest(req)
	rr := httptest.NewRecorder()
	ts.router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Len(t, gjson.ParseBytes(rr.Body.Bytes()).Array(), 2)
}
