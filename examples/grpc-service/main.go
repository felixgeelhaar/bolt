// gRPC Microservice Example
//
// This example demonstrates using Bolt in a production gRPC microservice with:
// - gRPC interceptors for logging
// - Request/response tracking
// - Error handling and status codes
// - Performance metrics
// - OpenTelemetry trace propagation
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/felixgeelhaar/bolt"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Server represents our gRPC server
type Server struct {
	logger *bolt.Logger
}

// UserService interface (simplified for example)
type UserServiceServer interface {
	GetUser(context.Context, *GetUserRequest) (*User, error)
	CreateUser(context.Context, *CreateUserRequest) (*User, error)
}

// Request/Response types (simplified - normally from protobuf)
type GetUserRequest struct {
	ID string
}

type CreateUserRequest struct {
	Email string
	Name  string
}

type User struct {
	ID        string
	Email     string
	Name      string
	CreatedAt time.Time
}

// GetUser implements the GetUser RPC method
func (s *Server) GetUser(ctx context.Context, req *GetUserRequest) (*User, error) {
	// Get logger with trace context
	logger := s.getContextLogger(ctx)

	logger.Info().
		Str("user_id", req.ID).
		Msg("getting user")

	// Simulate database lookup
	user, err := s.getUserFromDB(req.ID)
	if err != nil {
		logger.Warn().
			Str("user_id", req.ID).
			Str("error", err.Error()).
			Msg("user not found")

		return nil, status.Error(codes.NotFound, "user not found")
	}

	logger.Info().
		Str("user_id", user.ID).
		Str("user_email", user.Email).
		Msg("user retrieved")

	return user, nil
}

// CreateUser implements the CreateUser RPC method
func (s *Server) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
	logger := s.getContextLogger(ctx)

	logger.Info().
		Str("email", req.Email).
		Str("name", req.Name).
		Msg("creating user")

	// Validate request
	if req.Email == "" || req.Name == "" {
		logger.Warn().
			Str("email", req.Email).
			Str("name", req.Name).
			Msg("validation failed")

		return nil, status.Error(codes.InvalidArgument, "email and name are required")
	}

	// Create user
	user := &User{
		ID:        fmt.Sprintf("user_%d", time.Now().Unix()),
		Email:     req.Email,
		Name:      req.Name,
		CreatedAt: time.Now(),
	}

	logger.Info().
		Str("user_id", user.ID).
		Str("user_email", user.Email).
		Msg("user created")

	return user, nil
}

// getUserFromDB simulates database lookup
func (s *Server) getUserFromDB(id string) (*User, error) {
	time.Sleep(10 * time.Millisecond) // Simulate DB latency

	if id == "123" {
		return &User{
			ID:        "123",
			Email:     "john@example.com",
			Name:      "John Doe",
			CreatedAt: time.Now().Add(-24 * time.Hour),
		}, nil
	}

	return nil, fmt.Errorf("user not found")
}

// getContextLogger extracts trace context and returns a context-aware logger
func (s *Server) getContextLogger(ctx context.Context) *bolt.Logger {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return s.logger.Ctx(ctx)
	}
	return s.logger
}

// UnaryLoggingInterceptor logs all unary RPC calls
func UnaryLoggingInterceptor(logger *bolt.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Extract metadata
		md, _ := metadata.FromIncomingContext(ctx)
		userAgent := ""
		if ua := md.Get("user-agent"); len(ua) > 0 {
			userAgent = ua[0]
		}

		// Get context logger
		ctxLogger := logger
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			ctxLogger = logger.Ctx(ctx)
		}

		// Log request
		ctxLogger.Info().
			Str("method", info.FullMethod).
			Str("user_agent", userAgent).
			Msg("grpc request started")

		// Handle request
		resp, err := handler(ctx, req)

		// Log response
		duration := time.Since(start)
		logLevel := ctxLogger.Info()

		if err != nil {
			st, _ := status.FromError(err)
			if st.Code() >= codes.Internal {
				logLevel = ctxLogger.Error()
			} else if st.Code() >= codes.InvalidArgument {
				logLevel = ctxLogger.Warn()
			}

			logLevel.
				Str("method", info.FullMethod).
				Str("code", st.Code().String()).
				Str("error", st.Message()).
				Dur("duration", duration).
				Msg("grpc request failed")
		} else {
			logLevel.
				Str("method", info.FullMethod).
				Str("code", codes.OK.String()).
				Dur("duration", duration).
				Msg("grpc request completed")
		}

		return resp, err
	}
}

// RecoveryInterceptor recovers from panics in gRPC handlers
func RecoveryInterceptor(logger *bolt.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error().
					Str("method", info.FullMethod).
					Str("panic", fmt.Sprintf("%v", r)).
					Msg("panic recovered in grpc handler")

				err = status.Error(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// StreamLoggingInterceptor logs streaming RPC calls
func StreamLoggingInterceptor(logger *bolt.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		logger.Info().
			Str("method", info.FullMethod).
			Bool("is_client_stream", info.IsClientStream).
			Bool("is_server_stream", info.IsServerStream).
			Msg("grpc stream started")

		err := handler(srv, ss)

		duration := time.Since(start)
		if err != nil {
			st, _ := status.FromError(err)
			logger.Error().
				Str("method", info.FullMethod).
				Str("code", st.Code().String()).
				Dur("duration", duration).
				Msg("grpc stream failed")
		} else {
			logger.Info().
				Str("method", info.FullMethod).
				Dur("duration", duration).
				Msg("grpc stream completed")
		}

		return err
	}
}

func main() {
	// Initialize logger
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Str("service", "grpc-user-service").
		Str("version", "1.0.0").
		Msg("starting gRPC server")

	// Create server
	_ = &Server{
		logger: logger,
	}

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			RecoveryInterceptor(logger),
			UnaryLoggingInterceptor(logger),
		),
		grpc.StreamInterceptor(StreamLoggingInterceptor(logger)),
	)

	// Note: In a real implementation, you would register your service here:
	// pb.RegisterUserServiceServer(grpcServer, server)

	// Start listener
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		logger.Error().
			Str("error", err.Error()).
			Msg("failed to listen")
		os.Exit(1)
	}

	// Start server in goroutine
	go func() {
		logger.Info().
			Str("addr", lis.Addr().String()).
			Msg("grpc server listening")

		if err := grpcServer.Serve(lis); err != nil {
			logger.Error().
				Str("error", err.Error()).
				Msg("grpc server error")
			os.Exit(1)
		}
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("shutting down grpc server")

	// Graceful stop
	grpcServer.GracefulStop()

	logger.Info().Msg("grpc server stopped")
}
