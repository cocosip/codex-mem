package mcp

import (
	"context"
	"errors"
	"io"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServeStdio runs the go-sdk MCP server over newline-delimited JSON on the
// supplied stdio-like streams.
func ServeStdio(ctx context.Context, server *sdkmcp.Server, stdin io.Reader, stdout io.Writer) error {
	err := server.Run(ctx, &sdkmcp.IOTransport{
		Reader: io.NopCloser(stdin),
		Writer: nopWriteCloser{Writer: stdout},
	})
	if errors.Is(err, io.EOF) || (err != nil && strings.HasSuffix(err.Error(), ": EOF")) {
		return nil
	}
	return err
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }
