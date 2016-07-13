package main
import (
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"errors"
	"encoding/json"
	"strconv"
	"crypto/x509"
	"strings"
	"time"
	"bytes"
 	"encoding/gob"
)
type Stock struct{
	Symbol string
	Quantity int
}
type Option struct{
	Symbol string
	Quantity int
	OptionType string
	StockRate float64
	SettlementDate time.Time	
	OptionPrice float64
	EntityID string
	TradeID string
}
type Entity struct{
	EntityID string				// enrollmentID
	EntityName string
	EntityType string
	Portfolio []Stock
	Options []Option
	TradeHistory []string		// list of tradeIDs
}
type Trade struct				
{
	TradeID string				// rfq transaction id
	Symbol string
	Quantity int
	TradeType string			// buy/sell
	TransactionHistory []string // transactions belonging to this trade
	Status string				// "Quote requested" or "Responded" or "Trade executed" or "Trade settled" or "Trade timed out"
}
type Transaction struct{		// ledger transactions
	TransactionID string		// different for every transaction
	TradeID string				// same for all transactions corresponding to a single trade
	TransactionType string		// type of transaction rfq or resp or tradeExec or tradeSet
	OptionType string    		// buy/sell
	ClientID string				// entityId of client
	BankID string				// entityId of bank1 or bank2
	StockSymbol string				
	Quantity int
	OptionPrice float64
	StockRate float64	
	SettlementDate time.Time	
	Status string
}

const entity1 = "user_type1_708e3151c7"
const entity2 = "user_type1_5992b632c1"
const entity3 = "user_type1_6e041a6873"
//const entity4 = "user_type1_708e3151c7"

type SimpleChaincode struct {
}
func main() {
    err := shim.Start(new(SimpleChaincode))
    if err != nil {
        fmt.Printf("Error starting chaincode: %s", err)
    }
}
func (t *SimpleChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	// initialize entities	
	client:= Entity{		
		EntityID: entity1,	  
		EntityName:	"Client A",
		EntityType: "Client",
		Portfolio: []Stock{{Symbol:"GOOGL",Quantity:10},{Symbol:"AAPL",Quantity:20}},
		Options: []Option{{Symbol:"AMZN",Quantity:10,SettlementDate: time.Date(2016, 07, 01, 0, 0, 0, 0, time.UTC)}},
	}
	b, err := json.Marshal(client)
	if err == nil {
        err = stub.PutState(client.EntityID,b)
    } else {
		return nil, err
	}
	bank1:= Entity{
		EntityID: entity2,
		EntityName:	"Bank A",
		EntityType: "Bank",
		Portfolio: []Stock{{Symbol:"MSFT",Quantity:200},{Symbol:"AAPL",Quantity:250},{Symbol:"AMZN",Quantity:400}},
	}
	b, err = json.Marshal(bank1)
	if err == nil {
        err = stub.PutState(bank1.EntityID,b)
    } else {
		return nil, err
	}
	bank2:= Entity{
		EntityID: entity3,
		EntityName:	"Bank B",
		EntityType: "Bank",
		Portfolio: []Stock{{Symbol:"GOOGL",Quantity:150},{Symbol:"AAPL",Quantity:100}},
	}
	b, err = json.Marshal(bank2)
	if err == nil {
		err = stub.PutState(bank2.EntityID,b)
    } else {
		return nil, err
	}
	
	byteVal, err := stub.GetState("currentTransactionNum")
	if len(byteVal) == 0 {
		err = stub.PutState("currentTransactionNum", []byte("1000"))
	}
	ctidByte,err := stub.GetState("currentTransactionNum")
	if(err != nil){
		return nil, errors.New("Error while getting currentTransactionNum from ledger")
	}
	
	byteVal, err = stub.GetState("currentTradeNum")
	if len(byteVal) == 0 {
		err = stub.PutState("currentTradeNum", []byte("1000"))
	}
	ctidByte,err = stub.GetState("currentTradeNum")
	if(err != nil){
		return nil, errors.New("Error while getting currentTradeNum from ledger")
	}
    return ctidByte, nil
}
func (t *SimpleChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
    // Handle different functions
    if function == "init" {
        return t.Init(stub, "init", args)
    } else if function == "requestForQuote" {
        return t.requestForQuote(stub, args)
    } else if function == "respondToQuote" {
        return t.respondToQuote(stub, args)
    } else if function == "tradeExec" {
        return t.tradeExec(stub, args)
    } else if function == "tradeSet" {
        return t.tradeSet(stub, args)
    } else if function == "trial" {
        return t.trial(stub, args)
    } 
    fmt.Println("invoke did not find func: " + function)
    return nil, errors.New("Received unknown function invocation")
}
func (t *SimpleChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
    // Handle different functions
    if function == "readEntity" {
        return t.readEntity(stub, args)
    }	else if function =="readTransaction" {
		return t.readTransaction(stub,args)
	}	else if function =="getUserID" {
		return t.getUserID(stub,args)
	}	else if function =="getcurrentTransactionNum" {
		return t.getcurrentTransactionNum(stub,args)
	}	else if function == "getValue" {
        return t.getValue(stub, args)
	}	else if function == "readTradeIDsOfUser" {
        return t.readTradeIDsOfUser(stub, args)
    }	else if function == "readTrades" {
        return t.readTrades(stub, args)
    }	else if function == "readQuoteRequests" {
        return t.readQuoteRequests(stub, args)
    }	
	fmt.Println("query did not find func: " + function)
    return nil, errors.New("Received unknown function query")
}
func (t *SimpleChaincode) readEntity(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    var name, jsonResp string
    var err error
	var valAsbytes []byte
    if len(args) != 1 {
        return nil, errors.New("Incorrect number of arguments. Expecting name of the entity")
    }
    name = args[0]
	if name == "client" {
		valAsbytes, err = stub.GetState(entity1)
	} else if name == "bank1" {
		valAsbytes, err = stub.GetState(entity2)
	} else if name == "bank2" {
		valAsbytes, err = stub.GetState(entity3)
	}
    if err != nil {
        jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
        return nil, errors.New(jsonResp)
    }
    return valAsbytes, nil
}
func (t *SimpleChaincode) readTransaction(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    var tid, jsonResp string
    var err error
    if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting transaction ID")
    }
    tid = args[0]
    valAsbytes, err := stub.GetState(tid)
    if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + tid + "\"}"
		return nil, errors.New(jsonResp)
    }
    return valAsbytes, nil
}
// used by client to request for quotes for a particular stock, adds rfq transaction to ledger
/*			arg 0	:	OptionType
			arg 1	:	StockSymbol
			arg 2	:	Quantity
*/
func (t *SimpleChaincode) requestForQuote(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args)== 3{
		// get current Transaction number
		ctidByte, err := stub.GetState("currentTransactionNum")
		if(err != nil){
			return nil, errors.New("Error while getting currentTransactionNum from ledger")
		}
		tid,err := strconv.Atoi(string(ctidByte))
		if(err != nil){
			return nil, errors.New("Error while converting ctidByte to integer")
		}
		// get current Trade number
		ctidByte, err = stub.GetState("currentTradeNum")
		if(err != nil){
			return nil, errors.New("Error while getting currentTradeNum from ledger")
		}
		tradeID,err := strconv.Atoi(string(ctidByte))
		if(err != nil){
			return nil, errors.New("Error while converting ctidByte to integer")
		}
			
		q,err := strconv.Atoi(args[2])
		if(err != nil){
			return nil, errors.New("Error while converting args[2] to integer")
		}
		bytes, err := stub.GetCallerCertificate();
		if(err != nil){
			return nil, errors.New("Error while getting caller certificate")
		}
		// get client enrollmentID
		x509Cert, err := x509.ParseCertificate(bytes);
		if(err != nil){
			return nil, errors.New("Error while parsing caller certificate")
		}
		
		tid = tid + 1
		tradeID = tradeID + 1
		
		//Transaction
		t := Transaction{
		TransactionID: "trans"+strconv.Itoa(tid),
		TradeID: "trade"+strconv.Itoa(tradeID),			// create new TradeID
		TransactionType: "RFQ",
		OptionType: args[0],   						// based on input 
		ClientID:	x509Cert.Subject.CommonName,	// enrollmentID
		BankID: "",
		StockSymbol: args[1],						// based on input
		Quantity:	q,								// based on input
		OptionPrice: 0,
		StockRate: 0,
		//SettlementDate: "",
		}
		//Trade
		tr := Trade{
		TradeID: t.TradeID,
		Symbol: t.StockSymbol,
		Quantity: t.Quantity,
		TradeType: t.OptionType,
		}
		
		// convert to Transaction to JSON
		b, err := json.Marshal(t)
		// write to ledger
		if err == nil {
			err = stub.PutState(t.TransactionID,b)
			if(err != nil){
				return nil, errors.New("Error while writing Transaction to ledger")
			}
		} else {
			return nil, errors.New("Json Marshalling error")
		}
		
		// convert to Trade JSON
		b, err = json.Marshal(tr)
		// write to ledger
		if err == nil {
			err = stub.PutState(tr.TradeID,b)
			if(err != nil){
				return nil, errors.New("Error while writing Trade to ledger")
			}
		} else {
			return nil, errors.New("Json Marshalling error")
		}
		
		// update currentTransactionNum
		err = stub.PutState("currentTransactionNum", []byte(strconv.Itoa(tid)))
		if(err != nil){
			return nil, errors.New("Error while writing currentTransactionNum to ledger")
		}
		// update currentTradeNum
		err = stub.PutState("currentTradeNum", []byte(strconv.Itoa(tradeID)))
		if(err != nil){
			return nil, errors.New("Error while writing currentTransactionNum to ledger")
		}
		
		// add Trade ID to entity's trade history
		err = updateTradeHistory(stub, t.ClientID, t.TradeID)
		if err != nil {
			return nil, errors.New("Error while updating trade history")
		}	
		
		// update trade transaction history and status
		err = updateTradeState(stub, t.TradeID, t.TransactionID,"Quote requested")
		if err != nil {
			return nil, errors.New("Error while updating trade state")
		}	
		
		return []byte(t.TransactionID), nil
	}
	return nil, errors.New("Incorrect number of arguments")
}
/*			arg 0	:	TradeID
			arg 1	:	RequestID(QuoteID)
			arg 2	:	OptionPrice
			arg 3	:	StockRate
			arg 4	:	SettlementDate Year
			arg 5	:	SettlementDate Month
			arg 6	:	SettlementDate Day
*/
func (t *SimpleChaincode) respondToQuote(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args)== 7 {
		tradeID := args[0]
		quoteID := args[1]
		
		// get bank's enrollment id
		bytes, err := stub.GetCallerCertificate();
		if(err != nil){
			return nil, errors.New("Error while getting caller certificate")
		}
		x509Cert, err := x509.ParseCertificate(bytes);
		if(err != nil){
			return nil, errors.New("Error while parsing caller certificate")
		}		
		// get information from requestForQuote transaction
		rfqbyte,err := stub.GetState(quoteID)												
		if(err != nil){
			return nil, errors.New("Error while rfq transaction from ledger")
		}
		var rfq Transaction
		err = json.Unmarshal(rfqbyte, &rfq)
		if(err != nil){
			return nil, errors.New("Error while unmarshalling rfq data")
		}
		
		if rfq.TradeID != tradeID {
			return nil, errors.New("Mismatch in tradeIDs")	
		}		
		
		// check if bank has required stock quantity 
		bankbyte,err := stub.GetState(x509Cert.Subject.CommonName)																											
		if(err != nil){
			return nil, errors.New("Error while getting bank info from ledger")
		}
		var bank Entity
		err = json.Unmarshal(bankbyte, &bank)
		if(err != nil){
			return nil, errors.New("Error while unmarshalling bank data")
		}
		stockAvailable := false
		for i := 0; i< len(bank.Portfolio); i++ {
			if bank.Portfolio[i].Symbol == rfq.StockSymbol {
				if bank.Portfolio[i].Quantity >= rfq.Quantity {
					stockAvailable = true
					break
				}
			}
		}
		if stockAvailable == false {
			return nil, errors.New("Cannot respond to quote due to insufficient stock quantity")
		}
		ctidByte, err := stub.GetState("currentTransactionNum")
		if(err != nil){
			return nil, errors.New("Error while getting currentTransactionNum from ledger")
		}
		tid,err := strconv.Atoi(string(ctidByte))
		if(err != nil){
			return nil, errors.New("Error while converting ctidByte to integer")
		}	
		// get required data from input
		price, err := strconv.ParseFloat(args[2], 64)
		if(err != nil){
			return nil, errors.New("Error while converting args[1] to float")
		}
		rate, err := strconv.ParseFloat(args[3], 64)
		if(err != nil){
			return nil, errors.New("Error while converting args[2] to float")
		}
		year, err := strconv.Atoi(args[4])
		if(err != nil){
			return nil, errors.New("Incorrect settlement year data")
		}
		var m int
		m, err = strconv.Atoi(args[5])
		var month time.Month = time.Month(m)
		if(err != nil){
			return nil, errors.New("Incorrect settlement month data")
		}
		day, err := strconv.Atoi(args[6])
		if(err != nil){
			return nil, errors.New("Incorrect settlement day data")
		}
		settlementDate := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		// check if settlement date is greater than current date
		if settlementDate.Before(time.Now()) {
			return nil, errors.New("Cannot respond to quote due to incorrect settlement date")
		}

		tid = tid + 1
		t := Transaction {
		TransactionID: "trans"+strconv.Itoa(tid),
		TradeID: tradeID,																// based on input
		TransactionType: "RESP",
		OptionType: rfq.OptionType,														// get from rfq
		ClientID:	rfq.ClientID,														// get from rfq
		BankID: x509Cert.Subject.CommonName,											// enrollmentID
		StockSymbol: rfq.StockSymbol,													// get from rfq
		Quantity:	rfq.Quantity,														// get from rfq
		OptionPrice: price,																// based on input
		StockRate: rate,																// based on input
		SettlementDate: time.Date(year, month, day, 0, 0, 0, 0, time.UTC),				// based on input
		}

		// convert to JSON
		b, err := json.Marshal(t)
		
		// write to ledger
		if err == nil {
			err = stub.PutState(t.TransactionID,b)
			if(err != nil){
				return nil, errors.New("Error while writing Response transaction to ledger")
			}
		} else {
			return nil, errors.New("Json Marshalling error")
		}
		
		err = stub.PutState("currentTransactionNum", []byte(strconv.Itoa(tid)))
		if err != nil {
			return nil, errors.New("Error while writing currentTransactionNum to ledger")
		}
		
		// updating trade transaction history ans status
		err = updateTradeState(stub, t.TradeID, t.TransactionID,"Responded")
		if err != nil {
			return nil, errors.New("Error while updating trade state")
		}
		
		/*
		err = updateTransactionHistory(stub, t.BankID, t.TransactionID)
		if err != nil {
			return nil, errors.New("Error while updating bank's transaction history")
		}
		
		// update trade status
		err = stub.PutState(tradeId, []byte("Responded"))
		if err != nil {
			return nil, errors.New("Error while updating trade status")
		}
		*/
		return nil, nil
	}
	return nil, errors.New("Incorrect number of arguments")
}
/*			arg 0	:	TradeID
			arg 1	:	Selected quote's TransactionID
*/
//---------------------------------------------------------- consensus
func (t *SimpleChaincode) tradeExec(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args)== 2 {
		ctidByte, err := stub.GetState("currentTransactionNum")
		if err != nil {
			return nil, errors.New("Error while getting currentTransactionNum from ledger")
		}
		
		tid,err := strconv.Atoi(string(ctidByte))
		if err != nil {
			return nil, errors.New("Error while converting ctidByte to integer")
		}		
		
		tradeID := args[0]
		quoteId := args[1]
		
		// get client's enrollment id
		bytes, err := stub.GetCallerCertificate();
		if(err != nil){
			return nil, errors.New("Error while getting caller certificate")
		}
		x509Cert, err := x509.ParseCertificate(bytes);
		if(err != nil){
			return nil, errors.New("Error while parsing caller certificate")
		}		
		
		// get information from selected quote
		quotebyte,err := stub.GetState(quoteId)
		if(err != nil){
			return nil, errors.New("Error while getting quote transaction from ledger")
		}
		var quote Transaction
		err = json.Unmarshal(quotebyte, &quote)		
		if(err != nil){
			return nil, errors.New("Error while unmarshalling quote data")
		}
		
		if quote.TradeID != tradeID {
			return nil, errors.New("Mismatch in tradeIDs")	
		}
		
		// check if settlement Date is greater than current date
		if quote.SettlementDate.Before(time.Now()) {
			return nil, errors.New("Cannot execute trade due to invalid settlement date")
		}
		
		tid = tid + 1
		
		t := Transaction{
		TransactionID: "trans"+strconv.Itoa(tid),
		TradeID: tradeID,							// based on input
		TransactionType: "EXEC",
		OptionType: quote.OptionType,				// get from quote transaction
		ClientID: x509Cert.Subject.CommonName,		// get from quote transaction
		BankID: quote.BankID,						// get from quote transaction
		StockSymbol: quote.StockSymbol,				// get from quote transaction
		Quantity:	quote.Quantity,					// get from quote transaction
		OptionPrice: quote.OptionPrice,				// get from quote transaction
		StockRate: quote.StockRate,					// get from quote transaction
		SettlementDate: quote.SettlementDate,		// get from quote transaction
		}

		// convert to JSON
		b, err := json.Marshal(t)
		
		// write to ledger
		if err == nil {
			err = stub.PutState(t.TransactionID,b)
			if(err != nil){
				return nil, errors.New("Error while writing Response transaction to ledger")
			}
		} else {
			return nil, errors.New("Json Marshalling error")
		}
		
		err = stub.PutState("currentTransactionNum", []byte(strconv.Itoa(tid)))
		if(err != nil){
			return nil, errors.New("Error while writing currentTransactionNum to ledger")
		}
		
		// update client entity's options
		clientbyte,err := stub.GetState(t.ClientID)																										
		if(err != nil){
			return nil, errors.New("Error while getting client info from ledger")
		}
		var client Entity
		err = json.Unmarshal(clientbyte, &client)		
		if(err != nil){
			return nil, errors.New("Error while unmarshalling client data")
		}
		
		// optiontype
		// add option to clients data
		newOption := Option{Symbol: t.StockSymbol,Quantity: t.Quantity,OptionType: t.OptionType ,StockRate: t.StockRate ,SettlementDate: t.SettlementDate,OptionPrice: t.OptionPrice, EntityID: t.BankID, TradeID:t.TradeID}
		client.Options = append(client.Options,newOption)
		
		b, err = json.Marshal(client)
		if err == nil {
			err = stub.PutState(client.EntityID,b)
		} else {
			return nil, err
		}		
		
		// add option to bank data (switch optiontype)
		bankOptionType := "buy"
		if t.OptionType == "buy" {
			bankOptionType = "sell"
		}
		
		bankbyte,err := stub.GetState(t.BankID)																										
		if(err != nil){
			return nil, errors.New("Error while getting bank info from ledger")
		}
		var bank Entity
		err = json.Unmarshal(bankbyte, &bank)		
		if(err != nil){
			return nil, errors.New("Error while unmarshalling bank data")
		}
		newOption = Option{Symbol: t.StockSymbol,Quantity: t.Quantity,OptionType: bankOptionType ,StockRate: t.StockRate ,SettlementDate: t.SettlementDate,OptionPrice: t.OptionPrice, EntityID: t.BankID, TradeID:t.TradeID}
		bank.Options = append(bank.Options,newOption)
		
		b, err = json.Marshal(bank)
		if err == nil {
			err = stub.PutState(bank.EntityID,b)
		} else {
			return nil, err
		}
		
		// updating trade transaction history  and status
		err = updateTradeState(stub, t.TradeID, t.TransactionID,"Trade executed")
		if err != nil {
			return nil, errors.New("Error while updating trade state")
		}
		/*
		err = updateTransactionHistory(stub, t.BankID, t.TransactionID)
		if err != nil {
			return nil, errors.New("Error while updating bank's transaction history")
		}
		
		// update trade status
		err = stub.PutState(tradeId, []byte("Trade executed"))
		if(err != nil){
			return nil, errors.New("Error while updating trade status")
		}
		*/
		return nil, nil
	}
	return nil, errors.New("Incorrect number of arguments")
}
/*			arg 0	:	TradeID
			arg 1	:	Yes/ No
*/

// use only tradeID
// check trade type buy/ sell and settle accordingly

func (t *SimpleChaincode) tradeSet(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args)== 2 {
		tradeID := args[0]
		//tExecId := args[1]
		// get client's enrollment id
		bytes, err := stub.GetCallerCertificate();
		if(err != nil){
			return nil, errors.New("Error while getting caller certificate")
		}
		x509Cert, err := x509.ParseCertificate(bytes);
		if(err != nil){
			return nil, errors.New("Error while parsing caller certificate")
		}
		clientID := x509Cert.Subject.CommonName
		
		// update client entity's options
		clientbyte,err := stub.GetState(clientID)																												
		if(err != nil){
			return nil, errors.New("Error while getting client info from ledger")
		}
		var client Entity
		err = json.Unmarshal(clientbyte, &client)		
		if(err != nil){
			return nil, errors.New("Error while unmarshalling client data")
		}
		// remove option from clients data, check tradeID
		copyFlag := false
		for i := 0; i< len(client.Options); i++ {
			if client.Options[i].TradeID == tradeID {
				copyFlag = true
				continue
			}
			if copyFlag == true {
				client.Options[i-1]=client.Options[i]
			}
		}
		client.Options = client.Options[:len(client.Options)-1]

		// check if trade has to be settled
		if strings.ToLower(args[1]) == "yes" {

			ctidByte, err := stub.GetState("currentTransactionNum")
			if(err != nil){
				return nil, errors.New("Error while getting currentTransactionNum from ledger")
			}		

			tid,err := strconv.Atoi(string(ctidByte))
			if(err != nil){
				return nil, errors.New("Error while converting ctidByte to integer")
			}		
			// get transactionID from tradeID
			
			tradebyte,err := stub.GetState(tradeID)																										
			if err != nil {
				return nil,errors.New("Error while getting trade info from ledger")
			}
			var trade Trade
			err = json.Unmarshal(tradebyte, &trade)		
			if err != nil {
				return nil,errors.New("Error while unmarshalling trade data")
			}
			tExecId := trade.TransactionHistory[len(trade.TransactionHistory)-1]
			
			// get information from trade exec transaction
			tbyte,err := stub.GetState(tExecId)												
			if(err != nil){
				return nil, errors.New("Error while getting tradeExec transaction from ledger")
			}
			
			var tExec Transaction
			err = json.Unmarshal(tbyte, &tExec)		
			if(err != nil){
				return nil, errors.New("Error while unmarshalling tradeExec data")
			}

			if tExec.TradeID != tradeID {
			return nil, errors.New("Mismatch in tradeIDs")	
			}
			
			// check settlement date to see if option is still valid
			if time.Now().Before(tExec.SettlementDate) {
				tid = tid + 1
				t := Transaction{
				TransactionID: "trans"+strconv.Itoa(tid),
				TradeID: tradeID,							// based on input
				TransactionType: "SET",
				OptionType: tExec.OptionType,				// get from tradeExec transaction
				ClientID: x509Cert.Subject.CommonName,		// get from tradeExec transaction
				BankID: tExec.BankID,						// get from tradeExec transaction
				StockSymbol: tExec.StockSymbol,				// get from tradeExec transaction
				Quantity:	tExec.Quantity,					// get from tradeExec transaction
				OptionPrice: tExec.OptionPrice,				// get from tradeExec transaction
				StockRate: tExec.StockRate,					// get from tradeExec transaction
				SettlementDate: tExec.SettlementDate,		// get from tradeExec transaction
				}
				// convert to JSON
				b, err := json.Marshal(t)
				// write to ledger
				if err == nil {
					err = stub.PutState(t.TransactionID,b)
					if(err != nil){
						return nil, errors.New("Error while writing Response transaction to ledger")
					}
				} else {
					return nil, errors.New("Json Marshalling error")
				}
				err = stub.PutState("currentTransactionNum", []byte(strconv.Itoa(tid)))
				if(err != nil){
					return nil, errors.New("Error while writing currentTransactionNum to ledger")
				}
				
				// add stock to clients portfolio, check if stock already exists if yes increase quantity else create new stock entry 		
				stockExistFlag := false
				for i := 0; i< len(client.Portfolio); i++ {
					if client.Portfolio[i].Symbol == t.StockSymbol {
						stockExistFlag = true
						if strings.ToLower(t.OptionType) == "buy" {
							client.Portfolio[i].Quantity = client.Portfolio[i].Quantity + t.Quantity
						} else {	// sell option type
							if client.Portfolio[i].Quantity >= t.Quantity {
								client.Portfolio[i].Quantity = client.Portfolio[i].Quantity - t.Quantity
							} else {
								return nil, errors.New("Error insufficient stock quantity")
							}
						}
						break
					}
				}
				
				if (strings.ToLower(t.OptionType) == "sell") && (stockExistFlag == false) {
					return nil, errors.New("Error insufficient stock quantity")
				}
				
				// create new stock entry
				if stockExistFlag == false {
					newStock := Stock{Symbol: t.StockSymbol,Quantity: t.Quantity}
					client.Portfolio = append(client.Portfolio,newStock)
				}
				// update banks stock data
				bankbyte,err := stub.GetState(t.BankID)																											
				if(err != nil){
					return nil, errors.New("Error while getting bank info from ledger")
				}
				var bank Entity
				err = json.Unmarshal(bankbyte, &bank)		
				if(err != nil){
					return nil, errors.New("Error while unmarshalling bank data")
				}
				
				stockExistFlag = false
				for i := 0; i< len(bank.Portfolio); i++ {
					if bank.Portfolio[i].Symbol == t.StockSymbol {
						stockExistFlag = true
						if strings.ToLower(t.OptionType) == "buy" {		
								if bank.Portfolio[i].Quantity >= t.Quantity {
									bank.Portfolio[i].Quantity = bank.Portfolio[i].Quantity - t.Quantity
								} else {
									return nil, errors.New("Error insufficient stock quantity")
								}
						} else {
							bank.Portfolio[i].Quantity = bank.Portfolio[i].Quantity + t.Quantity
						}
						break
					}
				}
				// create new stock entry
				if stockExistFlag == false {
					newStock := Stock{Symbol: t.StockSymbol,Quantity: t.Quantity}
					bank.Portfolio = append(bank.Portfolio,newStock)
				}
				
				// update bank state
				b, err = json.Marshal(bank)
				if err == nil {
					err = stub.PutState(bank.EntityID,b)
				} else {
					return nil, err
				}
				// updating trade state
				err = updateTradeState(stub, t.TradeID, t.TransactionID,"Trade Settled")
				if err != nil {
					return nil, errors.New("Error while updating trade state")
				}
				
				/*
				err = updateTransactionHistory(stub, t.BankID, t.TransactionID)
				if err != nil {
					return nil, errors.New("Error while updating bank's transaction history")
				}
				// update trade status
				err = stub.PutState(tradeId, []byte("Trade Settled"))
				if(err != nil){
					return nil, errors.New("Error while updating trade status")
				}
				*/
			} else {	// trade expired
				
				// updating trade state
				err = updateTradeState(stub, tradeID,"" ,"Trade Expired")
				if err != nil {
					return nil, errors.New("Error while updating trade state")
				}
				/*
				// update trade status
				err = stub.PutState(tradeId, []byte("Trade Expired"))
				if(err != nil){
					return nil, errors.New("Error while updating trade status")
				}
				*/
			}
		} else {	// trade cancelled
			// updating trade state
			err = updateTradeState(stub, tradeID,"" ,"Trade Cancelled")
			if err != nil {
				return nil, errors.New("Error while updating trade state")
			}
			/*
			// update trade status
			err = stub.PutState(tradeId, []byte("Trade Cancelled"))
			if(err != nil){
				return nil, errors.New("Error while updating trade status")
			}
			*/
		}
		// update client state
		b, err := json.Marshal(client)
		if err == nil {
			err = stub.PutState(client.EntityID,b)
		} else {
			return nil, err
		}
		return nil, nil
	}
	return nil, errors.New("Incorrect number of arguments")
}
// get user id
func (t *SimpleChaincode) getUserID(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	bytes, err := stub.GetCallerCertificate()
	x509Cert, err := x509.ParseCertificate(bytes)
	return []byte(x509Cert.Subject.CommonName), err
}
func (t *SimpleChaincode) getcurrentTransactionNum(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	ctidByte,err := stub.GetState("currentTransactionNum")
	if err != nil {
		return nil, errors.New("Error retrieving currentTransactionNum")
	}
    return ctidByte, err
}
func (t *SimpleChaincode) getValue(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	byteVal,err := stub.GetState(args[0])
	if err != nil {
		return []byte(err.Error()), errors.New("Error retrieving key "+args[0])
	}
	if len(byteVal) == 0 {
		return []byte("Len is zero"), nil
	}
    return byteVal, nil
}
// read transactions IDs for a particular user
func (t *SimpleChaincode) readTradeIDsOfUser(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args)== 1 {
		// read entity state
		entitybyte,err := stub.GetState(args[0])																									
		if err != nil {
			return nil, errors.New("Error while getting entity info from ledger")
		}
		var entity Entity
		err = json.Unmarshal(entitybyte, &entity)		
		if(err != nil){
			return nil, errors.New("Error while unmarshalling entity data")
		}

		b, err := json.Marshal(entity.TradeHistory)
		if err != nil {
			return nil, errors.New("Error while marshalling trade history")
		}
		return b, nil
	}
	return nil, errors.New("Incorrect number of arguments")
}
func updateTradeHistory(stub *shim.ChaincodeStub, entityID string, tradeID string) (error) {
	// read entity state
	entitybyte,err := stub.GetState(entityID)																										
	if err != nil {
		return errors.New("Error while getting entity info from ledger")
	}
	var entity Entity
	err = json.Unmarshal(entitybyte, &entity)		
	if err != nil {
		return errors.New("Error while unmarshalling entity data")
	}
	// add tradeID to history
	entity.TradeHistory = append(entity.TradeHistory,tradeID)
	// write entity state to ledger
	b, err := json.Marshal(entity)
	if err == nil {
		err = stub.PutState(entity.EntityID,b)
	} else {
		return errors.New("Error while updating entity status")
	}
	return nil
}

func updateTradeState(stub *shim.ChaincodeStub, tradeID string, transactionID string, status string) (error) {
	// read trade state
	tradebyte,err := stub.GetState(tradeID)																										
	if err != nil {
		return errors.New("Error while getting trade info from ledger")
	}
	var trade Trade
	err = json.Unmarshal(tradebyte, &trade)		
	if err != nil {
		return errors.New("Error while unmarshalling trade data")
	}
	// add transactionID to history
	trade.TransactionHistory = append(trade.TransactionHistory,transactionID)
	
	// update status
	trade.Status = status
	
	// write trade state to ledger
	b, err := json.Marshal(trade)
	if err == nil {
		err = stub.PutState(trade.TradeID,b)
	} else {
		return errors.New("Error while updating trade status")
	}
	return nil
}

func (t *SimpleChaincode) trial(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	return nil, errors.New("********* TRIAL ERROR *********")
}

/* Function to read open requests
	Traverses through trade log, checks trade status returns open requests
	
*/

/* Function to read open quotes
	Traverses through trade log, checks trade status returns open requests
*/

// update trade status correctly & transaction status

//status
// check status
// error 



//readTrades() returns TradeID | Symbol | Quantity | Status | Buy/Sell and transactionIDs
/* args 0 userID
func (t *SimpleChaincode) readTradeHistory(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args)== 1 {
	}	
	return nil, errors.New("Incorrect number of arguments")
}
*/

// read trades of a client
func (t *SimpleChaincode) readTrades(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args)== 1 {
		// read entity state
		entitybyte,err := stub.GetState(args[0])																									
		if err != nil {
			return nil, errors.New("Error while getting entity info from ledger")
		}
		var entity Entity
		err = json.Unmarshal(entitybyte, &entity)		
		if(err != nil){
			return nil, errors.New("Error while unmarshalling entity data")
		}
		trades := make([]Trade,len(entity.TradeHistory))
		for i:=0; i<len(entity.TradeHistory); i++ {
			byteVal,err := stub.GetState(entity.TradeHistory[i])
			if err != nil {
				return nil, errors.New("Error while getting trades info from ledger")
			}
			err = json.Unmarshal(byteVal, &trades[i])	
			if err != nil {
				return nil, errors.New("Error while unmarshalling trades")
			}	
		}
		b, err := json.Marshal(trades)
		if err != nil {
			return nil, errors.New("Error while marshalling trades")
		}
		return b, nil
	}
	return nil, errors.New("Incorrect number of arguments")
}

func (t *SimpleChaincode) readQuoteRequests(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var quoteTransactions []string
	// get current Trade number
	ctidByte, err := stub.GetState("currentTradeNum")
	if(err != nil){
		return nil, errors.New("Error while getting currentTradeNum from ledger")
	}
	tradeNum,err := strconv.Atoi(string(ctidByte))
	if(err != nil){
		return nil, errors.New("Error while converting ctidByte to integer")
	}
	// check all trades
	for tradeNum > 1000 {
		// read trade state
		tradebyte,err := stub.GetState("trade"+strconv.Itoa(tradeNum))
		if err != nil {
			return nil, errors.New("Error while getting trade info from ledger")
		}
		var trade Trade
		err = json.Unmarshal(tradebyte, &trade)		
		if err != nil {
			return nil, errors.New("Error while unmarshalling trade data")
		}
		// check status
		if trade.Status == "Quote requested" {
			quoteTransactions = append(quoteTransactions,trade.TransactionHistory[0])
		} else if trade.Status == "Responded" { // check who has responded
			respondedFlag := false
			bytes, _ := stub.GetCallerCertificate()
			x509Cert, _ := x509.ParseCertificate(bytes)
			currentUserID := x509Cert.Subject.CommonName
			
			for i:=0; i< len(trade.TransactionHistory); i++ {
				tranbyte,err := stub.GetState(trade.TransactionHistory[i])
				if(err != nil){
					return nil, errors.New("Error while getting transaction from ledger")
				}
				var tran Transaction
				err = json.Unmarshal(tranbyte, &tran)		
				if(err != nil){
					return nil, errors.New("Error while unmarshalling tran data")
				}
				if tran.TransactionType == "RESP" {
					if tran.BankID == currentUserID {
						respondedFlag = true
						break
					}
				}
			}
			if respondedFlag == false {
				quoteTransactions = append(quoteTransactions,trade.TransactionHistory[0])
			}
		}	
		tradeNum--
	}
	buffer := &bytes.Buffer{}
	gob.NewEncoder(buffer).Encode(quoteTransactions)
 	byteSlice := buffer.Bytes()
	return []byte(byteSlice), nil
}
