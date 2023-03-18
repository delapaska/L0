package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
	csh  *Cache
	name string
}

func NewDB() *DB {
	db := DB{}
	db.Init()
	return &db
}

const (
	OrderID  = "SELECT order_id FROM cache WHERE app_key = '%s' ORDER BY id DESC LIMIT %d"
	ITEMSADD = `INSERT INTO items (ChrtID, Price, Rid, Name, Sale, Size, TotalPrice, NmID, Brand)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	PAYMANTSADD = `INSERT INTO payment (Transaction, Currency, Provider, Amount, PaymentDt, Bank, DeliveryCost,
		GoodsTotal) values ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	ORDERADD = `INSERT INTO orders (OrderUID, Entry, InternalSignature, payment_id_fk, Locale, 
		CustomerID, TrackNumber, DeliveryService, Shardkey, SmID) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`
	INSERTCACHE = `INSERT INTO cache (order_id, app_key) VALUES ($1, $2)`
	ORDERBYID   = `SELECT OrderUID, Entry, InternalSignature, payment_id_fk, Locale, CustomerID, 
	TrackNumber, DeliveryService, Shardkey, SmID FROM orders WHERE id = $1`
	ITEMSDATA = `SELECT ChrtID, Price, Rid, Name, Sale, Size, TotalPrice, NmID, Brand 
	FROM items WHERE id = $1`
	INSERTORDERITEMS  = `INSERT INTO order_items (order_id_fk, item_id_fk) values ($1, $2)`
	CLEARCACHE        = `DELETE FROM cache WHERE app_key = $1`
	SELECTTRANSACTION = `SELECT Transaction, Currency, Provider, Amount, PaymentDt, Bank, DeliveryCost,
	GoodsTotal FROM payment WHERE id = $1`
	SELECTITEMID = "SELECT item_id_fk FROM order_items WHERE order_id_fk = $1"
)

// Для обратных вызовов в кеш, сохраняем инстанс *Cache
func (db *DB) SetCahceInstance(csh *Cache) {
	db.csh = csh
}

// Загрузка объектов Orders (кеша) при его восстановлении
func (db *DB) GetCacheState(bufSize int) (map[int64]Order, []int64, int, error) {
	buffer := make(map[int64]Order, bufSize)
	queue := make([]int64, bufSize)
	var queueInd int

	// Выбираем все OrderID для нашей программы (APP_KEY) из таблицы кеша
	query := fmt.Sprintf(OrderID, os.Getenv("APP_KEY"), bufSize)
	rows, err := db.pool.Query(context.Background(), query)
	if err != nil {
		log.Printf("%v: unable to get order_id from database: %v\n", db.name, err)
	}
	defer rows.Close()
	var oid int64
	for rows.Next() {
		if err := rows.Scan(&oid); err != nil {
			log.Printf("%v: unable to get oid from database row: %v\n", db.name, err)
			return buffer, queue, queueInd, errors.New("unable to get oid from database row")
		}
		queue[queueInd] = oid
		queueInd++

		o, err := db.GetOrderByID(oid)
		if err != nil {
			log.Printf("%v: unable to get order from database: %v\n", db.name, err)
			continue
		}
		buffer[oid] = o
	}
	if queueInd == 0 {
		return buffer, queue, queueInd, errors.New("cache is empty")
	}

	for i := 0; i < int(queueInd/2); i++ {
		queue[i], queue[queueInd-i-1] = queue[queueInd-i-1], queue[i]
	}

	return buffer, queue, queueInd, nil
}

func ErrListener(err error, o Order) (Order, error) {
	return o, errors.New("Can't get data from database")
}
func (db *DB) GetOrderByID(oid int64) (Order, error) {
	var o Order
	var payment_id_fk int64
	err := db.pool.QueryRow(context.Background(), ORDERBYID, oid).Scan(&o.OrderUID, &o.Entry,
		&o.InternalSignature, &payment_id_fk, &o.Locale, &o.CustomerID, &o.TrackNumber, &o.DeliveryService, &o.Shardkey,
		&o.SmID)
	if err != nil {
		ErrListener(err, o)
	}

	err = db.pool.QueryRow(context.Background(), SELECTTRANSACTION, payment_id_fk).Scan(&o.Payment.Transaction, &o.Payment.Currency, &o.Payment.Provider,
		&o.Payment.Amount, &o.Payment.PaymentDt, &o.Payment.Bank, &o.Payment.DeliveryCost, &o.Payment.GoodsTotal)
	if err != nil {
		log.Printf("%v: unable to get payment from database: %v\n", db.name, err)
		return o, errors.New("unable to get payment from database")
	}

	rowsItems, err := db.pool.Query(context.Background(), SELECTITEMID, oid)
	if err != nil {
		ErrListener(err, o)
	}
	defer rowsItems.Close()

	// Цикл по списку ItemsID
	var itemID int64
	for rowsItems.Next() {
		var item Items
		if err := rowsItems.Scan(&itemID); err != nil {
			return o, errors.New("unable to get itemID from database row")
		}
		// Сбор данных об Items
		err = db.pool.QueryRow(context.Background(), ITEMSDATA, itemID).Scan(&item.ChrtID, &item.Price, &item.Rid, &item.Name, &item.Sale, &item.Size,
			&item.TotalPrice, &item.NmID, &item.Brand)
		if err != nil {
			ErrListener(err, o)
		}
		o.Items = append(o.Items, item)
	}
	return o, nil
}

// Сохранение Order в БД
func (db *DB) AddOrder(o Order) (int64, error) {
	var lastInsertId int64
	var itemsIds []int64 = []int64{}

	tx, err := db.pool.Begin(context.Background())
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(context.Background())

	// добавление Items
	for _, item := range o.Items {
		err := tx.QueryRow(context.Background(), ITEMSADD, item.ChrtID, item.Price, item.Rid, item.Name, item.Sale, item.Size,
			item.TotalPrice, item.NmID, item.Brand).Scan(&lastInsertId)
		if err != nil {
			log.Printf("%v: unable to insert data (items): %v\n", db.name, err)
			return -1, err
		}
		itemsIds = append(itemsIds, lastInsertId)
	}

	// Добавление Payment
	err = tx.QueryRow(context.Background(), PAYMANTSADD, o.Payment.Transaction, o.Payment.Currency, o.Payment.Provider,
		o.Payment.Amount, o.Payment.PaymentDt, o.Payment.Bank, o.Payment.DeliveryCost, o.Payment.GoodsTotal).Scan(&lastInsertId)
	if err != nil {
		log.Printf("%v: unable to insert data (payment): %v\n", db.name, err)
		return -1, err
	}
	paymentIdFk := lastInsertId

	// Добавление Order
	err = tx.QueryRow(context.Background(), ORDERADD,
		o.OrderUID, o.Entry, o.InternalSignature, paymentIdFk, o.Locale, o.CustomerID, o.TrackNumber, o.DeliveryService,
		o.Shardkey, o.SmID).Scan(&lastInsertId)
	if err != nil {
		log.Printf("%v: unable to insert data (orders): %v\n", db.name, err)
		return -1, err
	}
	orderIdFk := lastInsertId

	for _, itemId := range itemsIds {
		_, err := tx.Exec(context.Background(), INSERTORDERITEMS,
			orderIdFk, itemId)
		if err != nil {
			log.Printf("%v: unable to insert data (order_items): %v\n", db.name, err)
			return -1, err
		}
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return 0, err
	}

	log.Printf("%v: Order successfull added to DB\n", db.name)
	db.csh.SetOrder(orderIdFk, o)
	return orderIdFk, nil
}

func (db *DB) SendOrderIDToCache(oid int64) {
	db.pool.QueryRow(context.Background(), INSERTCACHE, oid, os.Getenv("APP_KEY"))
	log.Printf("%v: OrderID successfull added to Cache (DB)\n", db.name)
}

func (db *DB) ClearCache() {
	_, err := db.pool.Exec(context.Background(), CLEARCACHE, os.Getenv("APP_KEY"))
	if err != nil {
		log.Printf("%v: clear cache error: %s\n", db.name, err)
	}
	log.Printf("%v: cache successfull cleared from database\n", db.name)
}
