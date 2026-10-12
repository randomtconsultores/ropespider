package main

import (
    "encoding/csv"
    "fmt"
    "log"
    "os"
)

type ImageData struct {
    ProductID string
    ImagePath string
    SortOrder string
}

const imagesDir = "csv-images-output/"

func readImageCSV(filename string) (map[string]ImageData, map[string]bool, error) {
    filePath := imagesDir + filename
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

    data := make(map[string]ImageData)
    products := make(map[string]bool)

    for i, row := range records {
        if i == 0 {
            continue // skip header
        }
        key := row[0] + "_" + row[1] // unique key: product_id + image path
        data[key] = ImageData{
            ProductID: row[0],
            ImagePath: row[1],
            SortOrder: row[2],
        }
        products[row[0]] = true
    }
    return data, products, nil
}

func main() {
    if len(os.Args) < 3 {
        log.Fatalf("Uso: go run compare_images.go antigo.csv novo.csv")
    }

    oldFile := os.Args[1]
    newFile := os.Args[2]

    oldData, oldProducts, err := readImageCSV(oldFile)
    if err != nil {
        log.Fatalf("Erro a ler CSV antigo: %v", err)
    }

    newData, _, err := readImageCSV(newFile)
    if err != nil {
        log.Fatalf("Erro a ler CSV novo: %v", err)
    }

    fmt.Println("-- SQL gerado --")
    insertedProducts := make(map[string]bool)

    for key, newImg := range newData {
        _, prodExists := oldProducts[newImg.ProductID]
        oldImg, imgExists := oldData[key]

        if !prodExists {
            if !insertedProducts[newImg.ProductID] {
                fmt.Printf("-- Produto NOVO: %s\n", newImg.ProductID)
                fmt.Printf("INSERT INTO oc_product (product_id) VALUES (%s);\n", newImg.ProductID)
                insertedProducts[newImg.ProductID] = true
            }
            fmt.Printf("INSERT INTO oc_product_image (product_id, image, sort_order) VALUES (%s, '%s', %s);\n",
                newImg.ProductID, newImg.ImagePath, newImg.SortOrder)
            continue
        }

        if !imgExists {
            fmt.Printf("-- Nova imagem para produto %s\n", newImg.ProductID)
            fmt.Printf("INSERT INTO oc_product_image (product_id, image, sort_order) VALUES (%s, '%s', %s);\n",
                newImg.ProductID, newImg.ImagePath, newImg.SortOrder)
        } else if oldImg.SortOrder != newImg.SortOrder {
            fmt.Printf("-- Atualizar sort_order imagem %s produto %s\n", newImg.ImagePath, newImg.ProductID)
            fmt.Printf("UPDATE oc_product_image SET sort_order=%s WHERE product_id=%s AND image='%s';\n",
                newImg.SortOrder, newImg.ProductID, newImg.ImagePath)
        }
    }
}