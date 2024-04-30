package utils

import (
	"log"
	"net"
)

func readFromMicroservice(conn net.Conn) {
	defer conn.Close()

	for {
		buff := make([]byte, 1024)
		n, err := conn.Read(buff)
		if err != nil {
			log.Println("Error reading data from Python microservice:", err)
			return
		}

		log.Println("Received data from Python microservice:", buff[:n])
	}
}
