package configs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var MongoChat *mongo.Database
var MongoRRHH *mongo.Database

func ConnectDB() {
	viper.SetConfigFile(".env")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Error cargando archivo .env: %v", err)
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=America/La_Paz",
		viper.GetString("DB_HOST"),
		viper.GetString("DB_USER"),
		viper.GetString("DB_PASSWORD"),
		viper.GetString("DB_NAME"),
		viper.GetString("DB_PORT"),
	)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Falló la conexión a PostgreSQL: %v", err)
	}

	DB = database
	log.Println("Conexión a PostgreSQL Exitosa")

	mongoURI := fmt.Sprintf("mongodb://%s:%s@%s:27017",
		viper.GetString("MONGO_USER"),
		viper.GetString("MONGO_PASSWORD"),
		viper.GetString("MONGO_HOST"),
	)

	if viper.GetString("MONGO_USER") == "" {
		mongoURI = fmt.Sprintf("mongodb://%s:27017", viper.GetString("MONGO_HOST"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Printf("Error conectando a MongoDB: %v", err)
	} else {
		err = client.Ping(ctx, nil)
		if err != nil {
			log.Printf("MongoDB conectado pero no responde al Ping: %v", err)
		} else {
			log.Println("Conexión a MongoDB Exitosa")
			MongoChat = client.Database(viper.GetString("MONGO_USERS_DB"))
			MongoRRHH = client.Database(viper.GetString("MONGO_RRHH_DB"))
		}
	}
}
