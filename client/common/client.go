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

// ============================== STRUCT DEFINITION ============================== //

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID                         string
	ServerAddress              string
	MaxAmountOfBetsOnEachBatch int
	MaxKiBPerBatch             int
	AgencyFileName             string
}

// Client Entity that encapsulates how
type Client struct {
	config         ClientConfig
	conn           net.Conn
	clientShutdown bool
}

// ============================== BUILDER ============================== //

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{config: config, clientShutdown: false}
	return client
}

// ============================== PRIVATE - SIGNAL HANDLER ============================== //

func (client *Client) sigtermSignalHandler() {
	log.Infof("action: sigterm_signal_handler | result: in_progress | client_id: %v", client.config.ID)

	client.clientShutdown = true

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
	log.Debugf("action: connect | result: success | client_id: %v | server_address: %v",
		client.config.ID,
		client.config.ServerAddress,
	)
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
	msg, err := reader.ReadString(END_DELIMITER[0])
	if err != nil {
		log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
			client.config.ID,
			err,
		)
		return "", err
	}

	log.Debugf("action: receive_message | result: success | client_id: %v | msg: %v",
		client.config.ID,
		msg,
	)
	return msg, nil
}

// ============================= PRIVATE - READ BETS FROM CSV ============================== //

func (client *Client) withCsvReaderDo(function func(csv.Reader) error) error {
	file, err := os.Open(client.config.AgencyFileName)
	if err != nil {
		log.Errorf("action: agency_file_open | result: fail | client_id: %v | error: %v",
			client.config.ID,
			err,
		)
		return err
	}
	defer func() {
		file.Close()
		log.Debugf("action: agency_file_close | result: success | client_id: %v", client.config.ID)
	}()
	log.Debugf("action: agency_file_open | result: success | client_id: %v", client.config.ID)

	csvReader := csv.NewReader(file)
	csvReader.Comma = ','         // set as constant
	csvReader.Comment = '#'       // set as constant
	csvReader.FieldsPerRecord = 5 // set as constant

	return function(*csvReader)
}

func (client *Client) readBetFromCsvUsing(csvReader csv.Reader) (*Bet, error) {
	betRecord, err := csvReader.Read()
	if err != nil && err != io.EOF {
		log.Errorf("action: read_bet_from_csv | result: fail | client_id: %v | error: %v", client.config.ID, err)
		return nil, err
	} else if err == io.EOF {
		log.Debugf("action: no_bets_to_read_csv | result: success | client_id: %v", client.config.ID)
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

func (client *Client) readBetBatchFromCsvUsing(csvReader csv.Reader) ([]*Bet, error) {
	log.Infof("action: read_bet_batch_from_csv | result: in_progress | client_id: %v", client.config.ID)

	betBatch := []*Bet{}
	amountOfReadBytesOnBatch := 0
	amountOfReadBytesOnBatch += len(BET_MSG_TYPE) + len(START_DELIMITER) + len(END_DELIMITER)

	for len(betBatch) < client.config.MaxAmountOfBetsOnEachBatch && amountOfReadBytesOnBatch+MAX_BYTES_BET <= client.config.MaxKiBPerBatch*KiB {
		bet, err := client.readBetFromCsvUsing(csvReader)
		if err != nil && err != io.EOF {
			log.Errorf("action: read_bet_batch_from_csv | result: fail | client_id: %v | error: %v", client.config.ID, err)
			return nil, err
		} else if err == io.EOF {
			log.Infof("action: no_bet_batchs_to_read_csv | result: success | client_id: %v | bet_batch_size: %v | bytes_on_batch: %v",
				client.config.ID,
				len(betBatch),
				amountOfReadBytesOnBatch,
			)
			return betBatch, err
		}

		betBatch = append(betBatch, bet)
		amountOfReadBytesOnBatch += bet.LengthAsString() + 1
	}

	log.Infof("action: read_bet_batch_from_csv | result: success | client_id: %v | bet_batch_size: %v | bytes_on_batch: %v",
		client.config.ID,
		len(betBatch),
		amountOfReadBytesOnBatch,
	)
	return betBatch, nil
}

func (client *Client) whileConditionWithEachBetBatchDo(condition func() bool, function func([]*Bet) error) (bool, error) {
	allBatchesSuccessfullyRead := false

	err := client.withCsvReaderDo(func(csvReader csv.Reader) error {
		for condition() {
			betBatch, err := client.readBetBatchFromCsvUsing(csvReader)
			if err != nil && err != io.EOF {
				allBatchesSuccessfullyRead = false
				return err
			} else if err == io.EOF {
				if len(betBatch) == 0 {
					allBatchesSuccessfullyRead = true
					return nil
				}
				err := function(betBatch)
				allBatchesSuccessfullyRead = (err == nil)
				return err
			}

			err = function(betBatch)
			if err != nil {
				allBatchesSuccessfullyRead = false
				return err
			}
		}
		return nil
	})

	return allBatchesSuccessfullyRead, err
}

// ============================= PRIVATE - SEND & ACK BET INFORMATION ============================== //

func (client *Client) sendBetBatch(betBatch []*Bet) error {
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
		log.Errorf("action: ack_message_check | result: fail | client_id: %v | expected: %v | received: %v",
			client.config.ID,
			expectedMessage,
			receivedMessage,
		)
		return errors.New("batch ACK message is not as expected, bet batch not correctly processed by server")
	}

	log.Infof("action: send_bet_batch | result: success | client_id: %v | bet_batch_size: %v",
		client.config.ID,
		batchSize,
	)
	return nil
}

// ============================== PUBLIC ============================== //

func (client *Client) SendAllBetsToNationalLotteryHeadquarters() error {
	log.Infof("action: send_all_bets_to_national_lottery_headquarters | result: in_progress | client_id: %v", client.config.ID)

	signalReceiver := make(chan os.Signal, 1)
	defer func() {
		close(signalReceiver)
		log.Debugf("action: signal_channel_close | result: success | client_id: %v", client.config.ID)
	}()
	signal.Notify(signalReceiver, syscall.SIGTERM)

	allBetBatchSent, err := client.whileConditionWithEachBetBatchDo(func() bool { return !client.clientShutdown }, func(betBatch []*Bet) error {
		select {
		case <-signalReceiver:
			client.sigtermSignalHandler()
			return nil
		default:
			err := client.withNewClientSocketDo(func() error {
				return client.sendBetBatch(betBatch)
			})
			if err != nil {
				return err
			}
			return nil
		}
	})

	if !allBetBatchSent || err != nil {
		log.Errorf("action: send_all_bets_to_national_lottery_headquarters | result: fail | client_id: %v", client.config.ID)
		return err
	}

	log.Infof("action: send_all_bets_to_national_lottery_headquarters | result: success | client_id: %v", client.config.ID)
	return nil
}
