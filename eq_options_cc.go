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
	TradeType string			// Call/ Put
	TransactionHistory []string // transactions belonging to this trade
	Status string				// "Quote requested" or "Responded" or "Trade executed" or "Trade exercised" or "Trade timed out"
}
type Transaction struct{		// ledger transactions
	TransactionID string		// different for every transaction
	TradeID string				// same for all transactions corresponding to a single trade
	TransactionType string		// type of transaction rfq or resp or tradeExec or tradeSet	   Request	Response Execute	Exercise
	OptionType string    		// Call/ Put
	ClientID string				// entityId of client
	BankID string				// entityId of bank1 or bank2
	StockSymbol string				
	Quantity int
	OptionPrice float64
	StockRate float64	
	SettlementDate time.Time	
	Status string
}



const entity1 = "user_type1_0"
const entity2 = "user_type1_1"
const entity3 = "user_type1_2"
const entity4 = "user_type1_3"


type SimpleChaincode struct {
}
func main() {
    err := shim.Start(new(SimpleChaincode))
    if err != nil {
        fmt.Printf("Error starting chaincode: %s", err)
    }
}
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	// initialize entities	
	client:= Entity{		
		EntityID: entity1,	  
		EntityName:	"Client A",
		EntityType: "Client",
		Portfolio: []Stock{{Symbol:"GOOGL",Quantity:10},{Symbol:"AAPL",Quantity:20}},
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
	regBody:= Entity{
		EntityID: entity4,
		EntityName:	"Regulatory Body",
		EntityType: "RegBody",
	}
	b, err = json.Marshal(regBody)
	if err == nil {
		err = stub.PutState(regBody.EntityID,b)
    } else {
		return nil, err
	}
	
	EntityList := []string{entity1,entity2, entity3, entity4}

	b, err = json.Marshal(EntityList)
	if err == nil {
		err = stub.PutState("entityList",b)
    } else {
		return nil, err
	}
	
	// initialize trade num and transaction num
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
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
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
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
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
    }	else if function == "getAllTrades" {
        return t.getAllTrades(stub, args)
    }	else if function == "getEntityList" {
        return t.getEntityList(stub, args)
    }	else if function == "getTransactionStatus" {
        return t.getTransactionStatus(stub, args)
    }
	fmt.Println("query did not find func: " + function)
    return nil, errors.New("Received unknown function query")
}
func (t *SimpleChaincode) readEntity(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
    var jsonResp string
    var err error
	var valAsbytes []byte
    if len(args) != 1 {
        return nil, errors.New("Incorrect number of arguments. Expecting entity ID")
    }
	valAsbytes, err = stub.GetState(args[0])
    if err != nil {
        jsonResp = "{\"Error\":\"Failed to get state for " + args[0] + "\"}"
        return nil, errors.New(jsonResp)
    }
    return valAsbytes, nil
}
func (t *SimpleChaincode) readTransaction(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
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
	var tran Transaction
	err = json.Unmarshal(valAsbytes, &tran)
	if(err != nil){
		return nil, errors.New("Error while unmarshalling transaction data")
	}
	
	bytes, err := stub.GetCallerCertificate();
	if(err != nil){
		return nil, errors.New("Error while getting caller certificate")
	}
	x509Cert, err := x509.ParseCertificate(bytes);
	fmt.Print(x509Cert.Subject.CommonName)
	// check entity type and accordingly allow transaction to be read
	entityByte,err := stub.GetState(args[1]) //stub.GetState(x509Cert.Subject.CommonName)
	if(err != nil){
		return nil, errors.New("Error while getting bank info from ledger")
	}
	var entity Entity
	err = json.Unmarshal(entityByte, &entity)
	if(err != nil){
		return nil, errors.New("Error while unmarshalling entity data")
	}
	
	switch entity.EntityType {
		case "RegBody":	return valAsbytes, nil
		case "Client":	if tran.ClientID == args[1] {
							return valAsbytes, nil
						}
		case "Bank":	if tran.TransactionType == "Request" {
							return valAsbytes, nil
						} else if tran.BankID == args[1] {
							return valAsbytes, nil
						}
	}
    return nil, nil
}
// used by client to request for quotes for a particular stock, adds rfq transaction to ledger
/*			arg 0	:	OptionType
			arg 1	:	StockSymbol
			arg 2	:	Quantity
*/
func (t *SimpleChaincode) requestForQuote(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args)== 4{
		// get current Transaction number
		ctidByte, err := stub.GetState("currentTransactionNum")
		if(err != nil){
			return nil, errors.New("Error while getting currentTransactionNum from ledger")
		}
		tid,err := strconv.Atoi(string(ctidByte))
		if(err != nil){
			return nil, errors.New("Error while converting ctidByte to integer")
		}
		tid = tid + 1
		transactionID := "trans"+strconv.Itoa(tid)
		
		// get current Trade number
		ctidByte, err = stub.GetState("currentTradeNum")
		if err != nil {			
			_ = updateTransactionStatus(stub, transactionID, "Error while getting currentTradeNum from ledger")
			return nil, nil
		}
		tradeID,err := strconv.Atoi(string(ctidByte))
		if(err != nil){
			_ = updateTransactionStatus(stub, transactionID, "Error while converting ctidByte to integer")
			return nil, nil
		}
			
		q,err := strconv.Atoi(args[2])
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while converting quantity to integer")
			return nil, nil			
		}
		bytes, err := stub.GetCallerCertificate();
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while getting caller certificate")
			return nil, nil
		}
		// get client enrollmentID
		x509Cert, err := x509.ParseCertificate(bytes);
		fmt.Print(x509Cert.Subject.CommonName)
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while parsing caller certificate")
			return nil, nil
		}
		tradeID = tradeID + 1
		
		//Transaction
		t := Transaction{
		TransactionID: transactionID,
		TradeID: "trade"+strconv.Itoa(tradeID),			// create new TradeID
		TransactionType: "Request",
		OptionType: args[0],   						// based on input 
		ClientID:	args[3] ,//x509Cert.Subject.CommonName,	// enrollmentID
		BankID: "",
		StockSymbol: args[1],						// based on input
		Quantity:	q,								// based on input
		OptionPrice: 0,
		StockRate: 0,
		Status: "Success",
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
			if err != nil {
				_ = updateTransactionStatus(stub, transactionID, "Error while writing Transaction to ledger")
				return nil, nil
			}
		} else {
			_ = updateTransactionStatus(stub, transactionID, "Error while marshalling trade data")
			return nil, nil
		}
		
		// convert to Trade JSON
		b, err = json.Marshal(tr)
		// write to ledger
		if err == nil {
			err = stub.PutState(tr.TradeID,b)
			if err != nil {
				_ = updateTransactionStatus(stub, transactionID, "Error while writing Trade data to ledger")
				return nil, nil
			}
		} else {
			_ = updateTransactionStatus(stub, transactionID, "Error while marshalling trade data")
			return nil, nil
		}
		
		// update currentTransactionNum
		err = stub.PutState("currentTransactionNum", []byte(strconv.Itoa(tid)))
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while updating current transaction number")
			return nil, nil
		}
		// update currentTradeNum
		err = stub.PutState("currentTradeNum", []byte(strconv.Itoa(tradeID)))
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while updating current transaction number")
			return nil, nil
		}
		
		// add Trade ID to entity's trade history
		err = updateTradeHistory(stub, t.ClientID, t.TradeID)
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while updating trade history")
			return nil, nil
		}	
		
		// update trade transaction history and status
		err = updateTradeState(stub, t.TradeID, t.TransactionID,"Quote requested")
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while updating trade state")
			return nil, nil
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
func (t *SimpleChaincode) respondToQuote(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args)== 8 {
		tradeID := args[0]
		quoteID := args[1]
		
		ctidByte, err := stub.GetState("currentTransactionNum")
		if(err != nil){
			return nil, errors.New("Error while getting currentTransactionNum from ledger")
		}
		tid,err := strconv.Atoi(string(ctidByte))
		if(err != nil){
			return nil, errors.New("Error while converting ctidByte to integer")
		}
		tid = tid + 1
		transactionID := "trans"+strconv.Itoa(tid)
		
		// get bank's enrollment id
		bytes, err := stub.GetCallerCertificate();
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while getting caller certificate")
			return nil, nil
		}
		x509Cert, err := x509.ParseCertificate(bytes);
		fmt.Print(x509Cert.Subject.CommonName)
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while parsing caller certificate")
			return nil, nil
		}		
		// get information from requestForQuote transaction
		rfqbyte,err := stub.GetState(quoteID)												
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while reading quote request transaction from ledger")
			return nil, nil
		}
		var rfq Transaction
		err = json.Unmarshal(rfqbyte, &rfq)
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while unmarshalling quote request data")
			return nil, nil
		}
		
		if rfq.TradeID != tradeID {
			_ = updateTransactionStatus(stub, transactionID, "Error due to mismatch in tradeIDs")
			return nil, nil
		}		
		
		// add trade to bank's trade history
		err = updateTradeHistory(stub, args[7], tradeID)
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while updating trade history")
			return nil, nil
		}
		
		/*
		// check if bank has required stock quantity 
		bankbyte,err := stub.GetState(x509Cert.Subject.CommonName)																											
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while getting bank info from ledger")
			return nil, nil
		}
		var bank Entity
		err = json.Unmarshal(bankbyte, &bank)
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while unmarshalling bank data")
			return nil, nil
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
			_ = updateTransactionStatus(stub, transactionID, "Error while converting ctidByte to integer")
			return nil, nil
			return nil, errors.New("ErrorCannot respond to quote due to insufficient stock quantity")
		}
		*/
			
		// get required data from input
		price, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error invalid option price")
			return nil, nil
		}
		rate, err := strconv.ParseFloat(args[3], 64)
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error invalid stock rate")
			return nil, nil
		}
		year, err := strconv.Atoi(args[4])
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error invalid Expiration date")
			return nil, nil
		}
		var m int
		m, err = strconv.Atoi(args[5])
		var month time.Month = time.Month(m)
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error invalid Expiration date")
			return nil, nil
		}
		day, err := strconv.Atoi(args[6])
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error invalid Expiration date")
			return nil, nil
		}
		settlementDate := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		
		// check if settlement date is greater than current date
		if settlementDate.Before(time.Now()) {
			_ = updateTransactionStatus(stub, transactionID, "Error cannot respond to quote due to incorrect Expiration date")
		}

		
		t := Transaction {
		TransactionID: transactionID,
		TradeID: tradeID,																// based on input
		TransactionType: "Response",
		OptionType: rfq.OptionType,														// get from rfq
		ClientID:	rfq.ClientID,														// get from rfq
		BankID: args[7] ,//x509Cert.Subject.CommonName,											// enrollmentID
		StockSymbol: rfq.StockSymbol,													// get from rfq
		Quantity:	rfq.Quantity,														// get from rfq
		OptionPrice: price,																// based on input
		StockRate: rate,																// based on input
		SettlementDate: time.Date(year, month, day, 0, 0, 0, 0, time.UTC),				// based on input
		Status: "Success",
		}

		// convert to JSON
		b, err := json.Marshal(t)
		
		// write to ledger
		if err == nil {
			err = stub.PutState(t.TransactionID,b)
			if err != nil {
				_ = updateTransactionStatus(stub, transactionID, "Error while writing Response transaction to ledger")
				return nil, nil
			}
		} else {
			_ = updateTransactionStatus(stub, transactionID, "Error while marshalling transaction data")
			return nil, nil
		}
		
		err = stub.PutState("currentTransactionNum", []byte(strconv.Itoa(tid)))
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while writing current Transaction Number to ledger")
			return nil, nil
		}
		
		// updating trade transaction history ans status
		err = updateTradeState(stub, t.TradeID, t.TransactionID,"Responded")
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while updating trade state")
			return nil, nil
		}
		return nil, nil
	}
	return nil, errors.New("Incorrect number of arguments")
}
/*			arg 0	:	TradeID
			arg 1	:	Selected quote's TransactionID
*/
//---------------------------------------------------------- consensus
func (t *SimpleChaincode) tradeExec(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args)== 3 {
		
		ctidByte, err := stub.GetState("currentTransactionNum")
		if err != nil {
			return nil, errors.New("Error while getting current Transaction Number from ledger")
		}		
		tid,err := strconv.Atoi(string(ctidByte))
		if err != nil {
			return nil, errors.New("Error while converting ctidByte to integer")
		}
		tid = tid + 1
		transactionID := "trans"+strconv.Itoa(tid)
		
		tradeID := args[0]
		quoteId := args[1]
		
		// get client's enrollment id
		bytes, err := stub.GetCallerCertificate();
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while getting caller certificate")
			return nil, nil
		}
		x509Cert, err := x509.ParseCertificate(bytes);
		fmt.Print(x509Cert.Subject.CommonName)
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while parsing caller certificate")
			return nil, nil
		}

		// get information from selected quote
		quotebyte,err := stub.GetState(quoteId)
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while getting quote data")
			return nil, nil
		}
		var quote Transaction
		err = json.Unmarshal(quotebyte, &quote)		
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while unmarshalling quote data")
			return nil, nil
		}
		
		if quote.TradeID != tradeID {
			_ = updateTransactionStatus(stub, transactionID, "Error due to mismatch in tradeIDs")	
			return nil, nil
		}
		
		// check if settlement Date is greater than current date
		if quote.SettlementDate.Before(time.Now()) {
			_ = updateTransactionStatus(stub, transactionID, "Error cannot execute trade due to invalid Expiration date")
			return nil, nil
		}
		
		t := Transaction{
		TransactionID: transactionID,
		TradeID: tradeID,							// based on input
		TransactionType: "Execute",
		OptionType: quote.OptionType,				// get from quote transaction
		ClientID: args[2],//x509Cert.Subject.CommonName,		// get from quote transaction
		BankID: quote.BankID,						// get from quote transaction
		StockSymbol: quote.StockSymbol,				// get from quote transaction
		Quantity:	quote.Quantity,					// get from quote transaction
		OptionPrice: quote.OptionPrice,				// get from quote transaction
		StockRate: quote.StockRate,					// get from quote transaction
		SettlementDate: quote.SettlementDate,		// get from quote transaction
		Status: "Success",
		}

		// convert to JSON
		b, err := json.Marshal(t)
		
		// write to ledger
		if err == nil {
			err = stub.PutState(t.TransactionID,b)
			if(err != nil){
				_ = updateTransactionStatus(stub, transactionID, "Error while writing Response transaction to ledger")
				return nil, nil
			}
		} else {
			_ = updateTransactionStatus(stub, transactionID, "Error Json Marshalling error")
			return nil, nil
		}
		
		err = stub.PutState("currentTransactionNum", []byte(strconv.Itoa(tid)))
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while writing currentTransactionNum to ledger")
			return nil, nil
		}
		
		// update client entity's options
		clientbyte,err := stub.GetState(t.ClientID)																										
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while getting client info from ledger")
			return nil, nil
		}
		var client Entity
		err = json.Unmarshal(clientbyte, &client)		
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while unmarshalling client data")
			return nil, nil
		}
		
		newOption := Option{Symbol: t.StockSymbol,Quantity: t.Quantity,OptionType: t.OptionType ,StockRate: t.StockRate ,SettlementDate: t.SettlementDate,OptionPrice: t.OptionPrice, EntityID: t.BankID, TradeID:t.TradeID}
		client.Options = append(client.Options,newOption)
		
		b, err = json.Marshal(client)
		if err == nil {
			err = stub.PutState(client.EntityID,b)
		} else {
			_ = updateTransactionStatus(stub, transactionID, "Error while updating Client state")
			return nil, nil
		}		
		
		bankOptionType := t.OptionType
		
		bankbyte,err := stub.GetState(t.BankID)																										
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while getting bank information from ledger")
			return nil, nil
		}
		var bank Entity
		err = json.Unmarshal(bankbyte, &bank)		
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while unmarshalling bank data")
			return nil, nil
		}
		newOption = Option{Symbol: t.StockSymbol,Quantity: t.Quantity,OptionType: bankOptionType ,StockRate: t.StockRate ,SettlementDate: t.SettlementDate,OptionPrice: t.OptionPrice, EntityID: t.ClientID, TradeID:t.TradeID}
		bank.Options = append(bank.Options,newOption)
		
		b, err = json.Marshal(bank)
		if err == nil {
			err = stub.PutState(bank.EntityID,b)
		} else {
			_ = updateTransactionStatus(stub, transactionID, "Error while updating Bank state")
			return nil, nil
		}
		
		// updating trade transaction history  and status
		err = updateTradeState(stub, t.TradeID, t.TransactionID,"Trade Executed")
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while updating trade state")
			return nil, nil
		}
		return nil, nil
	}
	return nil, errors.New("Incorrect number of arguments")
}
/*			arg 0	:	TradeID
			arg 1	:	Yes/ No
*/
func (t *SimpleChaincode) tradeSet(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args)== 3 {
		tradeID := args[0]
		//tExecId := args[1]
		// get client's enrollment id
		
		ctidByte, err := stub.GetState("currentTransactionNum")
		if(err != nil){
			return nil, errors.New("Error while getting currentTransactionNum from ledger")
		}
		tid,err := strconv.Atoi(string(ctidByte))
		if(err != nil){
			return nil, errors.New("Error while converting ctidByte to integer")
		}	
		tid = tid + 1
		transactionID := "trans"+strconv.Itoa(tid)
		
		bytes, err := stub.GetCallerCertificate();
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while getting caller certificate")
			return nil, nil
		}
		x509Cert, err := x509.ParseCertificate(bytes);
		fmt.Print(x509Cert.Subject.CommonName)
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while parsing caller certificate")
			return nil, nil
		}
		clientID := args[2] //x509Cert.Subject.CommonName
		
		// update client entity's options
		clientbyte,err := stub.GetState(clientID)																												
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while getting client info from ledger")
			return nil, nil
		}
		var client Entity
		err = json.Unmarshal(clientbyte, &client)		
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while unmarshalling client data")
			return nil, nil
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
		client.Options = client.Options[:(len(client.Options)-1)]
		
		// get transactionID from tradeID
		tradebyte,err := stub.GetState(tradeID)
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while getting trade info from ledger")
			return nil, nil
		}
		var trade Trade
		err = json.Unmarshal(tradebyte, &trade)		
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while unmarshalling trade data")
			return nil, nil
		}
		tExecId := trade.TransactionHistory[len(trade.TransactionHistory)-1]
		
		// get information from trade exec transaction
		tbyte,err := stub.GetState(tExecId)												
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while getting tradeExec transaction from ledger")
			return nil, nil
		}
		
		var tExec Transaction
		err = json.Unmarshal(tbyte, &tExec)		
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while unmarshalling tradeExec data")
			return nil, nil
		}
		
		// update bank entity's options
		bankbyte,err := stub.GetState(tExec.BankID)																											
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while getting bank info from ledger")
			return nil, nil
		}
		var bank Entity
		err = json.Unmarshal(bankbyte, &bank)		
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while unmarshalling bank data")
			return nil, nil
		}
		// remove option from bank 
		copyFlag = false
		for i := 0; i< len(bank.Options); i++ {
			if bank.Options[i].TradeID == tradeID {
				copyFlag = true
				continue
			}
			if copyFlag == true {
				bank.Options[i-1]=bank.Options[i]
			}
		}
		bank.Options = bank.Options[:(len(bank.Options)-1)]
		// check if trade has to be settled
		if strings.ToLower(args[1]) == "yes" {
			if tExec.TradeID != tradeID {
				_ = updateTransactionStatus(stub, transactionID, "Error due to mismatch in tradeIDs")
				return nil, nil
			}
			
			// check settlement date to see if option is still valid
			if time.Now().Before(tExec.SettlementDate) {
				
				t := Transaction{
				TransactionID: transactionID,
				TradeID: tradeID,							// based on input
				TransactionType: "Exercise",
				OptionType: tExec.OptionType,				// get from tradeExec transaction
				ClientID: args[2] , //x509Cert.Subject.CommonName,		// get from tradeExec transaction
				BankID: tExec.BankID,						// get from tradeExec transaction
				StockSymbol: tExec.StockSymbol,				// get from tradeExec transaction
				Quantity:	tExec.Quantity,					// get from tradeExec transaction
				OptionPrice: tExec.OptionPrice,				// get from tradeExec transaction
				StockRate: tExec.StockRate,					// get from tradeExec transaction
				SettlementDate: tExec.SettlementDate,		// get from tradeExec transaction
				Status: "Success",
				}
				// convert to JSON
				b, err := json.Marshal(t)
				// write to ledger
				if err == nil {
					err = stub.PutState(t.TransactionID,b)
					if err != nil {
						_ = updateTransactionStatus(stub, transactionID, "Error while writing Response transaction to ledger")
						return nil, nil
					}
				} else {
					_ = updateTransactionStatus(stub, transactionID, "Error while marshalling transaction data")
					return nil, nil
				}
				
				// add stock to clients portfolio, check if stock already exists if yes increase quantity else create new stock entry 		
				stockExistFlag := false
				for i := 0; i< len(client.Portfolio); i++ {
					if client.Portfolio[i].Symbol == t.StockSymbol {
						stockExistFlag = true
						if strings.ToLower(t.OptionType) == "call" {
							client.Portfolio[i].Quantity = client.Portfolio[i].Quantity + t.Quantity
						} else {	// Put option type
							if client.Portfolio[i].Quantity >= t.Quantity {
								client.Portfolio[i].Quantity = client.Portfolio[i].Quantity - t.Quantity
							} else {
								_ = updateTransactionStatus(stub, transactionID, "Error insufficient stock quantity to complete the transaction")
								return nil, nil
							}
						}
						break
					}
				}
				
				if (strings.ToLower(t.OptionType) == "put") && (stockExistFlag == false) {
					_ = updateTransactionStatus(stub, transactionID, "Error insufficient stock quantity to complete the transaction")
					return nil, nil
				}
				
				// create new stock entry
				if stockExistFlag == false {
					newStock := Stock{Symbol: t.StockSymbol,Quantity: t.Quantity}
					client.Portfolio = append(client.Portfolio,newStock)
				}
				// update banks stock data
				stockExistFlag = false
				for i := 0; i< len(bank.Portfolio); i++ {
					if bank.Portfolio[i].Symbol == t.StockSymbol {
						stockExistFlag = true
						if strings.ToLower(t.OptionType) == "call" {
								if bank.Portfolio[i].Quantity >= t.Quantity {
									bank.Portfolio[i].Quantity = bank.Portfolio[i].Quantity - t.Quantity
								} else {
									_ = updateTransactionStatus(stub, transactionID, "Error insufficient stock quantity to complete the transaction")
									return nil, nil
								}
						} else {
							bank.Portfolio[i].Quantity = bank.Portfolio[i].Quantity + t.Quantity
						}
						break
					}
				}
				
				if (strings.ToLower(t.OptionType) == "call") && (stockExistFlag == false) {
					_ = updateTransactionStatus(stub, transactionID, "Error insufficient stock quantity to complete the transaction")
					return nil, nil
				}
				
				// create new stock entry
				if  (strings.ToLower(t.OptionType) == "put") && (stockExistFlag == false) {
					newStock := Stock{Symbol: t.StockSymbol,Quantity: t.Quantity}
					bank.Portfolio = append(bank.Portfolio,newStock)
				}				
				
				// updating trade state
				err = updateTradeState(stub, t.TradeID, t.TransactionID,"Trade Exercised")
				if err != nil {
					_ = updateTransactionStatus(stub, transactionID, "Error while updating trade state")
					return nil, nil
				}
				
			} else {	// trade expired
				
				
				_ = updateTransactionStatus(stub, transactionID, "")
				
				// updating trade state
				err = updateTradeState(stub, tradeID,"" ,"Trade Expired")
				if err != nil {
					_ = updateTransactionStatus(stub, transactionID, "Error while updating trade state")
					return nil, nil
				}
				
			}
		} else {	// trade cancelled
			_ = updateTransactionStatus(stub, transactionID, "")
			// updating trade state
			err = updateTradeState(stub, tradeID,"" ,"Trade Cancelled")
			if err != nil {
				_ = updateTransactionStatus(stub, transactionID, "Error while updating trade state")
				return nil, nil
			}
		}
		// update client state
		b, err := json.Marshal(client)
		if err == nil {
			err = stub.PutState(client.EntityID,b)
		} else {
			_ = updateTransactionStatus(stub, transactionID, "Error updating Client state")
			return nil, nil
		}
		// update bank state
		b, err = json.Marshal(bank)
		if err == nil {
			err = stub.PutState(bank.EntityID,b)
		} else {
			_ = updateTransactionStatus(stub, transactionID, "Error while updating Bank state")
			return nil, nil
		}
		// update transaction number
		err = stub.PutState("currentTransactionNum", []byte(strconv.Itoa(tid)))
		if err != nil {
			_ = updateTransactionStatus(stub, transactionID, "Error while writing currentTransactionNum to ledger")
			return nil, nil
		}
		return nil, nil
	}
	return nil, errors.New("Incorrect number of arguments")
}
// get user id
func (t *SimpleChaincode) getUserID(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	bytes, err := stub.GetCallerCertificate()
	x509Cert, err := x509.ParseCertificate(bytes)
	return []byte(x509Cert.Subject.CommonName), err
}
func (t *SimpleChaincode) getcurrentTransactionNum(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	ctidByte,err := stub.GetState("currentTransactionNum")
	if err != nil {
		return nil, errors.New("Error retrieving currentTransactionNum")
	}
    return ctidByte, err
}
func (t *SimpleChaincode) getValue(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
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
func (t *SimpleChaincode) readTradeIDsOfUser(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
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
func updateTradeHistory(stub shim.ChaincodeStubInterface, entityID string, tradeID string) (error) {
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

func updateTradeState(stub shim.ChaincodeStubInterface, tradeID string, transactionID string, status string) (error) {
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

func (t *SimpleChaincode) trial(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	return nil, errors.New("********* TRIAL ERROR *********")
}

/* error handling
	1. uuid return error
	2. no error returned check transactionID incremented or not
	3. maintain transaction status and check every time 
*/

/* if error 
update transaction status 
dont increment transaction number or trade number
dont include transaction in trade history
*/
// read trades of a client
func (t *SimpleChaincode) readTrades(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
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
func (t *SimpleChaincode) readQuoteRequests(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
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
		fmt.Print("Trade Status "+trade.Status)
		if trade.Status == "Quote requested" {
			quoteTransactions = append(quoteTransactions,trade.TransactionHistory[0])
		} else if trade.Status == "Responded" { // check who has responded
			respondedFlag := false
			bytes, _ := stub.GetCallerCertificate()
			fmt.Print(string(bytes))
			//x509Cert, _ := x509.ParseCertificate(bytes)
			currentUserID := args[0] //x509Cert.Subject.CommonName
			
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
				if tran.TransactionType == "Response" {
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
	b, err := json.Marshal(quoteTransactions)
	fmt.Print("Trade List"+string(b))
	return b, nil
}

func updateTransactionStatus(stub shim.ChaincodeStubInterface, transactionID string, status string) (error) {
		//Transaction
		t := Transaction{
		TransactionID: transactionID,
		Status: status,
		}
		// convert to Transaction to JSON
		b, err := json.Marshal(t)
		// write to ledger
		if err == nil {
			err = stub.PutState(t.TransactionID,b)
			if(err != nil){
				return errors.New("Error while writing Transaction to ledger")
			}
		} else {
			return errors.New("Json Marshalling error")
		}
		return nil
}
func (t *SimpleChaincode) getEntityList(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var allEntities []string
	var entities []string
	// get current Trade number
	ctidByte, err := stub.GetState("entityList")
	if(err != nil){
		return nil, errors.New("Error while getting entity list from ledger")
	}
	err = json.Unmarshal(ctidByte, &allEntities)		
	if(err != nil){
		return nil, errors.New("Error while unmarshalling entity data")
	}
	// check all entities
	for i:=0; i< len(allEntities); i++ {
		// read trade state
		entityByte,err := stub.GetState(allEntities[i])
		if err != nil {
			return nil, errors.New("Error while getting entity info from ledger")
		}
		var entity Entity
		err = json.Unmarshal(entityByte, &entity)		
		if err != nil {
			return nil, errors.New("Error while unmarshalling entity data")
		}
		// check type
		if entity.EntityType == "Client" || entity.EntityType == "Bank" {
			entities = append(entities,allEntities[i])
		}
	}
	b, err := json.Marshal(entities)
	return b, nil
}
func (t *SimpleChaincode) getAllTrades(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	// check entity type
	entitybyte,err := stub.GetState(args[0])																									
	if err != nil {
		return nil, errors.New("Error while getting entity info from ledger")
	}
	var entity Entity
	err = json.Unmarshal(entitybyte, &entity)		
	if err != nil {
		return nil, errors.New("Error while unmarshalling entity data")
	}
	if entity.EntityType == "RegBody" {		
			var tradeList []string
			// get current Trade number
			ctidByte, err := stub.GetState("currentTradeNum")
			if err != nil {
				return nil, errors.New("Error while getting currentTradeNum from ledger")
			}
			tradeNum,err := strconv.Atoi(string(ctidByte))
			if err != nil {
				return nil, errors.New("Error while converting ctidByte to integer")
			}
			for tradeNum > 1000 {
					tradeList = append(tradeList,"trade"+strconv.Itoa(tradeNum))
					tradeNum--
			}
			trades := make([]Trade,len(tradeList))
			for i:=0; i<len(tradeList); i++ {
				byteVal,err := stub.GetState(tradeList[i])
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
	return nil, errors.New("Error only Regulatory Body can access all trades")
}
func (t *SimpleChaincode) getTransactionStatus(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
		if len(args)== 1 {
				transactionID := "trans"+args[0]
				tbyte,err := stub.GetState(transactionID)
				if err != nil {
					return []byte("Error while getting transaction from ledger to get transaction status of "+transactionID), nil
				}
				var transaction Transaction
				err = json.Unmarshal(tbyte, &transaction)
				if err != nil {
					return []byte("Error while unmarshalling transaction data to get transaction status of "+transactionID), nil
				}
				return []byte(transaction.Status),nil
		}
		return nil, errors.New("Incorrect number of arguments")
}
