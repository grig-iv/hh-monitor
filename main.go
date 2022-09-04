/*
Hh-monitor looks up vacancy count for programming languages at spb.hh.ru

usage:
    hh-monitor [-f FILE] LANGS..

description:
    With no FILE, write to standard output.
    If FILE is present, add a blank line at the end of file.

    LANGS is a list of programming languages that will be search.

output format:
    [06-01-02 15:04]
    lang_1=10
    lang_2=50
    ...
    lang_n=20
*/

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type args struct {
	filePath string
	langs    []string
}

type monitorResult struct {
	lang         string
	vacancyCount uint16
	error        error
}

func main() {
	args := parseArgs()
	stats := monitor(args.langs)
	entry := formStatEntry(stats)

	if args.filePath != "" {
		saveToFile(args.filePath, entry)
	} else {
		fmt.Print(entry)
	}
}

func parseArgs() args {
	if len(os.Args) == 1 {
		log.Panic("argumets are missing")
	}

	var filePath string
	var langs []string

	if os.Args[1] == "-f" {
		filePath = os.Args[2]
		langs = os.Args[3:]
	} else {
		langs = os.Args[1:]
	}

	return args{filePath, langs}
}

func monitor(langs []string) []monitorResult {
	monitorRes := make([]monitorResult, len(langs))
	ch := make(chan monitorResult)

	for _, lang := range langs {
		go monitorLang(lang, ch)
	}

	for i := range langs {
		monitorRes[i] = <-ch
	}

	return monitorRes
}

func monitorLang(lang string, ch chan monitorResult) {
	count, err := getVacancyCount(lang)
	mr := monitorResult{lang, count, err}
	ch <- mr
}

func getVacancyCount(lang string) (uint16, error) {
	page, err := loadPage(lang)
	if err != nil {
		return 0, err
	}

	vacCount := findVacancyCount(page)
	return vacCount, nil
}

func loadPage(lang string) ([]byte, error) {
	requestURL := getUrl(lang)

	respond, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}

	defer respond.Body.Close()

	page, err := io.ReadAll(respond.Body)
	if err != nil {
		return nil, err
	}

	return page, nil
}

func findVacancyCount(page []byte) uint16 {
	headerRexp := regexp.MustCompile("\\d+ ваканс")
	header := string(headerRexp.Find(page))
	if header == "" {
		return 0
	}

	strVacCount := strings.Split(header, " ")[0]
	vacCount, _ := strconv.Atoi(strVacCount)
	return uint16(vacCount)
}

func getUrl(lang string) string {
	url := "https://spb.hh.ru/search/vacancy"
	query := "?area=2&professional_role=96&search_field=name&text=" + lang
	return url + query
}

func formStatEntry(monitorRes []monitorResult) string {
	sb := strings.Builder{}

	sb.WriteString(time.Now().Format("[06-01-02 15:04]\n"))

	for _, res := range monitorRes {
		var langLine string
		if res.error != nil {
			langLine = fmt.Sprintf("%s=%s\n", res.lang, res.error.Error())
		} else {
			langLine = fmt.Sprintf("%s=%s\n", res.lang, strconv.Itoa(int(res.vacancyCount)))
		}
		sb.WriteString(langLine)
	}

	return sb.String()
}

func saveToFile(filePath, entry string) {
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	file, err := os.OpenFile(filePath, flags, 0644)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()
	entry = fmt.Sprintln(entry) // add new line at the end
	_, err = file.WriteString(entry)
	if err != nil {
		log.Fatal(err)
	}
}
