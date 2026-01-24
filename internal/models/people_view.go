package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type SenadorData struct {
	Departamento string `bson:"departamento"`
	Sigla        string `bson:"sigla"`
	Tipo         string `bson:"tipo"`
	SuSuplenteCI string `bson:"su_suplente_ci"`
	SuTitularCI  string `bson:"su_titular_ci"`
	Gestion      string `bson:"gestion"`
	Active       bool   `bson:"active"`
}

type MongoPersonaView struct {
	ID              primitive.ObjectID `bson:"_id"`
	CI              string             `bson:"ci"`
	Firstname       string             `bson:"firstname"`
	Secondname      string             `bson:"secondname"`
	Lastname        string             `bson:"lastname"`
	Surname         string             `bson:"surname"`
	Phone           string             `bson:"phone"`
	Address         string             `bson:"address"`
	Email           string             `bson:"email"`
	Gender          string             `bson:"gender"`
	TipoFuncionario string             `bson:"tipo_funcionario"`
	Cargo           string             `bson:"cargo"`
	Dependencia     string             `bson:"dependencia"`
	SenadorData     SenadorData        `bson:"senador_data"`
}
