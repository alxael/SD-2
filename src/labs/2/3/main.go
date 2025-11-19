package main

import (
	"fmt"
	"time"
)

func processFunction(finishedSuccessfully chan bool) {
	// run compute-heavy program here
	time.Sleep(time.Second)
	finishedSuccessfully <- true
}

func functionFinished(finishedSuccessfully chan bool) {
	isSuccess := <-finishedSuccessfully
	if isSuccess {
		fmt.Println("Function finished successfully!")
	} else {
		fmt.Println("Function finished successfully!")
	}
}

func main() {
	finishedSuccessfully := make(chan bool)

	go processFunction(finishedSuccessfully)
	functionFinished(finishedSuccessfully)
}
