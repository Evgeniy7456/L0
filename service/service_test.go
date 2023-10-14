package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/Evgeniy7456/L0/postgresql"
	"github.com/Evgeniy7456/L0/source"
	"github.com/jackc/pgx"
	"github.com/nats-io/stan.go"
	. "github.com/smartystreets/goconvey/convey"
)

type Setup struct {
	NatsServer *exec.Cmd
	DB         *pgx.Conn
	SC         stan.Conn
	Sub        stan.Subscription
	WG         *sync.WaitGroup
	CacheDB    *Cache
	PublishSC  stan.Conn
}

func TestService(t *testing.T) {
	setup, err := setup()
	if err != nil {
		fmt.Println(err)
		return
	}

	t.Cleanup(func() {
		setup.NatsServer.Process.Kill()
		_, err := setup.DB.Exec("drop table test")
		if err != nil {
			fmt.Println("Table deletion error", err)
		}
		setup.DB.Close()
		setup.SC.Close()
		setup.PublishSC.Close()
	})

	// Тест подключений к БД, NATS streaming server, подписке на канал NATS streaming server
	Convey("Service test", t, func() {
		Convey("Database connection test", func() {
			So(setup.DB, ShouldNotHaveSameTypeAs, nil)
		})
		Convey("NATS streaming server subscription test", func() {
			So(setup.SC, ShouldNotHaveSameTypeAs, nil)
		})
		Convey("Channel subscription test in NATS streaming server", func() {
			So(setup.Sub, ShouldNotHaveSameTypeAs, nil)
		})
	})

	// Тест обработки полученных данных
	Convey("Data handling test", t, func() {
		// Корректные json данные
		Convey("Valid json test", func() {
			// Чтение данных из файла
			data, err := os.ReadFile("..\\source\\model1.json")
			if err != nil {
				fmt.Println(err)
				return
			}

			var json_data source.Model
			err = json.Unmarshal(data, &json_data)
			if err != nil {
				fmt.Println(err)
				return
			}
			// Тест записи данных в БД
			Convey("Test of writing data to the database", func() {
				err := setup.PublishSC.Publish("order-data", data)
				if err != nil {
					fmt.Println(err)
					return
				}
				time.Sleep(100 * time.Millisecond)
				db_data, err := postgresql.GetDataToDB(setup.DB, setup.CacheDB.tableName)
				if err != nil {
					fmt.Println(err)
					return
				}
				var json_db map[string]source.Model
				err = json.Unmarshal(db_data, &json_db)
				if err != nil {
					fmt.Println(err)
					return
				}
				So(json_data, ShouldEqual, json_db["b563feb7b2b84b6test"])
			})
			// Тест записи данных в map
			Convey("Cache write test", func() {
				So(json_data, ShouldEqual, setup.CacheDB.data["b563feb7b2b84b6test"])
			})
		})
		// Некорректные json данные
		Convey("Not a valid json test", func() {
			// Тест записи данных в БД
			Convey("Test of writing data to the database", func() {
				data := []byte(`{"uid": "b563feb7b2b84b6test2"}`)
				err := setup.PublishSC.Publish("order-data", data)
				if err != nil {
					fmt.Println(err)
					return
				}
				time.Sleep(100 * time.Millisecond)
				db_data, err := postgresql.GetDataToDB(setup.DB, setup.CacheDB.tableName)
				if err != nil {
					fmt.Println(err)
					return
				}
				var json_data source.Model
				var json_db map[string]source.Model
				err = json.Unmarshal(data, &json_data)
				if err != nil {
					fmt.Println(err)
					return
				}
				err = json.Unmarshal(db_data, &json_db)
				if err != nil {
					fmt.Println(err)
					return
				}
				So(json_db["b563feb7b2b84b6test2"], ShouldNotEqual, nil)
			})
			// Тест записи в map
			Convey("Cache write test", func() {
				So(setup.CacheDB.data["b563feb7b2b84b6test2"], ShouldEqual, setup.CacheDB.data[""])
			})
		})
	})
}

func setup() (*Setup, error) {
	// Запуск NATS streaming server
	natsServer := exec.Command("..\\build\\nats-streaming-server.exe")
	err := natsServer.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting NATS streaming server: %w", err)
	}

	cacheDB := Cache{
		data:      make(map[string]source.Model),
		tableName: "test",
	}

	// Подключение к БД, NATS streaming server и каналу передачи данных
	db := ConnectDB()
	sc := ConnectNSS("test-cluster", "test")
	sub, wg := SubscribeNSS(sc, db, &cacheDB)

	publishSC, err := stan.Connect("test-cluster", "test2")
	if err != nil {
		return nil, fmt.Errorf("unable to connection to NATS streaming server: %w", err)
	}

	setup := Setup{
		NatsServer: natsServer,
		DB:         db,
		SC:         sc,
		Sub:        sub,
		WG:         wg,
		CacheDB:    &cacheDB,
		PublishSC:  publishSC,
	}

	// Создание тестовой таблицы
	_, err = db.Exec("create table if not exists test (order_uid varchar not null primary key, json_data jsonb not null)")
	if err != nil {
		fmt.Println("Error creating database")
		panic(err)
	}

	return &setup, nil
}
