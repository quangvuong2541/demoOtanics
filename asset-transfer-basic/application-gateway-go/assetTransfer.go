/*
Copyright 2021 IBM All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	gwproto "github.com/hyperledger/fabric-protos-go/gateway"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

type Asset struct {
	AppraisedValue int    `json:"appraisedValue"`
	Color          string `json:"color"`
	ID             string `json:"id"`
	Owner          string `json:"owner"`
	Size           int    `json:"size"`
}

type TransferAssetRequest struct {
	ID    string `json:"id"`
	Owner string `json:"owner"`
}

const (
	mspID         = "Org1MSP"
	cryptoPath    = "../../test-network/organizations/peerOrganizations/org1.example.com"
	certPath      = cryptoPath + "/users/User1@org1.example.com/msp/signcerts/User1@org1.example.com-cert.pem"
	keyPath       = cryptoPath + "/users/User1@org1.example.com/msp/keystore/"
	tlsCertPath   = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint  = "localhost:7051"
	gatewayPeer   = "peer0.org1.example.com"
	channelName   = "channel1"
	chaincodeName = "basic"
)
// const (
// 	mspID         = "Org1MSP"
// 	cryptoPath    = "../../test-network/organizations/peerOrganizations/org1.example.com"
// 	certPath      = cryptoPath + "/users/User1@org1.example.com/msp/signcerts/cert.pem"
// 	keyPath       = cryptoPath + "/users/User1@org1.example.com/msp/keystore/"
// 	tlsCertPath   = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
// 	peerEndpoint  = "localhost:7051"
// 	gatewayPeer   = "peer0.org1.example.com"
// 	//channelName   = "mychannel"
// 	channelName   = "channel1"
// 	chaincodeName = "basic"
// )

var now = time.Now()
//var assetId = fmt.Sprintf("asset%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)
var asset_no int = 10

func main() {
	log.Println("============ application-golang starts ============")

	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection()
	defer clientConnection.Close()

	id := newIdentity()
	sign := newSign()

	// Create a Gateway connection for a specific client identity
	gateway, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gateway.Close()

	network := gateway.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	r := gin.Default()
	r.GET("/initLedger", func(c *gin.Context) {
		initLedger(contract)
		c.JSON(200, gin.H{
			"success": "ok",
		})
	})

	r.GET("/getAllAssets", func(c *gin.Context) {
		data, err := getAllAssets(contract)
		if err != nil {
			c.JSON(200, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.Status(200)
		c.Writer.Write(data)
	})

	r.POST("/createAsset", func(c *gin.Context) {
		req := &Asset{}
		err := c.ShouldBindJSON(req)
		if err != nil {
			c.JSON(200, gin.H{
				"error": err.Error(),
			})
			return
		}

		err = createAsset(contract, req)
		if err != nil {
			c.JSON(200, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": "ok",
		})
	})

	r.GET("/readAssetByID", func(c *gin.Context) {
		id, _ := c.GetQuery("id")
		data, err := readAssetByID(contract, id)
		if err != nil {
			c.JSON(200, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.Status(200)
		c.Writer.Write(data)
	})

	r.PUT("/transferAssetAsync", func(c *gin.Context) {
		req := &TransferAssetRequest{}
		err := c.ShouldBindJSON(req)
		if err != nil {
			c.JSON(200, gin.H{
				"error": err.Error(),
			})
			return
		}

		err = transferAssetAsync(contract, req)
		if err != nil {
			c.JSON(200, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": "ok",
		})
	})

	r.Run()

	log.Println("============ application-golang ends ============")
}

// newGrpcConnection creates a gRPC connection to the Gateway server.
func newGrpcConnection() *grpc.ClientConn {
	certificate, err := loadCertificate(tlsCertPath)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	connection, err := grpc.Dial(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return connection
}

// newIdentity creates a client identity for this Gateway connection using an X.509 certificate.
func newIdentity() *identity.X509Identity {
	certificate, err := loadCertificate(certPath)
	if err != nil {
		panic(err)
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		panic(err)
	}

	return id
}

func loadCertificate(filename string) (*x509.Certificate, error) {
	certificatePEM, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}
	return identity.CertificateFromPEM(certificatePEM)
}

// newSign creates a function that generates a digital signature from a message digest using a private key.
func newSign() identity.Sign {
	files, err := ioutil.ReadDir(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to read private key directory: %w", err))
	}
	privateKeyPEM, err := ioutil.ReadFile(path.Join(keyPath, files[0].Name()))

	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		panic(err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}

	return sign
}

/*
 This type of transaction would typically only be run once by an application the first time it was started after its
 initial deployment. A new version of the chaincode deployed later would likely not need to run an "init" function.
*/
func initLedger(contract *client.Contract) {
	fmt.Printf("Submit Transaction: InitLedger, function creates the initial set of assets on the ledger \n")

	_, err := contract.SubmitTransaction("InitLedger")
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Evaluate a transaction to query ledger state.
func getAllAssets(contract *client.Contract) ([]byte, error) {
	fmt.Println("Evaluate Transaction: GetAllAssets, function returns all the current assets on the ledger")

	evaluateResult, err := contract.EvaluateTransaction("GetAllAssets")
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate transaction: %w", err)
	}

	return evaluateResult, nil
}

// Submit a transaction synchronously, blocking until it has been committed to the ledger.
func createAsset(contract *client.Contract, req *Asset) error {
	fmt.Printf("Submit Transaction: CreateAsset, creates new asset with ID, Color, Size, Owner and AppraisedValue arguments \n")

	size := strconv.Itoa(req.Size)
	appValues := strconv.Itoa(req.AppraisedValue)
	
	asset_no = asset_no + 1
	assetId := fmt.Sprintf("asset%d", asset_no)
	_, err := contract.SubmitTransaction("CreateAsset", assetId, req.Color, size, req.Owner, appValues)
	if err != nil {
		return fmt.Errorf("failed to submit transaction: %w", err)
	}

	return nil
}

// Evaluate a transaction by assetID to query ledger state.
func readAssetByID(contract *client.Contract, id string) ([]byte, error) {
	fmt.Printf("Evaluate Transaction: ReadAsset, function returns asset attributes\n")

	evaluateResult, err := contract.EvaluateTransaction("ReadAsset", id)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate transaction: %w", err)
	}

	return evaluateResult, nil
}

/*
Submit transaction asynchronously, blocking until the transaction has been sent to the orderer, and allowing
this thread to process the chaincode response (e.g. update a UI) without waiting for the commit notification
*/
func transferAssetAsync(contract *client.Contract, req *TransferAssetRequest) error {
	fmt.Printf("Async Submit Transaction: TransferAsset, updates existing asset owner'\n")

	submitResult, commit, err := contract.SubmitAsync("TransferAsset", client.WithArguments(req.ID, req.Owner))
	if err != nil {
		return fmt.Errorf("failed to submit transaction asynchronously: %w", err)
	}

	fmt.Printf("Successfully submitted transaction to transfer ownership from %s to Mark. \n", string(submitResult))
	fmt.Println("Waiting for transaction commit.")

	if status, err := commit.Status(); err != nil {
		return fmt.Errorf("failed to get commit status: %w", err)
	} else if !status.Successful {
		return fmt.Errorf("transaction %s failed to commit with status: %d", status.TransactionID, int32(status.Code))
	}

	return nil
}

// Submit transaction, passing in the wrong number of arguments ,expected to throw an error containing details of any error responses from the smart contract.
func exampleErrorHandling(contract *client.Contract) {
	fmt.Println("Submit Transaction: UpdateAsset asset70, asset70 does not exist and should return an error")

	_, err := contract.SubmitTransaction("UpdateAsset")
	if err != nil {
		switch err := err.(type) {
		case *client.EndorseError:
			fmt.Printf("Endorse error with gRPC status %v: %s\n", status.Code(err), err)
		case *client.SubmitError:
			fmt.Printf("Submit error with gRPC status %v: %s\n", status.Code(err), err)
		case *client.CommitStatusError:
			if errors.Is(err, context.DeadlineExceeded) {
				fmt.Printf("Timeout waiting for transaction %s commit status: %s", err.TransactionID, err)
			} else {
				fmt.Printf("Error obtaining commit status with gRPC status %v: %s\n", status.Code(err), err)
			}
		case *client.CommitError:
			fmt.Printf("Transaction %s failed to commit with status %d: %s\n", err.TransactionID, int32(err.Code), err)
		}
		/*
		 Any error that originates from a peer or orderer node external to the gateway will have its details
		 embedded within the gRPC status error. The following code shows how to extract that.
		*/
		statusErr := status.Convert(err)
		for _, detail := range statusErr.Details() {
			errDetail := detail.(*gwproto.ErrorDetail)
			fmt.Printf("Error from endpoint: %s, mspId: %s, message: %s\n", errDetail.Address, errDetail.MspId, errDetail.Message)
		}
	}
}

//Format JSON data
func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, " ", ""); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}
