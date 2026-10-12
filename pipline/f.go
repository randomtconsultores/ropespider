package main

import (
    "encoding/csv"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
    "time"
)

const variantsDir = "csv-variants-output"

type VariantData struct {
    SupplierProductID string
    Model             string
    PriceBase         float64
    Available         string
}

func readCSV(filename string) (map[string][]VariantData, error) {
    filePath := filepath.Join(variantsDir, filename)
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    r := csv.NewReader(file)
    records, err := r.ReadAll()
    if err != nil {
        return nil, err
    }

    products := make(map[string][]VariantData)

    for i, row := range records {
        if i == 0 {
            continue
        }
        price, _ := strconv.ParseFloat(row[4], 64)
        variant := VariantData{
            SupplierProductID: row[0],
            Model:             row[2], // usando VariantName como model
            PriceBase:         price,
            Available:         row[3],
        }
        products[row[0]] = append(products[row[0]], variant)
    }

    return products, nil
}

func modelKey(s string) float64 {
    // Extrai número de string como, por exemplo, "5m" -> 5.0 para ordenação
    s = strings.ToLower(strings.TrimSpace(s))
    s = strings.TrimSuffix(s, "m")
    f, err := strconv.ParseFloat(s, 64)
    if err != nil {
        return 1e9 // colocar no fim se erro
    }
    return f
}

func main() {
    if len(os.Args) < 3 {
        log.Fatalf("Uso: go run f.go antigo.csv novo.csv")
    }
    oldFile := os.Args[1]
    newFile := os.Args[2]

    oldVariants, err := readCSV(oldFile)
    if err != nil {
        log.Fatalf("Erro lendo CSV antigo: %v", err)
    }
    newVariants, err := readCSV(newFile)
    if err != nil {
        log.Fatalf("Erro lendo CSV novo: %v", err)
    }

    // Criar mapa único para detectar variantes existentes no antigo arquivo
    oldMap := make(map[string]bool)
    for supplierID, variants := range oldVariants {
        for _, v := range variants {
            key := supplierID + "_" + v.Model
            oldMap[key] = true
        }
    }

    dateAdded := time.Now().Format("2006-01-02 15:04:05")

    // Ordenar supplier_product_id para saída ordenada
    var supplierIDs []string
    for supplier := range newVariants {
        supplierIDs = append(supplierIDs, supplier)
    }
    sort.Strings(supplierIDs)

    fmt.Println("-- SQL Gerado --")
    for _, supplierID := range supplierIDs {
        variants := newVariants[supplierID]

        // Ordenar por model numericamente
        sort.Slice(variants, func(i, j int) bool {
            return modelKey(variants[i].Model) < modelKey(variants[j].Model)
        })

        for _, v := range variants {
            key := supplierID + "_" + v.Model
            if oldMap[key] {
                continue // Excluir variantes já existentes
            }

            qty := 0
            if v.Available == "yes" || v.Available == "1" || strings.ToLower(v.Available) == "true" {
                qty = 9999
            }
            price := v.PriceBase * 1.4

            fmt.Printf(
                "INSERT INTO oc_product (supplier_product_id, model, price, quantity, status, date_added) VALUES ('%s', '%s', %.2f, %d, 1, '%s');\n",
                supplierID, v.Model, price, qty, dateAdded)
        }
    }
}