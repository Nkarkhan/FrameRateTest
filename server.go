package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

const (
	CONN_HOST = "localhost"
	CONN_PORT = "3333"
	CONN_TYPE = "tcp"

	FILE_SZ     = 10000 // 1 Mb files
	FILE_SZ_STR = "File Size:"
)

func main() {
	serverPtr := flag.Bool("server", false, "is this the server")
	addressPtr := flag.String("address", "localhost", "Bind/target address for server/client")
	portPtr := flag.Int("port", 3333, "Bind/target port for server/client")
	fileSzPtr := flag.Int("fSize", 100000, "File Size")

	flag.Parse()
	fmt.Println("ServerMode:", *serverPtr)
	fmt.Println("address:", *addressPtr)
	fmt.Println("port:", *portPtr)
	fmt.Println("fileSz:", *fileSzPtr)
	if *serverPtr {
		setupServer(*addressPtr, *portPtr, *fileSzPtr)
	} else {
		setupClient(*addressPtr, *portPtr, *fileSzPtr)
	}
}

// Setup Server
func setupServer(serverAddress string,
	serverPort int,
	fileSize int) {
	// Listen for incoming connections.
	l, err := net.Listen(CONN_TYPE, serverAddress+":"+strconv.Itoa(serverPort))
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + serverAddress + ":" + strconv.Itoa(serverPort))
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn, fileSize)
	}
}

func setupClient(serverAddress string,
	serverPort int,
	fileSize int) {
	con, err := net.Dial(CONN_TYPE, serverAddress+":"+strconv.Itoa(serverPort))
	if err != nil {
		log.Fatalln(err)
	}
	defer con.Close()
	buf := make([]byte, fileSize)
	for i := 0; i < fileSize; i++ {
		buf[i] = byte(i)
	}
	hz := 0
	for {
		wrLen, err := con.Write(buf)
		if err != nil {
			fmt.Println("Error writing:", err.Error())
		}
		if wrLen != fileSize {
			fmt.Println("Wrote :", wrLen)
		}
		hz = hz + 1
		if hz == 30 {
			fmt.Println("Send 30 frames")
			hz = 0
		}
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn, fSize int) {
	defer conn.Close()
	// Make a buffer to hold incoming data.
	buf := make([]byte, fSize+1000)
	// Print chunks of filebuffer received every second
	t := time.Now()
	hz := 0
	totalRead := 0
	for {
		rdLen, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading:", err.Error())
		}
		if rdLen != 0 {
			totalRead = totalRead + rdLen
			framesHere := totalRead / fSize
			totalRead = totalRead % fSize
			hz = hz + framesHere
		}
		if time.Since(t) >= time.Duration(time.Second) {
			fmt.Println("Frame Rate: ", hz)
			t = time.Now()
		}
	}
	// Close the connection when you're done with it.

}
