package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var aspRegex = regexp.MustCompile(`(?m)(\@page|\@model)`)
var out io.Writer = os.Stdout

func main() {
	var outputFileFlag string
	var payloadsFileFlag string
	var redirectTargetFlag string
	flag.StringVar(&outputFileFlag, "o", "", "Output file for identified leakd source")
	flag.StringVar(&payloadsFileFlag, "p", "", "File of open redirect payloads")
	flag.StringVar(&redirectTargetFlag, "t", "", "Target URL for redirect payloads")
	appendMissingProtocolsFlag := flag.Bool("x", false, "Automatically add http & https to all URLs (will likely increase overall amount of requests)")
	quietModeFlag := flag.Bool("q", false, "Only output the URL's with leaked source")
	flag.Parse()

	if payloadsFileFlag == "" || redirectTargetFlag == "" {
		fmt.Println("Usage: cat urls.txt | ./wrongturn -t \"https://redirect.to.me\" -p payloads.txt")
		os.Exit(0)
	}

	quietMode := *quietModeFlag
	appendMissingProtocols := *appendMissingProtocolsFlag
	saveOutput := outputFileFlag != ""
	outputToSave := []string{}

	preProcessedPayloads, payloadErr := readLines(payloadsFileFlag)
	if payloadErr != nil {
		panic(payloadErr)
	}

	payloads := []string{}
	for _, p := range preProcessedPayloads {
		np := strings.ReplaceAll(p, "<--target-->", redirectTargetFlag)
		payloads = append(payloads, np)
	}

	if !quietMode {
		banner()
		fmt.Println("")
		fmt.Println("[*] Payloads loaded:", len(payloads))
	}

	writer := bufio.NewWriter(out)
	urls := make(chan string, 1)
	var wg sync.WaitGroup

	ch := readStdin()
	go func() {
		//translate stdin channel to domains channel
		for u := range ch {
			urls <- u
		}
		close(urls)
	}()

	// flush to writer periodically
	t := time.NewTicker(time.Millisecond * 500)
	defer t.Stop()
	go func() {
		for {
			select {
			case <-t.C:
				writer.Flush()
			}
		}
	}()

	for u := range urls {
		wg.Add(1)
		go func(site string) {
			defer wg.Done()
			finalUrls := []string{}

			// create all the URLs here with the loaded payloads
			// If the identified URL has neither http or https infront of it. Create both and scan them.
			if appendMissingProtocols {
				if !strings.Contains(u, "http://") && !strings.Contains(u, "https://") {
					finalUrls = append(finalUrls, "http://"+u)
					finalUrls = append(finalUrls, "https://"+u)
				} else if strings.Contains(u, "http://") {
					finalUrls = append(finalUrls, "https://"+u)
				} else if strings.Contains(u, "https://") {
					finalUrls = append(finalUrls, "http://"+u)
				} else {
					// else, just scan the submitted one as it has either protocol
					finalUrls = append(finalUrls, u)
				}
			} else {
				// check for any protocol, if non-exist, add http only
				if !strings.Contains(u, "http://") && !strings.Contains(u, "https://") {
					finalUrls = append(finalUrls, "http://"+u)
				} else {
					finalUrls = append(finalUrls, "http://"+u)
				}
			}

			// now loop the slice of finalUrls (either submitted OR 2 urls with http/https appended to them)
			for _, uu := range finalUrls {
				for _, p := range payloads {
					foundRedirect := makeRequest(uu+p, quietMode, redirectTargetFlag)
					if foundRedirect {
						fmt.Printf("%s\n", uu+p)

						if saveOutput {
							outputToSave = append(outputToSave, uu+p)
						}
					}
				}
			}
		}(u)
	}

	wg.Wait()

	// just in case anything is still in buffer
	writer.Flush()

	if saveOutput {
		file, err := os.OpenFile(outputFileFlag, os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil && !quietMode {
			log.Fatalf("failed creating file: %s\n", err)
		}

		datawriter := bufio.NewWriter(file)

		for _, data := range outputToSave {
			_, _ = datawriter.WriteString(data + "\n")
		}

		datawriter.Flush()
		file.Close()
	}
}

func banner() {
	fmt.Println("---------------------------------------------------")
	fmt.Println("WrongTurn -> Crawl3r/@monobehaviour")
	fmt.Println("Looks for open redirect issues against target domains")
	fmt.Println("---------------------------------------------------")
}

func readStdin() <-chan string {
	lines := make(chan string)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			url := strings.ToLower(sc.Text())
			if url != "" {
				lines <- url
			}
		}
	}()
	return lines
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func makeRequest(url string, quietMode bool, targetUrl string) bool {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Failed to created GET request: %s\n", err.Error())
	}

	// TODO: make a client per thread now URL?
	client := new(http.Client)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return errors.New("Redirect")
	}

	response, err := client.Do(req)

	if err != nil {
		if strings.Contains(err.Error(), "Redirect") { // catch the redirect error we triggered as we know this is a good error for us
			if response.StatusCode == http.StatusFound { //status code 302
				location, _ := response.Location()
				if location.String() == targetUrl {
					return true
				}
			}
		} else {
			if !quietMode {
				log.Println("[err]", err)
			}
		}
	}

	return false
}
