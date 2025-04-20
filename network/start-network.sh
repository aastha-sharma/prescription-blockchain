#!/bin/bash

export FABRIC_CFG_PATH=$PWD


# Generate crypto materials
cryptogen generate --config=./crypto-config.yaml

# Generate genesis block for the orderer
configtxgen -profile OrdererGenesis -channelID system-channel -outputBlock ./channel-artifacts/genesis.block

# Generate channel transaction artifacts
configtxgen -profile PrescriptionChannel -outputCreateChannelTx ./channel-artifacts/prescription_channel.tx -channelID prescriptionchannel
configtxgen -profile RefillChannel -outputCreateChannelTx ./channel-artifacts/refill_channel.tx -channelID refillchannel
configtxgen -profile RegulatoryChannel -outputCreateChannelTx ./channel-artifacts/regulatory_channel.tx -channelID regulatorychannel

# Start the network
docker-compose -f docker-compose.yaml up -d

# Sleep to ensure network is up
sleep 10

# Create channels
export CORE_PEER_MSPCONFIGPATH=./organizations/peerOrganizations/doctor.example.com/users/Admin@doctor.example.com/msp
export CORE_PEER_ADDRESS=peer0.doctor.example.com:7051
export CORE_PEER_LOCALMSPID="DoctorMSP"
export CORE_PEER_TLS_ROOTCERT_FILE=./organizations/peerOrganizations/doctor.example.com/peers/peer0.doctor.example.com/tls/ca.crt

peer channel create -o orderer.example.com:7050 -c prescriptionchannel -f ./channel-artifacts/prescription_channel.tx
peer channel create -o orderer.example.com:7050 -c refillchannel -f ./channel-artifacts/refill_channel.tx
peer channel create -o orderer.example.com:7050 -c regulatorychannel -f ./channel-artifacts/regulatory_channel.tx

# Join peers to channels (Doctor org)
peer channel join -b prescriptionchannel.block
peer channel join -b refillchannel.block
peer channel join -b regulatorychannel.block

# Join Patient org to prescription and refill channels
export CORE_PEER_MSPCONFIGPATH=./organizations/peerOrganizations/patient.example.com/users/Admin@patient.example.com/msp
export CORE_PEER_ADDRESS=peer0.patient.example.com:8051
export CORE_PEER_LOCALMSPID="PatientMSP"
export CORE_PEER_TLS_ROOTCERT_FILE=./organizations/peerOrganizations/patient.example.com/peers/peer0.patient.example.com/tls/ca.crt

peer channel join -b prescriptionchannel.block
peer channel join -b refillchannel.block

# Add similar commands for Pharmacy and Regulatory orgs

echo "Network setup complete!"