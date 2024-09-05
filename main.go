package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Estrutura para armazenar links encurtados
type Data struct {
	Links map[string]string `json:"links"`
}

type RequestData struct {
	URL string `json:"url"` // Estrutura para decodificar o JSON com a URL
}

var data Data
var dataFile = "base.json" // Nome do arquivo JSON para armazenar as URLs

func main() {
	// Inicializa o mapa, caso não tenha sido carregado do arquivo
	data.Links = make(map[string]string)

	// Carregar as URLs encurtadas do arquivo JSON
	loadData()

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/new/", createShortenedUrl).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/{shortUrl}", redirectURL).Methods(http.MethodGet)

	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:8000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

// Função para encurtar URLs e salvar
func createShortenedUrl(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Erro ao ler o corpo da requisição", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var requestData RequestData
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		http.Error(w, "Erro ao decodificar JSON", http.StatusBadRequest)
		return
	}

	// Gerar o código encurtado usando o tempo atual como base
	shortUrl := EncodeBase62(time.Now().UnixNano())

	// Armazenar a URL original e sua versão encurtada
	data.Links[shortUrl] = requestData.URL

	// Salvar o arquivo JSON atualizado
	saveData()

	// Retornar a URL encurtada para o cliente
	response := map[string]string{"shortened_url": fmt.Sprintf("http://127.0.0.1:8000/api/v1/%s", shortUrl)}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Erro ao criar resposta JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// Função para redirecionar URLs encurtadas
func redirectURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortUrl := vars["shortUrl"]

	// Procurar a URL original no mapa
	originalUrl, found := data.Links[shortUrl]
	if !found {
		http.Error(w, "URL não encontrada", http.StatusNotFound)
		return
	}

	// Redirecionar para a URL original
	http.Redirect(w, r, originalUrl, http.StatusFound)
}

// EncodeBase62 converte um número inteiro em uma string Base62
func EncodeBase62(num int64) string {
	var encoded strings.Builder
	for num > 0 {
		remainder := num % 62
		encoded.WriteString(string(base62Chars[remainder]))
		num /= 62
	}

	// Inverter a string codificada
	encodedStr := encoded.String()
	runes := []rune(encodedStr)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

// Função para carregar os dados do arquivo JSON
func loadData() {
	// Verificar se o arquivo existe
	file, err := os.Open(dataFile)
	if err != nil {
		log.Println("Arquivo de dados não encontrado, criando novo...")
		data.Links = make(map[string]string)
		return
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Erro ao ler o arquivo: %v", err)
	}

	// Verifica se o arquivo está vazio
	if len(bytes) == 0 {
		log.Println("Arquivo JSON vazio, iniciando com um mapa vazio.")
		data.Links = make(map[string]string)
		return
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		log.Fatalf("Erro ao decodificar o arquivo JSON: %v", err)
	}
}

// Função para salvar os dados no arquivo JSON
func saveData() {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("Erro ao converter os dados para JSON: %v", err)
	}

	err = os.WriteFile(dataFile, bytes, 0644)
	if err != nil {
		log.Fatalf("Erro ao salvar o arquivo: %v", err)
	}
}
