// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: depot/cli/v1beta1/projects.proto

package cliv1beta1connect

import (
	context "context"
	errors "errors"
	connect_go "github.com/bufbuild/connect-go"
	v1beta1 "github.com/depot/cli/pkg/proto/depot/cli/v1beta1"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect_go.IsAtLeastVersion0_1_0

const (
	// ProjectsServiceName is the fully-qualified name of the ProjectsService service.
	ProjectsServiceName = "depot.cli.v1beta1.ProjectsService"
)

// ProjectsServiceClient is a client for the depot.cli.v1beta1.ProjectsService service.
type ProjectsServiceClient interface {
	ListProjects(context.Context, *connect_go.Request[v1beta1.ListProjectsRequest]) (*connect_go.Response[v1beta1.ListProjectsResponse], error)
	ResetProjectCache(context.Context, *connect_go.Request[v1beta1.ResetProjectCacheRequest]) (*connect_go.Response[v1beta1.ResetProjectCacheResponse], error)
}

// NewProjectsServiceClient constructs a client for the depot.cli.v1beta1.ProjectsService service.
// By default, it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped
// responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewProjectsServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) ProjectsServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &projectsServiceClient{
		listProjects: connect_go.NewClient[v1beta1.ListProjectsRequest, v1beta1.ListProjectsResponse](
			httpClient,
			baseURL+"/depot.cli.v1beta1.ProjectsService/ListProjects",
			opts...,
		),
		resetProjectCache: connect_go.NewClient[v1beta1.ResetProjectCacheRequest, v1beta1.ResetProjectCacheResponse](
			httpClient,
			baseURL+"/depot.cli.v1beta1.ProjectsService/ResetProjectCache",
			opts...,
		),
	}
}

// projectsServiceClient implements ProjectsServiceClient.
type projectsServiceClient struct {
	listProjects      *connect_go.Client[v1beta1.ListProjectsRequest, v1beta1.ListProjectsResponse]
	resetProjectCache *connect_go.Client[v1beta1.ResetProjectCacheRequest, v1beta1.ResetProjectCacheResponse]
}

// ListProjects calls depot.cli.v1beta1.ProjectsService.ListProjects.
func (c *projectsServiceClient) ListProjects(ctx context.Context, req *connect_go.Request[v1beta1.ListProjectsRequest]) (*connect_go.Response[v1beta1.ListProjectsResponse], error) {
	return c.listProjects.CallUnary(ctx, req)
}

// ResetProjectCache calls depot.cli.v1beta1.ProjectsService.ResetProjectCache.
func (c *projectsServiceClient) ResetProjectCache(ctx context.Context, req *connect_go.Request[v1beta1.ResetProjectCacheRequest]) (*connect_go.Response[v1beta1.ResetProjectCacheResponse], error) {
	return c.resetProjectCache.CallUnary(ctx, req)
}

// ProjectsServiceHandler is an implementation of the depot.cli.v1beta1.ProjectsService service.
type ProjectsServiceHandler interface {
	ListProjects(context.Context, *connect_go.Request[v1beta1.ListProjectsRequest]) (*connect_go.Response[v1beta1.ListProjectsResponse], error)
	ResetProjectCache(context.Context, *connect_go.Request[v1beta1.ResetProjectCacheRequest]) (*connect_go.Response[v1beta1.ResetProjectCacheResponse], error)
}

// NewProjectsServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewProjectsServiceHandler(svc ProjectsServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/depot.cli.v1beta1.ProjectsService/ListProjects", connect_go.NewUnaryHandler(
		"/depot.cli.v1beta1.ProjectsService/ListProjects",
		svc.ListProjects,
		opts...,
	))
	mux.Handle("/depot.cli.v1beta1.ProjectsService/ResetProjectCache", connect_go.NewUnaryHandler(
		"/depot.cli.v1beta1.ProjectsService/ResetProjectCache",
		svc.ResetProjectCache,
		opts...,
	))
	return "/depot.cli.v1beta1.ProjectsService/", mux
}

// UnimplementedProjectsServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedProjectsServiceHandler struct{}

func (UnimplementedProjectsServiceHandler) ListProjects(context.Context, *connect_go.Request[v1beta1.ListProjectsRequest]) (*connect_go.Response[v1beta1.ListProjectsResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("depot.cli.v1beta1.ProjectsService.ListProjects is not implemented"))
}

func (UnimplementedProjectsServiceHandler) ResetProjectCache(context.Context, *connect_go.Request[v1beta1.ResetProjectCacheRequest]) (*connect_go.Response[v1beta1.ResetProjectCacheResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("depot.cli.v1beta1.ProjectsService.ResetProjectCache is not implemented"))
}