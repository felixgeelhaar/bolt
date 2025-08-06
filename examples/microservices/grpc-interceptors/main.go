// Package main demonstrates gRPC client/server logging interceptors with Bolt.
// This example shows structured logging for gRPC services including request tracing,
// performance metrics, error handling, and streaming RPC logging.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/felixgeelhaar/bolt"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/felixgeelhaar/bolt/examples/microservices/grpc-interceptors/proto"
)

// Server implements the UserService gRPC server with logging
type Server struct {
	pb.UnimplementedUserServiceServer
	logger bolt.Logger
}

// NewServer creates a new gRPC server with logging
func NewServer() *Server {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		Level(bolt.InfoLevel).
		With().
		Str("service", "grpc-user-service").
		Str("version", "v1.0.0").
		Logger()

	return &Server{
		logger: logger,
	}
}

// GetUser implements the GetUser RPC method
func (s *Server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	correlationID := getCorrelationID(ctx)

	s.logger.Info().
		Str("correlation_id", correlationID).
		Str("method", "GetUser").
		Int64("user_id", req.UserId).
		Msg("Processing GetUser request")

	// Simulate business logic
	if req.UserId <= 0 {
		s.logger.Warn().
			Str("correlation_id", correlationID).
			Int64("user_id", req.UserId).
			Msg("Invalid user ID provided")
		return nil, status.Error(codes.InvalidArgument, "user ID must be positive")
	}

	// Simulate database lookup
	time.Sleep(10 * time.Millisecond)

	user := &pb.User{
		Id:        req.UserId,
		Name:      fmt.Sprintf("User %d", req.UserId),
		Email:     fmt.Sprintf("user%d@example.com", req.UserId),
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	s.logger.Info().
		Str("correlation_id", correlationID).
		Str("method", "GetUser").
		Int64("user_id", req.UserId).
		Str("user_email", user.Email).
		Msg("User retrieved successfully")

	return &pb.GetUserResponse{User: user}, nil
}

// CreateUser implements the CreateUser RPC method
func (s *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	correlationID := getCorrelationID(ctx)

	s.logger.Info().
		Str("correlation_id", correlationID).
		Str("method", "CreateUser").
		Str("user_name", req.Name).
		Str("user_email", req.Email).
		Msg("Processing CreateUser request")

	// Input validation
	if req.Name == "" || req.Email == "" {
		s.logger.Warn().
			Str("correlation_id", correlationID).
			Str("user_name", req.Name).
			Str("user_email", req.Email).
			Msg("Invalid user data provided")
		return nil, status.Error(codes.InvalidArgument, "name and email are required")
	}

	// Simulate database insertion
	time.Sleep(20 * time.Millisecond)

	user := &pb.User{
		Id:        123, // Simulate generated ID
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	s.logger.Info().
		Str("correlation_id", correlationID).
		Str("method", "CreateUser").
		Int64("user_id", user.Id).
		Str("user_name", user.Name).
		Str("user_email", user.Email).
		Msg("User created successfully")

	return &pb.CreateUserResponse{User: user}, nil
}

// ListUsers implements the ListUsers RPC method
func (s *Server) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	correlationID := getCorrelationID(ctx)

	s.logger.Info().
		Str("correlation_id", correlationID).
		Str("method", "ListUsers").
		Int32("page_size", req.PageSize).
		Str("page_token", req.PageToken).
		Msg("Processing ListUsers request")

	// Simulate database query
	time.Sleep(15 * time.Millisecond)

	// Create mock users
	users := make([]*pb.User, 0, req.PageSize)
	if req.PageSize == 0 {
		req.PageSize = 10 // Default page size
	}

	for i := int32(1); i <= req.PageSize && i <= 50; i++ { // Limit to 50 for demo
		users = append(users, &pb.User{
			Id:        int64(i),
			Name:      fmt.Sprintf("User %d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		})
	}

	s.logger.Info().
		Str("correlation_id", correlationID).
		Str("method", "ListUsers").
		Int("users_returned", len(users)).
		Int32("page_size", req.PageSize).
		Msg("Users listed successfully")

	return &pb.ListUsersResponse{
		Users:         users,
		NextPageToken: "next_page",
		TotalCount:    100, // Simulate total count
	}, nil
}

// StreamUsers implements the streaming StreamUsers RPC method
func (s *Server) StreamUsers(req *pb.StreamUsersRequest, stream pb.UserService_StreamUsersServer) error {
	ctx := stream.Context()
	correlationID := getCorrelationID(ctx)

	s.logger.Info().
		Str("correlation_id", correlationID).
		Str("method", "StreamUsers").
		Int("requested_users", len(req.UserIds)).
		Msg("Starting user stream")

	// Simulate streaming user events
	for i, userID := range req.UserIds {
		select {
		case <-ctx.Done():
			s.logger.Info().
				Str("correlation_id", correlationID).
				Str("method", "StreamUsers").
				Str("reason", "client_disconnected").
				Int("events_sent", i).
				Msg("Stream terminated by client")
			return ctx.Err()
		default:
		}

		// Simulate processing delay
		time.Sleep(100 * time.Millisecond)

		event := &pb.UserEvent{
			Type: pb.UserEvent_UPDATED,
			User: &pb.User{
				Id:        userID,
				Name:      fmt.Sprintf("User %d", userID),
				Email:     fmt.Sprintf("user%d@example.com", userID),
				CreatedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			},
			Timestamp: time.Now().Unix(),
		}

		if err := stream.Send(event); err != nil {
			s.logger.Error().
				Str("correlation_id", correlationID).
				Str("method", "StreamUsers").
				Int64("user_id", userID).
				Err(err).
				Msg("Failed to send user event")
			return err
		}

		s.logger.Debug().
			Str("correlation_id", correlationID).
			Str("method", "StreamUsers").
			Int64("user_id", userID).
			Str("event_type", "UPDATED").
			Msg("User event sent")
	}

	s.logger.Info().
		Str("correlation_id", correlationID).
		Str("method", "StreamUsers").
		Int("events_sent", len(req.UserIds)).
		Msg("User stream completed")

	return nil
}

// Interceptor implementations

// UnaryServerInterceptor provides logging for unary RPC calls
func UnaryServerInterceptor(logger bolt.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		
		// Extract or generate correlation ID
		correlationID := extractOrGenerateCorrelationID(ctx)
		ctx = context.WithValue(ctx, "correlation_id", correlationID)
		
		// Add correlation ID to outgoing metadata
		md := metadata.Pairs("correlation-id", correlationID)
		grpc.SetHeader(ctx, md)
		
		logger.Info().
			Str("correlation_id", correlationID).
			Str("method", info.FullMethod).
			Str("type", "unary").
			Interface("request", req).
			Msg("gRPC unary request started")

		// Call the handler
		resp, err := handler(ctx, req)
		
		duration := time.Since(start)
		
		// Log completion
		logEvent := logger.Info()
		if err != nil {
			logEvent = logger.Error()
		}
		
		logEvent.
			Str("correlation_id", correlationID).
			Str("method", info.FullMethod).
			Str("type", "unary").
			Dur("duration", duration).
			Float64("duration_ms", float64(duration.Nanoseconds())/1_000_000).
			Err(err).
			Msg("gRPC unary request completed")

		return resp, err
	}
}

// StreamServerInterceptor provides logging for streaming RPC calls
func StreamServerInterceptor(logger bolt.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		
		ctx := stream.Context()
		correlationID := extractOrGenerateCorrelationID(ctx)
		
		// Create wrapped stream with correlation ID
		wrappedStream := &correlationStream{
			ServerStream:  stream,
			correlationID: correlationID,
		}
		
		// Add correlation ID to outgoing metadata
		md := metadata.Pairs("correlation-id", correlationID)
		stream.SetHeader(md)
		
		logger.Info().
			Str("correlation_id", correlationID).
			Str("method", info.FullMethod).
			Str("type", "stream").
			Bool("client_streaming", info.IsClientStream).
			Bool("server_streaming", info.IsServerStream).
			Msg("gRPC stream request started")

		// Call the handler
		err := handler(srv, wrappedStream)
		
		duration := time.Since(start)
		
		// Log completion
		logEvent := logger.Info()
		if err != nil {
			logEvent = logger.Error()
		}
		
		logEvent.
			Str("correlation_id", correlationID).
			Str("method", info.FullMethod).
			Str("type", "stream").
			Dur("duration", duration).
			Float64("duration_ms", float64(duration.Nanoseconds())/1_000_000).
			Err(err).
			Msg("gRPC stream request completed")

		return err
	}
}

// correlationStream wraps ServerStream to include correlation ID in context
type correlationStream struct {
	grpc.ServerStream
	correlationID string
}

func (cs *correlationStream) Context() context.Context {
	return context.WithValue(cs.ServerStream.Context(), "correlation_id", cs.correlationID)
}

// Client interceptors

// UnaryClientInterceptor provides logging for outgoing unary RPC calls
func UnaryClientInterceptor(logger bolt.Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()
		
		correlationID := getOrGenerateCorrelationID(ctx)
		
		// Add correlation ID to outgoing metadata
		ctx = metadata.AppendToOutgoingContext(ctx, "correlation-id", correlationID)
		
		logger.Info().
			Str("correlation_id", correlationID).
			Str("method", method).
			Str("type", "unary_client").
			Str("target", cc.Target()).
			Interface("request", req).
			Msg("gRPC client request started")

		// Make the call
		err := invoker(ctx, method, req, reply, cc, opts...)
		
		duration := time.Since(start)
		
		// Log completion
		logEvent := logger.Info()
		if err != nil {
			logEvent = logger.Error()
		}
		
		logEvent.
			Str("correlation_id", correlationID).
			Str("method", method).
			Str("type", "unary_client").
			Str("target", cc.Target()).
			Dur("duration", duration).
			Float64("duration_ms", float64(duration.Nanoseconds())/1_000_000).
			Err(err).
			Msg("gRPC client request completed")

		return err
	}
}

// Utility functions

func extractOrGenerateCorrelationID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ids := md.Get("correlation-id"); len(ids) > 0 {
			return ids[0]
		}
	}
	return uuid.New().String()
}

func getOrGenerateCorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		return correlationID
	}
	return uuid.New().String()
}

func getCorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		return correlationID
	}
	return "unknown"
}

// Server main function
func runServer() error {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		Level(bolt.InfoLevel).
		With().
		Str("component", "grpc_server").
		Logger()

	// Create gRPC server with interceptors
	server := grpc.NewServer(
		grpc.UnaryInterceptor(UnaryServerInterceptor(logger)),
		grpc.StreamInterceptor(StreamServerInterceptor(logger)),
	)

	// Register service
	userService := NewServer()
	pb.RegisterUserServiceServer(server, userService)

	// Listen on port
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to listen on port 9090")
		return err
	}

	logger.Info().
		Str("address", lis.Addr().String()).
		Msg("Starting gRPC server")

	return server.Serve(lis)
}

// Client demo function
func runClient() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		Level(bolt.InfoLevel).
		With().
		Str("component", "grpc_client").
		Logger()

	// Connect to server with client interceptor
	conn, err := grpc.Dial("localhost:9090",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(UnaryClientInterceptor(logger)),
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to gRPC server")
		return
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)

	// Create context with correlation ID
	ctx := context.WithValue(context.Background(), "correlation_id", uuid.New().String())
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Test GetUser
	logger.Info().Msg("Testing GetUser RPC")
	getUserResp, err := client.GetUser(ctx, &pb.GetUserRequest{UserId: 1})
	if err != nil {
		logger.Error().Err(err).Msg("GetUser failed")
	} else {
		logger.Info().
			Int64("user_id", getUserResp.User.Id).
			Str("user_name", getUserResp.User.Name).
			Msg("GetUser succeeded")
	}

	// Test CreateUser
	logger.Info().Msg("Testing CreateUser RPC")
	createUserResp, err := client.CreateUser(ctx, &pb.CreateUserRequest{
		Name:  "John Doe",
		Email: "john@example.com",
	})
	if err != nil {
		logger.Error().Err(err).Msg("CreateUser failed")
	} else {
		logger.Info().
			Int64("user_id", createUserResp.User.Id).
			Str("user_name", createUserResp.User.Name).
			Msg("CreateUser succeeded")
	}

	// Test ListUsers
	logger.Info().Msg("Testing ListUsers RPC")
	listUsersResp, err := client.ListUsers(ctx, &pb.ListUsersRequest{
		PageSize: 5,
	})
	if err != nil {
		logger.Error().Err(err).Msg("ListUsers failed")
	} else {
		logger.Info().
			Int("users_count", len(listUsersResp.Users)).
			Int32("total_count", listUsersResp.TotalCount).
			Msg("ListUsers succeeded")
	}

	// Test StreamUsers
	logger.Info().Msg("Testing StreamUsers RPC")
	stream, err := client.StreamUsers(ctx, &pb.StreamUsersRequest{
		UserIds: []int64{1, 2, 3},
	})
	if err != nil {
		logger.Error().Err(err).Msg("StreamUsers failed")
		return
	}

	for {
		event, err := stream.Recv()
		if err == io.EOF {
			logger.Info().Msg("StreamUsers completed")
			break
		}
		if err != nil {
			logger.Error().Err(err).Msg("StreamUsers stream error")
			break
		}

		logger.Info().
			Int64("user_id", event.User.Id).
			Str("event_type", event.Type.String()).
			Int64("timestamp", event.Timestamp).
			Msg("Received user event")
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "client" {
		// Run as client
		time.Sleep(2 * time.Second) // Wait for server to start
		runClient()
		return
	}

	// Run as server
	go func() {
		if err := runServer(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("Shutting down server...")
}