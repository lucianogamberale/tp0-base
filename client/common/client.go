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
	config ClientConfig
	conn   net.Conn
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{config: config}
	return client
}

func (client *Client) sigtermSignalHandler() {
	log.Infof("action: sigterm_signal_handler | result: in_progress | client_id: %v", client.config.ID)

	if client.conn != nil {
		client.conn.Close()
		client.conn = nil
		log.Debugf("action: sigterm_client_connection_close | result: success | client_id: %v", client.config.ID)
	}

	log.Infof("action: sigterm_signal_handler | result: success | client_id: %v", client.config.ID)
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (client *Client) createClientSocket() {
	conn, err := net.Dial("tcp", client.config.ServerAddress)
	if err != nil {
		log.Fatalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			client.config.ID,
			err,
		)
	}
	client.conn = conn
}

func (client *Client) withNewClientSocketDo(function func() error) error {
	client.createClientSocket()
	defer func() {
		client.conn.Close()
		client.conn = nil
		log.Debugf("action: client_connection_close | result: success | client_id: %v", client.config.ID)
	}()
	return function()
}

func (client *Client) sendMessageAndReceiveReply(msgID int) error {
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
		return err
	}

	log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
		client.config.ID,
		msg,
	)
	return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (client *Client) StartClientLoop() error {
	signalReceiver := make(chan os.Signal, 1)
	defer func() {
		close(signalReceiver)
		log.Debugf("action: signal_channel_close | result: success | client_id: %v", client.config.ID)
	}()
	signal.Notify(signalReceiver, syscall.SIGTERM)

	err := error(nil)

	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; msgID <= client.config.LoopAmount; msgID++ {
		select {
		case <-signalReceiver:
			client.sigtermSignalHandler()
			return nil
		default:
			// Create the client socket. If it is created successfully
			// send the message and receive the reply
			err = client.withNewClientSocketDo(func() error {
				err := client.sendMessageAndReceiveReply(msgID)
				// Wait a time between sending one message and the next one
				time.Sleep(client.config.LoopPeriod)
				return err
			})
			if err != nil {
				log.Errorf("action: loop_finished | result: fail | client_id: %v", client.config.ID)
				return err
			}
		}
	}

	log.Infof("action: loop_finished | result: success | client_id: %v", client.config.ID)
	return nil
}
