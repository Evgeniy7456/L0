package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Evgeniy7456/L0/postgresql"
	"github.com/Evgeniy7456/L0/source"
	"github.com/jackc/pgx"
	"github.com/nats-io/stan.go"
)

type Cache struct {
	data      map[string]source.Model
	tableName string
}

func (c *Cache) init(db *pgx.Conn) {
	// Получение данных из БД
	res, err := postgresql.GetDataToDB(db, c.tableName)
	if err != nil {
		log.Fatal("Error retrieving data from database:", err)
	}

	// Сохранение данных в map
	json.Unmarshal(res, &c.data)
}

var cacheDB = Cache{
	tableName: "order_data",
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Подключение к NATS streaming server
	sc := ConnectNSS("test-cluster", "test")
	// Подключение к базе данных
	db := ConnectDB()
	// Инициализация кэша
	cacheDB.init(db)
	// Подписка на канал в NATS streaming server
	sub, wg := SubscribeNSS(sc, db, &cacheDB)

	// Запуск http-сервера
	http.HandleFunc("/getData", getData)
	go http.ListenAndServe("localhost:8080", nil)
	log.Println("Http-server start listening on port 8080")

	// Ожидание от системы сигнала на закрытие сервиса
	<-ctx.Done()

	// Завершение работы сервиса
	shutdown(sc, db, sub, wg)
}

func ConnectNSS(stanClasterID string, clientID string, options ...stan.Option) stan.Conn {
	sc, err := stan.Connect(stanClasterID, clientID)
	if err != nil {
		log.Fatal("Unable to connection to NATS streaming server!")
	}
	log.Print("Successful to connection to the NATS streaming server!")
	return sc
}

func ConnectDB() *pgx.Conn {
	db, err := postgresql.NewClient()
	if err != nil {
		log.Fatalf("Unable to connection to database: %v\n", err)
	}
	log.Println("Successful connection to the database!")
	return db
}

func SubscribeNSS(sc stan.Conn, db *pgx.Conn, cache *Cache) (stan.Subscription, *sync.WaitGroup) {
	// Mutex для избежания блокировки при одновременном подключении к БД и записи в map
	mu := sync.Mutex{}
	// WaitGroup для ожидания записи данных в БД при закрытии сервиса
	wg := sync.WaitGroup{}

	sub, err := sc.Subscribe("order-data", func(m *stan.Msg) {
		wg.Add(1)
		go func() {
			log.Print("Received a message")
			defer wg.Done()
			// Проверка полученных данных на соответсвие json модели
			json_model, err := valid(m.Data)
			if err != nil {
				log.Println("Error json validation:", err)
				return
			}

			mu.Lock()
			cache.data[json_model.Order_uid] = json_model
			postgresql.InsertIntoDB(db, cache.tableName, m.Data, json_model.Order_uid)
			mu.Unlock()
		}()
	}, stan.DeliverAllAvailable())
	if err != nil {
		log.Fatal("Unable to subscribe to NATS streaming server!")
	}
	log.Println("Successfully subscribed to a NATS streaming server channel!")
	return sub, &wg
}

func valid(data []byte) (json_model source.Model, err error) {
	err = json.Unmarshal(data, &json_model)
	if err != nil || len(json_model.Order_uid) == 0 {
		return json_model, fmt.Errorf("the received message is not in json format")
	}
	return json_model, nil
}

// Получение данных от клиента и отправка json
func getData(w http.ResponseWriter, r *http.Request) {
	var order_uid map[string]string
	json.NewDecoder(r.Body).Decode(&order_uid)
	json_data := cacheDB.data[order_uid["order_uid"]]
	data_byte, _ := json.Marshal(json_data)
	json.NewEncoder(w).Encode(data_byte)
}

func shutdown(sc stan.Conn, db *pgx.Conn, sub stan.Subscription, wg *sync.WaitGroup) {
	// Отписка от канала NATS streaming server
	sub.Unsubscribe()
	log.Println("Unsubscribing from a NATS streaming service channel")

	// Отключение от NATS streaming server
	sc.Close()
	log.Println("Closing the connection to the NATS streaming server")

	// Ожидание завершения записи полученных сообщений в БД
	log.Println("Waiting for data recording to complete")
	wg.Wait()

	// Отключение от БД
	db.Close()
	log.Println("Closing a database connection")
}
