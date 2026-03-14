package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// FramedIOTransport adapts Content-Length framed stdio streams to the go-sdk
// transport interface.
type FramedIOTransport struct {
	Reader io.Reader
	Writer io.Writer
}

// NewFramedTransport constructs a go-sdk transport that preserves the
// repository's existing Content-Length framing over stdio-like streams.
func NewFramedTransport(reader io.Reader, writer io.Writer) *FramedIOTransport {
	return &FramedIOTransport{
		Reader: reader,
		Writer: writer,
	}
}

// Connect implements the go-sdk transport interface.
func (t *FramedIOTransport) Connect(context.Context) (sdkmcp.Connection, error) {
	return newFramedConn(t.Reader, t.Writer), nil
}

type framedConn struct {
	reader *bufio.Reader
	writer *bufio.Writer

	writeMu sync.Mutex

	incoming <-chan rawFrameOrErr
	queue    []jsonrpc.Message

	closeOnce sync.Once
	closed    chan struct{}
	closeErr  error
}

type rawFrameOrErr struct {
	payload []byte
	err     error
}

func newFramedConn(reader io.Reader, writer io.Writer) *framedConn {
	incoming := make(chan rawFrameOrErr)
	closed := make(chan struct{})
	bufferedReader := bufio.NewReader(reader)

	go func() {
		for {
			payload, err := readFrame(bufferedReader)
			select {
			case incoming <- rawFrameOrErr{payload: payload, err: err}:
			case <-closed:
				return
			}
			if err != nil {
				return
			}
		}
	}()

	return &framedConn{
		reader:   bufferedReader,
		writer:   bufio.NewWriter(writer),
		incoming: incoming,
		closed:   closed,
	}
}

func (c *framedConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if len(c.queue) > 0 {
		next := c.queue[0]
		c.queue = c.queue[1:]
		return next, nil
	}

	var item rawFrameOrErr
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case item = <-c.incoming:
	case <-c.closed:
		return nil, io.EOF
	}

	if item.err != nil {
		return nil, item.err
	}

	messages, err := decodeFramedMessages(item.payload)
	if err != nil {
		return nil, err
	}
	c.queue = messages[1:]
	return messages[0], nil
}

func (c *framedConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	body, err := jsonrpc.EncodeMessage(msg)
	if err != nil {
		return fmt.Errorf("marshal jsonrpc message: %w", err)
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return writeRawFrame(c.writer, body)
}

func (c *framedConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return c.closeErr
}

func (c *framedConn) SessionID() string { return "" }

func decodeFramedMessages(payload []byte) ([]jsonrpc.Message, error) {
	var rawBatch []json.RawMessage
	if err := json.Unmarshal(payload, &rawBatch); err == nil {
		if len(rawBatch) == 0 {
			return nil, errors.New("empty batch")
		}

		messages := make([]jsonrpc.Message, 0, len(rawBatch))
		for _, raw := range rawBatch {
			msg, err := jsonrpc.DecodeMessage(raw)
			if err != nil {
				return nil, err
			}
			messages = append(messages, msg)
		}
		return messages, nil
	}

	msg, err := jsonrpc.DecodeMessage(payload)
	if err != nil {
		return nil, err
	}
	return []jsonrpc.Message{msg}, nil
}

func writeRawFrame(writer *bufio.Writer, body []byte) error {
	if _, err := fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	if _, err := writer.Write(body); err != nil {
		return err
	}
	return writer.Flush()
}
