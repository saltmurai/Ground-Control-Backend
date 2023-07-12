package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/saltmurai/drone-api-service/gendb"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

// global database connection
var db *sql.DB
var queries *gendb.Queries
var redisClient *redis.Client
var channel *amqp.Channel
var connRabbitMQ *amqp.Connection

func InitDatabase() error {
	var err error
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), 5432, os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_USER"))

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		zap.L().Sugar().Errorf("Can't connect DB")
		return err
	}

	queries = gendb.New(db)
	opt, err := redis.ParseURL(fmt.Sprintf("redis://%s:6379", os.Getenv("REDIS_HOST")))
	if err != nil {
		zap.L().Sugar().Error(err)
		return err
	}
	redisClient = redis.NewClient(opt)

	connRabbitMQ, err = amqp.Dial(os.Getenv("AMQP_URL"))
	if err != nil {
		zap.L().Sugar().Error(err)
		return err
	}
	// Create a channel and declare a queue
	channel, err = connRabbitMQ.Channel()
	if err != nil {
		zap.L().Sugar().Error(err)
		return err
	}
	return nil
}

func GetDB() *sql.DB {
	return db
}

func GetQueries() *gendb.Queries {
	return queries
}

func CloseDB() {
	db.Close()
}

func GetRedisClient() *redis.Client {
	return redisClient
}

func GetChannel() *amqp.Channel {
	return channel
}

func closeChannel() {
	connRabbitMQ.Close()
	channel.Close()
}
