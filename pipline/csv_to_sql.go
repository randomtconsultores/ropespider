package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
)

func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Use: go run csv_to_sql.go <arquivo_csv_traduzido>")
	}
	inputCSV := os.Args[1]
	languageID := 2 // ID do idioma português pt-pt no OpenCart, ajuste se necessário

	inFile, err := os.Open(inputCSV)
	if err != nil {
		log.Fatalf("Erro ao abrir arquivo CSV: %v", err)
	}
	defer inFile.Close()

	reader := csv.NewReader(inFile)

	// Ler e ignorar cabeçalho
	_, err = reader.Read()
	if err != nil {
		log.Fatalf("Erro ao ler cabeçalho: %v", err)
	}

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		productID := record[0]
		title := escapeSQLString(record[1])
		description := escapeSQLString(record[2])
		supplierProductID := record[3] // novo campo no CSV para supplier_product_id

		// SQL para tabela oc_product_description com supplier_product_id
		sqlDesc := fmt.Sprintf(
			"INSERT INTO oc_product_description (product_id, language_id, name, description, supplier_product_id) VALUES (%s, %d, '%s', '%s', %s) "+
				"ON DUPLICATE KEY UPDATE name='%s', description='%s', supplier_product_id=%s;",
			productID, languageID, title, description, supplierProductID,
			title, description, supplierProductID)

		// SQL para tabela oc_product para atualizar supplier_product_id
		sqlProd := fmt.Sprintf(
			"UPDATE oc_product SET supplier_product_id = %s WHERE product_id = %s;",
			supplierProductID, productID)

		fmt.Println(sqlDesc)
		fmt.Println(sqlProd)
	}
}