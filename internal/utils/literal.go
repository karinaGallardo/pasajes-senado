package utils

import (
	"fmt"
	"math"
)

// NumeroALetras convierte un float64 a formato literal monetario (es).
func NumeroALetras(numero float64) string {
	entero := int64(math.Trunc(numero))
	decimales := int(math.Round((numero - float64(entero)) * 100))

	decStr := ""
	if decimales > 0 {
		decStr = fmt.Sprintf(" CON %d/100", decimales)
	} else {
		decStr = " CON 00/100"
	}

	res := ToText(entero) + decStr
	return res
}

// ToText realiza el mapeo de enteros a su representación literal (es).
func ToText(valor int64) string {
	if valor == 0 {
		return "CERO"
	} else if valor == 1 {
		return "UNO"
	} else if valor == 2 {
		return "DOS"
	} else if valor == 3 {
		return "TRES"
	} else if valor == 4 {
		return "CUATRO"
	} else if valor == 5 {
		return "CINCO"
	} else if valor == 6 {
		return "SEIS"
	} else if valor == 7 {
		return "SIETE"
	} else if valor == 8 {
		return "OCHO"
	} else if valor == 9 {
		return "NUEVE"
	} else if valor == 10 {
		return "DIEZ"
	} else if valor == 11 {
		return "ONCE"
	} else if valor == 12 {
		return "DOCE"
	} else if valor == 13 {
		return "TRECE"
	} else if valor == 14 {
		return "CATORCE"
	} else if valor == 15 {
		return "QUINCE"
	} else if valor < 20 {
		return "DIECI" + ToText(valor-10)
	} else if valor == 20 {
		return "VEINTE"
	} else if valor < 30 {
		return "VEINTI" + ToText(valor-20)
	} else if valor == 30 {
		return "TREINTA"
	} else if valor == 40 {
		return "CUARENTA"
	} else if valor == 50 {
		return "CINCUENTA"
	} else if valor == 60 {
		return "SESENTA"
	} else if valor == 70 {
		return "SETENTA"
	} else if valor == 80 {
		return "OCHENTA"
	} else if valor == 90 {
		return "NOVENTA"
	} else if valor < 100 {
		return ToText(int64(math.Trunc(float64(valor)/10))*10) + " Y " + ToText(valor%10)
	} else if valor == 100 {
		return "CIEN"
	} else if valor < 200 {
		return "CIENTO " + ToText(valor-100)
	} else if (valor == 200) || (valor == 300) || (valor == 400) || (valor == 600) || (valor == 800) {
		return ToText(int64(math.Trunc(float64(valor)/100))) + "CIENTOS"
	} else if valor == 500 {
		return "QUINIENTOS"
	} else if valor == 700 {
		return "SETECIENTOS"
	} else if valor == 900 {
		return "NOVECIENTOS"
	} else if valor < 1000 {
		return ToText(int64(math.Trunc(float64(valor)/100))*100) + " " + ToText(valor%100)
	} else if valor == 1000 {
		return "MIL"
	} else if valor < 2000 {
		return "MIL " + ToText(valor%1000)
	} else if valor < 1000000 {
		numText := ToText(int64(math.Trunc(float64(valor)/1000))) + " MIL"
		if (valor % 1000) > 0 {
			numText += " " + ToText(valor%1000)
		}
		return numText
	} else if valor == 1000000 {
		return "UN MILLON"
	} else if valor < 2000000 {
		return "UN MILLON " + ToText(valor%1000000)
	} else if valor < 1000000000000 {
		numText := ToText(int64(math.Trunc(float64(valor)/1000000))) + " MILLONES"
		if (valor - int64(math.Trunc(float64(valor)/1000000))*1000000) > 0 {
			numText += " " + ToText(valor-int64(math.Trunc(float64(valor)/1000000))*1000000)
		}
		return numText
	}

	return "NUMERO_MUY_GRANDE"
}

// DiasALetras convierte un float64 de días a formato literal (es).
func DiasALetras(dias float64) string {
	entero := int64(dias)
	decimal := dias - float64(entero)

	texto := ToText(entero)

	if decimal == 0.5 {
		if entero == 0 {
			return "MEDIO"
		}
		return texto + " Y MEDIO"
	}

	return texto
}
