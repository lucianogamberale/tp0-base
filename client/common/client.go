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

func (c *Client) sigtermSignalHandler() {
	log.Infof("action: sigterm_signal_handler | result: in_progress | client_id: %v", c.config.ID)

	c.clientRunning = false

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	log.Infof("action: sigterm_signal_handler | result: success | client_id: %v", c.config.ID)
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return err
}

func (c *Client) withNewClientSocketDo(function func()) {
	if err := c.createClientSocket(); err == nil {
		defer func() {
			c.conn.Close()
			c.conn = nil
		}()
		function()
	}
}

func (c *Client) sendMessageAndReceiveReply(msgID int) {
	// TODO: Modify the send to avoid short-write
	fmt.Fprintf(
		c.conn,
		"[CLIENT %v] Message N°%v\n",
		c.config.ID,
		msgID,
	)

	msg, err := bufio.NewReader(c.conn).ReadString('\n')
	if err != nil {
		log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	} else {
		log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
			c.config.ID,
			msg,
		)
	}
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	c.clientRunning = true

	signalReceiver := make(chan os.Signal, 1)
	signal.Notify(signalReceiver, syscall.SIGTERM)

	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; c.clientRunning && msgID <= c.config.LoopAmount; msgID++ {
		select {
		case <-signalReceiver:
			c.sigtermSignalHandler()
		default:
			// Create the client socket. If it is created successfully
			// send the message and receive the reply
			c.withNewClientSocketDo(func() {
				c.sendMessageAndReceiveReply(msgID)
				// Wait a time between sending one message and the next one
				time.Sleep(c.config.LoopPeriod)
			})
		}
	}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
