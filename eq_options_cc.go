/*
update trade status after every transaction

*/
package main
import (
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"errors"
	"encoding/json"
	"strconv"
	"crypto/x509"
)
type Stock struct{
	Symbol string
	Quantity int
}
type Option struct{
	Symbol string
	Quantity int
	StockRate float64
	SettlementDate string	
}
type Entity struct{
	EntityId string				// enrollmentID
	EntityName string
	Portfolio []Stock
	Options []Option
}
type Trade struct				
{
	TradeId int					// rfq transaction id
	Status string				// "Quote requested" or "Responded" or "Trade executed" or "Trade settled" or "Trade timed out"
}
type Transaction struct{		// ledger transactions
	TransactionID int			// different for every transaction
	TradeId int					// same for all transactions corresponding to a single trade
	TransactionType string		// type of transaction rfq or resp or tradeExec or tradeSet
	OptionType string    		// buy/sell
	ClientID string				// entityId of client
	BankID string				// entityId of bank1 or bank2
	StockSymbol string				
	Quantity int
	OptionPrice float64
	StockRate float64	
	SettlementDate string	
}
type SimpleChaincode struct {
}
func main() {
    err := shim.Start(new(SimpleChaincode))
    if err != nil {
        fmt.Printf("Error starting Simple chaincode: %s", err)
    }
}
func (t *SimpleChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	
	// initialize entities	
	client:= Entity{		
		EntityId: "entity1",	  
		EntityName:	"Client A",
		Portfolio: []Stock{{Symbol:"GOOGL",Quantity:10},{Symbol:"AAPL",Quantity:20}},
		Options: []Option{{Symbol:"AMZN",Quantity:10,SettlementDate:"07/01/2016"}},
	}
	b, err := json.Marshal(client)
	if err == nil {
        err = stub.PutState(client.EntityId,b)
    } else {
		return nil, err
	}
	
	bank1:= Entity{
		EntityId: "entity2",
		EntityName:	"Bank A",
		Portfolio: []Stock{{Symbol:"MSFT",Quantity:200},{Symbol:"AAPL",Quantity:250},{Symbol:"AMZN",Quantity:400}},
	}
	b, err = json.Marshal(bank1)
	if err == nil {
        err = stub.PutState(bank1.EntityId,b)
    } else {
		return nil, err
	}
	
	bank2:= Entity{
		EntityId: "entity3",
		EntityName:	"Bank B",
		Portfolio: []Stock{{Symbol:"GOOGL",Quantity:150},{Symbol:"AAPL",Quantity:100}},
	}
	b, err = json.Marshal(bank2)
	if err == nil {
        err = stub.PutState(bank2.EntityId,b)
    } else {
		return nil, err
	}
	
	/*
	_, err = stub.GetState("currentTransactionID")
    if err != nil {
        err = stub.PutState("currentTransactionID", []byte("0"))
    }
	*/
	
	err = stub.PutState("currentTransactionID", []byte("1000"))
	
	ctidByte,err := stub.GetState("currentTransactionID")
	
	str:= "current TransactionID: "+string(ctidByte)
    return []byte(str), err
}
func (t *SimpleChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
    fmt.Println("invoke is running " + function)

    // Handle different functions
    if function == "init" {
        return t.Init(stub, "init", args)
    } else if function == "requestForQuote" {
        return t.requestForQuote(stub, args)
    } else if function == "respondToQuote" {
        return t.respondToQuote(stub, args)
    } else if function == "getValue" {
        return t.getValue(stub, args)
	}
    fmt.Println("invoke did not find func: " + function)
    return nil, errors.New("Received unknown function invocation")
}
func (t *SimpleChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
    fmt.Println("query is running " + function)

    // Handle different functions
    if function == "readEntity" {                       
        return t.readEntity(stub, args)
    }	else if function =="readTransaction" {
		return t.readTransaction(stub,args)
	}	else if function =="getUserID" {
		return t.getUserID(stub,args)
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
		valAsbytes, err = stub.GetState("entity1")
	} else if name == "bank1" {
		valAsbytes, err = stub.GetState("entity2")
	} else if name == "bank2" {
		valAsbytes, err = stub.GetState("entity3")
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
		ctidByte, err := stub.GetState("currentTransactionID")
		
		tid,err := strconv.Atoi(string(ctidByte))
		q,err := strconv.Atoi(args[3])
		
		bytes, err := stub.GetCallerCertificate();
		x509Cert, err := x509.ParseCertificate(bytes);
		
		t := Transaction{
		TransactionID: tid + 1,
		TradeId: tid + 1,							// create new tradeID
		TransactionType: "RFQ",
		OptionType: args[0],   						// based on input 
		ClientID:	x509Cert.Subject.CommonName,	// enrollmentID
		BankID: "",
		StockSymbol: args[1],						// based on input
		Quantity:	q,								// based on input
		OptionPrice: 0,
		StockRate: 0,
		SettlementDate: "",
		}
		err = stub.PutState("currentTransactionID", []byte(strconv.Itoa(tid + 1)))
		// convert to JSON
		b, err := json.Marshal(t)
		
		// write to ledger
		if err == nil {
			err = stub.PutState(strconv.Itoa(t.TransactionID),b)
			return []byte(strconv.Itoa(t.TransactionID)), err
		}
	}
	return nil, errors.New("Incorrect number of arguments")
}
/*			arg 0	:	TradeId or TransactionID of rfq
			arg 1	:	OptionPrice
			arg 2	:	StockRate
			arg 3	:	SettlementDate
*/
func (t *SimpleChaincode) respondToQuote(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args)== 4{
		
		ctidByte, err := stub.GetState("currentTransactionID")
		
		tid,err := strconv.Atoi(string(ctidByte))
		
		// get required data from input
		rate, err := strconv.ParseFloat(args[2], 64)
		price, err := strconv.ParseFloat(args[3], 64)
		
		tradeId,err := strconv.Atoi(args[0])
		
		// get bank's enrollment id
		bytes, err := stub.GetCallerCertificate();
		x509Cert, err := x509.ParseCertificate(bytes);
		
		// tradeID
		rfqbyte,err := stub.GetState("currentTransactionID")												// <<<<<<<<<<<<<<<<<<<<<<<<<<----------------------------------------
		var rfq Transaction
		err = json.Unmarshal(rfqbyte, &rfq)		
		
		t := Transaction{
		TransactionID: tid + 1,
		TradeId: tradeId,							// based on input
		TransactionType: "RESP",
		OptionType: rfq.OptionType,					// get from rfq
		ClientID:	rfq.ClientID,					// get from rfq
		BankID: x509Cert.Subject.CommonName,		// enrollmentID
		StockSymbol: rfq.StockSymbol,				// get from rfq
		Quantity:	rfq.Quantity,					// get from rfq
		OptionPrice: price,							// based on input
		StockRate: rate,							// based on input
		SettlementDate: args[3],					// based on input
		}
		err = stub.PutState("currentTransactionID", []byte(strconv.Itoa(tid + 1)))
		
		// convert to JSON
		b, err := json.Marshal(t)
		
		// write to ledger
		if err == nil {
			err = stub.PutState(strconv.Itoa(t.TransactionID),b)
			return []byte(strconv.Itoa(t.TransactionID)), err
		}
	}
	return nil, errors.New("Incorrect number of arguments")
}
func (t *SimpleChaincode) tradeExec(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	return nil,nil
}
func (t *SimpleChaincode) tradeSet(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	return nil,nil
}
func (t *SimpleChaincode) getEntityState(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	return nil,nil
}
// get user id
func (t *SimpleChaincode) getUserID(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	bytes, err := stub.GetCallerCertificate();
	x509Cert, err := x509.ParseCertificate(bytes);
	return []byte(x509Cert.Subject.CommonName), err
}
func (t *SimpleChaincode) getCurrentTransactionID(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	ctidByte,err := stub.GetState("currentTransactionID")
    return []byte(string(ctidByte)), err
}
func (t *SimpleChaincode) getValue(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	_,err := stub.GetState("xyzabc")
    return []byte(err.Error()), err
}
