package main

import (
    "bufio"
    "context"
    "encoding/csv"
    "encoding/json"
    "fmt"
    "log"
    "math/rand"
    "os"
    "strings"
    "time"

    "github.com/chromedp/chromedp"
)

type SizeInfo struct {
    Key                 string      `json:"key"`
    Name                string      `json:"name"`
    ProductID           interface{} `json:"product_id"` // Pode ser int ou string conforme dados
    Available           bool        `json:"available"`
    PriceGrossValue     interface{} `json:"price_gross_value"`
    ShippingDays        int         `json:"shipping_days"`
}

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

func parsePrice(val interface{}) float64 {
    var price float64
    switch v := val.(type) {
    case float64:
        price = v
    case string:
        cleaned := strings.ReplaceAll(v, ",", ".")
        fmt.Sscanf(cleaned, "%f", &price)
    }
    return price
}

func main() {
    rand.Seed(time.Now().UnixNano())

    urls, err := readURLs("urls.txt")
    if err != nil {
        log.Fatalf("Erro a ler ficheiro de links: %v", err)
    }

    // Definir diretório de saída
    outputDir := "./csv-variants-output"
    if _, err := os.Stat(outputDir); os.IsNotExist(err) {
        os.MkdirAll(outputDir, 0755)
    }

    now := time.Now().Format("20060102_150405")
    filename := fmt.Sprintf("%s/product_variants_%s.csv", outputDir, now)

    file, err := os.Create(filename)
    if err != nil {
        log.Fatalf("Erro a criar o ficheiro CSV: %v", err)
    }
    defer file.Close()

    writer := csv.NewWriter(file)
    defer writer.Flush()

    // Cabeçalho do CSV
    writer.Write([]string{
        "product_id",
        "variant_key",
        "variant_name",
        "available",
        "price_gross_value",
        "shipping_days",
    })

    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.UserAgent(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 `+
            `(KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36`))

    allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), opts...)
    defer cancelAllocator()

    for _, url := range urls {
        fmt.Printf("\n--- Analisando: %s ---\n", url)

        ctx, cancelCtx := chromedp.NewContext(allocatorCtx)
        ctx, cancelCtx = context.WithTimeout(ctx, 40*time.Second)

        var sizesJson string
        err := chromedp.Run(ctx,
            chromedp.Navigate(url),
            chromedp.Sleep(time.Duration(rand.Intn(6)+2)*time.Second),
            chromedp.ScrollIntoView("body"),
            chromedp.Sleep(time.Duration(rand.Intn(6)+2)*time.Second),
            chromedp.Evaluate(`(() => {
                if(!window.product_data || !Array.isArray(window.product_data) || window.product_data.length===0) return "[]";
                let sizes = Array.isArray(window.product_data[0].sizes) ? window.product_data[0].sizes : [];
                return JSON.stringify(sizes.map(s => ({
                    key: s.id || null,
                    name: s.name || null,
                    product_id: s.product_id || null,
                    available: !!(s.availability && s.availability.visible && s.availability.status !== 'disable'),
                    price_gross_value: s.price && s.price.price && s.price.price.gross && (s.price.price.gross.value != null ? s.price.price.gross.value : null) || null,
                    shipping_days: s.shippingTime && s.shippingTime.shippingTime && typeof s.shippingTime.shippingTime.time.days === 'number' ? s.shippingTime.shippingTime.time.days : null
                })));
            })()`, &sizesJson),
        )
        cancelCtx()

        if err != nil {
            fmt.Printf("Falha ao analisar %s: %v\n", url, err)
            continue
        }

        var sizes []SizeInfo
        err = json.Unmarshal([]byte(sizesJson), &sizes)
        if err != nil {
            fmt.Printf("Falha no parse do JSON em %s: %v\n", url, err)
            continue
        }

        for _, s := range sizes {
            priceValue := parsePrice(s.PriceGrossValue)
            writer.Write([]string{
                fmt.Sprintf("%v", s.ProductID),
                fmt.Sprintf("%v", s.Key),
                s.Name,
                fmt.Sprintf("%v", s.Available),
                fmt.Sprintf("%.2f", priceValue),
                fmt.Sprintf("%d", s.ShippingDays),
            })
        }
    }

    fmt.Printf("\nExportação para CSV concluída: %s\n", filename)
}