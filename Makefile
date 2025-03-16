export PATH := ${PWD}/fabric-samples/bin:${PATH}
export FABRIC_CFG_PATH := ${PWD}/fabric-samples/config/
export CC_PACKAGE_ID := ehr-payment_1.0:f4ff2b146eb6849e4dd7055ec731bcde2931b7742755852c6584e02b0f67b103
# Renamed to download-fabric to avoid conflict with other install target
download-fabric: 
	curl -sSL https://bit.ly/2ysbOFE | bash -s

start:
	cd fabric-samples/test-network && ./network.sh down
	cd fabric-samples/test-network && ./network.sh up createChannel -ca

package:
	peer lifecycle chaincode package ehr-payment.tar.gz --path ./assets/chaincode --lang golang --label ehr-payment_1.0

install-smart-org1:
	# Environment variables for Org1
	export CORE_PEER_TLS_ENABLED=true && \
	export CORE_PEER_LOCALMSPID="Org1MSP" && \
	export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt && \
	export CORE_PEER_MSPCONFIGPATH=${PWD}/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp && \
	export CORE_PEER_ADDRESS=localhost:7051 && \
	peer lifecycle chaincode install ehr-payment.tar.gz

install-smart-org2:
	# Environment variables for Org2
	export CORE_PEER_TLS_ENABLED=true && \
	export CORE_PEER_LOCALMSPID="Org2MSP" && \
	export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/fabric-samples/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt && \
	export CORE_PEER_MSPCONFIGPATH=${PWD}/fabric-samples/test-network/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp && \
	export CORE_PEER_ADDRESS=localhost:9051 && \
	peer lifecycle chaincode install ehr-payment.tar.gz


install: package install-smart-org1 install-smart-org2

check-installation:
	peer lifecycle chaincode queryinstalled

approve-org1:
	export CORE_PEER_LOCALMSPID="Org1MSP" && \
	export CORE_PEER_MSPCONFIGPATH=${PWD}/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp && \
	export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt && \
	export CORE_PEER_ADDRESS=localhost:7051 && \
	peer lifecycle chaincode approveformyorg --channelID mychannel -o localhost:7050 --name ehr-payment --version 1.0 --package-id ${CC_PACKAGE_ID} --sequence 1 --tls --cafile "${PWD}/fabric-samples/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"

approve-org2:
	export CORE_PEER_LOCALMSPID="Org2MSP" && \
	export CORE_PEER_MSPCONFIGPATH=${PWD}/fabric-samples/test-network/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp && \
	export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/fabric-samples/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt && \
	export CORE_PEER_ADDRESS=localhost:9051 && \
	peer lifecycle chaincode approveformyorg -o localhost:7050 --channelID mychannel --name ehr-payment --version 1.0 --package-id ${CC_PACKAGE_ID} --sequence 1 --tls --cafile "${PWD}/fabric-samples/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"

approve: approve-org1 approve-org2

commit:	
	peer lifecycle chaincode checkcommitreadiness --channelID mychannel --name ehr-payment --version 1.0 --sequence 1 --tls --cafile "${PWD}/fabric-samples/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --output json
	peer lifecycle chaincode commit -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID mychannel --name ehr-payment --version 1.0 --sequence 1 --tls --cafile "${PWD}/fabric-samples/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/fabric-samples/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt"

committed:
	peer lifecycle chaincode querycommitted --channelID mychannel --name ehr-payment --cafile "${PWD}/fabric-samples/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
	
invoke:
	peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/fabric-samples/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n ehr-payment --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/fabric-samples/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt" -c '{"function":"InitLedgerEHR","Args":[]}'

get-patient1:
	peer chaincode query -C mychannel -n ehr-payment -c '{"Args":["GetPatient","patient1"]}'

get-patient2:
	peer chaincode query -C mychannel -n ehr-payment -c '{"Args":["GetPatient","patient2"]}'

channel-list:
	peer channel getinfo -c mychannel

.PHONY: download-fabric start deploy-smart-org1 deploy-smart-org2 approve check-installation package install commit committed invoke get-patient1 get-patient2 channel-list