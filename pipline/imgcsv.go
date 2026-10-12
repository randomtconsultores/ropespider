package main

import (
    "bufio"
    "context"
    "encoding/csv"
    "fmt"
    "log"
    "math/rand"
    "os"
    "regexp"
    "strings"
    "time"

    "github.com/chromedp/chromedp"
)

var productIDRegexp = regexp.MustCompile(`product-ger-(\d+)`)

func extractProductID(url string) string {
    m := productIDRegexp.FindStringSubmatch(url)
    if len(m) > 1 {
        return m[1]
    }
    return ""
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
        if len(line) > 0 {
            urls = append(urls, line)
        }
    }
    return urls, scanner.Err()
}

func main() {
    rand.Seed(time.Now().UnixNano())

    urls, err := readURLs("urls.txt")
    if err != nil {
        log.Fatalf("Erro a ler ficheiro de links: %v", err)
    }

    outputDir := "./csv-output"
    if _, err := os.Stat(outputDir); os.IsNotExist(err) {
        os.MkdirAll(outputDir, 0755)
    }

    now := time.Now().Format("20060102_150405")
    filePath := fmt.Sprintf("%s/product_images_%s.csv", outputDir, now)

    file, err := os.Create(filePath)
    if err != nil {
        log.Fatalf("Erro a criar ficheiro CSV: %v", err)
    }
    defer file.Close()

    writer := csv.NewWriter(file)
    defer writer.Flush()

    // Cabeçalho CSV
    writer.Write([]string{"product_id", "image", "sort_order"})

    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.UserAgent(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 `+
            `(KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36`))

    allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), opts...)
    defer cancelAllocator()

    for _, url := range urls {
        productID := extractProductID(url)
        fmt.Printf("\n--- Extraindo imagens de: %s | product_id: %s ---\n", url, productID)

        ctx, cancelCtx := chromedp.NewContext(allocatorCtx)
        ctx, cancelCtx = context.WithTimeout(ctx, 40*time.Second)

        var imageSrcs []string

        jsExtractImages := `
            Array.from(document.querySelectorAll('#photos_slider img.photos__photo')).map(img => {
                return img.getAttribute('data-img_high_res') || img.src;
            })
        `

        err := chromedp.Run(ctx,
            chromedp.Navigate(url),

            chromedp.Sleep(time.Duration(rand.Intn(4)+2)*time.Second),
            chromedp.Evaluate(`window.scrollBy(0, window.innerHeight/2)`, nil),
            chromedp.Sleep(time.Duration(rand.Intn(3)+1)*time.Second),
            chromedp.Evaluate(`window.scrollBy(0, window.innerHeight/2)`, nil),
            chromedp.Sleep(time.Duration(rand.Intn(3)+1)*time.Second),
            chromedp.Evaluate(`window.scrollTo(0, 0)`, nil),
            chromedp.Sleep(time.Duration(rand.Intn(2)+1)*time.Second),

            chromedp.WaitVisible("#photos_slider", chromedp.ByID),
            chromedp.Evaluate(jsExtractImages, &imageSrcs),
        )
        cancelCtx()

        if err != nil {
            log.Printf("Erro a extrair imagens em %s: %v\n", url, err)
            continue
        }

        if len(imageSrcs) == 0 {
            fmt.Println("  Nenhuma imagem encontrada")
        } else {
            for i, src := range imageSrcs {
                if strings.HasPrefix(src, "/") {
                    src = "https://seiletech.de" + src
                }
                fmt.Printf("  Imagem %d: %s\n", i+1, src)

                err := writer.Write([]string{productID, src, fmt.Sprintf("%d", i+1)})
                if err != nil {
                    log.Printf("Erro a escrever no CSV: %v\n", err)
                }
            }
            writer.Flush()
        }
    }

    fmt.Printf("\nFicheiro CSV salvo em: %s\n", filePath)
}