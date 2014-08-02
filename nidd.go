package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	Name    string = "nidd"
	Version string = "0.0.1"

	Port    int           = 113
	Timeout time.Duration = 10
)

func main() {
	var (
		ident string

		lsnr net.Listener
		conn net.Conn

		err error
	)

	// Set up the listener.
	lsnr, err = net.Listen("tcp", fmt.Sprintf(":%d", Port))
	if err != nil {
		log.Fatalf("Error binding to socket: %s", err)
	}

	// Print out some helpful information.
	log.Printf("%s %s", Name, Version)
	log.Printf("Listening on port %d.", Port)

	// Set the ident response to the first argument.
	if len(os.Args) > 1 {
		ident = os.Args[1]

		log.Printf("Responding with '%s'.", ident)
	} else {
		log.Print("No response specified.")
	}

	for {
		// Wait for client connections.
		conn, err = lsnr.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %s", err)
		}

		// Handle the connection in a goroutine.
		go func(conn net.Conn) {
			var (
				response chan string
				data     string
			)

			// Always close the client connection.
			defer func() {
				log.Printf("Closed connection from %s.", conn.RemoteAddr())
				conn.Close()
			}()

			log.Printf("Accepted connection from %s.", conn.RemoteAddr())

			// Create a channel to get the response from.
			response = make(chan string)
			// Spawn a goroutine to read the request.
			go func(conn net.Conn, response chan<- string) {
				var (
					rdr  *bufio.Reader
					regx *regexp.Regexp

					data string
					resp string

					err error
				)

				// Create a buffered reader.
				rdr = bufio.NewReader(conn)

				// Compile the regexp.
				regx = regexp.MustCompile("^[0-9]{1,5} ?, ?[0-9]{1,5}$")

				// Read data from the client.
				data, err = rdr.ReadString('\n')
				if err != nil {
					return
				}

				// Trim CRLF.
				resp = strings.TrimSpace(data)

				// Check if the request is valid and build a response.
				if regx.MatchString(resp) {
					if len(ident) > 0 {
						// Return ident if it's set.
						resp = fmt.Sprintf("%s : USERID : UNIX : %s\r\n", resp, ident)
					} else {
						// Otherwise return NO-USER.
						resp = fmt.Sprintf("%s : ERROR : NO-USER", resp)
					}
				} else {
					// Otherwise just return UNKNOWN-ERROR.
					resp = fmt.Sprintf("%s : ERROR : UNKNOWN-ERROR", resp)
				}

				// Send back the response.
				response <- resp
			}(conn, response)

			// Block until a response is ready or we time out.
			select {
			case data = <-response:
				var wrt *bufio.Writer

				// Create a buffered writer
				wrt = bufio.NewWriter(conn)

				log.Printf("Sent response to %s: %s", conn.RemoteAddr(), data)

				// Buffer the response
				_, err = wrt.WriteString(data)
				if err != nil {
					return
				}

				// Write the response.
				wrt.Flush()
			case <-time.After(time.Second * Timeout):
				// Let us know we timed out.
				log.Printf("Request timed out from %s.", conn.RemoteAddr())
			}
		}(conn)
	}
}

// vi: ts=4
