# first run
```sh
cd ./test-network
./network.sh up
./network.sh createChannel -c channel1
./network.sh deployCC -ccn basic -ccp ../asset-transfer-basic/chaincode-go -ccl go -c channel1
```

# to stop
```sh
./network.sh down
```


# Run api
```sh
cd asset-transfer-basic/application-gateway-go
go run assetTransfer.go
```

# test api
import ./asset-transfer-basic/application-gateway-go/API Hyperledger Example.postman_collection.json to postman

# view db

- localhost:7984/_utils/
- localhost:5984/_utils

admin / adminpw

