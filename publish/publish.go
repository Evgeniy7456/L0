package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/nats-io/stan.go"
)

func main() {
	// Подключение к NATS streaming server
	sc, err := stan.Connect("test-cluster", "test2")
	if err != nil {
		log.Fatal("Unable to connection to NATS streaming server!")
	}
	defer sc.Close()

	wg := sync.WaitGroup{}
	for i := 1; i <= 2; i++ {
		i := i
		wg.Add(1)
		go func() {
			name := fmt.Sprintf("..\\source\\model%d.json", i)

			// Чтение файлов с данными в формате json
			data, err := os.ReadFile(name)
			if err != nil {
				log.Println(err)
			} else {
				// Отправляем json данные в канал NATS streaming server
				err := sc.Publish("order-data", data)
				if err != nil {
					log.Fatal(err)
				}
			}

			wg.Done()
		}()
	}
	wg.Wait()
}
