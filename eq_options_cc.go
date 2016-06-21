package main
import (
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"errors"
	"encoding/json"
	"strconv"
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
type Transaction struct{		// ledger transactions
	TransactionID int			// different for every transaction
	TradeId int					// same for all transactions corresponding to a single trade
	TransactionType string		// type of transaction rfq or resp or tradeExec or tradeSet
	OptionType string    				// buy/sell
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
	if err != nil {
        err = stub.PutState(client.EntityId,b)
    }
	
	bank1:= Entity{
		EntityId: "entity2",
		EntityName:	"Bank A",
		Portfolio: []Stock{{Symbol:"MSFT",Quantity:200},{Symbol:"AAPL",Quantity:250},{Symbol:"AMZN",Quantity:400}},
	}
	b, err = json.Marshal(bank1)
	if err != nil {
        err = stub.PutState(bank1.EntityId,b)
    }
	
	bank2:= Entity{
		EntityId: "entity3",
		EntityName:	"Bank B",
		Portfolio: []Stock{{Symbol:"GOOGL",Quantity:150},{Symbol:"AAPL",Quantity:100}},
	}
	b, err = json.Marshal(bank2)
	if err != nil {
        err = stub.PutState(bank2.EntityId,b)
    }	
	/*ctidByte, err := stub.GetState("currentTransactionID")
    if err != nil {
        err = stub.PutState("currentTransactionID", []byte("0"))
    }
	*/
    return nil, err
}
func (t *SimpleChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
    fmt.Println("invoke is running " + function)

    // Handle different functions
    if function == "init" {
        return t.Init(stub, "init", args)
    } else if function == "requestForQuote" {
        return t.requestForQuote(stub, args)
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
/*			arg 0	: 
			arg 1	:	OptionType
			arg 2	:	StockSymbol
			arg 3	:	Quantity
			arg 4	:
*/
func (t *SimpleChaincode) requestForQuote(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args)== 4{
		ctidByte, err := stub.GetState("currentTransactionID")
		
		tid,err := strconv.Atoi(string(ctidByte))
		q,err := strconv.Atoi(args[3])
		
		t := Transaction{
		TransactionID: tid + 1,
		TradeId: tid + 1,							// create new tradeID
		TransactionType: "RFQ",
		OptionType: args[1],   						// based on input 
		ClientID:	"",								// get enrollmentID
		BankID: "",
		StockSymbol: args[2],						// based on input
		Quantity:	q,								// based on input
		OptionPrice: 0,
		StockRate: 0,
		SettlementDate: "",
		}
		
		// convert to JSON
		b, err := json.Marshal(t)
		
		// write to ledger
		if err == nil {
			err = stub.PutState(strconv.Itoa(t.TransactionID),b)
			return nil, err
		}
	}
	return nil, errors.New("Incorrect number of arguments")
}

func (t *SimpleChaincode) respondToQuote(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	return nil,nil	
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


