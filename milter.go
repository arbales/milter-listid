/*milter service for postfix*/
package main

import (
	"flag"
	"log"
	"net"
	"net/textproto"
	"os"
	"strings"
	"fmt"
	"github.com/andybalholm/milter"
)

/* replyMilter object */
type replyMilter struct {
	milter.Milter
	listId string
	listUnsub string
}

// Connect is called when a new SMTP connection is received. The values for
// network and address are in the same format that would be passed to net.Dial.
func (b replyMilter) Connect(hostname string, network string, address string, macros map[string]string) milter.Response {
	return milter.Continue
}

// Helo is called when the client sends its HELO or EHLO message.
func (b replyMilter) Helo(name string, macros map[string]string) milter.Response {
	return milter.Continue
}

// To is called when the client sends a RCPT TO message. The recipient's
// address is passed without <> brackets. If it returns a rejection milter.Response,
// only the one recipient is rejected.
func (b replyMilter) To(recipient string, macros map[string]string) milter.Response {
	return milter.Continue
}

// From is called when the client sends its MAIL FROM message. The sender's
// address is passed without <> brackets.
func (b replyMilter) From(from string, macros map[string]string) milter.Response {
	return milter.Continue
}

// Headers is called when the message headers have been received.
func (b replyMilter) Headers(headers textproto.MIMEHeader) milter.Response {
	toAddress := headers.Get("To");
	toAddress = toAddress[1 : len(toAddress) - 1]
	toParts := strings.Split(toAddress, "@")
	if len(toParts) > 1  && toParts[1] == "lists.giraffic.world" {
		b.listId = toAddress
		b.listUnsub = "https://giraffic.world/lists"
  }

	return milter.Continue
}

// Body is called when the message body has been received. It gives an
// opportunity for the milter to modify the message before it is delivered.
func (b replyMilter) Body(body []byte, m milter.Modifier) milter.Response {
	if len(b.listId) > 0 {
		m.AddHeader("List-Unsubscribe", fmt.Sprintf("<%s>",b.listUnsub))
		m.AddHeader("List-ID", fmt.Sprintf("<%s>",b.listId))
	}
	return milter.Accept
}

/* NewObject creates new BogoMilter instance */
func runServer(socket net.Listener) {
	// declare milter init function
	init := func() milter.Milter {
		return replyMilter{};
	}
	// start server
	if err := milter.Serve(socket, init); err != nil {
		log.Fatal(err)
	}
}

/* main program */
func main() {
	// parse commandline arguments
	var protocol, address string
	flag.StringVar(&protocol,
		"proto",
		"unix",
		"Protocol family (unix or tcp)")
	flag.StringVar(&address,
		"addr",
		"./milter-replyto.sock",
		"Bind to address or unix domain socket")
	flag.Parse()

	var isProtoUnix = (protocol == "unix")

	// make sure the specified protocol is either unix or tcp
	if !isProtoUnix && protocol != "tcp" {
		log.Fatal("invalid protocol name")
	}

	// make sure socket does not exist
	if isProtoUnix {
		// ignore os.Remove errors
		os.Remove(address)
	}

	// bind to listening address
	socket, err := net.Listen(protocol, address)
	if err != nil {
		log.Fatal(err)
	}
	defer socket.Close()

	if isProtoUnix {
		// set mode 0660 for unix domain sockets
		if err := os.Chmod(address, 0660); err != nil {
			log.Fatal(err)
		}
		// remove socket on exit
		defer os.Remove(address)
	}

	// run server
	go runServer(socket)

	// sleep forever
	select {}
}
