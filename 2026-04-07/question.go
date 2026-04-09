
package main

import (
	"fmt"
	"sync"
)

func main() {
	letter, number := make(chan bool), make(chan bool)
	wait := sync.WaitGroup{}

	wait.Add(1)
	go func() {
		i := 1
		for {
			<-number
			fmt.Print(i)
			i++
			fmt.Print(i)
			i++
			letter <- true
		}
	}()

	go func(wait *sync.WaitGroup) {
		i := 'A'
		for {
			<-letter
			if i >= 'Z' {
				wait.Done()
				return
			}
			fmt.Print(string(i))
			i++
			fmt.Print(string(i))
			i++
			number <- true
		}
	}(&wait)

	number <- true
	wait.Wait()
}
