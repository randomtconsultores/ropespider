package main

import (
    "encoding/csv"
    "fmt"
    "log"
    "os"
    "sort"
    "strconv"
    "strings"
)

type VariantData struct {
    ProductID    string
    VariantKey   string
    VariantName  string
    Available    string
    PriceBase    float64
    ShippingDays string
}

const variantsDir = "csv-variants-output/"
const fixedOptionID = 4 // ID fixo da opção (ex: 4 = Comprimento)

func readCSV(filename string) (map[string]VariantData, map[string][]VariantData, error) {
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
    grouped := make(map[string][]VariantData)

    for i, row := range records {
        if i == 0 {
            continue
        }
        price, _ := strconv.ParseFloat(row[4], 64)
        key := row[0] + "_" + row[1]
        variant := VariantData{
            ProductID:    row[0],
            VariantKey:   row[1],
            VariantName:  row[2],
            Available:    row[3],
            PriceBase:    price,
            ShippingDays: row[5],
        }
        data[key] = variant
        grouped[row[0]] = append(grouped[row[0]], variant)
    }
    return data, grouped, nil
}

func parseMeters(s string) (int, error) {
    s = strings.ToLower(strings.TrimSpace(s))
    s = strings.ReplaceAll(s, " ", "")
    s = strings.TrimSuffix(s, "m")
    return strconv.Atoi(s)
}

func main() {
    if len(os.Args) < 3 {
        log.Fatalf("Uso: go run compare.go antigo.csv novo.csv")
    }

    oldFile := os.Args[1]
    newFile := os.Args[2]

    oldData, _, err := readCSV(oldFile)
    if err != nil {
        log.Fatalf("Erro ao ler CSV antigo: %v", err)
    }

    _, newGrouped, err := readCSV(newFile)
    if err != nil {
        log.Fatalf("Erro ao ler CSV novo: %v", err)
    }

    fmt.Println("-- SQL gerado --")
    fmt.Println("SET @product_option_id = 0;")

    insertedProducts := make(map[string]bool)

    for productID, variants := range newGrouped {
        sort.Slice(variants, func(i, j int) bool {
            vi, err1 := strconv.ParseFloat(variants[i].VariantKey, 64)
            vj, err2 := strconv.ParseFloat(variants[j].VariantKey, 64)
            if err1 != nil || err2 != nil {
                return variants[i].VariantKey < variants[j].VariantKey
            }
            return vi < vj
        })

        if !insertedProducts[productID] {
            fmt.Printf("-- Produto NOVO: %s\n", productID)
            fmt.Printf("INSERT INTO oc_product (product_id) VALUES (%s);\n", productID)
            // Sem o campo value na inserção abaixo
            fmt.Printf("INSERT INTO oc_product_option (product_id, option_id, required) VALUES (%s, %d, 1);\n", productID, fixedOptionID)
            fmt.Println("SET @product_option_id = LAST_INSERT_ID();")
            insertedProducts[productID] = true
        }

        for _, v := range variants {
            priceWithMargin := v.PriceBase * 1.4
            priceStr := fmt.Sprintf("%.2f", priceWithMargin)

            key := v.ProductID + "_" + v.VariantKey
            _, exists := oldData[key]

            quantity := "0"
            subtract := "1"
            if v.Available == "yes" || v.Available == "1" || v.Available == "true" {
                quantity = "9999"
                subtract = "0"
            }

            if !exists {
                variantNameLower := strings.ToLower(strings.TrimSpace(v.VariantName))
                if strings.Contains(variantNameLower, "universal") {
                    // Insere sem option_value_id para Universal
                    fmt.Printf("-- Variante Universal para produto %s\n", productID)
                    fmt.Printf(
                        "INSERT INTO oc_product_option_value (product_option_id, product_id, option_id, quantity, subtract, price, price_prefix, points, points_prefix, weight, weight_prefix) "+
                            "VALUES (@product_option_id, %s, %d, %s, %s, %s, '+', 0, '+', 0, '+');\n",
                        productID, fixedOptionID, quantity, subtract, priceStr)
                    continue
                }

                meters, err := parseMeters(v.VariantName)
                if err != nil || quantity == "0" {
                    continue
                }

                fmt.Printf("-- Variante: %s, produto: %s\n", v.VariantName, productID)
                fmt.Printf(
                    "INSERT INTO oc_product_option_value (product_option_id, product_id, option_id, option_value_id, quantity, subtract, price, price_prefix, points, points_prefix, weight, weight_prefix) "+
                        "VALUES (@product_option_id, %s, %d, %d, %s, %s, %s, '+', 0, '+', 0, '+');\n",
                    productID, fixedOptionID, meters, quantity, subtract, priceStr)
            }
        }
    }
}