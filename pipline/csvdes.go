package main

import (
    "encoding/csv"
    "fmt"
    "log"
    "os"
    "strings"
)

func escapeSQLString(s string) string {
    s = strings.ReplaceAll(s, "'", "''")  // escapa aspas simples
    s = strings.ReplaceAll(s, "\n", " ")  // remove quebras de linha
    s = strings.ReplaceAll(s, "\r", " ")  // remove retorno de carro
    return s
}

func main() {
    if len(os.Args) < 2 {
        log.Fatal("Use: go run csv_to_sql.go <arquivo_csv_traduzido>")
    }
    inputCSV := os.Args[1]
    languageID := 2 // Id do idioma, ajuste se necessário

    inFile, err := os.Open(inputCSV)
    if err != nil {
        log.Fatalf("Erro ao abrir arquivo CSV: %v", err)
    }
    defer inFile.Close()

    reader := csv.NewReader(inFile)

    // Ignorar cabeçalho
    _, err = reader.Read()
    if err != nil {
        log.Fatalf("Erro ao ler cabeçalho: %v", err)
    }

    seenSupplierProducts := make(map[string]bool)

    for {
        record, err := reader.Read()
        if err != nil {
            break
        }

        if len(record) < 3 {
            log.Printf("Linha ignorada, menos de 3 colunas: %v\n", record)
            continue
        }

        supplierProductID := record[0]
        title := escapeSQLString(record[1])
        description := escapeSQLString(record[2])

        if seenSupplierProducts[supplierProductID] {
            continue
        }
        seenSupplierProducts[supplierProductID] = true

        fmt.Printf(
            "INSERT INTO oc_product_description (product_id, language_id, name, description) "+
                "SELECT product_id, %d, '%s', '%s' FROM oc_product WHERE supplier_product_id = '%s';\n",
            languageID, title, description, supplierProductID)
    }
}