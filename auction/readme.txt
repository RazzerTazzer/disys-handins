How to run the program:
- Open up 5 terminals (3 for the server, 2 for the clients bidding)
- Write in terminal 1: go run server/server.go -port 5555 -next 5556 -nextnext 5557
- Write in terminal 2: go run server/server.go -port 5556 -next 5557 -nextnext 5555
- Write in terminal 3: go run server/server.go -port 5557 -next 5555 -nextnext 5556

Now that the server-nodes are setup they can be connected to eachother.
- Write in each of the the three terminals: connect

Now it is time to find a leader
- Write one of the server terminals: vote

Now the server is ready and it is time to setup the clients
- write in terminal 4: go run client/client.go -port 8080 -name Bella
- write in terminal 5: go run client/client.go -port 8090 -name Ole

To start the auction, one of the clients have to make a bid.
- write in one of the client termnals: bid
- prompted with how much write an integer

To ask about the status of auction and see the current highest bid and if it is still active
- write in terminal of a client: currentBid

To crash one of the server-nodes. 
- in one of the serverterminals press CTRL + C


good luck have fun 