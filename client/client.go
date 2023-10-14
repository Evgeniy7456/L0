package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/Evgeniy7456/L0/source"
)

func main() {
	// Контекст ожидающий отмены от системы
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	client := &http.Client{}
	ch := make(chan int)
	go response(client, ch)

	for {
		select {
		// Закрытие сервера после сигнала от операционной системы
		case <-ctx.Done():
			fmt.Println("Client is closing")
			return
		// Повторный вызов функции для ввода id заказа
		case <-ch:
			go response(client, ch)
		}
	}
}

func response(client *http.Client, ch chan int) {
	defer func() {
		ch <- 1
	}()

	// Запрос id заказа у пользователя
	fmt.Print("Enter order ID: ")
	var order_uid string
	_, err := fmt.Scan(&order_uid)
	if err != nil {
		if err == io.EOF {
			fmt.Println()
		} else {
			fmt.Println(err)
		}
	}

	// Отправка id заказа POST запросом на сервер в формате json
	json_data := map[string]string{
		"order_uid": order_uid,
	}
	data, _ := json.Marshal(json_data)
	r := bytes.NewReader(data)
	resp, err := client.Post("http://localhost:8080/getData", "application/json", r)
	if err != nil {
		fmt.Println("Server is not available")
		return
	}
	// Вывод полученных данных
	var resp_json []byte
	json.NewDecoder(resp.Body).Decode(&resp_json)
	var json_model source.Model
	err = json.Unmarshal(resp_json, &json_model)
	if err != nil {
		fmt.Println("Invalid data format")
		return
	}
	fmt.Println(json_model.Order_uid)
	if json_model.Order_uid == "" {
		fmt.Println("This id does not exist")
		fmt.Println()
		return
	}

	fmt.Println("Track number:", json_model.Track_number)
	fmt.Println("Entry:", json_model.Entry)
	fmt.Println("Delivery:")
	fmt.Println("    Name:", json_model.Delivery.Name)
	fmt.Println("    Phone:", json_model.Delivery.Phone)
	fmt.Println("    Zip:", json_model.Delivery.Zip)
	fmt.Println("    City:", json_model.Delivery.City)
	fmt.Println("    Address:", json_model.Delivery.Address)
	fmt.Println("    Region", json_model.Delivery.Region)
	fmt.Println("    Email", json_model.Delivery.Email)
	fmt.Println("Payment:")
	fmt.Println("    Transaction:", json_model.Payment.Transaction)
	fmt.Println("    Request id", json_model.Payment.Request_id)
	fmt.Println("    Currency:", json_model.Payment.Currency)
	fmt.Println("    Provider:", json_model.Payment.Provider)
	fmt.Println("    Amount:", json_model.Payment.Amount)
	fmt.Println("    Payment dt:", json_model.Payment.Payment_dt)
	fmt.Println("    Bank:", json_model.Payment.Bank)
	fmt.Println("    Delivery cost:", json_model.Payment.Delivery_cost)
	fmt.Println("    Goods total:", json_model.Payment.Goods_total)
	fmt.Println("    Custom fee:", json_model.Payment.Custom_fee)
	fmt.Printf("Items count %d:\n", len(json_model.Items))
	for i := 0; i < len(json_model.Items); i++ {
		fmt.Printf("    Item №%d:\n", i+1)
		fmt.Println("        Chrt id:", json_model.Items[i].Chrt_id)
		fmt.Println("        Track number:", json_model.Items[i].Track_number)
		fmt.Println("        Price:", json_model.Items[i].Price)
		fmt.Println("        Rid:", json_model.Items[i].Rid)
		fmt.Println("        Name:", json_model.Items[i].Name)
		fmt.Println("        Sale:", json_model.Items[i].Sale)
		fmt.Println("        Size:", json_model.Items[i].Size)
		fmt.Println("        Total price:", json_model.Items[i].Total_price)
		fmt.Println("        nm id:", json_model.Items[i].Nm_id)
		fmt.Println("        Brand:", json_model.Items[i].Brand)
		fmt.Println("        Status:", json_model.Items[i].Status)
	}
	fmt.Println("Locale:", json_model.Locale)
	fmt.Println("Internal signature:", json_model.Internal_signature)
	fmt.Println("Customer id:", json_model.Customer_id)
	fmt.Println("Delivery service:", json_model.Delivery_service)
	fmt.Println("Shardkey:", json_model.Shardkey)
	fmt.Println("sm id:", json_model.Sm_id)
	fmt.Println("Date created:", json_model.Date_created)
	fmt.Println("oof shard:", json_model.Oof_shard)
}
