package models

type SenadorData struct {
	Departamento string `bson:"departamento"`
	Sigla        string `bson:"sigla"`
	Tipo         string `bson:"tipo"`
	Suplente     string `bson:"suplente"`
	Titular      string `bson:"titular"`
	Gestion      string `bson:"gestion"`
	Active       bool   `bson:"active"`
}

type ItemData struct {
	Unit string `bson:"unit"`
}

type FuncionarioPermanente struct {
	ItemData ItemData `bson:"item_data"`
}

type ItemUnitData struct {
	Name string `bson:"name"`
}

type FuncionarioEventual struct {
	UnitData ItemUnitData `bson:"unit_data"`
}

type MongoPersonaView struct {
	ID                    interface{}           `bson:"_id"`
	CI                    string                `bson:"ci"`
	Firstname             interface{}           `bson:"firstname"`
	Secondname            interface{}           `bson:"secondname"`
	Lastname              interface{}           `bson:"lastname"`
	Surname               interface{}           `bson:"surname"`
	Phone                 interface{}           `bson:"phone"`
	Address               interface{}           `bson:"address"`
	Email                 interface{}           `bson:"email"`
	Gender                interface{}           `bson:"gender"`
	TipoFuncionario       interface{}           `bson:"tipo_funcionario"`
	SenadorData           SenadorData           `bson:"senador_data"`
	FuncionarioPermanente FuncionarioPermanente `bson:"funcionario_permanente"`
	FuncionarioEventual   FuncionarioEventual   `bson:"funcionario_eventual"`
}
