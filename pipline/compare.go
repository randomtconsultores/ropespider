package main

import (
    "encoding/csv"
    "fmt"
    "log"
    "os"
    "strconv"
)

type VariantData struct {
    ProductID   string
    VariantKey  string
    VariantName string
    Available   string
    PriceBase   float64
    ShippingDays string
}

const variantsDir = "csv-variants-output/"

func readCSV(filename string) (map[string]VariantData, map[string]bool, error) {
    filePath := variantsDir + filename
    file, err := os.Open(filePath)
    if err != nil {
        return nil, nil, err
    }
    defer file.Close()

    reader := csv.NewReader(file)
    records, err := reader.ReadAll()
    if err != nil {
        return nil, nil, err
    }

    data := make(map[string]VariantData)
    products := make(map[string]bool)

    for i, row := range records {
        if i == 0 {
            continue // skip header
        }
        price, _ := strconv.ParseFloat(row[4], 64)
        key := row[0] + "_" + row[1] // unique key product_variant
        data[key] = VariantData{
            ProductID:   row[0],
            VariantKey:  row[1],
            VariantName: row[2],
            Available:   row[3],
            PriceBase:   price,
            ShippingDays: row[5],
        }
        products[row[0]] = true
    }
    return data, products, nil
}

func main() {
    if len(os.Args) < 3 {
        log.Fatalf("Uso: go run compare.go antigo.csv novo.csv")
    }

    oldFile := os.Args[1]
    newFile := os.Args[2]

    oldData, oldProducts, err := readCSV(oldFile)
    if err != nil {
        log.Fatalf("Erro a ler CSV antigo: %v", err)
    }

    newData, _, err := readCSV(newFile)
    if err != nil {
        log.Fatalf("Erro a ler CSV novo: %v", err)
    }

    fmt.Println("-- SQL gerado --")

    insertedProducts := make(map[string]bool)

    for key, newVar := range newData {
        priceWithMargin := newVar.PriceBase * 1.4
        priceStr := fmt.Sprintf("%.2f", priceWithMargin)

        _, prodExists := oldProducts[newVar.ProductID]
        oldVar, varExists := oldData[key]

        if !prodExists {
            if !insertedProducts[newVar.ProductID] {
                fmt.Printf("-- Produto NOVO: %s\n", newVar.ProductID)
                fmt.Printf("INSERT INTO oc_product (product_id) VALUES (%s);\n", newVar.ProductID)
                insertedProducts[newVar.ProductID] = true
            }
            fmt.Printf("INSERT INTO oc_product_option_value (product_id, value, price) VALUES (%s, '%s', %s);\n",
                newVar.ProductID, newVar.VariantKey, priceStr)
            continue
        }

        if !varExists {
            fmt.Printf("-- Variante NOVA para produto %s\n", newVar.ProductID)
            fmt.Printf("INSERT INTO oc_product_option_value (product_id, value, price) VALUES (%s, '%s', %s);\n",
                newVar.ProductID, newVar.VariantKey, priceStr)
        } else {
            if oldVar.PriceBase != newVar.PriceBase || oldVar.VariantName != newVar.VariantName {
                fmt.Printf("-- Atualizar variante %s produto %s\n", newVar.VariantKey, newVar.ProductID)
                fmt.Printf("UPDATE oc_product_option_value SET price=%s WHERE product_id=%s AND value='%s';\n",
                    priceStr, newVar.ProductID, newVar.VariantKey)
            }
        }
    }
}