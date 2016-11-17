package main

type MyStruct struct {
	aPointer *int8
	aSlice   []string
	anArray  [4]float64
	fromChan <-chan int8
	toChan   chan<- int64
}

func main() {}
