package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	externalAPIURL = "https://economia.awesomeapi.com.br/json/last/USD-BRL"
	apiTimeout     = 200 * time.Millisecond
	dbTimeout      = 10 * time.Millisecond
	dbFileName     = "./cotacoes.db"
	serverPort     = ":8080"
)

type ExchangeRateApiResponse struct {
	USDBRL ExchangeRateData `json:"USDBRL"`
}

type ExchangeRateData struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

var db *sql.DB

func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", dbFileName)
	if err != nil {
		return fmt.Errorf("erro ao abrir banco de dados SQLite: %w", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS quotations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		bid TEXT NOT NULL,
		full_response TEXT,
		api_timestamp TEXT,
		server_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		db.Close()
		return fmt.Errorf("erro ao criar tabela 'quotations': %w", err)
	}
	log.Println("[DATABASE] Banco de dados SQLite inicializado e tabela 'quotations' pronta.")
	return nil
}

func fetchExchangeRate(ctx context.Context) (*ExchangeRateApiResponse, []byte, error) {
	log.Println("[API_CLIENT] Buscando cotação da API externa...")
	req, err := http.NewRequestWithContext(ctx, "GET", externalAPIURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("falha ao criar requisição para API externa: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[API_CLIENT] Erro: Timeout de %s excedido ao buscar cotação da API externa.", apiTimeout)
			return nil, nil, fmt.Errorf("timeout ao buscar cotação da API externa: %w", ctx.Err())
		}
		return nil, nil, fmt.Errorf("falha ao executar requisição para API externa: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("API externa retornou status não OK: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("falha ao ler corpo da resposta da API externa: %w", err)
	}

	var apiResponse ExchangeRateApiResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, nil, fmt.Errorf("falha ao decodificar JSON da API externa: %w", err)
	}
	log.Println("[API_CLIENT] Cotação recebida com sucesso da API externa. Bid: " + apiResponse.USDBRL.Bid)
	return &apiResponse, body, nil
}

func saveQuotationToDB(ctx context.Context, quotationData *ExchangeRateData, fullResponseJson []byte) error {
	log.Println("[DATABASE] Salvando cotação no banco de dados...")
	stmt, err := db.PrepareContext(ctx, "INSERT INTO quotations (bid, full_response, api_timestamp) VALUES (?, ?, ?)")
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[DATABASE] Erro: Timeout de %s excedido ao preparar statement.", dbTimeout)
			return fmt.Errorf("timeout ao preparar statement SQL: %w", ctx.Err())
		}
		return fmt.Errorf("erro ao preparar statement SQL: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, quotationData.Bid, string(fullResponseJson), quotationData.Timestamp)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[DATABASE] Erro: Timeout de %s excedido ao salvar cotação.", dbTimeout)
			return fmt.Errorf("timeout ao salvar cotação no banco de dados: %w", ctx.Err())
		}
		return fmt.Errorf("erro ao executar statement SQL para salvar cotação: %w", err)
	}
	log.Println("[DATABASE] Cotação salva com sucesso. Bid: " + quotationData.Bid)
	return nil
}

func cotacaoHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[SERVER] Requisição recebida em /cotacao")

	apiCtx, apiCancel := context.WithTimeout(r.Context(), apiTimeout)
	defer apiCancel()

	exchangeRateApiResponse, originalJsonBody, err := fetchExchangeRate(apiCtx)
	if err != nil {
		log.Printf("[SERVER] Erro ao buscar cotação da API externa: %v", err)
		http.Error(w, "Erro ao buscar cotação externa: "+err.Error(), http.StatusInternalServerError)
		return
	}

	dbCtx, dbCancel := context.WithTimeout(context.Background(), dbTimeout)
	defer dbCancel()

	err = saveQuotationToDB(dbCtx, &exchangeRateApiResponse.USDBRL, originalJsonBody)
	if err != nil {
		log.Printf("[SERVER] Aviso: Falha ao salvar cotação no banco de dados: %v. A resposta ainda será enviada ao cliente.", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(originalJsonBody)
	if err != nil {
		log.Printf("[SERVER] Erro ao escrever resposta para o cliente: %v", err)
	} else {
		log.Println("[SERVER] Resposta JSON enviada ao cliente.")
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(os.Stdout)

	if err := initDB(); err != nil {
		log.Fatalf("[SERVER] Falha fatal ao inicializar banco de dados: %v", err)
	}
	defer func() {
		if db != nil {
			db.Close()
			log.Println("[DATABASE] Conexão com banco de dados SQLite fechada.")
		}
	}()

	http.HandleFunc("/cotacao", cotacaoHandler)
	log.Printf("[SERVER] Servidor HTTP ouvindo na porta %s", serverPort)
	log.Printf("[SERVER] Endpoint: http://localhost%s/cotacao (local) ou http://server%s/cotacao (via client Docker)", serverPort, serverPort)

	if err := http.ListenAndServe(serverPort, nil); err != nil {
		log.Fatalf("[SERVER] Falha fatal ao iniciar servidor HTTP: %v", err)
	}
}
