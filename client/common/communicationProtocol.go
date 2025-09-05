package common

import (
	"fmt"
	"strings"
)

const (
	// MESSAGE_TYPE_LENGTH es la longitud fija (en bytes) del prefijo de tipo de mensaje.
	MESSAGE_TYPE_LENGTH = 3

	// --- Tipos de Mensajes ---
	ACK_MSG_TYPE             = "ACK"
	BET_MSG_TYPE             = "BET"
	NO_MORE_BETS_MSG_TYPE    = "NMB"
	ASK_FOR_WINNERS_MSG_TYPE = "ASK"
	WINNERS_MSG_TYPE         = "WIN"

	// --- Delimitadores y Separadores del Protocolo ---
	START_MSG_DELIMITER = "["
	END_MSG_DELIMITER   = "]"

	START_BET_DELIMITER  = "{"
	END_BET_DELIMITER    = "}"
	BET_BATCH_SEPARATOR  = ";"
	BET_FIELDS_SEPARATOR = ","

	WINNERS_SEPARATOR = ","
)

// ============================= ENCODE ============================== //

// encodeMessage es el constructor base para todos los mensajes del protocolo.
// Envuelve un payload (contenido) con su tipo y los delimitadores estándar.
// Formato de salida: TIPO[payload]
func encodeMessage(messageType string, encodedPayload string) string {
	encodedMessage := messageType
	encodedMessage += START_MSG_DELIMITER
	encodedMessage += encodedPayload
	encodedMessage += END_MSG_DELIMITER
	return encodedMessage
}

// encodeField formatea un único par clave-valor en el formato string personalizado del protocolo.
// Formato de salida: "clave":"valor"
func encodeField(fieldName string, fieldValue string) string {
	return fmt.Sprintf(`"%s":"%s"`, fieldName, fieldValue)
}

// EncodeBet serializa una única estructura Bet al formato de payload de apuesta.
// Formato de salida: {"clave1":"valor1","clave2":"valor2",...}
func EncodeBet(bet *Bet) string {
	encodedBet := START_BET_DELIMITER
	encodedBet += encodeField("agency", bet.Agency) + BET_FIELDS_SEPARATOR
	encodedBet += encodeField("first_name", bet.FirstName) + BET_FIELDS_SEPARATOR
	encodedBet += encodeField("last_name", bet.LastName) + BET_FIELDS_SEPARATOR
	encodedBet += encodeField("document", bet.Document) + BET_FIELDS_SEPARATOR
	encodedBet += encodeField("birthdate", bet.Birthdate) + BET_FIELDS_SEPARATOR
	encodedBet += encodeField("number", bet.Number)
	encodedBet += END_BET_DELIMITER
	return encodedBet
}

// EncodeBetBatchMessage serializa un slice de estructuras Bet en un único mensaje de tipo BET.
// Las apuestas individuales dentro del payload se separan por BET_BATCH_SEPARATOR (punto y coma).
// Formato de salida: BET[{"bet1"};{"bet2"};...]
func EncodeBetBatchMessage(betBatch []*Bet) string {
	encodedPayload := ""
	for i, bet := range betBatch {
		encodedPayload += EncodeBet(bet)
		if i < len(betBatch)-1 {
			encodedPayload += BET_BATCH_SEPARATOR
		}
	}
	return encodeMessage(BET_MSG_TYPE, encodedPayload)
}

// EncodeAckMessage crea un mensaje de confirmación (ACK) estándar con el payload provisto.
// Ejemplos de salida: ACK[1] o ACK[NMB]
func EncodeAckMessage(message string) string {
	return encodeMessage(ACK_MSG_TYPE, message)
}

// EncodeNoMoreBetsMessage crea el mensaje de notificación "No More Bets" (NMB).
// El payload identifica a la agencia que finaliza su envío.
func EncodeNoMoreBetsMessage(agency string) string {
	encodedPayload := encodeField("agency", agency)
	return encodeMessage(NO_MORE_BETS_MSG_TYPE, encodedPayload)
}

// EncodeAskForWinnersMessage crea el mensaje de consulta "Ask For Winners" (ASK).
// El payload identifica a la agencia que realiza la consulta.
func EncodeAskForWinnersMessage(agency string) string {
	encodedPayload := encodeField("agency", agency)
	return encodeMessage(ASK_FOR_WINNERS_MSG_TYPE, encodedPayload)
}

// ============================= DECODE ============================== //

// DecodeMessageType extrae el prefijo de tipo de mensaje (los primeros 3 bytes) de un string de mensaje crudo.
func DecodeMessageType(message string) (string, error) {
	if len(message) < MESSAGE_TYPE_LENGTH {
		return "", fmt.Errorf("message too short to contain message type")
	}
	return message[0:MESSAGE_TYPE_LENGTH], nil
}

// assertMessageFormat valida que un mensaje tenga el tipo esperado y los delimitadores correctos.
// Devuelve un error si el formato no coincide.
func assertMessageFormat(message string, expectedMessageType string) error {
	receivedMessageType, err := DecodeMessageType(message)
	if err != nil {
		return err
	}

	if receivedMessageType != expectedMessageType {
		return fmt.Errorf("unexpected message type: expected %s but received %s", expectedMessageType, receivedMessageType)
	}

	// Verifica que el mensaje comience con "TIPO[" y termine con "]"
	if !strings.HasPrefix(message, expectedMessageType+START_MSG_DELIMITER) || !strings.HasSuffix(message, END_MSG_DELIMITER) {
		return fmt.Errorf("unexpected message format")
	}
	return nil
}

// getMessagePayload extrae el contenido (payload) crudo de un mensaje, quitando el prefijo de tipo y los delimitadores.
// Ejemplo de entrada: TIPO[contenido_del_payload] -> Salida: contenido_del_payload
func getMessagePayload(message string) string {
	payload := message[MESSAGE_TYPE_LENGTH:]

	payload = payload[len(START_MSG_DELIMITER) : len(payload)-len(END_MSG_DELIMITER)]

	return payload
}

// DecodeWinnersMessage parsea un mensaje de tipo WIN, valida su formato, y extrae la lista de DNIs ganadores.
// Maneja correctamente un payload vacío (si no hay ganadores) y limpia las comillas de cada DNI.
// Ejemplo de entrada: WIN["12135000","87654321"]
func DecodeWinnersMessage(message string) ([]string, error) {
	err := assertMessageFormat(message, WINNERS_MSG_TYPE)
	if err != nil {
		return nil, err
	}

	payload := getMessagePayload(message)
	if payload == "" {
		return []string{}, nil
	}

	winners := strings.Split(payload, WINNERS_SEPARATOR)
	for i := range winners {
		winners[i] = strings.Trim(winners[i], `"`)
	}

	return winners, nil
}