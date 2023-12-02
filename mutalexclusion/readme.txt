The setup steps: 
- open 3 terminals
- write in terminal 1:go run node/node.go -port 5555 -next 5556 -nextnext 5557
- write in terminal 2:go run node/node.go -port 5556 -next 5557 -nextnext 5555
- write in terminal 3:go run node/node.go -port 5557 -next 5555 -nextnext 5556 
- Then in each of the terminal write: connect
- In any of the terminals write: vote

Now the program is ready to have nodes accessing the critical section
- to request access write: access