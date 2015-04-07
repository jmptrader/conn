This is the socket connection handling part from one of my server applications


### when to use
if the structure of your networking message fits in this pattern: 

```
message = head + body

in which head is a fixed length byte array, holding the length value of the message body and some other info.
```

this package might help

for example, I used this package to read/write messages of structure like the following:

```
message = head + body

head = 1 byte + 4 bytes (the first 1 byte indicates message type, and the next 4 bytes indicating the length of body)

body = the real message playload. can be anything. In may case, it's protocol buf messages.
```

### usage
As of usage, you can see it in conn_test.go file.
Basically, the steps should be like this:

1. create a server instance with appropriate configurations.

2. start listening coming connection by called the serve() method of pre-created server instance.

3. create instance which conforms to the Delegate interface, and set it to coming Conn object.

4. expect the delegate methods to be called.

5. notice that there are some methods on the Conn object, which can be used to send message to peer, to close the connection, and to extend/reduce the timeout deadline.  

6. for any others stuff you can go right through the code, it's quite simple

bug reports are welcome.
