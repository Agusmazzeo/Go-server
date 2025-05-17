package services

import (
	"context"
	"fmt"
	"server/src/clients/esco"
	"server/src/schemas"
	"server/src/utils"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ESCOServiceI interface {
	GetAccountByID(ctx context.Context, token, id string) (*esco.CuentaSchema, error)
	GetAccountState(ctx context.Context, token, id string, date time.Time) (*schemas.AccountState, error)
	GetAccountStateWithTransactions(ctx context.Context, token, id string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error)
	GetAccountStateDateRange(ctx context.Context, token, id string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error)
	GetLiquidacionesDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error)
	GetBoletosDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error)
	GetMultiAccountStateWithTransactions(ctx context.Context, token string, ids []string, startDate, endDate time.Time, interval time.Duration) ([]*schemas.AccountState, error)
	GetMultiAccountStateByCategory(ctx context.Context, token string, ids []string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountStateByCategory, error)
	GetCtaCteConsolidadoDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error)
}

type ESCOService struct {
	client esco.ESCOServiceClientI
}

func NewESCOService(client esco.ESCOServiceClientI) *ESCOService {
	return &ESCOService{client: client}
}

func (s *ESCOService) GetAccountByID(ctx context.Context, token, id string) (*esco.CuentaSchema, error) {
	acc, err := s.client.BuscarCuentas(token, id)
	if err != nil {
		return nil, err
	}
	if len(acc) == 0 {
		return nil, fmt.Errorf("no accounts received for given id %s", id)
	}
	if len(acc) != 1 {
		return nil, fmt.Errorf("more than 1 account received for given id %s", id)
	}
	return &acc[0], nil
}

func (s *ESCOService) GetAccountState(ctx context.Context, token, id string, date time.Time) (*schemas.AccountState, error) {
	account, err := s.GetAccountByID(ctx, token, id)
	if err != nil {
		return nil, err
	}
	accStateData, err := s.client.GetEstadoCuenta(token, account.ID, account.FI, strconv.Itoa(account.N), "0", date, false)
	if err != nil {
		return nil, err
	}
	return s.parseEstadoToAccountState(&accStateData, &date)
}

func (s *ESCOService) GetAccountStateWithTransactions(ctx context.Context, token, id string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error) {
	var err error
	logger := utils.LoggerFromContext(ctx)
	var wg sync.WaitGroup
	wg.Add(4)

	var accountState *schemas.AccountState
	var liquidaciones *schemas.AccountState
	var boletos *schemas.AccountState
	var instrumentos *schemas.AccountState

	go func() {
		accountState, err = s.GetAccountStateDateRange(ctx, token, id, startDate, endDate, interval)
		if err != nil {
			logger.Errorf("error while on GetAccountStateDateRange: %v", err)
		}
		wg.Done()
	}()

	go func() {
		retries := 3
		for {
			liquidaciones, err = s.GetLiquidacionesDateRange(ctx, token, id, startDate, endDate)
			if err != nil {
				logger.Errorf("error while on GetLiquidacionesDateRange: %v. Retrying...", err)
				retries--
			} else {
				break
			}
			if retries == 0 {
				logger.Errorf("exhausted retries on GetLiquidacionesDateRange: %v", err)
				break
			}
		}
		wg.Done()
	}()

	go func() {
		retries := 3
		for {
			boletos, err = s.GetBoletosDateRange(ctx, token, id, startDate, endDate)
			if err != nil {
				logger.Errorf("error while on GetBoletosDateRange: %v. Retrying...", err)
				retries--
			} else {
				break
			}
			if retries == 0 {
				logger.Errorf("exhausted retries on GetBoletosDateRange: %v", err)
				break
			}
		}
		wg.Done()
	}()

	go func() {
		retries := 3
		for {
			instrumentos, err = s.GetCtaCteConsolidadoDateRange(ctx, token, id, startDate, endDate)
			if err != nil {
				logger.Errorf("error while on GetCtaCteConsolidadoDateRange: %v. Retrying...", err)
				retries--
			} else {
				break
			}
			if retries == 0 {
				logger.Errorf("exhausted retries on GetCtaCteConsolidadoDateRange: %v", err)
				break
			}
		}
		wg.Done()
	}()

	wg.Wait()
	if err != nil {
		return nil, err
	}
	if accountState == nil {
		logger.Warn("empty account state received")
		return nil, fmt.Errorf("empty account state received")
	}

	for id := range *accountState.Assets {
		asset := (*accountState.Assets)[id]
		if boletos != nil {
			if boleto, ok := (*boletos.Assets)[id]; ok {
				asset.Transactions = append(asset.Transactions, boleto.Transactions...)
				(*accountState.Assets)[id] = asset
			}
		}
		if liquidaciones != nil {
			if liquidacion, ok := (*liquidaciones.Assets)[id]; ok {
				asset.Transactions = append(asset.Transactions, liquidacion.Transactions...)
				(*accountState.Assets)[id] = asset
			}
		}
		if instrumentos != nil {
			if instrumento, ok := (*instrumentos.Assets)[id]; ok {
				asset.Transactions = append(asset.Transactions, instrumento.Transactions...)
				(*accountState.Assets)[id] = asset
			}
		}
	}
	return accountState, nil
}

func (s *ESCOService) GetAccountStateDateRange(ctx context.Context, token, id string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error) {
	logger := utils.LoggerFromContext(ctx)
	account, err := s.GetAccountByID(ctx, token, id)
	if err != nil {
		logger.Errorf("error while on GetAccountByID: %v", err)
		return nil, err
	}
	intervalHours := interval.Hours()
	numDays := int(endDate.Sub(startDate).Hours()/intervalHours) + 1
	assets := make(map[string]schemas.Asset)

	var wg sync.WaitGroup
	var errChan = make(chan error, numDays)
	var assetChan = make(chan *schemas.AccountState, numDays)
	wg.Add(numDays)

	for i := 0; i < numDays; i++ {
		go func(i int) {
			defer wg.Done()
			var retries = 3
			var accStateData []esco.EstadoCuentaSchema
			date := startDate.AddDate(0, 0, i*int(intervalHours/24))
			for {
				accStateData, err = s.client.GetEstadoCuenta(token, account.ID, account.FI, strconv.Itoa(account.N), "0", date, false)
				if err != nil || accStateData == nil {
					retries -= 1
					logger.Warnf("error while on GetEstadoCuenta: %v. Retrying..", err)
					time.Sleep(100 * time.Millisecond)
				} else {
					break
				}
				if retries == 0 {
					errChan <- err
					logger.Errorf("retries exceeded for GetEstadoCuenta: %v", err)
					return
				}
			}
			accountState, err := s.parseEstadoToAccountState(&accStateData, &date)
			if err != nil {
				errChan <- err
				return
			}
			assetChan <- accountState
		}(i)
	}

	go func() {
		wg.Wait()
		close(errChan)
		close(assetChan)
	}()

	for accountState := range assetChan {
		for key, value := range *accountState.Assets {
			if v, ok := assets[key]; ok {
				v.Holdings = append(v.Holdings, value.Holdings...)
				assets[key] = v
			} else {
				assets[key] = value
			}
		}
	}
	for _, asset := range assets {
		sortHoldingsByDateRequested(&asset)
	}
	return &schemas.AccountState{Assets: &assets}, <-errChan
}

func (s *ESCOService) GetLiquidacionesDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error) {
	account, err := s.GetAccountByID(ctx, token, id)
	if err != nil {
		return nil, err
	}
	liquidaciones, err := s.client.GetLiquidaciones(token, account.ID, account.FI, strconv.Itoa(account.N), "0", startDate, endDate, false)
	if err != nil {
		return nil, err
	}
	return s.parseLiquidacionesToAccountState(&liquidaciones)
}

func (s *ESCOService) GetBoletosDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error) {
	account, err := s.GetAccountByID(ctx, token, id)
	if err != nil {
		return nil, err
	}
	boletos, err := s.client.GetBoletos(token, account.ID, account.FI, strconv.Itoa(account.N), "0", startDate, endDate, false)
	if err != nil {
		return nil, err
	}
	return s.parseBoletosToAccountState(&boletos)
}

func (s *ESCOService) GetMultiAccountStateWithTransactions(ctx context.Context, token string, ids []string, startDate, endDate time.Time, interval time.Duration) ([]*schemas.AccountState, error) {
	logger := utils.LoggerFromContext(ctx)
	accountsStates := make([]*schemas.AccountState, 0, len(ids))
	var err error

	for _, id := range ids {
		var accountState *schemas.AccountState
		accountState, err = s.GetAccountStateWithTransactions(ctx, token, id, startDate, endDate, interval)
		if err != nil {
			logger.Error(err)
			return nil, err
		}
		accountsStates = append(accountsStates, accountState)
	}

	return accountsStates, nil
}

func (s *ESCOService) GetMultiAccountStateByCategory(ctx context.Context, token string, ids []string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountStateByCategory, error) {
	accountStates, err := s.GetMultiAccountStateWithTransactions(ctx, token, ids, startDate, endDate, interval)
	if err != nil {
		logger := utils.LoggerFromContext(ctx)
		logger.Error(err)
		return nil, err
	}

	return s.CollapseAndGroupAccountsStates(accountStates), nil
}

func (s *ESCOService) CollapseAndGroupAccountsStates(accountsStates []*schemas.AccountState) *schemas.AccountStateByCategory {
	collapsedAccountState := s.collapseAccountStates(accountsStates)
	return s.groupTotalHoldingsAndTransactionsByDate(&collapsedAccountState)
}

func (s *ESCOService) parseEstadoToAccountState(accStateData *[]esco.EstadoCuentaSchema, date *time.Time) (*schemas.AccountState, error) {
	var categoryKey string
	categoryMap := s.client.GetCategoryMap()
	accStateRes := schemas.NewAccountState()
	for _, accData := range *accStateData {
		var asset schemas.Asset
		var exists bool
		var parsedDate *time.Time
		if asset, exists = (*accStateRes.Assets)[accData.A]; !exists {
			categoryKey = fmt.Sprintf("%s - %s", accData.A, accData.D)
			var category string
			var exists bool
			if category, exists = categoryMap[categoryKey]; !exists {
				category = "S / C"
			}
			(*accStateRes.Assets)[accData.A] = schemas.Asset{
				ID:           accData.A,
				Type:         accData.TI,
				Denomination: accData.D,
				Category:     category,
				Holdings:     make([]schemas.Holding, 0, len(*accStateData)),
			}
			asset = (*accStateRes.Assets)[accData.A]
		}
		if accData.F != "" {
			p, err := time.Parse(utils.ShortSlashDateLayout, accData.F)
			if err != nil {
				return nil, err
			}
			parsedDate = &p
		} else {
			parsedDate = nil
		}
		asset.Holdings = append(asset.Holdings, schemas.Holding{
			Currency:      accData.M,
			CurrencySign:  accData.MS,
			Value:         accData.N,
			Units:         accData.C,
			DateRequested: date,
			Date:          parsedDate,
		})
		(*accStateRes.Assets)[accData.A] = asset
	}

	return accStateRes, nil
}

func (s *ESCOService) parseBoletosToAccountState(boletos *[]esco.Boleto) (*schemas.AccountState, error) {
	var categoryKey string
	categoryMap := s.client.GetCategoryMap()
	accStateRes := schemas.NewAccountState()
	for _, boleto := range *boletos {
		var asset schemas.Asset
		var exists bool
		var parsedDate *time.Time
		id := strings.Split(boleto.I, " - ")[0]
		categoryKey = boleto.I
		if asset, exists = (*accStateRes.Assets)[id]; !exists {
			var category string
			var exists bool
			if category, exists = categoryMap[categoryKey]; !exists {
				category = "S / C"
			}
			(*accStateRes.Assets)[id] = schemas.Asset{
				ID:           id,
				Type:         boleto.T,
				Denomination: boleto.FL,
				Category:     category,
				Transactions: make([]schemas.Transaction, 0, len(*boletos)),
			}
			asset = (*accStateRes.Assets)[id]
		}
		if boleto.FL != "" {
			p, err := time.Parse(utils.ShortSlashDateLayout, boleto.FL)
			if err != nil {
				return nil, err
			}
			loc, _ := time.LoadLocation("America/Argentina/Buenos_Aires")
			p = time.Date(p.Year(), p.Month(), p.Day(), 23, 0, 0, 0, loc)
			parsedDate = &p
		} else {
			parsedDate = nil
		}
		asset.Transactions = append(asset.Transactions, schemas.Transaction{
			Currency:     "Pesos",
			CurrencySign: boleto.NS,
			Value:        -boleto.N,
			Units:        boleto.C,
			Date:         parsedDate,
		})
		(*accStateRes.Assets)[id] = asset
	}

	return accStateRes, nil
}

func (s *ESCOService) parseLiquidacionesToAccountState(liquidaciones *[]esco.Liquidacion) (*schemas.AccountState, error) {
	var categoryKey string
	categoryMap := s.client.GetCategoryMap()
	accStateRes := schemas.NewAccountState()
	for _, liquidacion := range *liquidaciones {
		var asset schemas.Asset
		var exists bool
		var parsedDate *time.Time
		id := strings.Split(liquidacion.F, " - ")[0]
		categoryKey = fmt.Sprintf("%s / %s", liquidacion.F, id)
		if asset, exists = (*accStateRes.Assets)[id]; !exists {
			var category string
			var exists bool
			if category, exists = categoryMap[categoryKey]; !exists {
				category = "S / C"
			}
			(*accStateRes.Assets)[id] = schemas.Asset{
				ID:           id,
				Type:         "",
				Denomination: categoryKey,
				Category:     category,
				Transactions: make([]schemas.Transaction, 0, len(*liquidaciones)),
			}
			asset = (*accStateRes.Assets)[id]
		}
		if liquidacion.FL != "" {
			p, err := time.Parse(utils.ShortSlashDateLayout, liquidacion.FL)
			if err != nil {
				return nil, err
			}
			loc, _ := time.LoadLocation("America/Argentina/Buenos_Aires")
			p = time.Date(p.Year(), p.Month(), p.Day(), 23, 0, 0, 0, loc)
			parsedDate = &p
		} else {
			parsedDate = nil
		}
		asset.Transactions = append(asset.Transactions, schemas.Transaction{
			Currency:     "Pesos",
			CurrencySign: liquidacion.MS,
			Value:        -liquidacion.I,
			Units:        liquidacion.Q,
			Date:         parsedDate,
		})
		(*accStateRes.Assets)[id] = asset
	}

	return accStateRes, nil
}

func (s *ESCOService) GetCtaCteConsolidadoDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error) {
	account, err := s.GetAccountByID(ctx, token, id)
	if err != nil {
		return nil, err
	}
	instrumentos, err := s.client.GetCtaCteConsolidado(token, account.ID, account.FI, strconv.Itoa(account.N), "0", startDate, endDate, false)
	if err != nil {
		return nil, err
	}
	return s.parseInstrumentosRecoveriesToAccountState(&instrumentos)
}

func (s *ESCOService) parseInstrumentosRecoveriesToAccountState(instrumentos *[]esco.Instrumentos) (*schemas.AccountState, error) {
	if instrumentos == nil {
		return nil, nil
	}
	var id, currencySign, categoryKey string
	var units, value float64
	categoryMap := s.client.GetCategoryMap()
	accStateRes := schemas.NewAccountState()
	for _, ins := range *instrumentos {
		if ins.C < float64(0) && strings.Contains(ins.D, "Retiro de Títulos") {
			id = strings.Split(strings.Split(ins.I, " - ")[1], " /")[0]
			currencySign = ins.PR_S
			units = -ins.C
			value = ins.N
			categoryKey = fmt.Sprintf("%s / %s", ins.F, id)
		} else if strings.Contains(ins.D, "Renta") || strings.Contains(ins.D, "Boleto") {
			id = strings.Split(ins.I, " - ")[1]
			if id == "$" {
				continue
			}
			currencySign = "$"
			units = -ins.C
			value = ins.N
			categoryKey = "CCL"
		} else {
			continue
		}
		var asset schemas.Asset
		var exists bool
		var parsedDate *time.Time

		if asset, exists = (*accStateRes.Assets)[id]; !exists {
			var category string
			var exists bool
			if category, exists = categoryMap[categoryKey]; !exists {
				category = "S / C"
			}
			(*accStateRes.Assets)[id] = schemas.Asset{
				ID:           id,
				Type:         "",
				Denomination: categoryKey,
				Category:     category,
				Transactions: make([]schemas.Transaction, 0, len(*instrumentos)),
			}
			asset = (*accStateRes.Assets)[id]
		}
		if ins.FL != "" {
			p, err := time.Parse(utils.ShortSlashDateLayout, ins.FL)
			if err != nil {
				return nil, err
			}
			loc, _ := time.LoadLocation("America/Argentina/Buenos_Aires")
			p = time.Date(p.Year(), p.Month(), p.Day(), 23, 0, 0, 0, loc)
			parsedDate = &p
		} else {
			parsedDate = nil
		}
		asset.Transactions = append(asset.Transactions, schemas.Transaction{
			Currency:     "Pesos",
			CurrencySign: currencySign,
			Value:        value,
			Units:        -units,
			Date:         parsedDate,
		})
		(*accStateRes.Assets)[id] = asset
	}

	return accStateRes, nil
}

func (s *ESCOService) collapseAccountStates(states []*schemas.AccountState) schemas.AccountState {
	holdingMapByAssetID := make(map[string]map[string]schemas.Holding)
	transactionMapByAssetID := make(map[string]map[string]schemas.Transaction)
	assetMapByID := make(map[string]schemas.Asset)

	for _, state := range states {
		if state.Assets == nil {
			continue
		}
		var holdingMap map[string]schemas.Holding
		var transactionMap map[string]schemas.Transaction
		var found bool
		for assetID, asset := range *state.Assets {
			if _, found := assetMapByID[assetID]; !found {
				assetMapByID[assetID] = asset
			}
			if holdingMap, found = holdingMapByAssetID[assetID]; !found {
				holdingMapByAssetID[assetID] = make(map[string]schemas.Holding)
				holdingMap = holdingMapByAssetID[assetID]
			}
			if transactionMap, found = transactionMapByAssetID[assetID]; !found {
				transactionMapByAssetID[assetID] = make(map[string]schemas.Transaction)
				transactionMap = transactionMapByAssetID[assetID]
			}
			for _, holding := range asset.Holdings {
				date := holding.DateRequested.Format("2006-01-02")
				if existing, found := holdingMap[date]; !found {
					holdingMap[date] = holding
				} else {
					existing.Value += holding.Value
					holdingMap[date] = existing
				}
			}
			holdingMapByAssetID[assetID] = holdingMap

			for _, transaction := range asset.Transactions {
				key := transaction.Date.Format("2006-01-02")
				if existing, found := transactionMap[key]; !found {
					transactionMap[key] = transaction
				} else {
					existing.Value += transaction.Value
					existing.Units += transaction.Units
					transactionMap[key] = existing
				}
			}
			transactionMapByAssetID[assetID] = transactionMap
		}
	}
	collapsed := make(map[string]schemas.Asset)
	for assetID, asset := range assetMapByID {
		var existing schemas.Asset
		var ok bool
		if existing, ok = collapsed[assetID]; !ok {
			collapsed[assetID] = schemas.Asset{
				ID:           assetID,
				Type:         asset.Type,
				Category:     asset.Category,
				Denomination: asset.Denomination,
				Holdings:     []schemas.Holding{},
				Transactions: []schemas.Transaction{},
			}
			existing = collapsed[assetID]
		}
		holdings := existing.Holdings
		for _, holding := range holdingMapByAssetID[assetID] {
			holdings = append(holdings, holding)
		}
		existing.Holdings = holdings
		sort.SliceStable(existing.Holdings, func(i, j int) bool {
			return existing.Holdings[i].DateRequested.Before(*existing.Holdings[j].DateRequested)
		})

		transactions := existing.Transactions
		for _, transaction := range transactionMapByAssetID[assetID] {
			transactions = append(transactions, transaction)
		}
		existing.Transactions = transactions
		sort.SliceStable(existing.Transactions, func(i, j int) bool {
			return existing.Transactions[i].Date.Before(*existing.Transactions[j].Date)
		})

		collapsed[assetID] = existing
	}
	return schemas.AccountState{Assets: &collapsed}
}

func (s *ESCOService) groupTotalHoldingsAndTransactionsByDate(state *schemas.AccountState) *schemas.AccountStateByCategory {
	totalHoldingsByDate := make(map[string]schemas.Holding)
	totalTransactionsByDate := make(map[string]schemas.Transaction)
	assetsByCategory := make(map[string][]schemas.Asset)
	categoryHoldingsByDate := make(map[string]map[string]schemas.Holding)
	categoryTransactionsByDate := make(map[string]map[string]schemas.Transaction)

	for _, asset := range *state.Assets {
		category := asset.Category
		assetsByCategory[category] = append(assetsByCategory[category], asset)

		if _, exists := categoryHoldingsByDate[category]; !exists {
			categoryHoldingsByDate[category] = make(map[string]schemas.Holding)
		}
		if _, exists := categoryTransactionsByDate[category]; !exists {
			categoryTransactionsByDate[category] = make(map[string]schemas.Transaction)
		}

		for _, holding := range asset.Holdings {
			date := *holding.DateRequested
			dateStr := date.Format("2006-01-02")
			if _, exists := totalHoldingsByDate[dateStr]; !exists {
				totalHoldingsByDate[dateStr] = schemas.Holding{
					Currency:      "Pesos",
					CurrencySign:  "$",
					Value:         0,
					DateRequested: &date,
					Date:          &date,
				}
			}
			total := totalHoldingsByDate[dateStr]
			total.Value += holding.Value
			totalHoldingsByDate[dateStr] = total

			if _, exists := categoryHoldingsByDate[category][dateStr]; !exists {
				categoryHoldingsByDate[category][dateStr] = schemas.Holding{
					Currency:      "Pesos",
					CurrencySign:  "$",
					Value:         0,
					DateRequested: &date,
					Date:          &date,
				}
			}
			categoryHolding := categoryHoldingsByDate[category][dateStr]
			categoryHolding.Value += holding.Value
			categoryHoldingsByDate[category][dateStr] = categoryHolding
		}

		for _, transaction := range asset.Transactions {
			date := *transaction.Date
			dateStr := date.Format("2006-01-02")
			if _, exists := totalTransactionsByDate[dateStr]; !exists {
				totalTransactionsByDate[dateStr] = schemas.Transaction{
					Currency:     "Pesos",
					CurrencySign: "$",
					Value:        0,
					Date:         &date,
				}
			}
			total := totalTransactionsByDate[dateStr]
			total.Value += transaction.Value
			totalTransactionsByDate[dateStr] = total

			if _, exists := categoryTransactionsByDate[category][dateStr]; !exists {
				categoryTransactionsByDate[category][dateStr] = schemas.Transaction{
					Currency:     "Pesos",
					CurrencySign: "$",
					Value:        0,
					Units:        0,
					Date:         &date,
				}
			}
			categoryTransaction := categoryTransactionsByDate[category][dateStr]
			categoryTransaction.Value += transaction.Value
			categoryTransactionsByDate[category][dateStr] = categoryTransaction
		}
	}

	for category := range assetsByCategory {
		sort.Slice(assetsByCategory[category], func(i, j int) bool {
			return assetsByCategory[category][i].ID < assetsByCategory[category][j].ID
		})
	}

	categoryAssets := s.generateCategoryAssets(categoryHoldingsByDate, categoryTransactionsByDate)

	return &schemas.AccountStateByCategory{
		AssetsByCategory:        &assetsByCategory,
		CategoryAssets:          &categoryAssets,
		TotalHoldingsByDate:     &totalHoldingsByDate,
		TotalTransactionsByDate: &totalTransactionsByDate,
	}
}

func (s *ESCOService) generateCategoryAssets(
	categoryHoldings map[string]map[string]schemas.Holding,
	categoryTransactions map[string]map[string]schemas.Transaction,
) map[string]schemas.Asset {
	categoryAssets := map[string]schemas.Asset{}
	for category, holdingsByDate := range categoryHoldings {
		if _, exist := categoryAssets[category]; !exist {
			categoryAssets[category] = schemas.Asset{
				ID:           category,
				Type:         "Category",
				Denomination: category,
				Category:     category,
				Holdings:     []schemas.Holding{},
				Transactions: []schemas.Transaction{},
			}
		}
		categoryAsset := categoryAssets[category]
		for _, holding := range holdingsByDate {
			categoryAsset.Holdings = append(categoryAsset.Holdings, holding)
		}
		categoryAssets[category] = categoryAsset
	}

	for category, transactionsByDate := range categoryTransactions {
		if _, exist := categoryAssets[category]; !exist {
			categoryAssets[category] = schemas.Asset{
				ID:           category,
				Type:         "Category",
				Denomination: category,
				Category:     category,
				Holdings:     []schemas.Holding{},
				Transactions: []schemas.Transaction{},
			}
		}
		categoryAsset := categoryAssets[category]
		for _, transaction := range transactionsByDate {
			categoryAsset.Transactions = append(categoryAsset.Transactions, transaction)
		}
		categoryAssets[category] = categoryAsset
	}
	return categoryAssets
}

func sortHoldingsByDateRequested(asset *schemas.Asset) {
	sort.Slice(asset.Holdings, func(i, j int) bool {
		return asset.Holdings[i].DateRequested.Before(*asset.Holdings[j].DateRequested)
	})
}
