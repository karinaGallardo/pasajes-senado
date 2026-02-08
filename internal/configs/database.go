package configs

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
var MongoChat *mongo.Database
var MongoRRHH *mongo.Database

func ConnectDB() {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Aviso: No se pudo cargar el archivo .env (%v). Se usarán las variables de entorno del sistema.", err)
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=America/La_Paz",
		viper.GetString("DB_HOST"),
		viper.GetString("DB_USER"),
		viper.GetString("DB_PASSWORD"),
		viper.GetString("DB_NAME"),
		viper.GetString("DB_PORT"),
	)

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			Colorful:                  true,
		},
	)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            false,
		Logger:                 newLogger,
	})
	if err != nil {
		log.Fatalf("Falló la conexión a PostgreSQL: %v", err)
	}

	sqlDB, err := database.DB()
	if err == nil {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	DB = database
	RegisterAuditCallbacks(DB)
	// log.Println("Conexión a PostgreSQL Exitosa")
	log.Printf("Conexión a PostgreSQL Exitosa (%s:%s)\n", viper.GetString("DB_HOST"), viper.GetString("DB_PORT"))

	mongoPort := viper.GetString("MONGO_PORT")
	if mongoPort == "" {
		mongoPort = "27017"
	}

	mongoURI := fmt.Sprintf("mongodb://%s:%s@%s:%s",
		viper.GetString("MONGO_USER"),
		viper.GetString("MONGO_PASSWORD"),
		viper.GetString("MONGO_HOST"),
		mongoPort,
	)

	if viper.GetString("MONGO_USER") == "" {
		mongoURI = fmt.Sprintf("mongodb://%s:%s", viper.GetString("MONGO_HOST"), mongoPort)
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
