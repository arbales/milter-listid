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
	"golang.org/x/exp/slices"
)

/* replyMilter object */
type replyMilter struct {
	milter.Milter
	listId string
	listUnsub string
}

// Connect is called when a new SMTP connection is received. The values for
// network and address are in the same format that would be passed to net.Dial.
func (b *replyMilter) Connect(hostname string, network string, address string, macros map[string]string) milter.Response {
	return milter.Continue
}

// Helo is called when the client sends its HELO or EHLO message.
func (b *replyMilter) Helo(name string, macros map[string]string) milter.Response {
	return milter.Continue
}

// To is called when the client sends a RCPT TO message. The recipient's
// address is passed without <> brackets. If it returns a rejection milter.Response,
// only the one recipient is rejected.
func (b *replyMilter) To(recipient string, macros map[string]string) milter.Response {
	return milter.Continue
}

// From is called when the client sends its MAIL FROM message. The sender's
// address is passed without <> brackets.
func (b *replyMilter) From(from string, macros map[string]string) milter.Response {
	return milter.Continue
}

// Headers is called when the message headers have been received.
func (b *replyMilter) Headers(headers textproto.MIMEHeader) milter.Response {
	toAddress := headers.Get("To");
	// toAddress = toAddress[1 : len(toAddress) - 1]
	log.Printf("to: %s", toAddress)
	toParts := strings.Split(toAddress, "@")

	// If we know the address is a list, it is a list.
	lists := []string{
		"utilities@giraffic.world",
		"houseboats-21@giraffic.world",
		"houseboats-22@giraffic.world",
		"22-leads@giraffic.world",
		"membership@giraffic.world",
		"2018@giraffic.world",
		"2021@giraffic.world",
		"test-email-list@giraffic.world",
	}
	doesMatch := slices.Contains(lists, toAddress)

	// If the list is from a List Domain it is also a list.
	listDomains := []string{"lists.giraffes.camp", "lists.giraffic.world"}
	listAddress := ""

	if len(toParts) > 1 {
		listAddress = fmt.Sprintf("%s@lists.giraffic.world", toParts[0])
	}

	domainDoesMatch := (len(toParts) > 1) && (slices.Contains(listDomains, toParts[1]))

	if doesMatch || domainDoesMatch {
		log.Print("Should add headers!")
		b.listId = listAddress
		b.listUnsub = "https://giraffic.world/lists"
  } else {
		log.Print("Not adding headers.")
		b.listId = ""
		b.listUnsub = ""
	}

	return milter.Continue
}

// Body is called when the message body has been received. It gives an
// opportunity for the milter to modify the message before it is delivered.
func (b *replyMilter) Body(body []byte, m milter.Modifier) milter.Response {
	log.Printf("b.listId: %s", b.listId)
	if len(b.listId) > 0 {
		log.Print("Adding headers")
		m.AddHeader("List-Unsubscribe", fmt.Sprintf("<%s>",b.listUnsub))
		m.AddHeader("List-ID", fmt.Sprintf("<%s>",b.listId))
		log.Print("Added headers")
	}
	return milter.Continue
}

/* NewObject creates new BogoMilter instance */
func runServer(socket net.Listener) {
	// declare milter init function
	init := func() milter.Milter {
		m := replyMilter{};
		m.listId = ""
		m.listUnsub = ""
		return &m;
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
		"./milter-listid.sock",
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
