package common

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config        ClientConfig
	conn          net.Conn
	clientRunning bool
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{config: config, clientRunning: false}
	return client
}

func (client *Client) sigtermSignalHandler(signalReceiver chan os.Signal) {
	log.Infof("action: sigterm_signal_handler | result: in_progress | client_id: %v", client.config.ID)

	client.clientRunning = false

	if client.conn != nil {
		client.conn.Close()
		client.conn = nil
	}

	close(signalReceiver)

	log.Infof("action: sigterm_signal_handler | result: success | client_id: %v", client.config.ID)
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (client *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", client.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			client.config.ID,
			err,
		)
	}
	client.conn = conn
	return err
}

func (client *Client) withNewClientSocketDo(function func()) {
	if err := client.createClientSocket(); err == nil {
		defer func() {
			client.conn.Close()
			client.conn = nil
		}()
		function()
	}
}

func (client *Client) sendMessageAndReceiveReply(msgID int) {
	// TODO: Modify the send to avoid short-write
	fmt.Fprintf(
		client.conn,
		"[CLIENT %v] Message N°%v\n",
		client.config.ID,
		msgID,
	)

	msg, err := bufio.NewReader(client.conn).ReadString('\n')
	if err != nil {
		log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
			client.config.ID,
			err,
		)
	} else {
		log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
			client.config.ID,
			msg,
		)
	}
}

// StartClientLoop Send messages to the client until some time threshold is met
func (client *Client) StartClientLoop() {
	client.clientRunning = true

	signalReceiver := make(chan os.Signal, 1)
	signal.Notify(signalReceiver, syscall.SIGTERM)

	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; client.clientRunning && msgID <= client.config.LoopAmount; msgID++ {
		select {
		case <-signalReceiver:
			client.sigtermSignalHandler(signalReceiver)
		default:
			// Create the client socket. If it is created successfully
			// send the message and receive the reply
			client.withNewClientSocketDo(func() {
				client.sendMessageAndReceiveReply(msgID)
				// Wait a time between sending one message and the next one
				time.Sleep(client.config.LoopPeriod)
			})
		}
	}
	log.Infof("action: loop_finished | result: success | client_id: %v", client.config.ID)
}
