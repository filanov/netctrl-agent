package client

import (
	"context"
	"fmt"

	v1 "github.com/filanov/netctrl-server/pkg/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps the gRPC connection and AgentService client.
type Client struct {
	conn          *grpc.ClientConn
	agentService  v1.AgentServiceClient
}

// NewClient creates a new gRPC client connected to the specified address.
func NewClient(address string) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	return &Client{
		conn:         conn,
		agentService: v1.NewAgentServiceClient(conn),
	}, nil
}

// RegisterAgent sends a registration request to the server.
func (c *Client) RegisterAgent(ctx context.Context, req *v1.RegisterAgentRequest) (*v1.RegisterAgentResponse, error) {
	resp, err := c.agentService.RegisterAgent(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}
	return resp, nil
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
