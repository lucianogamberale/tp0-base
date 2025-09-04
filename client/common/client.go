package common

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ============================== CONSTANTS ============================== //

const (
	CSV_FIELD_DELIMITER   = ','
	CSV_COMMENT           = '#'
	CSV_FIELDS_PER_RECORD = 5
)

// ============================== STRUCT DEFINITION ============================== //

type ClientConfig struct {
	ID                         string
	ServerAddress              string
	MaxAmountOfBetsOnEachBatch int
	MaxKiBPerBatch             int
	AgencyFileName             string
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

// ============================= PRIVATE - READ BETS FROM CSV ============================== //

func (client *Client) withCsvReaderDo(function func(*csv.Reader) error) error {
	file, err := os.Open(client.config.AgencyFileName)
	if err != nil {
		log.Errorf("action: agency_file_open | result: fail | client_id: %v | error: %v", client.config.ID, err)
		return err
	}
	defer func() {
		file.Close()
		log.Debugf("action: agency_file_close | result: success | client_id: %v", client.config.ID)
	}()
	log.Debugf("action: agency_file_open | result: success | client_id: %v", client.config.ID)

	csvReader := csv.NewReader(file)
	csvReader.Comma = CSV_FIELD_DELIMITER
	csvReader.Comment = CSV_COMMENT
	csvReader.FieldsPerRecord = CSV_FIELDS_PER_RECORD
	return function(csvReader)
}

func (client *Client) readBetFromCsvUsing(csvReader *csv.Reader) (*Bet, error) {
	betRecord, err := csvReader.Read()
	if err != nil && err != io.EOF {
		log.Errorf("action: read_bet_from_csv | result: fail | client_id: %v | error: %v", client.config.ID, err)
		return nil, err
	} else if err == io.EOF {
		log.Debugf("action: no_more_bets_to_read_csv | result: success | client_id: %v", client.config.ID)
		return nil, err
	}

	agency := client.config.ID
	firstName := betRecord[0]
	lastName := betRecord[1]
	document := betRecord[2]
	birthdate := betRecord[3]
	number := betRecord[4]

	log.Debugf("action: read_bet_from_csv | result: success | client_id: %v | bet: %v", client.config.ID, betRecord)
	return NewBet(
		agency,
		firstName,
		lastName,
		document,
		birthdate,
		number,
	), nil
}

func (client *Client) readBetBatchFromCsvUsing(csvReader *csv.Reader) ([]*Bet, error) {
	log.Infof("action: read_bet_batch_from_csv | result: in_progress | client_id: %v", client.config.ID)

	betBatch := []*Bet{}
	amountOfReadBytesOnBatch := 0
	amountOfReadBytesOnBatch += len(BET_MSG_TYPE) + len(START_MSG_DELIMITER) + len(END_MSG_DELIMITER)

	for len(betBatch) < client.config.MaxAmountOfBetsOnEachBatch && amountOfReadBytesOnBatch+MAX_BYTES_BET <= client.config.MaxKiBPerBatch*KiB {
		bet, err := client.readBetFromCsvUsing(csvReader)
		if err != nil && err != io.EOF {
			log.Errorf("action: read_bet_batch_from_csv | result: fail | client_id: %v | error: %v", client.config.ID, err)
			return nil, err
		} else if err == io.EOF {
			log.Infof("action: no_more_bet_batchs_to_read_csv | result: success | client_id: %v | bet_batch_size: %v | bytes_on_batch: %v",
				client.config.ID,
				len(betBatch),
				amountOfReadBytesOnBatch,
			)
			return betBatch, err
		}

		betBatch = append(betBatch, bet)
		amountOfReadBytesOnBatch += bet.LengthWhenEncoded() + 1
	}

	log.Infof("action: read_bet_batch_from_csv | result: success | client_id: %v | bet_batch_size: %v | bytes_on_batch: %v",
		client.config.ID,
		len(betBatch),
		amountOfReadBytesOnBatch,
	)
	return betBatch, nil
}

func (client *Client) whileConditionWithEachBetBatchDo(condition func() bool, function func([]*Bet) error) error {
	return client.withCsvReaderDo(func(csvReader *csv.Reader) error {
		for condition() {
			betBatch, err := client.readBetBatchFromCsvUsing(csvReader)
			if err != nil && err != io.EOF {
				return err
			} else if err == io.EOF {
				if len(betBatch) == 0 {
					return nil
				}
				return function(betBatch)
			}

			if err = function(betBatch); err != nil {
				return err
			}
		}
		return nil
	})
}

// ============================= PRIVATE - SEND BET BATCHS ============================== //

func (client *Client) sendBetBatchMessage(betBatch []*Bet) error {
	log.Debugf("action: send_bet_batch_message | result: in_progress | client_id: %v", client.config.ID)

	messageToSend := EncodeBetBatchMessage(betBatch)
	err := client.sendMessage(messageToSend)
	if err != nil {
		return err
	}

	receivedMessage, err := client.receiveMessage()
	if err != nil {
		return err
	}

	batchSize := len(betBatch)
	expectedMessage := EncodeAckMessage(fmt.Sprintf("%d", batchSize))
	if receivedMessage != expectedMessage {
		log.Errorf("action: ack_verification | result: fail | client_id: %v | expected: %v | received: %v",
			client.config.ID,
			expectedMessage,
			receivedMessage,
		)
		return errors.New("bad ack message, bet batch not correctly processed by server")
	}

	log.Debugf("action: send_bet_batch_message | result: success | client_id: %v | bet_batch_size: %v", client.config.ID, batchSize)
	return nil
}

func (client *Client) sendAllBetsUsingBetBatchs(signalReceiver chan os.Signal) error {
	log.Infof("action: send_all_bets_using_bet_batchs | result: in_progress | client_id: %v", client.config.ID)

	err := client.whileConditionWithEachBetBatchDo(
		client.isRunning,
		func(betBatch []*Bet) error {
			return client.whenNoSigtermReceivedDo(signalReceiver, func() error { return client.sendBetBatchMessage(betBatch) })
		})
	if err != nil {
		log.Errorf("action: send_all_bets_using_bet_batchs | result: fail | client_id: %v", client.config.ID)
		return err
	}

	log.Infof("action: send_all_bets_using_bet_batchs | result: success | client_id: %v", client.config.ID)
	return nil
}

// ============================= PRIVATE - SEND NO MORE BETS ============================== //

func (client *Client) sendNoMoreBetsMessage() error {
	messageToSend := EncodeNoMoreBetsMessage(client.config.ID)
	err := client.sendMessage(messageToSend)
	if err != nil {
		return err
	}

	receivedMessage, err := client.receiveMessage()
	if err != nil {
		return err
	}

	expectedMessage := EncodeAckMessage(NO_MORE_BETS_MSG_TYPE)
	if receivedMessage != expectedMessage {
		log.Errorf("action: ack_verification | result: fail | client_id: %v | expected: %v | received: %v",
			client.config.ID,
			expectedMessage,
			receivedMessage,
		)
		return errors.New("bad ack message, no more bets message not correctly processed by server")
	}
	return nil
}

func (client *Client) notifyNoMoreBets(signalReceiver chan os.Signal) error {
	return client.whenNoSigtermReceivedDo(signalReceiver, func() error {
		log.Infof("action: send_no_more_bets_message | result: in_progress | client_id: %v", client.config.ID)

		err := client.sendNoMoreBetsMessage()
		if err != nil {
			log.Errorf("action: send_no_more_bets_message | result: fail | client_id: %v", client.config.ID)
			return err
		}

		log.Infof("action: send_no_more_bets_message | result: success | client_id: %v", client.config.ID)
		return nil
	})
}

// ============================== PUBLIC ============================== //

func (client *Client) SendAllBetsToNationalLotteryHeadquarters() error {
	client.clientRunning = true

	signalReceiver := make(chan os.Signal, 1)
	defer func() {
		close(signalReceiver)
		log.Debugf("action: signal_channel_close | result: success | client_id: %v", client.config.ID)
	}()
	signal.Notify(signalReceiver, syscall.SIGTERM)

	return client.withNewClientSocketDo(func() error {
		err := client.sendAllBetsUsingBetBatchs(signalReceiver)
		if err != nil {
			return err
		}

		return client.notifyNoMoreBets(signalReceiver)
	})
}
