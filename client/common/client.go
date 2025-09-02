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
	"time"

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
	WaitLoopPeriod             time.Duration
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

func (client *Client) handleSigtermDuring(signalReceiver chan os.Signal, function func() error) error {
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
	msg, err := reader.ReadString(END_MSG_DELIMITER[0])
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
	amountOfReadBytesOnBatch += len(BET_MSG_TYPE) + len(START_MSG_DELIMITER) + len(END_MSG_DELIMITER)

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
		amountOfReadBytesOnBatch += bet.LengthWhenEncoded() + 1
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

// ============================= PRIVATE - SEND BET BATCHS ============================== //

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
		log.Errorf("action: ack_bet_message_check | result: fail | client_id: %v | expected: %v | received: %v",
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

func (client *Client) sendAllBetsUsingBatchs(signalReceiver chan os.Signal) (bool, error) {
	log.Infof("action: send_all_bets_to_lottery | result: in_progress | client_id: %v", client.config.ID)

	allBetBatchSent, err := client.whileConditionWithEachBetBatchDo(
		client.isRunning,
		func(betBatch []*Bet) error {
			return client.handleSigtermDuring(signalReceiver, func() error {
				return client.sendBetBatch(betBatch)
			})
		})
	if err != nil {
		log.Errorf("action: send_all_bets_to_lottery | result: in_progress | client_id: %v", client.config.ID)
		return false, err
	}

	log.Infof("action: send_all_bets_to_lottery | result: success | client_id: %v", client.config.ID)
	return allBetBatchSent, nil
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
		log.Errorf("action: ack_no_more_bets_message_check | result: fail | client_id: %v | expected: %v | received: %v",
			client.config.ID,
			expectedMessage,
			receivedMessage,
		)
		return errors.New("no more bets ACK message is not as expected")
	}
	return nil
}

func (client *Client) notifyNoMoreBets() error {
	log.Infof("action: notify_no_more_bets_message | result: in_progress | client_id: %v", client.config.ID)
	if !client.isRunning() {
		log.Infof("action: notify_no_more_bets_message | result: fail | client_id: %v", client.config.ID)
		return nil
	}

	err := client.sendNoMoreBetsMessage()
	if err != nil {
		log.Errorf("action: notify_no_more_bets_message | result: fail | client_id: %v", client.config.ID)
		return err
	}

	log.Infof("action: notify_no_more_bets_message | result: success | client_id: %v", client.config.ID)
	return nil
}

// ============================= PRIVATE - QUERY FOR WINNERS ============================== //

func (client *Client) tryToAskForWinners() (bool, []string, error) {
	isDrawHeld := false
	winners := []string{}

	err := client.withNewClientSocketDo(func() error {
		messageToSend := EncodeAskForWinnersMessage(client.config.ID)
		err := client.sendMessage(messageToSend)
		if err != nil {
			return err
		}

		receivedMessage, err := client.receiveMessage()
		if err != nil {
			return err
		}

		switch GetMessageType(receivedMessage) {
		case WAIT_MSG_TYPE:
			isDrawHeld = false
		case WINNERS_MSG_TYPE:
			isDrawHeld = true
			winners, err = DecodeWinnersMessage(receivedMessage)
			if err != nil {
				return err
			}
		default:
			log.Errorf("action: unknown_message_type | result: fail | client_id: %v | msg: %v",
				client.config.ID,
				receivedMessage,
			)
			return errors.New("unknown message type received from server")
		}

		log.Debugf("action: try_ask_for_winners | result: success | client_id: %v | is_draw_held: %v",
			client.config.ID,
			isDrawHeld,
		)
		return nil
	})

	return isDrawHeld, winners, err
}

func (client *Client) whileConditionKeepTryingToAskForWinners(
	condition func() bool,
	whenDrawNotHeldFunction func(),
	whenDrawHeldFunction func(winners []string),
) error {
	for condition() {
		isDrawHeld, winners, err := client.tryToAskForWinners()
		if err != nil {
			return err
		}

		if !isDrawHeld {
			log.Infof("action: lottery_has_not_held_the_draw_yet | result: success | client_id: %v", client.config.ID)
			whenDrawNotHeldFunction()
		} else {
			log.Infof("action: lottery_held_the_draw | result: success | client_id: %v | winners: %v", client.config.ID, winners)
			whenDrawHeldFunction(winners)
			return nil
		}
	}
	return nil
}

func (client *Client) askForWinners(signalReceiver chan os.Signal) error {
	log.Infof("action: ask_for_winners | result: in_progress | client_id: %v", client.config.ID)

	err := client.whileConditionKeepTryingToAskForWinners(
		client.isRunning,
		func() {
			client.handleSigtermDuring(signalReceiver, func() error {
				time.Sleep(client.config.WaitLoopPeriod)
				return nil
			})
		},
		func(winners []string) {
			client.handleSigtermDuring(signalReceiver, func() error {
				log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v", len(winners))
				return nil
			})
		},
	)
	if err != nil {
		log.Errorf("action: ask_for_winners | result: fail | client_id: %v", client.config.ID)
		return err
	}

	log.Infof("action: ask_for_winners | result: success | client_id: %v", client.config.ID)
	return nil
}

// ============================== PUBLIC ============================== //

func (client *Client) SendAllBetsToNationalLotteryHeadquartersThenAskForWinners() error {
	client.clientRunning = true

	signalReceiver := make(chan os.Signal, 1)
	defer func() {
		close(signalReceiver)
		log.Debugf("action: signal_channel_close | result: success | client_id: %v", client.config.ID)
	}()
	signal.Notify(signalReceiver, syscall.SIGTERM)

	err := client.withNewClientSocketDo(func() error {
		_, err := client.sendAllBetsUsingBatchs(signalReceiver)
		if err != nil {
			return err
		}

		return client.notifyNoMoreBets()
	})
	if err != nil {
		return err
	}

	return client.askForWinners(signalReceiver)
}
