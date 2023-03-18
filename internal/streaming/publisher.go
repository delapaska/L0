package streaming

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"wb-test-task/internal/db"

	stan "github.com/nats-io/stan.go"
)

type Publisher struct {
	sc   *stan.Conn
	name string
}

func NewPublisher(conn *stan.Conn) *Publisher {
	return &Publisher{
		name: "Publisher",
		sc:   conn,
	}
}

type itemrand struct {
	ChrtID int
}

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func RandomNum() int {
	num := rand.Intn(1000)
	return num
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
func (p *Publisher) Publish() {
	//Отправление данных из привера model.json

	item := db.Items{ChrtID: RandomNum(), Price: RandomNum(), Rid: RandStringBytes(10), Name: RandStringBytes(10), Sale: RandomNum(), Size: RandStringBytes(10), TotalPrice: RandomNum(), NmID: RandomNum(), Brand: RandStringBytes(10)}

	payment := db.Payment{Transaction: RandomString(10), Currency: RandStringBytes(10), Provider: RandStringBytes(10), Amount: RandomNum(), PaymentDt: RandomNum(),
		Bank: RandStringBytes(10), DeliveryCost: 1500, GoodsTotal: 317}

	order := db.Order{OrderUID: RandomString(10), Entry: RandStringBytes(10), InternalSignature: RandStringBytes(10), Payment: payment, Items: []db.Items{item},
		Locale: RandStringBytes(10), CustomerID: RandStringBytes(10), TrackNumber: RandomString(10), DeliveryService: RandStringBytes(10), Shardkey: RandStringBytes(10), SmID: RandomNum()}

	// Проверка работы с другими значениями
	/*
		item := db.Items{ChrtID: 9934930, Price: 453, Rid: "ab4219087a764ae0btest", Name: "Mascaras", Sale: 30, Size: "0", TotalPrice: 10, NmID: 2389212, Brand: "Vivienne Sabo"}

		payment := db.Payment{Transaction: "b563feb7b2b84b6test", Currency: "USD", Provider: "wbpay", Amount: 1817, PaymentDt: 1637907727,
			Bank: "alpha", DeliveryCost: 1500, GoodsTotal: 317}
		order := db.Order{OrderUID: "1", Entry: "2", InternalSignature: "3", Payment: payment, Items: []db.Items{item},
			Locale: "4", CustomerID: "5", TrackNumber: "6", DeliveryService: "7", Shardkey: "8", SmID: 9}
	*/
	orderData, err := json.Marshal(order)

	if err != nil {
		log.Printf("%s: json.Marshal error: %v\n", p.name, err)
	}

	ackHandler := func(ackedNuid string, err error) {

		if err != nil {
			log.Printf("%s: error publishing msg id %s: %v\n", p.name, ackedNuid, err.Error())
		} else {
			log.Printf("%s: received ack for msg id: %s\n", p.name, ackedNuid)
		}
	}

	log.Printf("%s: publishing data ...\n", p.name)
	nuid, err := (*p.sc).PublishAsync(os.Getenv("NATS_SUBJECT"), orderData, ackHandler) // returns immediately
	if err != nil {
		log.Printf("%s: error publishing msg %s: %v\n", p.name, nuid, err.Error())
	}
}
