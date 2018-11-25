package main

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"os"
	"strings"
	"errors"
	"strconv"
)

type Actor interface {
    addMessage(message []string)
    dispatch(message []string)
}

func send(reciever Actor, message []string){
    reciever.addMessage(message)
}

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

type DataStorageManager struct{
    data []string
    swm *StopWordManager
    messages chan []string
}

func (dsm *DataStorageManager) addMessage(message []string){
    dsm.messages <- message
}

func (dsm *DataStorageManager) init(path string) {
    book, err := ioutil.ReadFile(path)
    if err != nil {
        panic(err)
    }
    dsm.data = RegSplit(strings.ToLower(string(book)), "[^a-zA-Z]")
}

func (dsm *DataStorageManager) processWords() {
    for _, w := range dsm.data {
        send(dsm.swm, []string{"filter", w})
    }
    send(dsm.swm, []string{"top25"})
}

func (dsm *DataStorageManager) dispatch(message []string) {
    if (message[0] == "init"){
        dsm.init(message[1]);
    } else if (message[0] == "processWords") {
        dsm.processWords()
    } else if (message[0] == "kill") {
        close(dsm.messages)  
    } else {
        send(dsm.swm, message)
    }
}

func (dsm *DataStorageManager) run() {
    for message := range dsm.messages {
        dsm.dispatch(message)
    }
}

type StopWordManager struct{
    stopWords []string
    wfm *WordFrequencyManager
    messages chan []string
}

func (swm *StopWordManager) addMessage(message []string){
    swm.messages <- message
}

func (swm *StopWordManager) init() {
    swFile, err := ioutil.ReadFile("./stop_words.txt")
    if err != nil {
        panic(err)
    }
    swm.stopWords = RegSplit(strings.ToLower(string(swFile)), "[^a-zA-Z]")
}

func (swm *StopWordManager) filter(word string) {
    sendMsg := true
    for _, sw := range swm.stopWords {
        if sw == word {
            sendMsg = false
            break
        }
    }
    if (sendMsg && len(word) > 1){
        send(swm.wfm, []string{"increment", word})
    }
}

func (swm *StopWordManager) dispatch(message []string) {
    if (message[0] == "init"){
        swm.init();
    } else if (message[0] == "filter"){
        swm.filter(message[1]);
    } else if (message[0] == "kill") {
        close(swm.messages)  
    } else {
        send(swm.wfm, message)
    }
}

func (swm *StopWordManager) run() {
    for message := range swm.messages {
        swm.dispatch(message)
    }
}

type Frequency struct {
    key string
    value int
}

type WordFrequencyManager struct {
    freqs map[string]int
    messages chan []string
    wfc *WordFrequencyController
}

func (wfm *WordFrequencyManager) addMessage(message []string){
    wfm.messages <- message
}

func (wfm *WordFrequencyManager) increment(word string) {
    wfm.freqs[word] = wfm.freqs[word] + 1
}

func (wfm *WordFrequencyManager) top25() {
    message := []string{"top25"}
    for i := 0; i < 25; i++ {
        maxKey := ""
        maxVal := 0
        for k, v := range wfm.freqs{
            if (v > maxVal) {
                maxKey = k
                maxVal = v
            }
        }
        message = append(message, maxKey + " - " + strconv.Itoa(maxVal))
        wfm.freqs[maxKey] = 0
    }
    send(wfm.wfc, message)
}

func (wfm *WordFrequencyManager) dispatch(message []string) {
    if (message[0] == "increment"){
        wfm.increment(message[1]);
    } else if (message[0] == "top25"){
        wfm.top25();
    } else if (message[0] == "kill") {
        close(wfm.messages)  
    } else {
        panic(errors.New(message[0] + " not understood"))
    }
}

func (wfm *WordFrequencyManager) run() {
    for message := range wfm.messages {
        wfm.dispatch(message)
    }
}

type WordFrequencyController struct {
    dsm *DataStorageManager
    messages chan []string
}

func (wfc *WordFrequencyController) addMessage(message []string){
    wfc.messages <- message
}

func (wfc *WordFrequencyController) execute() {
    send(wfc.dsm, []string{"processWords"})
}

func (wfc *WordFrequencyController) display(lines []string){
    for _, line := range lines {
        fmt.Println(line)
    }
    send(wfc.dsm, []string{"kill"})
    send(wfc.dsm.swm, []string{"kill"})
    send(wfc.dsm.swm.wfm, []string{"kill"})
    send(wfc, []string{"kill"})
}

func (wfc *WordFrequencyController) dispatch(message []string) {
    if (message[0] == "run"){
        wfc.execute();
    } else if (message[0] == "top25"){
        wfc.display(message[1:]);
    } else if (message[0] == "kill") {
        close(wfc.messages)  
    } else {
        panic(errors.New(message[0] + " not understood"))
    }
}

func (wfc *WordFrequencyController) run() {
    for message := range wfc.messages {
        wfc.dispatch(message)
    }
}

func main() {
	wfc := WordFrequencyController{messages: make(chan []string, 100)}
	wfm := WordFrequencyManager{wfc: &wfc, freqs: make(map[string]int), messages: make(chan []string, 100)}
	swm := StopWordManager{wfm: &wfm, messages: make(chan []string, 100)}
	dsm := DataStorageManager{swm: &swm, messages: make(chan []string, 100)}
	wfc.dsm = &dsm
	
	send(&swm, []string{"init"})
	send(&dsm, []string{"init", os.Args[1]})
	send(&wfc, []string{"run"})
	
	go dsm.run()
	go swm.run()
	go wfm.run()
	wfc.run()
}
