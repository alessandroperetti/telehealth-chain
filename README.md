# telehealth-chain
A framework for secure and transparent telemedicine transactions on the blockchain


## HOW TO CONFIGURE

Install all the pre-requisites of Hyperledger Fabric and the smart contract according to your OS:

[https://hyperledger-fabric.readthedocs.io/en/latest/prereqs.html](https://hyperledger-fabric.readthedocs.io/en/latest/prereqs.html)

Install `make` tool for your OS:

* windows: [https://gnuwin32.sourceforge.net/packages/make.htm](https://gnuwin32.sourceforge.net/packages/make.htm)

* mac: `brew install make`

* linux: `sudo apt-get install make`


In the project root folder to start up the network run:

`make download-fabric`  

`make start`

Go to `assets/chaincode` folder and run:

`go mod vendor`

It will create the `vendor` folder with the all go module dependencies locally.

`make install`

Once the installation process is done you have to change in the make file (Makefile) the variable `CC_PACKAGE_ID` with the output of the command:

`make check-installation`



Substitute the value of `CC_PACKAGE_ID` with the output of the command above:

Example:


`export CC_PACKAGE_ID :=ehr-payment_1.0:f4ff2b146eb6849e4dd7055ec731bcde2931b7742755852c6584e02b0f67b103`

Then, you can go ahead and approve and commit the cc:

`make approve`

`make commit`

`make committed`

Invoke the chaincode to initialize the ledger:

`make invoke`


To test if everything is working run (fetching the cc ):

`make get-patient1`

You should see something like this:

    peer chaincode query -C mychannel -n ehr-payment -c '{"Args":["GetPatient","patient1"]}'
    {"accessLog":[],"dob":"1990-01-01","id":"patient1","history":[],"name":"John Doe"}



Congratulations, you have successfully deployed the cc and initialized the ledger as well as tested the cc and getting a patient.


## HOW TO USE AN APPLICATION

A sample application is available in the `applications` folder.

Run the applications with:

`go run applications/ehr.go`





