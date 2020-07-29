package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/tadeokondrak/ircdiscord/internal/server"
)

// listen returns a net.Listener listening on port.
//
// If port is 0, the listening port will be 6667.
func listen(port int) (net.Listener, error) {
	if port == 0 {
		port = 6667
	}

	addr := fmt.Sprintf(":%d", port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	return listener, nil
}

// listenTLS returns a net.Listener listening on port.
//
// If port is 0, the listening port will be 6667.
// certfile and keyfile are paths to their respective files,
// to listen using TLS.
func listenTLS(port int, certfile, keyfile string) (net.Listener, error) {
	if certfile == "" || keyfile == "" {
		return nil, errors.New(
			"certfile and keyfile are required for TLS")
	}

	cert, err := tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		return nil, fmt.Errorf("failed to load keypair: %w", err)
	}

	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	if port == 0 {
		port = 6697
	}

	addr := fmt.Sprintf(":%d", port)

	listener, err := tls.Listen("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	return listener, nil
}

func main() {
	var (
		debug        bool
		ircDebug     bool
		discordDebug bool
		port         int
		tlsEnabled   bool
		certfile     string
		keyfile      string
	)

	flag.BoolVar(&debug, "debug", false,
		"enable verbose logging")
	flag.BoolVar(&ircDebug, "ircdebug", false,
		"enable verbose logging of irc communication")
	flag.BoolVar(&discordDebug, "discorddebug", false,
		"enable verbose logging of discord communication")
	flag.IntVar(&port, "port", 0,
		"port to run on, defaults to 6667/6697 depending on tls")
	flag.BoolVar(&tlsEnabled, "tls", false, "enable tls encryption")
	flag.StringVar(&certfile, "cert", "", "tls certificate file")
	flag.StringVar(&keyfile, "key", "", "tls key file")
	flag.Parse()

	if debug {
		log.SetFlags(log.Lshortfile)
	} else {
		log.SetFlags(0)
	}

	var ln net.Listener
	if tlsEnabled {
		var err error
		ln, err = listenTLS(port, certfile, keyfile)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		var err error
		ln, err = listen(port)
		if err != nil {
			log.Fatalln(err)
		}
	}

	server := server.New(ln, debug, ircDebug, discordDebug)
	defer server.Close()

	errors := make(chan error)

	go func() {
		if err := server.Run(); err != nil {
			errors <- err
		}
	}()

	log.Printf("listening on %v", ln.Addr())

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)

	select {
	case err := <-errors:
		log.Println(err)
	case sig := <-sigch:
		log.Printf("received signal '%v'", sig)
	}
}
