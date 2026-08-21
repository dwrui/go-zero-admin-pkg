package authscope

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryClientInterceptor 将 context 中的 scope 写入 gRPC metadata。
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if scope, ok := ctx.Value(scopeContextKey{}).(string); ok && scope != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, metadataKey, scope)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// UnaryServerInterceptor 从 gRPC metadata 解析 scope 写入 context。
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		scope := FromIncoming(ctx)
		ctx = context.WithValue(ctx, scopeContextKey{}, scope)
		return handler(ctx, req)
	}
}

type scopeContextKey struct{}

// WithScope 在 HTTP 网关层写入 scope，供客户端拦截器读取。
func WithScope(ctx context.Context, scope string) context.Context {
	if scope == "" {
		scope = Admin
	}
	return context.WithValue(ctx, scopeContextKey{}, scope)
}

// FromContext 读取 scope（HTTP context 或 gRPC 服务端 context）。
func FromContext(ctx context.Context) string {
	if scope, ok := ctx.Value(scopeContextKey{}).(string); ok && scope != "" {
		return scope
	}
	return FromIncoming(ctx)
}
