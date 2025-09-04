package common

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ============================== STRUCT DEFINITION ============================== //

type ClientConfig struct {
	ID            string
	ServerAddress string
}

type Client struct {
	config        ClientConfig
	conn          net.Conn
	clientRunning bool
}

// ============================== BUILDER ============================== //

func NewClient(config ClientConfig) *Client {
	client := &Client{config: config, clientRunning: false}
	return client
}

// ============================== PRIVATE - SIGNAL HANDLER ============================== //

func (client *Client) isRunning() bool {
	return client.clientRunning
}

// ============================== PRIVATE - SIGNAL HANDLER ============================== //

func (client *Client) sigtermSignalHandler() {
	log.Infof("action: sigterm_signal_handler | result: in_progress | client_id: %v", client.config.ID)

	client.clientRunning = false

	if client.conn != nil {
		client.conn.Close()
		client.conn = nil
		log.Debugf("action: sigterm_client_connection_close | result: success | client_id: %v", client.config.ID)
	}

	log.Infof("action: sigterm_signal_handler | result: success | client_id: %v", client.config.ID)
}

func (client *Client) whenNoSigtermReceivedDo(signalReceiver chan os.Signal, function func() error) error {
	if !client.isRunning() {
		return nil
	}

	select {
	case <-signalReceiver:
		client.sigtermSignalHandler()
		return nil
	default:
		return function()
	}
}

// ============================== PRIVATE - CREATE CLIENT CONNECTION ============================== //

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (client *Client) createClientSocket() {
	conn, err := net.Dial("tcp", client.config.ServerAddress)
	if err != nil {
		log.Fatalf("action: connect | result: fail | client_id: %v | error: %v", client.config.ID, err)
	}
	client.conn = conn
	log.Debugf("action: connect | result: success | client_id: %v | server_address: %v", client.config.ID, client.config.ServerAddress)
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
	log.Debugf("action: send_message | result: in_progress | client_id: %v | msg: %v", client.config.ID, message)

	writer := bufio.NewWriter(client.conn)
	_, err := writer.WriteString(message)
	if err != nil {
		log.Errorf("action: send_message | result: fail | client_id: %v | error: %v", client.config.ID, err)
		return err
	}

	err = writer.Flush()
	if err != nil {
		log.Errorf("action: flush_message | result: fail | client_id: %v | error: %v", client.config.ID, err)
	}

	log.Debugf("action: send_message | result: success | client_id: %v | msg: %v", client.config.ID, message)
	return nil
}

func (client *Client) receiveMessage() (string, error) {
	log.Debugf("action: receive_message | result: in_progress | client_id: %v", client.config.ID)

	reader := bufio.NewReader(client.conn)
	msg, err := reader.ReadString(END_MSG_DELIMITER[0])
	if err != nil {
		log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v", client.config.ID, err)
		return "", err
	}

	log.Debugf("action: receive_message | result: success | client_id: %v | msg: %v", client.config.ID, msg)
	return msg, nil
}

// ============================= PRIVATE - SEND BET BATCHS ============================== //

func (client *Client) sendBetMessage(bet *Bet) error {
	messageToSend := EncodeBetMessage(bet)
	err := client.sendMessage(messageToSend)
	if err != nil {
		return err
	}

	receivedMessage, err := client.receiveMessage()
	if err != nil {
		return err
	}

	batchSize := 1
	expectedMessage := EncodeAckMessage(fmt.Sprintf("%d", batchSize))
	if receivedMessage != expectedMessage {
		log.Errorf("action: ack_verification | result: fail | client_id: %v | expected: %v | received: %v",
			client.config.ID,
			expectedMessage,
			receivedMessage,
		)
		return errors.New("bad ack message, bet not correctly processed by server")
	}

	log.Infof("action: apuesta_enviada | result: success | dni: %s | numero: %s",
		bet.Document,
		bet.Number,
	)
	return nil
}

func (client *Client) sendBet(signalReceiver chan os.Signal, bet *Bet) error {
	return client.whenNoSigtermReceivedDo(signalReceiver, func() error {
		log.Infof("action: send_bet | result: in_progress | client_id: %v", client.config.ID)

		err := client.sendBetMessage(bet)
		if err != nil {
			log.Errorf("action: send_bet | result: fail | client_id: %v | error: %v", client.config.ID, err)
			return err
		}

		log.Infof("action: send_bet | result: in_progress | client_id: %v", client.config.ID)
		return nil
	})
}

// ============================== PUBLIC ============================== //

func (client *Client) SendBetToNationalLotteryHeadquarters(bet *Bet) error {
	client.clientRunning = true

	signalReceiver := make(chan os.Signal, 1)
	defer func() {
		close(signalReceiver)
		log.Debugf("action: signal_channel_close | result: success | client_id: %v", client.config.ID)
	}()
	signal.Notify(signalReceiver, syscall.SIGTERM)

	return client.withNewClientSocketDo(func() error {
		return client.sendBet(signalReceiver, bet)
	})
}
