package dtos

type CreateDestinoRequest struct {
	IATA               string `form:"iata" binding:"required"`
	Ciudad             string `form:"ciudad" binding:"required"`
	Aeropuerto         string `form:"aeropuerto"`
	AmbitoCodigo       string `form:"ambito_codigo" binding:"required"`
	DepartamentoCodigo string `form:"departamento_codigo"`
	Pais               string `form:"pais"`
	Estado             string `form:"estado"`
}

type UpdateDestinoRequest struct {
	Ciudad             string `form:"ciudad" binding:"required"`
	Aeropuerto         string `form:"aeropuerto"`
	AmbitoCodigo       string `form:"ambito_codigo" binding:"required"`
	DepartamentoCodigo string `form:"departamento_codigo"`
	Pais               string `form:"pais"`
	Estado             string `form:"estado"`
}
