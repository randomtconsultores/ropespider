package main

import (
    "encoding/csv"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"
)

const csvDir = "csv-output"
const imagesBaseDir = "images-download"

type ImageData struct {
    ProductID string
    ImagePath string
    SortOrder string
}

// Lê CSV do diretório csv-output e cria mapa chave única productID_image_sortOrder
func readImageCSV(filename string) (map[string]ImageData, error) {
    filePath := filepath.Join(csvDir, filename)
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    reader := csv.NewReader(file)
    records, err := reader.ReadAll()
    if err != nil {
        return nil, err
    }

    data := make(map[string]ImageData)
    for i, row := range records {
        if i == 0 {
            continue // pular header
        }
        key := row[0] + "_" + row[1] + "_" + row[2]
        data[key] = ImageData{
            ProductID: row[0],
            ImagePath: row[1],
            SortOrder: row[2],
        }
    }
    return data, nil
}

// Cria diretório de sessão para guardar as imagens baixadas, usando timestamp
func createSessionDir() string {
    timestamp := time.Now().Format("20060102_150405")
    sessionDir := filepath.Join(imagesBaseDir, timestamp)
    err := os.MkdirAll(sessionDir, 0755)
    if err != nil {
        log.Fatalf("Erro ao criar diretório de sessão %s: %v", sessionDir, err)
    }
    return sessionDir
}

// Download do arquivo da internet para arquivo local
func downloadImage(url, dest string) error {
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("erro HTTP: %s", resp.Status)
    }

    out, err := os.Create(dest)
    if err != nil {
        return err
    }
    defer out.Close()

    _, err = io.Copy(out, resp.Body)
    return err
}

func main() {
    if len(os.Args) < 3 {
        log.Fatalf("Uso: go run download_new_images.go antigo.csv novo.csv")
    }

    oldFile := os.Args[1]
    newFile := os.Args[2]

    oldData, err := readImageCSV(oldFile)
    if err != nil {
        log.Fatalf("Erro a ler CSV antigo: %v", err)
    }

    newData, err := readImageCSV(newFile)
    if err != nil {
        log.Fatalf("Erro a ler CSV novo: %v", err)
    }

    sessionDir := createSessionDir()

    for key, img := range newData {
        if _, exists := oldData[key]; !exists {
            url := img.ImagePath
            filename := filepath.Base(strings.Split(url, "?")[0])
            dest := filepath.Join(sessionDir, filename)

            fmt.Printf("A descarregar imagem nova: %s -> %s\n", url, dest)
            err := downloadImage(url, dest)
            if err != nil {
                log.Printf("Erro ao descarregar %s: %v\n", url, err)
            }
        }
    }

    fmt.Printf("\nDownload concluído. Imagens guardadas em: %s\n", sessionDir)
}