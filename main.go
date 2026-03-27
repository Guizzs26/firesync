package main

import "fmt"

type Engine struct {
	Horsepower int
}

func (e Engine) Start() {
	fmt.Println("Engine started")
}

type Car struct {
	Engine
	Model string
}

func main() {
	c := Car{
		Engine: Engine{Horsepower: 300},
		Model:  "Sedan",
	}

	fmt.Println(c.Horsepower)
	c.Start()

	fmt.Println(c.Engine.Horsepower)
}
