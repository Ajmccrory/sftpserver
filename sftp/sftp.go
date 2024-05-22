package sftp

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func HandleConnection(nConn net.Conn, config *ssh.ServerConfig, debugStream io.Writer, readOnly bool) {
	_, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		log.Fatal("Failed to handshake: ", err)
	}
	fmt.Fprintf(debugStream, "SSH server established\n")

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		fmt.Fprintf(debugStream, "Incoming channel: %s\n", newChannel.ChannelType())
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			fmt.Fprintf(debugStream, "Unknown channel type: %s\n", newChannel.ChannelType())
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Fatal("Could not accept channel: ", err)
		}
		fmt.Fprintf(debugStream, "Channel accepted\n")

		go handleRequests(requests, debugStream)

		serverOptions := []sftp.ServerOption{
			sftp.WithDebug(debugStream),
		}

		if readOnly {
			serverOptions = append(serverOptions, sftp.ReadOnly())
			fmt.Fprintf(debugStream, "Read-only server\n")
		} else {
			fmt.Fprintf(debugStream, "Read-write server\n")
		}

		server, err := sftp.NewServer(channel, serverOptions...)
		if err != nil {
			log.Fatal(err)
		}
		if err := server.Serve(); err == io.EOF {
			server.Close()
			log.Print("SFTP client exited session.")
		} else if err != nil {
			log.Fatal("SFTP server completed with error: ", err)
		}
	}
}

func handleRequests(in <-chan *ssh.Request, debugStream io.Writer) {
	for req := range in {
		fmt.Fprintf(debugStream, "Request: %v\n", req.Type)
		ok := false
		if req.Type == "subsystem" && string(req.Payload[4:]) == "sftp" {
			ok = true
		}
		fmt.Fprintf(debugStream, " - accepted: %v\n", ok)
		req.Reply(ok, nil)
	}
}
