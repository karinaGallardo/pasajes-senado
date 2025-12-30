package models

type SenadorData struct {
	Departamento string `bson:"departamento"`
	Sigla        string `bson:"sigla"`
	Tipo         string `bson:"tipo"`
	Suplente     string `bson:"suplente"`
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
	CI                    string                `bson:"ci"`
	SenadorData           SenadorData           `bson:"senador_data"`
	FuncionarioPermanente FuncionarioPermanente `bson:"funcionario_permanente"`
	FuncionarioEventual   FuncionarioEventual   `bson:"funcionario_eventual"`
}
