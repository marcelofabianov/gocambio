package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	serverURL      string
	requestTimeout = 300 * time.Millisecond
	outputFileName = "./cotacao.txt"
)

type ServerResponse struct {
	USDBRL struct {
		Bid string `json:"bid"`
	} `json:"USDBRL"`
}

func init() {
	serverURL = os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080/cotacao"
		log.Println("[CLIENT] SERVER_URL não definida. Usando default: " + serverURL)
	} else {
		log.Println("[CLIENT] SERVER_URL definida como: " + serverURL)
	}
}

func fetchQuotationFromServer(ctx context.Context) (string, error) {
	log.Println("[CLIENT] Solicitando cotação do servidor em " + serverURL)
	req, err := http.NewRequestWithContext(ctx, "GET", serverURL, nil)
	if err != nil {
		return "", fmt.Errorf("falha ao criar requisição: %w", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[CLIENT] Erro: Timeout de %s excedido ao solicitar cotação.", requestTimeout)
			return "", fmt.Errorf("timeout ao solicitar cotação: %w", ctx.Err())
		}
		return "", fmt.Errorf("falha ao executar requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("servidor retornou status não OK: %d - %s", resp.StatusCode, string(bodyBytes))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("falha ao ler corpo da resposta: %w", err)
	}

	var serverResp ServerResponse
	if err := json.Unmarshal(body, &serverResp); err != nil {
		return "", fmt.Errorf("falha ao decodificar JSON da resposta: %w", err)
	}

	if serverResp.USDBRL.Bid == "" {
		return "", fmt.Errorf("campo 'bid' não encontrado ou vazio na resposta")
	}
	log.Println("[CLIENT] Cotação recebida. Bid: " + serverResp.USDBRL.Bid)
	return serverResp.USDBRL.Bid, nil
}

func saveQuotationToFile(bidValue string) error {
	log.Println("[CLIENT] Salvando cotação no arquivo " + outputFileName)
	fileContent := fmt.Sprintf("Dólar: %s", bidValue)
	err := os.WriteFile(outputFileName, []byte(fileContent), 0644)
	if err != nil {
		return fmt.Errorf("falha ao salvar cotação no arquivo '%s': %w", outputFileName, err)
	}
	log.Printf("[CLIENT] Cotação salva com sucesso em %s: %s", outputFileName, fileContent)
	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(os.Stdout)

	clientCtx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	bidValue, err := fetchQuotationFromServer(clientCtx)
	if err != nil {
		log.Fatalf("[CLIENT] Erro crítico ao obter cotação: %v", err)
	}
	if err := saveQuotationToFile(bidValue); err != nil {
		log.Fatalf("[CLIENT] Erro crítico ao salvar cotação: %v", err)
	}
	log.Println("[CLIENT] Operação concluída com sucesso!")
}
