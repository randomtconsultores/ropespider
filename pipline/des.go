package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/k3a/html2text"
)

func readURLs(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			urls = append(urls, line)
		}
	}
	return urls, scanner.Err()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	urls, err := readURLs("urls.txt")
	if err != nil {
		log.Fatalf("Erro a ler ficheiro URLs: %v", err)
	}

	file, err := os.Create("products.csv")
	if err != nil {
		log.Fatalf("Erro ao criar CSV: %v", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"product_id", "title", "description"})

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 `+
			`(KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36`))
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	for _, url := range urls {
		fmt.Printf("\n--- Analisando: %s ---\n", url)
		ctx, cancel := chromedp.NewContext(allocCtx)
		ctx, cancel = context.WithTimeout(ctx, 100*time.Second)
		defer cancel()

		var productID, title, descHTML string

		err := chromedp.Run(ctx,
			chromedp.Navigate(url),
			chromedp.Sleep(time.Duration(rand.Intn(5)+3)*time.Second),
			chromedp.ScrollIntoView("body"),
			chromedp.Sleep(time.Duration(rand.Intn(5)+3)*time.Second),

			chromedp.Evaluate(`(() => {
				if (window.product_data && window.product_data[0] && window.product_data[0].id) {
					return window.product_data[0].id.toString();
				}
				return "";
			})()`, &productID),

			chromedp.Text(`h1.product_name__name.m-0`, &title, chromedp.NodeVisible),

			chromedp.InnerHTML(`#projector_longdescription`, &descHTML, chromedp.ByID),
		)

		if err != nil {
			fmt.Printf("Falha ao extrair %s: %v\n", url, err)
			continue
		}

		descText := html2text.HTML2Text(descHTML)

		writer.Write([]string{productID, title, descText})
		writer.Flush()

		fmt.Printf("Extraído: ID=%s Título=%s\n", productID, title)
	}
	fmt.Println("Extração concluída")
}