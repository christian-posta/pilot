// Copyright 2016 Google Inc.
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

package envoy

import (
	"net/http"
	"strconv"

	restful "github.com/emicklei/go-restful"
	"github.com/golang/glog"

	"istio.io/manager/model"
)

// DiscoveryService publishes services, clusters, and routes for proxies
type DiscoveryService struct {
	services model.ServiceDiscovery
	server   *http.Server
}

type hosts struct {
	Hosts []host `json:"hosts"`
}

type host struct {
	Address string `json:"ip_address"`
	Port    int    `json:"port"`
	// Weight is an integer in the range [1, 100] or empty
	Weight int `json:"load_balancing_weight,omitempty"`
}

// NewDiscoveryService creates an Envoy discovery service on a given port
func NewDiscoveryService(services model.ServiceDiscovery, port int) *DiscoveryService {
	out := &DiscoveryService{
		services: services,
	}
	container := restful.NewContainer()
	out.Register(container)
	out.server = &http.Server{Addr: ":" + strconv.Itoa(port), Handler: container}
	return out
}

func (ds *DiscoveryService) Register(container *restful.Container) {
	ws := &restful.WebService{}
	ws.Produces(restful.MIME_JSON)
	ws.Route(ws.
		GET("/v1/registration/{service-key}").
		To(ds.ListEndpoints).
		Doc("SDS registration").
		Param(ws.PathParameter("service-key", "tuple of service name and tag name").DataType("string")).
		Writes(hosts{}))
	container.Add(ws)
}

func (ds *DiscoveryService) Run() error {
	glog.Infof("Starting discovery service at %v", ds.server.Addr)
	return ds.server.ListenAndServe()
}

func (ds *DiscoveryService) ListEndpoints(request *restful.Request, response *restful.Response) {
	key := request.PathParameter("service-key")
	svc := model.ParseServiceString(key)
	out := make([]host, 0)
	for _, ep := range ds.services.Endpoints(svc) {
		out = append(out, host{
			Address: ep.Endpoint.Address,
			Port:    ep.Endpoint.Port.Port,
		})
	}
	response.WriteEntity(hosts{out})
}