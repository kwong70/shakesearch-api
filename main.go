package main

import (
	"encoding/json"
	"fmt"
	"index/suffixarray"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sort"
)

type Searcher struct {
	Titles map[string]bool `json:"Titles"`
	Works []Work `json:"Works"`
}

type Work struct {
	Title string `json:"Title"` 
	SuffixArray *suffixarray.Index `json:"SuffixArray"` 
	CompleteWorks string `json:"CompleteWorks"` 
	LineNumberShift int `json:"LineNumberShift`
}

type WorkSnippet struct {
	Title string `json:"Title"`
	Snippet string `json:"Snippet"`
}

func main() {

	// Initialize 
	s := Searcher{}	
	s.load();

	fmt.Println("Loaded", len(s.Works), "works")

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/search", handleSearch(s))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	fmt.Printf("Listening on port %s...\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *Searcher) load() error {
	var titleFilePath string = "./titles.txt"
	var completeworksPath string = "./completeworks.txt"
	s.Titles = map[string]bool{}

	// Get a set of titles 
	dat, err := ioutil.ReadFile(titleFilePath)
	if err != nil {
		fmt.Println("Failed to load title file") 
		return fmt.Errorf("failed loading titles: %w", err)
	}
	titles := string(dat)
	for _, title := range strings.Split(titles, "\n") {
		title = strings.TrimSpace(title)
		s.Titles[title] = true
	}

	// Load completeworks organized by story 
	dat, err = ioutil.ReadFile(completeworksPath)
	if err != nil {
		fmt.Println("failed to load completeworks.txt file")
		return fmt.Errorf("failed loading complete works: %w", err)
	}
	completeWorks := string(dat)
	unseenTitles := s.Titles
	currentWork := Work{}
	currentWork.Title = "Prework"
	currentWork.CompleteWorks = ""
	currentWork.LineNumberShift = 0
	for lineNumber, line := range strings.Split(completeWorks, "\n") {
		trimmedLine := strings.TrimSpace( line )
		if lineNumber > 130 && unseenTitles[trimmedLine] { // The start of a new work 
			currentWork.SuffixArray = suffixarray.New([]byte( strings.ToLower(currentWork.CompleteWorks) ))
			currentWork.LineNumberShift = lineNumber
			if(currentWork.Title != "Prework") {
				s.Works = append(s.Works, currentWork)
			}
			currentWork.Title = trimmedLine;
			currentWork.CompleteWorks = ""
			unseenTitles[trimmedLine] = false
		} else if trimmedLine == "FINIS" {
			currentWork.SuffixArray = suffixarray.New([]byte( strings.ToLower(currentWork.CompleteWorks) ))
			currentWork.LineNumberShift = lineNumber
			s.Works = append(s.Works, currentWork)
			break
		}else {
			currentWork.CompleteWorks = currentWork.CompleteWorks + strconv.Itoa(lineNumber - currentWork.LineNumberShift) + " " + line
		}
	}
	return nil
}

func handleSearch(s Searcher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w)
		
		queries, _, selectedTitles := parseRequest(s, r)
		fmt.Println("/search", "query:", queries)
		
		var results []WorkSnippet = s.Search(queries, selectedTitles)

		w.Header().Set("Content-Type", "application/json")
		//w.Header().Set("Content-Length", contentLength)
		json.NewEncoder(w).Encode(results)
	}
}

func (s *Searcher) Search(queries []string, selectedTitles map[string]bool) []WorkSnippet {
	var results []WorkSnippet
	const SNIPPET_SIZE int = 150
	for _, work := range s.Works {
		if ( selectedTitles[work.Title] ) {
			var foundIndices []int
			for _, query := range queries {
				idxs := work.SuffixArray.Lookup([]byte(query), -1)
				foundIndices = append(foundIndices, idxs...)
			}
			if(len(foundIndices) > 0) {
				sort.Ints(foundIndices)
				var i int = 0
				for i < len( foundIndices ) {
					var currEnd = foundIndices[i]
					var j int = i + 1  
					for( j < len(foundIndices) && foundIndices[j] <= currEnd + SNIPPET_SIZE) {
						currEnd = foundIndices[j]
						j++
					}
					beginning, end := getBounds(foundIndices[i], currEnd, len(work.CompleteWorks))
					text := work.CompleteWorks[beginning: end]
					snippet :=  WorkSnippet{work.Title, text}
					results = append(results, snippet)
					i = j 
				}
			}
		}
	}

	return results
}

func getBounds(startIndex int, endIndex int,  workLength int) (int, int){
	var SNIPPET_SIZE int = 150
	var beginning, end int
	if startIndex - SNIPPET_SIZE < 0{
		beginning = 0
	} else {
		beginning = startIndex - SNIPPET_SIZE
	}

	if endIndex + SNIPPET_SIZE > workLength {
		end = workLength
	} else {
		end = endIndex + SNIPPET_SIZE
	}
	return beginning, end
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func parseRequest(s Searcher, r *http.Request) ([]string, bool, map[string]bool) {
	exactMatchStr, _ := r.URL.Query()["exactMatch"]
	exactMatch, _ := strconv.ParseBool(exactMatchStr[0])

	queries, _ := r.URL.Query()["q"]
	rawQuery := queries[0]
	var query []string
	if !exactMatch {
		splitQueries := strings.Split(rawQuery, " ")
		if len(splitQueries) > 1 {
			query = strings.Split(rawQuery, " ")
		}
	}	
	query = append(query, rawQuery)
	
	// Get list of titles and convert it into a set 
	bodyStr, _ := ioutil.ReadAll(r.Body)
	var body []string
	_ = json.Unmarshal(bodyStr, &body)
	selectedTitles := map[string]bool{}
	if(len(body) == 0 ) {
		for _, work := range s.Works {
			selectedTitles[work.Title] = true  
		}
	}else {
		for _, selectedTitle := range body { 
			trimmedTitle := strings.TrimSpace(selectedTitle)
			selectedTitles[trimmedTitle] = true
		}
	}

	return query, exactMatch, selectedTitles
}