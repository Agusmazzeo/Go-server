package bcra

type Divisa struct {
	Codigo       string `json:"codigo"`
	Denominacion string `json:"denominacion"`
}

type GetDivisasResponse struct {
	Status  int      `json:"status"`
	Results []Divisa `json:"results"`
}

type CotizacionDetalle struct {
	CodigoMoneda   string  `json:"codigoMoneda"`
	Descripcion    string  `json:"descripcion"`
	TipoPase       float64 `json:"tipoPase"`
	TipoCotizacion float64 `json:"tipoCotizacion"`
}

type Cotizacion struct {
	Fecha   string              `json:"fecha"`
	Detalle []CotizacionDetalle `json:"detalle"`
}

type GetCotizacionesResponse struct {
	Status  int        `json:"status"`
	Results Cotizacion `json:"results"`
}

type CotizacionByMonedaDetalle struct {
	CodigoMoneda   string  `json:"codigoMoneda"`
	Descripcion    string  `json:"descripcion"`
	TipoPase       float64 `json:"tipoPase"`
	TipoCotizacion float64 `json:"tipoCotizacion"`
}

type CotizacionByMoneda struct {
	Fecha   string                      `json:"fecha"`
	Detalle []CotizacionByMonedaDetalle `json:"detalle"`
}

type Metadata struct {
	ResultSet struct {
		Count  int `json:"count"`
		Offset int `json:"offset"`
		Limit  int `json:"limit"`
	} `json:"resultset"`
}

type GetCotizacionesByMonedaResponse struct {
	Status   int                  `json:"status"`
	Metadata Metadata             `json:"metadata"`
	Results  []CotizacionByMoneda `json:"results"`
}

type GetVariablesResponse struct {
	Status        int        `json:"status"`
	Results       []Variable `json:"results"`
	ErrorMessages []string   `json:"errorMessages"`
}

type Variable struct {
	IDVariable  int     `json:"idVariable"`
	CDSerie     int     `json:"cdSerie"`
	Descripcion string  `json:"descripcion"`
	Fecha       string  `json:"fecha"`
	Valor       float64 `json:"valor"`
}
