package server

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strconv"

	"github.com/pkg/errors"

	"github.com/ivolkoff/tcp-pow-go/internal/pkg/cache"
	"github.com/ivolkoff/tcp-pow-go/internal/pkg/clock"
	"github.com/ivolkoff/tcp-pow-go/internal/pkg/config"
	"github.com/ivolkoff/tcp-pow-go/internal/pkg/pow"
	"github.com/ivolkoff/tcp-pow-go/internal/pkg/protocol"
)

// Quotes - const array of quotes to respond on client's request
var Quotes = []string{
	"All saints who remember to keep and do these sayings, " +
		"walking in obedience to the commandments, " +
		"shall receive health in their navel and marrow to their bones",

	"And shall find wisdom and great treasures of knowledge, even hidden treasures",

	"And shall run and not be weary, and shall walk and not faint",

	"And I, the Lord, give unto them a promise, " +
		"that the destroying angel shall pass by them, " +
		"as the children of Israel, and not slay them",
}

var (
	ErrQuit = errors.New("client requests to close connection")
)

type Server interface {
	Run(address string) error
}

type server struct {
	di *Dependency
}

type Dependency struct {
	Config *config.Config
	Clock  clock.Clock
	Cache  cache.Cache
	Rand   *rand.Rand
}

func NewServer(di *Dependency) Server {
	return &server{di: di}
}

// Run - main function, launches server to listen on given address and handle new connections
func (s *server) Run(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	// Close the listener when the application closes.
	defer listener.Close()
	fmt.Println("listening", listener.Addr())
	for {
		// Listen for an incoming connection.
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("error accept connection: %w", err)
		}
		// Handle connections in a new goroutine.
		go s.handleConnection(conn)
	}
}

func (s *server) handleConnection(conn net.Conn) {
	fmt.Println("new client:", conn.RemoteAddr())
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		req, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("err read connection:", err)
			return
		}
		msg, err := s.processRequest(req, conn.RemoteAddr().String())
		if err != nil {
			fmt.Println("err process request:", err)
			return
		}
		if msg != nil {
			err := s.sendMsg(*msg, conn)
			if err != nil {
				fmt.Println("err send message:", err)
			}
		}
	}
}

// processRequest - process request from client
// returns not-nil pointer to Message if needed to send it back to client
func (s *server) processRequest(msgStr string, clientInfo string) (*protocol.Message, error) {
	ctx := context.Background()
	msg, err := protocol.ParseMessage(msgStr)
	if err != nil {
		return nil, err
	}
	// switch by header of msg
	switch msg.Header {
	case protocol.Quit:
		return nil, ErrQuit
	case protocol.RequestChallenge:
		fmt.Printf("client %s requests challenge\n", clientInfo)
		// create new challenge for client
		date := s.di.Clock.Now()

		// add new created rand value to cache to check it later on RequestResource stage
		// with duration in seconds
		randValue := s.di.Rand.Intn(100000)
		err := s.di.Cache.Add(ctx, s.cacheKey(clientInfo, randValue), s.di.Config.HashcashDuration)
		if err != nil {
			return nil, fmt.Errorf("err add rand to cache: %w", err)
		}

		hashcash := pow.HashcashData{
			Version:    1,
			ZerosCount: s.di.Config.HashcashZerosCount,
			Date:       date.Unix(),
			Resource:   clientInfo,
			Rand:       base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", randValue))),
			Counter:    0,
		}
		hashcashMarshaled, err := json.Marshal(hashcash)
		if err != nil {
			return nil, fmt.Errorf("err marshal hashcash: %v", err)
		}
		msg := protocol.Message{
			Header:  protocol.ResponseChallenge,
			Payload: string(hashcashMarshaled),
		}
		return &msg, nil
	case protocol.RequestResource:
		fmt.Printf("client %s requests resource with payload %s\n", clientInfo, msg.Payload)
		// parse client's solution
		var hashcash pow.HashcashData
		err := json.Unmarshal([]byte(msg.Payload), &hashcash)
		if err != nil {
			return nil, fmt.Errorf("err unmarshal hashcash: %w", err)
		}
		// validate hashcash params
		if hashcash.Resource != clientInfo {
			return nil, fmt.Errorf("invalid hashcash resource")
		}

		// decoding rand from base64 field in received client's hashcash
		randValueBytes, err := base64.StdEncoding.DecodeString(hashcash.Rand)
		if err != nil {
			return nil, fmt.Errorf("err decode rand: %w", err)
		}
		randValue, err := strconv.Atoi(string(randValueBytes))
		if err != nil {
			return nil, fmt.Errorf("err decode rand: %w", err)
		}

		// if rand exists in cache, it means, that hashcash is valid and really challenged by this server in past
		exists, err := s.di.Cache.Exist(ctx, s.cacheKey(clientInfo, randValue))
		if err != nil {
			return nil, fmt.Errorf("err get rand from cache: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("challenge expired or not sent")
		}

		// sent solution should not be outdated
		if s.di.Clock.Now().Unix()-hashcash.Date > s.di.Config.HashcashDuration {
			return nil, fmt.Errorf("challenge expired")
		}
		//to prevent indefinite computing on server if client sent hashcash with 0 counter
		maxIter := hashcash.Counter
		if maxIter == 0 {
			maxIter = 1
		}
		_, err = hashcash.ComputeHashcash(maxIter)
		if err != nil {
			return nil, fmt.Errorf("invalid hashcash")
		}
		//get random quote
		fmt.Printf("client %s succesfully computed hashcash %s\n", clientInfo, msg.Payload)
		msg := protocol.Message{
			Header:  protocol.ResponseResource,
			Payload: Quotes[s.di.Rand.Intn(len(Quotes))],
		}
		// delete rand from cache to prevent duplicated request with same hashcash value
		s.di.Cache.Delete(ctx, s.cacheKey(clientInfo, randValue))
		return &msg, nil
	default:
		return nil, fmt.Errorf("unknown header")
	}
}

// cacheKey - generate cache key for client
func (s *server) cacheKey(clientInfo string, rand int) string {
	return fmt.Sprintf("%s:%d", clientInfo, rand)
}

// sendMsg - send protocol message to connection
func (s *server) sendMsg(msg protocol.Message, conn net.Conn) error {
	msgStr := fmt.Sprintf("%s\n", msg.Stringify())
	_, err := conn.Write([]byte(msgStr))
	return err
}
