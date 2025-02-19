package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/ivolkoff/tcp-pow-go/internal/pkg/config"
	"github.com/ivolkoff/tcp-pow-go/internal/pkg/pow"
	"github.com/ivolkoff/tcp-pow-go/internal/pkg/protocol"
)

type Client interface {
	Run(address string) error
}

type client struct {
	di *Dependency
}

type Dependency struct {
	Config *config.Config
}

func NewClient(di *Dependency) Client {
	return &client{di: di}
}

// Run - main function, launches client to connect and work with server on address
func (c *client) Run(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}

	fmt.Println("connected to", address)
	defer conn.Close()

	// client will send new request every 5 seconds endlessly
	for {
		message, err := c.handleConnection(conn, conn)
		if err != nil {
			return err
		}
		fmt.Println("quote result:", message)
		time.Sleep(5 * time.Second)
	}
}

// handleConnection - scenario for TCP-client
// 1. request challenge from server
// 2. compute hashcash to check Proof of Work
// 3. send hashcash solution back to server
// 4. get result quote from server
// readerConn and writerConn divided to more convenient mock on testing
func (c *client) handleConnection(readerConn io.Reader, writerConn io.Writer) (string, error) {
	reader := bufio.NewReader(readerConn)

	// 1. requesting challenge
	err := c.sendMsg(protocol.Message{
		Header: protocol.RequestChallenge,
	}, writerConn)
	if err != nil {
		return "", fmt.Errorf("err send request: %w", err)
	}

	// reading and parsing response
	msgStr, err := c.readConnMsg(reader)
	if err != nil {
		return "", fmt.Errorf("err read msg: %w", err)
	}
	msg, err := protocol.ParseMessage(msgStr)
	if err != nil {
		return "", fmt.Errorf("err parse msg: %w", err)
	}
	var hashcash pow.HashcashData
	err = json.Unmarshal([]byte(msg.Payload), &hashcash)
	if err != nil {
		return "", fmt.Errorf("err parse hashcash: %w", err)
	}
	fmt.Println("got hashcash:", hashcash)

	// 2. got challenge, compute hashcash
	hashcash, err = hashcash.ComputeHashcash(c.di.Config.HashcashMaxIterations)
	if err != nil {
		return "", fmt.Errorf("err compute hashcash: %w", err)
	}
	fmt.Println("hashcash computed:", hashcash)
	// marshal solution to json
	byteData, err := json.Marshal(hashcash)
	if err != nil {
		return "", fmt.Errorf("err marshal hashcash: %w", err)
	}

	// 3. send challenge solution back to server
	err = c.sendMsg(protocol.Message{
		Header:  protocol.RequestResource,
		Payload: string(byteData),
	}, writerConn)
	if err != nil {
		return "", fmt.Errorf("err send request: %w", err)
	}
	fmt.Println("challenge sent to server")

	// 4. get result quote from server
	msgStr, err = c.readConnMsg(reader)
	if err != nil {
		return "", fmt.Errorf("err read msg: %w", err)
	}
	msg, err = protocol.ParseMessage(msgStr)
	if err != nil {
		return "", fmt.Errorf("err parse msg: %w", err)
	}
	return msg.Payload, nil
}

// readConnMsg - read string message from connection
func (c *client) readConnMsg(reader *bufio.Reader) (string, error) {
	return reader.ReadString('\n')
}

// sendMsg - send protocol message to connection
func (c *client) sendMsg(msg protocol.Message, conn io.Writer) error {
	msgStr := fmt.Sprintf("%s\n", msg.Stringify())
	_, err := conn.Write([]byte(msgStr))
	return err
}
