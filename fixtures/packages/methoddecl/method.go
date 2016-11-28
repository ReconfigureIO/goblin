package main

type Thing struct {
	count int8
}

func (t Thing) Inc() {
	t.count += 1
}

func main() {
	t := Thing { 1 }
	t.Inc()
	println(t.count)
}
