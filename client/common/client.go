package common

import (
	"bufio"
	"errors"
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
	config ClientConfig
	conn   net.Conn
}

// ============================== BUILDER ============================== //

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{config: config}
	return client
}

// ============================== PRIVATE - SIGNAL HANDLER ============================== //

func (client *Client) sigtermSignalHandler() {
	log.Infof("action: sigterm_signal_handler | result: in_progress | client_id: %v", client.config.ID)

	if client.conn != nil {
		client.conn.Close()
		client.conn = nil
		log.Debugf("action: sigterm_client_connection_close | result: success | client_id: %v", client.config.ID)
	}

	log.Infof("action: sigterm_signal_handler | result: success | client_id: %v", client.config.ID)
}

// ============================== PRIVATE - CREATE CLIENT CONNECTION ============================== //

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

	log.Debugf("action: send_message | result: success | client_id: %v | msg: %v",
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

func (client *Client) sendBetInformationAckReceipt(bet *Bet) error {
	messageToSend := EncodeBetMessage(bet)
	err := client.sendMessage(messageToSend)
	if err != nil {
		return err
	}

	receivedMessage, err := client.receiveMessage()
	if err != nil {
		return err
	}

	expectedMessage := EncodeAckMessage("1")
	if receivedMessage != expectedMessage {
		return errors.New("bad ack message received")
	}

	log.Infof("action: apuesta_enviada | result: success | dni: %s | numero: %s",
		bet.document,
		bet.number,
	)
	return nil
}

// ============================== PUBLIC ============================== //

func (client *Client) SendBetInformation(bet *Bet) error {
	log.Infof("action: send_bet | result: in_progress | client_id: %v", client.config.ID)

	signalReceiver := make(chan os.Signal, 1)
	defer func() {
		close(signalReceiver)
		log.Debugf("action: signal_channel_close | result: success | client_id: %v", client.config.ID)
	}()
	signal.Notify(signalReceiver, syscall.SIGTERM)

	select {
	case <-signalReceiver:
		client.sigtermSignalHandler()
		log.Errorf("action: send_bet | result: fail | client_id: %v", client.config.ID)
		return nil
	default:
		err := client.withNewClientSocketDo(func() error {
			return client.sendBetInformationAckReceipt(bet)
		})
		if err != nil {
			log.Errorf("action: send_bet | result: fail | client_id: %v", client.config.ID)
			return err
		}
	}

	log.Infof("action: send_bet | result: success | client_id: %v", client.config.ID)
	return nil
}
