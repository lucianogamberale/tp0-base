package common

import (
	"bufio"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ============================== STRUCT DEFINITION ============================== //

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
}

// Client Entity that encapsulates how
type Client struct {
	config        ClientConfig
	conn          net.Conn
	clientRunning bool
}

// ============================== BUILDER ============================== //

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{config: config, clientRunning: false}
	return client
}

// ============================== PRIVATE - SIGNAL HANDLER ============================== //

func (client *Client) buildSignalReceiver() chan os.Signal {
	signalReceiver := make(chan os.Signal, 1)
	signal.Notify(signalReceiver, syscall.SIGTERM)
	return signalReceiver
}

func (client *Client) sigtermSignalHandler(signalReceiver chan os.Signal) {
	log.Infof("action: sigterm_signal_handler | result: in_progress | client_id: %v", client.config.ID)

	client.clientRunning = false

	if client.conn != nil {
		client.conn.Close()
		client.conn = nil
		log.Debugf("action: closing_connection | result: success | client_id: %v", client.config.ID)
	}

	close(signalReceiver)
	log.Debugf("action: closing_signal_receiver | result: success | client_id: %v", client.config.ID)

	log.Infof("action: sigterm_signal_handler | result: success | client_id: %v", client.config.ID)
}

// ============================== PRIVATE - CREATE CLIENT CONNECTION ============================== //

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

// ============================== PRIVATE - SEND/RECEIVE MESSAGES ============================== //

func (client *Client) sendMessage(message string) error {
	writer := bufio.NewWriter(client.conn)
	_, err := writer.WriteString(message)
	if err != nil {
		log.Errorf("action: send_message | result: fail | client_id: %v | error: %v",
			client.config.ID,
			err,
		)
		return err
	}

	err = writer.Flush()
	if err != nil {
		log.Errorf("action: flush_message | result: fail | client_id: %v | error: %v",
			client.config.ID,
			err,
		)
	}

	log.Infof("action: send_message | result: success | client_id: %v | msg: %v",
		client.config.ID,
		message,
	)
	return nil
}

func (client *Client) receiveMessage() (string, error) {
	reader := bufio.NewReader(client.conn)
	msg, err := reader.ReadString(DELIMITER)
	if err != nil {
		log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
			client.config.ID,
			err,
		)
		return "", err
	}

	log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
		client.config.ID,
		msg,
	)
	return msg, nil
}

// ============================= PRIVATE - SEND & ACK BET INFORMATION ============================== //

func (client *Client) sendBetInformationAckReceipt(bet *Bet) {
	messageToSend := BetMessageFor(bet)
	err := client.sendMessage(messageToSend)
	if err != nil {
		return
	}

	receivedMessage, err := client.receiveMessage()
	if err != nil {
		return
	}

	if receivedMessage == AckMessage("1") {
		log.Infof("action: apuesta_enviada | result: success | dni: %s | numero: %s",
			bet.document,
			bet.number,
		)
	} else {
		log.Errorf("action: apuesta_enviada | result: fail | dni: %s | numero: %s",
			bet.document,
			bet.number,
		)
	}
}

// ============================== PUBLIC ============================== //

func (client *Client) SendBetInformation(bet *Bet) {
	client.clientRunning = true
	signalReceiver := client.buildSignalReceiver()

	select {
	case <-signalReceiver:
		client.sigtermSignalHandler(signalReceiver)
	default:
		client.withNewClientSocketDo(func() {
			client.sendBetInformationAckReceipt(bet)
		})
	}
}
