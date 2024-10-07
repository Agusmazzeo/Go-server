package schemas

type CurrenciesResponse struct {
	Currencies []Currency
}

type Currency struct {
	ID          string
	Description string
}

type CurrencyWithValuationResponse struct {
	ID          string
	Description string
	Valuations  []CurrencyValuation
}

type CurrencyValuation struct {
	Date                string
	ArgCurrencyRelation float32
	UsdCurrencyRelation float32
}
