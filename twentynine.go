package main

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"os"
	"strings"
	"strconv"
)

func RegSplit(text string, delimeter string) []string {
    reg := regexp.MustCompile(delimeter)
    indexes := reg.FindAllStringIndex(text, -1)
    laststart := 0
    result := make([]string, len(indexes) + 1)
    for i, element := range indexes {
            result[i] = text[laststart:element[0]]
            laststart = element[1]
    }
    result[len(indexes)] = text[laststart:len(text)]
    return result
}

func main () {
    wordSpace := make(chan string, 200000)
    freqSpace := make(chan map[string]int, 100)
    
    swFile, err := ioutil.ReadFile("./stop_words.txt")
    if err != nil {
        panic(err)
    }
    stopWords := RegSplit(strings.ToLower(string(swFile)), "[^a-zA-Z]")
    
    processWords := func(){
        freqs := make(map[string]int)
        for word := range wordSpace {
            increment := len(word) > 1
            if !increment {
                continue
            }
            for _, sw := range stopWords {
                if sw == word {
                    increment = false
                    break
                }
            }
            if increment {
                freqs[word] = freqs[word] + 1
            }
        }
        freqSpace <- freqs
    }
    
    book, err := ioutil.ReadFile(os.Args[1])
    if err != nil {
        panic(err)
    }
    wordList := RegSplit(strings.ToLower(string(book)), "[^a-zA-Z]")
    go func() {
        defer close(wordSpace)
        for _, w := range wordList{
            wordSpace <- w
        }
    }()
    
    numWorkers := 5
    for i := 0; i < numWorkers; i++ {
        go func() {
            processWords()
        }()
    }
    
    wordFreqs := make(map[string]int)
    for i := 0; i < numWorkers; i++ {
        freqs := <-freqSpace
        for key, value := range freqs {
            wordFreqs[key] = wordFreqs[key] + value
        }
    }
    
    for i := 0; i < 25; i++ {
        maxKey := ""
        maxVal := 0
        for k, v := range wordFreqs{
            if (v > maxVal) {
                maxKey = k
                maxVal = v
            }
        }
        fmt.Println(maxKey + " - " + strconv.Itoa(maxVal))
        wordFreqs[maxKey] = 0
    }
    
    close(freqSpace)
}
