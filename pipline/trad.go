package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"time"

	translate "cloud.google.com/go/translate/apiv3"
	translatepb "google.golang.org/genproto/googleapis/cloud/translate/v3"

	"google.golang.org/api/option"
)

func translateText(ctx context.Context, client *translate.TranslationClient, projectID, text, sourceLang, targetLang string) (string, error) {
	req := &translatepb.TranslateTextRequest{
		Parent:             fmt.Sprintf("projects/%s/locations/global", projectID),
		Contents:           []string{text},
		MimeType:           "text/plain",
		SourceLanguageCode: sourceLang,
		TargetLanguageCode: targetLang,
	}
	resp, err := client.TranslateText(ctx, req)
	if err != nil {
		return "", err
	}
	if len(resp.Translations) == 0 {
		return "", fmt.Errorf("no translation returned")
	}
	return resp.Translations[0].TranslatedText, nil
}

func main() {
	const projectID = "opespidertranslate"
	const credsPath = "opespidertranslate-e78c1e4ad09a.json"
	const inputPath = "products.csv"
	outputPath := fmt.Sprintf("products_translated_%s.csv", time.Now().Format("20060102_150405"))

	ctx := context.Background()
	client, err := translate.NewTranslationClient(ctx, option.WithCredentialsFile(credsPath))
	if err != nil {
		log.Fatalf("Failed to create translation client: %v", err)
	}
	defer client.Close()

	inFile, err := os.Open(inputPath)
	if err != nil {
		log.Fatalf("Failed to open input CSV: %v", err)
	}
	defer inFile.Close()
	reader := csv.NewReader(inFile)

	outFile, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output CSV: %v", err)
	}
	defer outFile.Close()
	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	_, err = reader.Read() // descartar header original
	if err != nil {
		log.Fatalf("Failed to read header: %v", err)
	}

	// Escreve header novo só com colunas que quer guardar
	writer.Write([]string{"product_id", "title_pt", "description_pt"})

	for {
		record, err := reader.Read()
		if err != nil {
			break // EOF
		}

		productID := record[0]
		title := record[1]
		description := record[2]

		titlePt, err := translateText(ctx, client, projectID, title, "de", "pt")
		if err != nil {
			log.Printf("Erro ao traduzir título: %v", err)
			titlePt = ""
		}

		descPt, err := translateText(ctx, client, projectID, description, "de", "pt")
		if err != nil {
			log.Printf("Erro ao traduzir descrição: %v", err)
			descPt = ""
		}

		writer.Write([]string{productID, titlePt, descPt})
		fmt.Printf("Traduzido: ID=%s Título: %s -> %s\n", productID, title, titlePt)
	}

	fmt.Printf("Tradução concluída, arquivo salvo em %s\n", outputPath)
}